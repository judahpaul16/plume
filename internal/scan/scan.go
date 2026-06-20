package scan

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/judahpaul16/plume/internal/graph"
	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// Stats summarizes a scan for the CLI to report.
type Stats struct {
	Files     int
	Parsed    int
	Infra     int
	Languages map[string]int
}

var ignoreDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true, "build": true,
	".next": true, "target": true, "__pycache__": true, ".venv": true, "venv": true,
	".idea": true, ".vscode": true, "obj": true, "_build": true, ".terraform": true,
	"coverage": true, ".cache": true, ".pytest_cache": true, "deps": true,
}

const maxFileSize = 1 << 20 // skip files larger than 1 MiB
const parseTimeout = 3 * time.Second

// skipExt are data/generated formats that hold no call flows; parsing them only
// costs time.
var skipExt = map[string]bool{
	".json": true, ".lock": true, ".map": true, ".csv": true, ".tsv": true,
	".svg": true, ".snap": true, ".sum": true, ".min": true, ".ipynb": true,
}
var lockfiles = map[string]bool{
	"package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true,
	"composer.lock": true, "poetry.lock": true, "gemfile.lock": true, "cargo.lock": true,
}

type infraFile struct{ rel, src string }
type codeFile struct {
	base, rel string
	src       []byte
	entry     *grammars.LangEntry
}

// langCache loads each grammar at most once; the embedded grammars are large
// blobs and re-decompressing one per file dominates the scan otherwise.
var (
	langMu    sync.Mutex
	langCache = map[string]*gts.Language{}
)

func getLang(entry *grammars.LangEntry) *gts.Language {
	langMu.Lock()
	defer langMu.Unlock()
	if l, ok := langCache[entry.Name]; ok {
		return l
	}
	l := entry.Language()
	langCache[entry.Name] = l
	return l
}

// Scan walks the given roots and builds a user-information flow graph. Files are
// parsed in parallel; the immutable parse tables are shared, graph writes are
// serialized.
func Scan(roots []string) (*graph.Graph, Stats) {
	g := graph.New()
	st := Stats{Languages: map[string]int{}}
	userID := g.AddNode(graph.Node{ID: "source:user", Kind: graph.Source, Label: "User", System: "People"})

	var files []codeFile
	var infra []infraFile
	for _, root := range roots {
		root = filepath.Clean(root)
		base := filepath.Base(root)
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if path != root && (ignoreDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
					return filepath.SkipDir
				}
				return nil
			}
			info, err := d.Info()
			if err != nil || info.Size() == 0 || info.Size() > maxFileSize {
				return nil
			}
			name := strings.ToLower(d.Name())
			if skipExt[strings.ToLower(filepath.Ext(name))] || lockfiles[name] || strings.Contains(name, ".min.") {
				return nil
			}
			st.Files++
			rel, _ := filepath.Rel(root, path)
			entry := grammars.DetectLanguage(filepath.Base(path))
			src, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if isInfra(entry, path) {
				st.Infra++
				infra = append(infra, infraFile{rel, string(src)})
				return nil
			}
			if entry != nil {
				files = append(files, codeFile{base, rel, src, entry})
			}
			return nil
		})
	}

	// warm each distinct grammar once so workers never race a first load
	seen := map[string]bool{}
	for _, f := range files {
		if !seen[f.entry.Name] {
			seen[f.entry.Name] = true
			getLang(f.entry)
		}
	}

	workers := runtime.NumCPU()
	if workers > 12 {
		workers = 12
	}
	if workers < 1 {
		workers = 1
	}
	var mu sync.Mutex
	ch := make(chan codeFile)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range ch {
				scanCode(g, &mu, f, userID, &st)
			}
		}()
	}
	for _, f := range files {
		ch <- f
	}
	close(ch)
	wg.Wait()

	for _, f := range infra {
		enrichFromInfra(g, f.src)
	}
	g.Finalize()
	return g, st
}

func scanCode(g *graph.Graph, mu *sync.Mutex, f codeFile, userID string, st *Stats) {
	lang := getLang(f.entry)
	if lang == nil {
		return
	}
	p := gts.NewParser(lang)
	cf := p.CancellationFlag()
	type parsed struct {
		tree *gts.Tree
		err  error
	}
	done := make(chan parsed, 1)
	go func() { tr, e := p.Parse(f.src); done <- parsed{tr, e} }()
	var tree *gts.Tree
	select {
	case r := <-done:
		if r.err != nil || r.tree == nil {
			return
		}
		tree = r.tree
	case <-time.After(parseTimeout):
		if cf != nil {
			atomic.StoreUint32(cf, 1) // best-effort cancel; the goroutine exits on its own
		}
		return // a pathologically slow file is skipped rather than stalling the scan
	}

	mu.Lock()
	st.Parsed++
	st.Languages[f.entry.Name]++
	mu.Unlock()

	svcID := "svc:" + f.base + "/" + f.rel

	gts.Walk(tree.RootNode(), func(n *gts.Node, depth int) gts.WalkAction {
		if !callKinds[n.Type(lang)] {
			return gts.WalkContinue
		}
		cc := n.ChildCount()
		if cc < 1 {
			return gts.WalkContinue
		}
		full := n.Text(f.src)
		argsText := n.Child(cc - 1).Text(f.src) // last child of a call is its argument list
		callee := strings.TrimSpace(strings.TrimSuffix(full, argsText))
		tgt, ok := classifyCall(callee, full)
		if !ok {
			return gts.WalkContinue
		}
		cats := categoriesIn(argsText)
		if len(cats) == 0 {
			return gts.WalkContinue
		}
		evidence := f.rel + ":" + strconv.Itoa(lineAt(f.src, n.StartByte()))
		mu.Lock()
		catIDs := make([]string, 0, len(cats))
		for _, c := range cats {
			g.AddCategory(graph.Category{ID: c.ID, Label: c.Label, Sensitivity: c.Sensitivity})
			catIDs = append(catIDs, c.ID)
		}
		g.AddNode(graph.Node{ID: svcID, Kind: graph.Service, Label: filepath.Base(f.rel), System: f.base, Location: f.base + "/" + f.rel})
		g.AddNode(graph.Node{ID: tgt.id, Kind: tgt.kind, Label: tgt.label, System: targetSystem(tgt.kind)})
		g.AddFlow(graph.Flow{From: userID, To: svcID, Categories: catIDs})
		g.AddFlow(graph.Flow{From: svcID, To: tgt.id, Categories: catIDs, Via: tgt.via, Evidence: evidence})
		mu.Unlock()
		return gts.WalkContinue
	})
}

func targetSystem(k graph.Kind) string {
	switch k {
	case graph.External:
		return "Third parties"
	case graph.Sink:
		return "Sinks"
	case graph.Store:
		return "Data stores"
	default:
		return ""
	}
}

func lineAt(src []byte, off uint32) int {
	n := 1
	if int(off) > len(src) {
		off = uint32(len(src))
	}
	for i := 0; i < int(off); i++ {
		if src[i] == '\n' {
			n++
		}
	}
	return n
}

// Package graph defines Plume's normalized user-information flow graph: the
// nodes personal data passes through, the categories of data, and the flows
// between nodes. Collectors build a Graph; renderers consume it.
package graph

import "sort"

type Sensitivity string

const (
	Public     Sensitivity = "public"
	PII        Sensitivity = "pii"
	Financial  Sensitivity = "financial"
	Health     Sensitivity = "health"
	Credential Sensitivity = "credential"
	Special    Sensitivity = "special"
)

func (s Sensitivity) rank() int {
	switch s {
	case Health, Credential, Special:
		return 4
	case Financial:
		return 3
	case PII:
		return 2
	case Public:
		return 1
	}
	return 0
}

type Kind string

const (
	Source   Kind = "source"   // where user data enters
	Service  Kind = "service"  // processes it
	Store    Kind = "store"    // data at rest
	Sink     Kind = "sink"     // logs, files, stdout
	External Kind = "external" // a third party
)

type Category struct {
	ID          string      `json:"id"`
	Label       string      `json:"label"`
	Sensitivity Sensitivity `json:"sensitivity,omitempty"`
}

type Node struct {
	ID       string `json:"id"`
	Kind     Kind   `json:"kind"`
	Label    string `json:"label"`
	System   string `json:"system,omitempty"`
	Location string `json:"location,omitempty"`
}

type Flow struct {
	From       string   `json:"from"`
	To         string   `json:"to"`
	Categories []string `json:"categories,omitempty"`
	Via        string   `json:"via,omitempty"`
	Evidence   string   `json:"evidence,omitempty"`
}

// Graph is the whole model. Build it through the Add* methods, which dedup by
// identity and merge, so collectors can emit the same node/flow from many sites.
type Graph struct {
	Categories []Category `json:"categories"`
	Nodes      []Node     `json:"nodes"`
	Flows      []Flow     `json:"flows"`

	nodeIdx map[string]int
	catIdx  map[string]int
	flowIdx map[string]int
}

func New() *Graph {
	return &Graph{nodeIdx: map[string]int{}, catIdx: map[string]int{}, flowIdx: map[string]int{}}
}

func (g *Graph) AddCategory(c Category) {
	if i, ok := g.catIdx[c.ID]; ok {
		if c.Sensitivity.rank() > g.Categories[i].Sensitivity.rank() {
			g.Categories[i].Sensitivity = c.Sensitivity
		}
		return
	}
	g.catIdx[c.ID] = len(g.Categories)
	g.Categories = append(g.Categories, c)
}

// AddNode inserts or merges a node by ID, filling blank fields, and returns the ID.
func (g *Graph) AddNode(n Node) string {
	if i, ok := g.nodeIdx[n.ID]; ok {
		ex := &g.Nodes[i]
		if ex.Label == "" {
			ex.Label = n.Label
		}
		if ex.System == "" {
			ex.System = n.System
		}
		if ex.Location == "" {
			ex.Location = n.Location
		}
		return n.ID
	}
	if n.Label == "" {
		n.Label = n.ID
	}
	g.nodeIdx[n.ID] = len(g.Nodes)
	g.Nodes = append(g.Nodes, n)
	return n.ID
}

func (g *Graph) HasNode(id string) bool { _, ok := g.nodeIdx[id]; return ok }

// Relabel overwrites an existing node's label. Infra uses it to make a generic
// code-detected store concrete, e.g. "Database" -> "PostgreSQL (Amazon RDS)".
func (g *Graph) Relabel(id, label string) {
	if i, ok := g.nodeIdx[id]; ok {
		g.Nodes[i].Label = label
	}
}

// AddFlow inserts or merges a directed flow, unioning its data categories.
func (g *Graph) AddFlow(f Flow) {
	if f.From == "" || f.To == "" || f.From == f.To {
		return
	}
	key := f.From + "\x00" + f.To
	if i, ok := g.flowIdx[key]; ok {
		ex := &g.Flows[i]
		ex.Categories = unionSorted(ex.Categories, f.Categories)
		if ex.Via == "" {
			ex.Via = f.Via
		}
		if ex.Evidence == "" {
			ex.Evidence = f.Evidence
		}
		return
	}
	f.Categories = unionSorted(nil, f.Categories)
	g.flowIdx[key] = len(g.Flows)
	g.Flows = append(g.Flows, f)
}

func unionSorted(a, b []string) []string {
	set := map[string]bool{}
	for _, x := range a {
		set[x] = true
	}
	for _, x := range b {
		set[x] = true
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for x := range set {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

// Finalize drops nodes with no flows touching them and sorts everything for
// stable, diff-friendly output.
func (g *Graph) Finalize() {
	used := map[string]bool{}
	for _, f := range g.Flows {
		used[f.From] = true
		used[f.To] = true
	}
	kept := g.Nodes[:0]
	for _, n := range g.Nodes {
		if used[n.ID] {
			kept = append(kept, n)
		}
	}
	g.Nodes = kept
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })
	sort.Slice(g.Flows, func(i, j int) bool {
		if g.Flows[i].From != g.Flows[j].From {
			return g.Flows[i].From < g.Flows[j].From
		}
		return g.Flows[i].To < g.Flows[j].To
	})
	sort.Slice(g.Categories, func(i, j int) bool { return g.Categories[i].ID < g.Categories[j].ID })
}

// Collapse merges every node of the given kind into one node, rewriting flows
// through it and dropping per-flow evidence. Blackbox mode uses it to hide code
// internals behind a single Application node.
func Collapse(src *Graph, kind Kind, id, label string) *Graph {
	isKind := map[string]bool{}
	for _, n := range src.Nodes {
		if n.Kind == kind {
			isKind[n.ID] = true
		}
	}
	mapID := func(nid string) string {
		if isKind[nid] {
			return id
		}
		return nid
	}
	g := New()
	g.AddNode(Node{ID: id, Kind: kind, Label: label})
	for _, n := range src.Nodes {
		if !isKind[n.ID] {
			g.AddNode(n)
		}
	}
	for _, c := range src.Categories {
		g.AddCategory(c)
	}
	for _, f := range src.Flows {
		g.AddFlow(Flow{From: mapID(f.From), To: mapID(f.To), Categories: f.Categories})
	}
	g.Finalize()
	return g
}

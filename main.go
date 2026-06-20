// Plume maps how user information flows through a codebase, its infrastructure,
// or a set of repos, and renders it as a readable interactive graphic.
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/judahpaul16/plume/internal/report"
	"github.com/judahpaul16/plume/internal/scan"
)

//go:embed web
var webFS embed.FS

func main() {
	out := flag.String("out", "plume.html", "output HTML file")
	noOpen := flag.Bool("no-open", false, "write the report but do not serve or open a browser")
	asJSON := flag.Bool("json", false, "print the flow graph as JSON and exit")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "usage: plume [flags] [path ...]\n\nScans the given paths (default: current dir) and opens a flow graphic.\n\nflags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}

	g, st := scan.Scan(roots)
	fmt.Fprintf(os.Stderr, "plume: %d files scanned (%d parsed, %d infra) across %d languages; %d nodes, %d flows\n",
		st.Files, st.Parsed, st.Infra, len(st.Languages), len(g.Nodes), len(g.Flows))

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(g)
		return
	}

	html, err := report.Render(webFS, g)
	if err != nil {
		fmt.Fprintln(os.Stderr, "plume:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, html, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "plume:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "plume: wrote %s (%d KB)\n", *out, len(html)/1024)
	if *noOpen {
		return
	}
	if err := report.Serve(html); err != nil {
		fmt.Fprintln(os.Stderr, "plume:", err)
		os.Exit(1)
	}
}

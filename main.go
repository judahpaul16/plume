package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/judahpaul16/plume/internal/graph"
	"github.com/judahpaul16/plume/internal/report"
	"github.com/judahpaul16/plume/internal/scan"
)

//go:embed web
var webFS embed.FS

// version is overwritten at release build time via -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

const usageText = `Plume maps how user information flows through a codebase, its infrastructure,
or a set of repos, and renders it as a readable graphic.

usage:
  plume [flags] [path ...]   scan paths (default: current dir) and open a flow graphic
  plume open <file|dir>      open a saved report, or pick from a folder of reports
  plume version              print the version
  plume help                 print this help

flags:
  --out FILE     output file; .html is interactive, .svg is a static image (default plume.html)
  --no-open      write the report but do not serve or open a browser
  --blackbox     collapse code files into one Application node and hide file paths
  --sankey       render the Sankey (flow-volume) view for .svg/.png/.jpg output
  --json         print the flow graph as JSON and exit
`

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Println("plume " + version)
			return
		case "help", "--help", "-h":
			fmt.Print(usageText)
			return
		case "open":
			openReport(os.Args[2:])
			return
		}
	}

	out := flag.String("out", "plume.html", "output file (.html or .svg)")
	noOpen := flag.Bool("no-open", false, "write the report but do not serve or open a browser")
	asJSON := flag.Bool("json", false, "print the flow graph as JSON and exit")
	blackbox := flag.Bool("blackbox", false, "collapse code files into one Application node and hide file paths")
	sankey := flag.Bool("sankey", false, "render the Sankey (flow-volume) view for image output")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usageText) }
	flag.Parse()

	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}

	g, st := scan.Scan(roots)
	fmt.Fprintf(os.Stderr, "plume: %d files scanned (%d parsed, %d infra) across %d languages; %d nodes, %d flows\n",
		st.Files, st.Parsed, st.Infra, len(st.Languages), len(g.Nodes), len(g.Flows))

	if *asJSON {
		og := g
		if *blackbox {
			og = graph.Collapse(g, graph.Service, "svc:app", "Application")
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(og)
		return
	}

	data, err := renderTo(*out, g, *sankey, *blackbox)
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fail(err)
	}
	fmt.Fprintf(os.Stderr, "plume: wrote %s (%d KB)\n", *out, len(data)/1024)
	if *noOpen {
		return
	}
	if err := report.ServeContent(data, report.ContentType(*out)); err != nil {
		fail(err)
	}
}

// renderTo renders an interactive HTML page, or a static image (.svg, .png,
// .jpg) when the output name carries an image extension. With sankey, image
// output is the flow-volume Sankey view.
func renderTo(out string, g *graph.Graph, sankey, blackbox bool) ([]byte, error) {
	lo := strings.ToLower(out)
	switch {
	case strings.HasSuffix(lo, ".svg"):
		gg := maybeCollapse(g, blackbox)
		if sankey {
			return report.RenderSankeySVG(gg), nil
		}
		return report.RenderSVG(gg), nil
	case strings.HasSuffix(lo, ".png"):
		gg := maybeCollapse(g, blackbox)
		if sankey {
			return report.RenderSankeyRaster(gg, "png")
		}
		return report.RenderRaster(gg, "png")
	case strings.HasSuffix(lo, ".jpg"), strings.HasSuffix(lo, ".jpeg"):
		gg := maybeCollapse(g, blackbox)
		if sankey {
			return report.RenderSankeyRaster(gg, "jpg")
		}
		return report.RenderRaster(gg, "jpg")
	default:
		return report.Render(webFS, g, blackbox)
	}
}

func maybeCollapse(g *graph.Graph, blackbox bool) *graph.Graph {
	if blackbox {
		return graph.Collapse(g, graph.Service, "svc:app", "Application")
	}
	return g
}

// openReport serves and opens a saved report, or a picker gallery when given a
// directory of reports.
func openReport(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: plume open <file|dir>")
		os.Exit(1)
	}
	info, err := os.Stat(args[0])
	if err != nil {
		fail(err)
	}
	if info.IsDir() {
		if err := report.ServeDir(args[0]); err != nil {
			fail(err)
		}
		return
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		fail(err)
	}
	if err := report.ServeContent(data, report.ContentType(args[0])); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "plume:", err)
	os.Exit(1)
}

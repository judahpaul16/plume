package report

import (
	"fmt"
	"math"

	"github.com/judahpaul16/plume/internal/graph"
)

var kindStroke = map[graph.Kind]string{
	graph.Source:   "#34d399",
	graph.Service:  "#60a5fa",
	graph.Store:    "#f59e0b",
	graph.Sink:     "#9ca3af",
	graph.External: "#f87171",
}

var sevColor = map[graph.Sensitivity]string{
	graph.PII:        "#f59e0b",
	graph.Financial:  "#ef4444",
	graph.Health:     "#a855f7",
	graph.Credential: "#ec4899",
	graph.Special:    "#a855f7",
	graph.Public:     "#64748b",
}

var sevLabel = map[graph.Sensitivity]string{
	graph.PII: "PII", graph.Financial: "Financial", graph.Health: "Health",
	graph.Credential: "Credential", graph.Special: "Special", graph.Public: "Public",
}

func sevRank(s graph.Sensitivity) int {
	switch s {
	case graph.Health, graph.Credential, graph.Special:
		return 4
	case graph.Financial:
		return 3
	case graph.PII:
		return 2
	case graph.Public:
		return 1
	}
	return 0
}

const (
	svgPad     = 40.0
	nodeW      = 210.0
	nodeH      = 48.0
	colGap     = 130.0
	rowGap     = 26.0
	contentTop = 64.0
	legendH    = 36.0
)

type placedNode struct {
	X, Y                float64
	Label, Kind, Stroke string
}

type placedEdge struct {
	X1, Y1, C1x, C2x, X2, Y2 float64
	Color                    string
}

type legendItem struct{ Color, Label string }

type diagram struct {
	W, H     float64
	Subtitle string
	Nodes    []placedNode
	Edges    []placedEdge
	Legend   []legendItem
}

// layout places the graph in kind-ordered columns (source to service to
// store/sink/external) and computes edge curves. Both the SVG and raster
// renderers consume the result so the two outputs always agree.
func layout(g *graph.Graph) diagram {
	order := []graph.Kind{graph.Source, graph.Service, graph.Store, graph.Sink, graph.External}
	byKind := map[graph.Kind][]graph.Node{}
	for _, n := range g.Nodes {
		byKind[n.Kind] = append(byKind[n.Kind], n)
	}
	var cols []graph.Kind
	maxRows := 0
	for _, k := range order {
		if len(byKind[k]) > 0 {
			cols = append(cols, k)
			if len(byKind[k]) > maxRows {
				maxRows = len(byKind[k])
			}
		}
	}

	type pos struct{ x, y float64 }
	topLeft := map[string]pos{}
	colStride := nodeW + colGap
	rowStride := nodeH + rowGap

	var d diagram
	for ci, k := range cols {
		ns := byKind[k]
		x := svgPad + float64(ci)*colStride
		colTop := contentTop + float64(maxRows-len(ns))*rowStride/2
		for ri, n := range ns {
			y := colTop + float64(ri)*rowStride
			topLeft[n.ID] = pos{x, y}
			stroke := kindStroke[n.Kind]
			if stroke == "" {
				stroke = "#1f2a3a"
			}
			d.Nodes = append(d.Nodes, placedNode{
				X: x, Y: y, Stroke: stroke, Kind: string(n.Kind),
				Label: truncate(n.Label, int(math.Floor((nodeW-20)/7))),
			})
		}
	}
	d.W = svgPad*2 + float64(len(cols))*nodeW + float64(maxi(len(cols)-1, 0))*colGap
	d.H = contentTop + float64(maxi(maxRows, 1))*rowStride - rowGap + svgPad + legendH

	sevOf := map[string]graph.Sensitivity{}
	for _, c := range g.Categories {
		sevOf[c.ID] = c.Sensitivity
	}
	for _, f := range g.Flows {
		a, ok1 := topLeft[f.From]
		t, ok2 := topLeft[f.To]
		if !ok1 || !ok2 {
			continue
		}
		dx := math.Max(40, math.Abs(t.x-a.x)*0.35)
		e := placedEdge{Y1: a.y + nodeH/2, Y2: t.y + nodeH/2, Color: edgeColor(f, sevOf)}
		if t.x < a.x {
			e.X1, e.C1x = a.x, a.x-dx
			e.X2, e.C2x = t.x+nodeW, t.x+nodeW+dx
		} else {
			e.X1, e.C1x = a.x+nodeW, a.x+nodeW+dx
			e.X2, e.C2x = t.x, t.x-dx
		}
		d.Edges = append(d.Edges, e)
	}

	d.Subtitle = fmt.Sprintf("user information flow · %d nodes, %d flows", len(g.Nodes), len(g.Flows))

	seen := map[graph.Sensitivity]bool{}
	for _, c := range g.Categories {
		if c.Sensitivity != "" && !seen[c.Sensitivity] {
			seen[c.Sensitivity] = true
			d.Legend = append(d.Legend, legendItem{Color: sevColor[c.Sensitivity], Label: sevLabel[c.Sensitivity]})
		}
	}
	return d
}

func edgeColor(f graph.Flow, sevOf map[string]graph.Sensitivity) string {
	best := graph.Sensitivity("")
	for _, c := range f.Categories {
		if s, ok := sevOf[c]; ok && sevRank(s) > sevRank(best) {
			best = s
		}
	}
	if col, ok := sevColor[best]; ok {
		return col
	}
	return "#33415a"
}

func truncate(s string, max int) string {
	r := []rune(s)
	if max > 1 && len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

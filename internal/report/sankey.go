package report

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"image/png"
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/judahpaul16/plume/internal/graph"
)

const (
	skBarW       = 16.0
	skColGap     = 230.0
	skNodeGap    = 14.0
	skTargetColH = 480.0
	skTop        = 64.0
	skLabelW     = 160.0
)

type skNode struct {
	X, Y, H   float64
	Label     string
	Fill      string
	LabelLeft bool
}

type skRibbon struct {
	X1, Y1, X2, Y2, W float64
	Color             string
}

type skDiagram struct {
	W, H     float64
	Subtitle string
	Nodes    []skNode
	Ribbons  []skRibbon
	Legend   []legendItem
}

// sankeyLayout sizes each node bar by its throughput (max of in/out flow weight)
// and each ribbon by its flow weight, in the same kind-ordered columns as the
// graph view. Both Sankey renderers consume it.
func sankeyLayout(g *graph.Graph) skDiagram {
	order := []graph.Kind{graph.Source, graph.Service, graph.Store, graph.Sink, graph.External}
	byKind := map[graph.Kind][]graph.Node{}
	for _, n := range g.Nodes {
		byKind[n.Kind] = append(byKind[n.Kind], n)
	}
	var cols []graph.Kind
	for _, k := range order {
		if len(byKind[k]) > 0 {
			cols = append(cols, k)
		}
	}

	weight := func(f graph.Flow) float64 {
		if w := float64(len(f.Categories)); w > 1 {
			return w
		}
		return 1
	}
	inW, outW := map[string]float64{}, map[string]float64{}
	for _, f := range g.Flows {
		w := weight(f)
		outW[f.From] += w
		inW[f.To] += w
	}
	thru := map[string]float64{}
	for _, n := range g.Nodes {
		t := inW[n.ID]
		if outW[n.ID] > t {
			t = outW[n.ID]
		}
		if t < 1 {
			t = 1
		}
		thru[n.ID] = t
	}

	maxCol, maxNodes := 1.0, 0
	for _, k := range cols {
		s := 0.0
		for _, n := range byKind[k] {
			s += thru[n.ID]
		}
		if s > maxCol {
			maxCol = s
		}
		if len(byKind[k]) > maxNodes {
			maxNodes = len(byKind[k])
		}
	}
	unit := (skTargetColH - float64(maxNodes-1)*skNodeGap) / maxCol
	if unit < 4 {
		unit = 4
	}

	colHeights := make([]float64, len(cols))
	maxColH := skTargetColH
	for ci, k := range cols {
		total := float64(len(byKind[k])-1) * skNodeGap
		for _, n := range byKind[k] {
			total += thru[n.ID] * unit
		}
		colHeights[ci] = total
		if total > maxColH {
			maxColH = total
		}
	}

	colStride := skBarW + skColGap
	posByID := map[string]skNode{}
	var d skDiagram
	for ci, k := range cols {
		x := svgPad + float64(ci)*colStride
		y := skTop + (maxColH-colHeights[ci])/2
		for _, n := range byKind[k] {
			h := thru[n.ID] * unit
			fill := kindStroke[n.Kind]
			if fill == "" {
				fill = "#33415a"
			}
			sn := skNode{X: x, Y: y, H: h, Label: truncate(n.Label, 22), Fill: fill, LabelLeft: ci == len(cols)-1}
			posByID[n.ID] = sn
			d.Nodes = append(d.Nodes, sn)
			y += h + skNodeGap
		}
	}
	d.W = svgPad*2 + float64(len(cols)-1)*colStride + skBarW + skLabelW
	d.H = skTop + maxColH + svgPad + legendH

	sevOf := map[string]graph.Sensitivity{}
	for _, c := range g.Categories {
		sevOf[c.ID] = c.Sensitivity
	}
	outCur, inCur := map[string]float64{}, map[string]float64{}
	for _, f := range g.Flows {
		s, ok1 := posByID[f.From]
		t, ok2 := posByID[f.To]
		if !ok1 || !ok2 {
			continue
		}
		w := weight(f) * unit
		var x1, x2 float64
		if t.X >= s.X {
			x1, x2 = s.X+skBarW, t.X
		} else {
			x1, x2 = s.X, t.X+skBarW
		}
		d.Ribbons = append(d.Ribbons, skRibbon{
			X1: x1, Y1: s.Y + outCur[f.From], X2: x2, Y2: t.Y + inCur[f.To], W: w, Color: edgeColor(f, sevOf),
		})
		outCur[f.From] += w
		inCur[f.To] += w
	}

	d.Subtitle = fmt.Sprintf("flow volume · %d nodes, %d flows", len(g.Nodes), len(g.Flows))
	seen := map[graph.Sensitivity]bool{}
	for _, c := range g.Categories {
		if c.Sensitivity != "" && !seen[c.Sensitivity] {
			seen[c.Sensitivity] = true
			d.Legend = append(d.Legend, legendItem{Color: sevColor[c.Sensitivity], Label: sevLabel[c.Sensitivity]})
		}
	}
	return d
}

// RenderSankeySVG draws the flow graph as a static Sankey diagram.
func RenderSankeySVG(g *graph.Graph) []byte {
	d := sankeyLayout(g)
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f" font-family="ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif">`, d.W, d.H, d.W, d.H)
	fmt.Fprintf(&b, `<rect width="%.0f" height="%.0f" fill="#0b0f17"/>`, d.W, d.H)
	fmt.Fprintf(&b, `<text x="%.0f" y="34" fill="#e5edf7" font-size="18" font-weight="700">Plume</text>`, svgPad)
	fmt.Fprintf(&b, `<text x="%.0f" y="34" fill="#8aa0b8" font-size="12">%s</text>`, svgPad+66, esc(d.Subtitle))

	for _, r := range d.Ribbons {
		mx := (r.X1 + r.X2) / 2
		fmt.Fprintf(&b, `<path d="M%.1f %.1f C%.1f %.1f %.1f %.1f %.1f %.1f L%.1f %.1f C%.1f %.1f %.1f %.1f %.1f %.1f Z" fill="%s" fill-opacity="0.45"/>`,
			r.X1, r.Y1, mx, r.Y1, mx, r.Y2, r.X2, r.Y2,
			r.X2, r.Y2+r.W, mx, r.Y2+r.W, mx, r.Y1+r.W, r.X1, r.Y1+r.W, r.Color)
	}
	for _, n := range d.Nodes {
		fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.0f" height="%.1f" rx="3" fill="%s"/>`, n.X, n.Y, skBarW, n.H, n.Fill)
		if n.LabelLeft {
			fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" fill="#e5edf7" font-size="11.5" text-anchor="end">%s</text>`, n.X-6, n.Y+n.H/2+4, esc(n.Label))
		} else {
			fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" fill="#e5edf7" font-size="11.5">%s</text>`, n.X+skBarW+6, n.Y+n.H/2+4, esc(n.Label))
		}
	}
	x, y := svgPad, d.H-14
	for _, li := range d.Legend {
		fmt.Fprintf(&b, `<rect x="%.0f" y="%.0f" width="11" height="11" rx="2" fill="%s"/>`, x, y-9, li.Color)
		fmt.Fprintf(&b, `<text x="%.0f" y="%.0f" fill="#8aa0b8" font-size="11">%s</text>`, x+16, y, esc(li.Label))
		x += 24 + float64(len(li.Label))*7
	}
	b.WriteString(`</svg>`)
	return []byte(b.String())
}

// RenderSankeyRaster draws the flow graph as a static Sankey PNG or JPEG.
func RenderSankeyRaster(g *graph.Graph, format string) ([]byte, error) {
	d := sankeyLayout(g)
	dc := gg.NewContext(int(d.W*rasterScale), int(d.H*rasterScale))
	dc.SetHexColor("#0b0f17")
	dc.Clear()
	dc.Scale(rasterScale, rasterScale)

	reg, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	bold, err := truetype.Parse(gobold.TTF)
	if err != nil {
		return nil, err
	}

	dc.SetFontFace(truetype.NewFace(bold, &truetype.Options{Size: 18}))
	dc.SetHexColor("#e5edf7")
	dc.DrawString("Plume", svgPad, 34)
	dc.SetFontFace(truetype.NewFace(reg, &truetype.Options{Size: 12}))
	dc.SetHexColor("#8aa0b8")
	dc.DrawString(d.Subtitle, svgPad+66, 34)

	for _, r := range d.Ribbons {
		mx := (r.X1 + r.X2) / 2
		dc.MoveTo(r.X1, r.Y1)
		dc.CubicTo(mx, r.Y1, mx, r.Y2, r.X2, r.Y2)
		dc.LineTo(r.X2, r.Y2+r.W)
		dc.CubicTo(mx, r.Y2+r.W, mx, r.Y1+r.W, r.X1, r.Y1+r.W)
		dc.ClosePath()
		dc.SetHexColor(r.Color + "73")
		dc.Fill()
	}

	label := truetype.NewFace(reg, &truetype.Options{Size: 11.5})
	for _, n := range d.Nodes {
		dc.DrawRoundedRectangle(n.X, n.Y, skBarW, n.H, 3)
		dc.SetHexColor(n.Fill)
		dc.Fill()
		dc.SetFontFace(label)
		dc.SetHexColor("#e5edf7")
		if n.LabelLeft {
			dc.DrawStringAnchored(n.Label, n.X-6, n.Y+n.H/2, 1, 0.4)
		} else {
			dc.DrawStringAnchored(n.Label, n.X+skBarW+6, n.Y+n.H/2, 0, 0.4)
		}
	}

	dc.SetFontFace(truetype.NewFace(reg, &truetype.Options{Size: 11}))
	x, y := svgPad, d.H-14
	for _, li := range d.Legend {
		dc.DrawRoundedRectangle(x, y-9, 11, 11, 2)
		dc.SetHexColor(li.Color)
		dc.Fill()
		dc.SetHexColor("#8aa0b8")
		dc.DrawString(li.Label, x+16, y)
		x += 24 + float64(len(li.Label))*7
	}

	var buf bytes.Buffer
	img := dc.Image()
	switch format {
	case "jpg", "jpeg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92})
	default:
		err = png.Encode(&buf, img)
	}
	return buf.Bytes(), err
}

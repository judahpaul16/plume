package report

import (
	"fmt"
	"strings"

	"github.com/judahpaul16/plume/internal/graph"
)

// RenderSVG draws the flow graph as a single static SVG image. Edges are colored
// by the highest sensitivity they carry; node detail lives in the HTML report.
func RenderSVG(g *graph.Graph) []byte {
	d := layout(g)
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f" font-family="ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif">`, d.W, d.H, d.W, d.H)
	fmt.Fprintf(&b, `<rect width="%.0f" height="%.0f" fill="#0b0f17"/>`, d.W, d.H)
	fmt.Fprintf(&b, `<text x="%.0f" y="34" fill="#e5edf7" font-size="18" font-weight="700">Plume</text>`, svgPad)
	fmt.Fprintf(&b, `<text x="%.0f" y="34" fill="#8aa0b8" font-size="12">%s</text>`, svgPad+66, esc(d.Subtitle))

	for _, e := range d.Edges {
		fmt.Fprintf(&b, `<path d="M%.1f %.1f C%.1f %.1f %.1f %.1f %.1f %.1f" fill="none" stroke="%s" stroke-width="1.6" opacity="0.8"/>`,
			e.X1, e.Y1, e.C1x, e.Y1, e.C2x, e.Y2, e.X2, e.Y2, e.Color)
	}
	for _, e := range d.Edges {
		if e.Label == "" {
			continue
		}
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" fill="#8aa0b8" font-size="10" text-anchor="middle">%s</text>`,
			(e.X1+e.X2)/2, (e.Y1+e.Y2)/2-5, esc(e.Label))
	}

	for _, n := range d.Nodes {
		fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.0f" height="%.0f" rx="9" fill="#111826" stroke="%s" stroke-width="1.5"/>`,
			n.X, n.Y, nodeW, nodeH, n.Stroke)
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" fill="#e5edf7" font-size="12.5" text-anchor="middle">%s</text>`,
			n.X+nodeW/2, n.Y+nodeH/2-1, esc(n.Label))
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" fill="%s" font-size="9.5" text-anchor="middle">%s</text>`,
			n.X+nodeW/2, n.Y+nodeH/2+13, n.Stroke, esc(n.Kind))
	}

	x := svgPad
	y := d.H - 14
	for _, li := range d.Legend {
		fmt.Fprintf(&b, `<rect x="%.0f" y="%.0f" width="11" height="11" rx="2" fill="%s"/>`, x, y-9, li.Color)
		fmt.Fprintf(&b, `<text x="%.0f" y="%.0f" fill="#8aa0b8" font-size="11">%s</text>`, x+16, y, esc(li.Label))
		x += 24 + float64(len(li.Label))*7
	}

	b.WriteString(`</svg>`)
	return []byte(b.String())
}

var xmlesc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")

func esc(s string) string { return xmlesc.Replace(s) }

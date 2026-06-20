package report

import (
	"bytes"
	"image/jpeg"
	"image/png"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/judahpaul16/plume/internal/graph"
)

const rasterScale = 2.0

// RenderRaster draws the flow graph as a PNG or JPEG (format "png", "jpg", or
// "jpeg"). It rasterizes with pure-Go drawing and embedded fonts, so it needs no
// browser or external toolchain.
func RenderRaster(g *graph.Graph, format string) ([]byte, error) {
	d := layout(g)
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
	face := func(f *truetype.Font, size float64) font.Face {
		return truetype.NewFace(f, &truetype.Options{Size: size})
	}
	title, label, tag, small, legend, elabel := face(bold, 18), face(reg, 12.5), face(reg, 9.5), face(reg, 12), face(reg, 11), face(reg, 10)

	dc.SetFontFace(title)
	dc.SetHexColor("#e5edf7")
	dc.DrawString("Plume", svgPad, 34)
	dc.SetFontFace(small)
	dc.SetHexColor("#8aa0b8")
	dc.DrawString(d.Subtitle, svgPad+66, 34)

	for _, e := range d.Edges {
		dc.MoveTo(e.X1, e.Y1)
		dc.CubicTo(e.C1x, e.Y1, e.C2x, e.Y2, e.X2, e.Y2)
		dc.SetHexColor(e.Color + "cc")
		dc.SetLineWidth(1.6)
		dc.Stroke()
	}
	dc.SetFontFace(elabel)
	dc.SetHexColor("#8aa0b8")
	for _, e := range d.Edges {
		if e.Label == "" {
			continue
		}
		dc.DrawStringAnchored(e.Label, (e.X1+e.X2)/2, (e.Y1+e.Y2)/2-5, 0.5, 0.5)
	}

	for _, n := range d.Nodes {
		dc.DrawRoundedRectangle(n.X, n.Y, nodeW, nodeH, 9)
		dc.SetHexColor("#111826")
		dc.FillPreserve()
		dc.SetHexColor(n.Stroke)
		dc.SetLineWidth(1.5)
		dc.Stroke()
		dc.SetFontFace(label)
		dc.SetHexColor("#e5edf7")
		dc.DrawStringAnchored(n.Label, n.X+nodeW/2, n.Y+nodeH/2-1, 0.5, 0.5)
		dc.SetFontFace(tag)
		dc.SetHexColor(n.Stroke)
		dc.DrawStringAnchored(n.Kind, n.X+nodeW/2, n.Y+nodeH/2+13, 0.5, 0.5)
	}

	dc.SetFontFace(legend)
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

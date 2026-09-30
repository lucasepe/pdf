package main

import (
	"image/color"
	"log"
	"math"

	pdf "github.com/lucasepe/pdf"
)

func main() {
	doc := pdf.NewPDF("A4")
	c := pdf.NewContext(&doc)

	c.SetFillColor(color.RGBA{R: 245, G: 247, B: 250, A: 255}).
		DrawRectangle(0, 0, doc.PageWidth(), doc.PageHeight()).Fill()

	c.SetFillColor(color.RGBA{R: 47, G: 128, B: 237, A: 255}).
		DrawRoundedRectangle(45, 55, 220, 105, 18).Fill()
	c.SetStrokeColor(color.RGBA{R: 18, G: 52, B: 86, A: 255}).
		SetLineWidth(4).SetLineJoin(pdf.JoinRound).
		DrawRoundedRectangle(45, 55, 220, 105, 18).Stroke()

	c.SetFillColor(color.RGBA{R: 242, G: 153, B: 74, A: 255}).
		DrawCircle(370, 105, 55).FillPreserve().
		SetStrokeColor(color.RGBA{R: 111, G: 78, B: 55, A: 255}).Stroke()

	c.SetStrokeColor(color.RGBA{R: 155, G: 81, B: 224, A: 255}).
		SetLineWidth(5).SetLineCap(pdf.CapRound).SetDash(0, 14, 8).
		MoveTo(55, 225).CubicTo(155, 145, 250, 310, 360, 220).
		QuadraticTo(430, 165, 525, 235).Stroke()

	// A clipped and rotated stripe panel.
	c.Push().DrawRoundedRectangle(55, 300, 480, 190, 24).Clip()
	c.SetFillColor(color.RGBA{R: 224, G: 242, B: 241, A: 255}).
		DrawRectangle(55, 300, 480, 190).Fill()
	c.RotateAbout(-math.Pi/10, 295, 395)
	colors := []color.RGBA{
		{R: 39, G: 174, B: 96, A: 255},
		{R: 86, G: 204, B: 242, A: 255},
		{R: 235, G: 87, B: 87, A: 255},
	}
	for i := -3; i < 8; i++ {
		c.SetFillColor(colors[(i+6)%len(colors)]).
			DrawRectangle(float64(i)*75, 330, 42, 220).Fill()
	}
	c.Pop()

	c.SetStrokeColor(color.RGBA{R: 45, G: 52, B: 54, A: 255}).
		SetLineWidth(7).SetDash(0).
		DrawArc(295, 650, 105, -math.Pi*.85, math.Pi*.55).Stroke()

	if err := c.SaveFile("_examples/basic_shapes/output.pdf"); err != nil {
		log.Fatal(err)
	}
}

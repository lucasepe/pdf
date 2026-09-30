package main

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"log"
	"path/filepath"

	pdf "github.com/lucasepe/pdf"
)

//go:embed testdata/*.txt
var fixtures embed.FS

type pointingEmojiProvider struct{ image image.Image }

func (p pointingEmojiProvider) LookupEmoji(sequence string) (pdf.EmojiAsset, bool, error) {
	if sequence != "☝️" {
		return pdf.EmojiAsset{}, false, nil
	}
	return pdf.EmojiAsset{Image: p.image, HeightEm: 1.08, BaselineEm: -.12, AdvanceEm: 1}, true, nil
}

func main() {
	spec := pdf.PageSpec{
		Width: pdf.Inches(6), Height: pdf.Inches(9),
		MarginTop: 18, MarginRight: 18, MarginBottom: 18, MarginLeft: 18,
		BindingOffset: 6, HeaderHeight: 24, FooterHeight: 24,
	}
	doc := pdf.NewPDF(spec.PaperSize())
	if err := doc.SetEmojiProvider(pointingEmojiProvider{image: pointingHand(96)}); err != nil {
		log.Fatal(err)
	}
	c := pdf.NewContext(&doc)
	flow, err := pdf.NewFlow(c, spec, decoratePage)
	if err != nil {
		log.Fatal(err)
	}
	names, err := fs.Glob(fixtures, "testdata/*.txt")
	if err != nil {
		log.Fatal(err)
	}
	for i, name := range names {
		if i > 0 {
			flow.NewPage()
		}
		source, err := fixtures.ReadFile(name)
		if err != nil {
			log.Fatal(err)
		}
		flow.DrawASCIIDiagram(string(source), pdf.ASCIIDiagramOptions{
			StrokeColor: color.RGBA{R: 44, G: 57, B: 72, A: 255},
			TextColor:   color.RGBA{R: 31, G: 78, B: 121, A: 255},
		})
	}
	if err := flow.Error(); err != nil {
		log.Fatal(err)
	}
	if err := c.SaveFile(filepath.Join("_examples", "ascii_diagrams", "output.pdf")); err != nil {
		log.Fatal(err)
	}
}

func decoratePage(c *pdf.Context, page pdf.PageInfo) error {
	background := color.RGBA{R: 252, G: 250, B: 246, A: 255}
	muted := color.RGBA{R: 91, G: 102, B: 116, A: 255}
	c.SetFillColor(background).DrawRectangle(0, 0, page.Spec.Width, page.Spec.Height).Fill()
	c.SetStrokeColor(color.RGBA{R: 211, G: 216, B: 221, A: 255}).SetLineWidth(.45).
		DrawRectangle(page.Body.X, page.Body.Y, page.Body.Width, page.Body.Height).Stroke()
	c.SetFillColor(muted).UseFontRole(pdf.FontRoleMonoBold, 8).
		DrawString(fmt.Sprintf("ASCII DIAGRAM FIXTURE %02d", page.Number), page.Body.X, page.Spec.MarginTop+10).
		UseFontRole(pdf.FontRoleMono, 7).
		DrawStringAnchored(fmt.Sprintf("— %d / 9 —", page.Number), page.Spec.Width/2, page.Spec.Height-page.Spec.MarginBottom-5, .5, 0)
	return c.Error()
}

func pointingHand(size int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	skin := color.NRGBA{R: 244, G: 184, B: 116, A: 255}
	outline := color.NRGBA{R: 127, G: 82, B: 48, A: 255}
	// A deliberately small, dependency-free stand-in for the fixture emoji.
	for y := size / 16; y < size*11/16; y++ {
		for x := size * 7 / 16; x < size*10/16; x++ {
			img.SetNRGBA(x, y, skin)
		}
	}
	for y := size * 9 / 16; y < size*15/16; y++ {
		for x := size * 3 / 16; x < size*13/16; x++ {
			dx, dy := x-size/2, y-size*11/16
			if dx*dx+dy*dy < size*size/7 {
				img.SetNRGBA(x, y, skin)
			}
		}
	}
	for y := size / 16; y < size*15/16; y++ {
		for x := size * 3 / 16; x < size*13/16; x++ {
			if img.NRGBAAt(x, y).A != 0 {
				for _, p := range [][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}} {
					if p[0] >= 0 && p[1] >= 0 && p[0] < size && p[1] < size && img.NRGBAAt(p[0], p[1]).A == 0 {
						img.SetNRGBA(p[0], p[1], outline)
					}
				}
			}
		}
	}
	return img
}

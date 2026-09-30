package main

import (
	"image"
	"image/color"
	"log"

	pdf "github.com/lucasepe/pdf"
)

type emojiProvider struct {
	target  image.Image
	warning image.Image
}

func (p emojiProvider) LookupEmoji(sequence string) (pdf.EmojiAsset, bool, error) {
	switch sequence {
	case "🎯":
		return pdf.EmojiAsset{Image: p.target, HeightEm: 1.15, BaselineEm: -.18, AdvanceEm: 1.22}, true, nil
	case "⚠":
		return pdf.EmojiAsset{Image: p.warning, HeightEm: 1.15, BaselineEm: -.18, AdvanceEm: 1.22}, true, nil
	default:
		return pdf.EmojiAsset{}, false, nil
	}
}

func main() {
	doc := pdf.NewPDF("A4")
	target := targetImage(96)
	warning := warningImage(96)
	if err := doc.SetEmojiProvider(emojiProvider{target: target, warning: warning}); err != nil {
		log.Fatal(err)
	}
	c := pdf.NewContext(&doc)
	c.SetFillColor(color.RGBA{R: 248, G: 249, B: 251, A: 255}).DrawRectangle(0, 0, doc.PageWidth(), doc.PageHeight()).Fill()
	c.SetFillColor(color.RGBA{R: 35, G: 45, B: 58, A: 255}).UseFontRole(pdf.FontRoleMonoBold, 18).DrawString("image.Image + soft mask", 45, 65)
	c.DrawImageScaledAnchored(target, 155, 210, 180, 180, .5, .5)
	c.Push().RotateAbout(.25, 420, 210).DrawImageScaledAnchored(warning, 420, 210, 145, 145, .5, .5).Pop()
	c.SetFillColor(color.RGBA{R: 35, G: 45, B: 58, A: 255}).UseFontRole(pdf.FontRoleMono, 14).
		DrawString("Emoji inline:", 55, 380).DrawEmojiAt("🎯", 175, 380, 22).
		DrawString("target acquisito", 205, 380).
		DrawString("Attenzione", 55, 425).DrawEmojiAt("⚠", 145, 425, 22).
		DrawString("provider PNG/raster", 175, 425)
	// The same pixels are reused: only one resource per unique image is embedded.
	for i := 0; i < 6; i++ {
		c.DrawImageScaledAnchored(target, 90+float64(i)*78, 550, 52, 52, .5, .5)
	}
	if err := c.SaveFile("_examples/images_and_emoji/output.pdf"); err != nil {
		log.Fatal(err)
	}
}

func targetImage(size int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	cx := float64(size-1) / 2
	bands := []struct {
		radius float64
		c      color.NRGBA
	}{{.48, color.NRGBA{R: 231, G: 76, B: 60, A: 235}}, {.34, color.NRGBA{R: 250, G: 250, B: 250, A: 245}}, {.21, color.NRGBA{R: 231, G: 76, B: 60, A: 255}}, {.08, color.NRGBA{R: 250, G: 220, B: 70, A: 255}}}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := float64(x)-cx, float64(y)-cx
			d := dx*dx + dy*dy
			for _, b := range bands {
				r := b.radius * float64(size)
				if d <= r*r {
					img.SetNRGBA(x, y, b.c)
				}
			}
		}
	}
	return img
}
func warningImage(size int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			nx, ny := float64(x)/float64(size-1), float64(y)/float64(size-1)
			left, right := .5-ny*.46, .5+ny*.46
			if ny > .06 && ny < .94 && nx >= left && nx <= right {
				img.SetNRGBA(x, y, color.NRGBA{R: 242, G: 201, B: 76, A: 245})
			}
		}
	}
	for y := 30; y < 65; y++ {
		for x := 45; x < 51; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 55, G: 55, B: 55, A: 255})
		}
	}
	return img
}

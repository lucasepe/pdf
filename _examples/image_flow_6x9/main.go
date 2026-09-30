package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"os"

	pdf "github.com/lucasepe/pdf"
)

func main() {
	spec := pdf.PageSpec{
		Width: pdf.Inches(6), Height: pdf.Inches(9),
		MarginTop: 18, MarginRight: 18, MarginBottom: 18, MarginLeft: 18,
		BindingOffset: 6, HeaderHeight: 24, FooterHeight: 24,
	}
	doc := pdf.NewPDF(spec.PaperSize())
	fontData, err := os.ReadFile("testdata/fonts/Vera.ttf")
	if err != nil {
		log.Fatal(err)
	}
	if err := doc.RegisterFont("Body", fontData); err != nil {
		log.Fatal(err)
	}
	for _, role := range []pdf.FontRole{pdf.FontRoleBody, pdf.FontRoleBodyBold, pdf.FontRoleBodyItalic} {
		if err := doc.BindFontRole(role, "Body"); err != nil {
			log.Fatal(err)
		}
	}

	photo, err := doc.RegisterImage(makeJPEG())
	if err != nil {
		log.Fatal(err)
	}
	cards := make([]*pdf.ImageResource, 0, 3)
	for _, accent := range []color.RGBA{
		{R: 38, G: 101, B: 160, A: 255},
		{R: 224, G: 104, B: 74, A: 255},
		{R: 63, G: 148, B: 112, A: 255},
	} {
		resource, err := doc.RegisterImage(makeTransparentCard(accent))
		if err != nil {
			log.Fatal(err)
		}
		cards = append(cards, resource)
	}

	c := pdf.NewContext(&doc)
	flow, err := pdf.NewFlow(c, spec, decoratePage)
	if err != nil {
		log.Fatal(err)
	}
	dark := color.RGBA{R: 39, G: 49, B: 63, A: 255}
	blue := color.RGBA{R: 38, G: 101, B: 160, A: 255}
	title := pdf.TextStyle{FontRole: pdf.FontRoleBodyBold, FontSize: 20, Color: blue}
	body := pdf.TextStyle{FontRole: pdf.FontRoleBody, FontSize: 9.5, Color: dark}
	caption := pdf.TextStyle{FontRole: pdf.FontRoleBodyItalic, FontSize: 8.5, Color: color.RGBA{R: 91, G: 102, B: 116, A: 255}}

	flow.DrawParagraph([]pdf.TextRun{{Text: "Immagini efficienti e paginabili", Style: title}}, pdf.ParagraphStyle{LineHeight: 27, SpaceAfter: 5})
	flow.DrawParagraph([]pdf.TextRun{{
		Text:  fmt.Sprintf("La fotografia sintetica è registrata una volta come %s, %d×%d px, e riutilizzata senza ricodifica.", photo.Format(), photo.PixelWidth(), photo.PixelHeight()),
		Style: body,
	}}, pdf.ParagraphStyle{LineHeight: 13, SpaceAfter: 7})

	drawScaleExample(flow, photo, .50, "scale=0.50 — larghezza 0.39 del body", caption)
	drawScaleExample(flow, photo, 1.00, "scale=1.00 — larghezza predefinita 0.78", caption)
	drawScaleExample(flow, photo, 2.00, "scale=2.00 — clamp automatico a 0.98", caption)

	flow.DrawParagraph([]pdf.TextRun{{Text: "Gruppo flow compatto", Style: title}}, pdf.ParagraphStyle{LineHeight: 27, SpaceBefore: 8, SpaceAfter: 4})
	flow.DrawParagraph([]pdf.TextRun{{Text: "Tre PNG trasparenti condividono la stessa riga con gap pari a 0.02 del body.", Style: body}}, pdf.ParagraphStyle{LineHeight: 13, SpaceAfter: 6})
	flow.DrawImageGroup([]pdf.ImageItem{
		{Image: cards[0], Scale: .28},
		{Image: cards[1], Scale: .28},
		{Image: cards[2], Scale: .22},
	}, pdf.ImageGroupOptions{SpaceAfter: 10})

	flow.DrawParagraph([]pdf.TextRun{{Text: "Wrapping e page break delle righe", Style: title}}, pdf.ParagraphStyle{LineHeight: 27, SpaceAfter: 4})
	flow.DrawParagraph([]pdf.TextRun{{Text: "Con scale=0.80 le immagini non entrano affiancate: ogni riga resta atomica e passa interamente alla pagina seguente quando necessario.", Style: body}}, pdf.ParagraphStyle{LineHeight: 13, SpaceAfter: 6})
	flow.DrawImageGroup([]pdf.ImageItem{
		{Image: cards[0], Scale: .80},
		{Image: cards[1], Scale: .80},
	}, pdf.ImageGroupOptions{SpaceAfter: 9})
	flow.DrawParagraph([]pdf.TextRun{{Text: "Fine del flusso immagini.", Style: caption}}, pdf.ParagraphStyle{LineHeight: 12})

	if err := flow.Error(); err != nil {
		log.Fatal(err)
	}
	if err := c.SaveFile("_examples/image_flow_6x9/output.pdf"); err != nil {
		log.Fatal(err)
	}
}

func drawScaleExample(flow *pdf.Flow, image *pdf.ImageResource, scale float64, label string, style pdf.TextStyle) {
	fraction := .78 * scale
	if fraction > .98 {
		fraction = .98
	}
	width := flow.Body().Width * fraction
	height := width * float64(image.PixelHeight()) / float64(image.PixelWidth())
	flow.EnsureSpace(12 + 3 + height + 5)
	flow.DrawParagraph([]pdf.TextRun{{Text: label, Style: style}}, pdf.ParagraphStyle{LineHeight: 12, SpaceBefore: 4, SpaceAfter: 3})
	flow.DrawImage(image, pdf.ImageOptions{Scale: scale, SpaceAfter: 5})
}

func decoratePage(c *pdf.Context, page pdf.PageInfo) error {
	background := color.RGBA{R: 252, G: 250, B: 246, A: 255}
	ink := color.RGBA{R: 91, G: 102, B: 116, A: 255}
	c.SetFillColor(background).DrawRectangle(0, 0, page.Spec.Width, page.Spec.Height).Fill()
	c.SetStrokeColor(color.RGBA{R: 218, G: 222, B: 226, A: 255}).SetLineWidth(.45).
		DrawRectangle(page.Body.X, page.Body.Y, page.Body.Width, page.Body.Height).Stroke()
	c.SetFillColor(ink).UseFontRole(pdf.FontRoleMono, 7.5).
		DrawString("PDF · IMAGE FLOW", page.Body.X, page.Spec.MarginTop+10).
		DrawStringAnchored(fmt.Sprintf("— %d —", page.Number), page.Spec.Width/2, page.Spec.Height-page.Spec.MarginBottom-5, .5, 0)
	return c.Error()
}

func makeJPEG() []byte {
	const width, height = 1200, 675
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			t := float64(x) / width
			u := float64(y) / height
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(28 + 70*t),
				G: uint8(70 + 95*(1-u)),
				B: uint8(125 + 95*u),
				A: 255,
			})
		}
	}
	for i := 0; i < 7; i++ {
		x0 := 80 + i*155
		y0 := 100 + (i%3)*125
		draw.Draw(img, image.Rect(x0, y0, x0+115, y0+115), &image.Uniform{C: color.RGBA{R: 245, G: 194, B: uint8(75 + i*12), A: 255}}, image.Point{}, draw.Src)
	}
	var output bytes.Buffer
	if err := jpeg.Encode(&output, img, &jpeg.Options{Quality: 86}); err != nil {
		panic(err)
	}
	return output.Bytes()
}

func makeTransparentCard(accent color.RGBA) image.Image {
	const width, height = 500, 300
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 18; y < height-18; y++ {
		for x := 18; x < width-18; x++ {
			alpha := uint8(220)
			if x < 28 || x >= width-28 || y < 28 || y >= height-28 {
				alpha = 120
			}
			img.SetNRGBA(x, y, color.NRGBA{R: accent.R, G: accent.G, B: accent.B, A: alpha})
		}
	}
	for y := 90; y < 210; y++ {
		for x := 190; x < 310; x++ {
			dx, dy := x-250, y-150
			if dx*dx+dy*dy < 55*55 {
				img.SetNRGBA(x, y, color.NRGBA{R: 252, G: 250, B: 246, A: 245})
			}
		}
	}
	return img
}

package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"

	pdf "github.com/lucasepe/pdf"
)

type emojiProvider struct{ target image.Image }

func (p emojiProvider) LookupEmoji(sequence string) (pdf.EmojiAsset, bool, error) {
	if sequence != "🎯" {
		return pdf.EmojiAsset{}, false, nil
	}
	return pdf.EmojiAsset{Image: p.target, HeightEm: 1.05, BaselineEm: -0.16, AdvanceEm: 1.18}, true, nil
}

func main() {
	spec := pdf.PageSpec{
		Width: pdf.Inches(6), Height: pdf.Inches(9),
		MarginTop: pdf.Inches(.25), MarginRight: pdf.Inches(.25),
		MarginBottom: pdf.Inches(.25), MarginLeft: pdf.Inches(.25),
		BindingOffset: pdf.Inches(.08),
		HeaderHeight:  24, FooterHeight: 24,
	}
	doc := pdf.NewPDF(spec.PaperSize())
	fontData, err := os.ReadFile("testdata/fonts/Vera.ttf")
	if err != nil {
		log.Fatal(err)
	}
	if err := doc.RegisterFont("Body", fontData); err != nil {
		log.Fatal(err)
	}
	for _, role := range []pdf.FontRole{pdf.FontRoleBody, pdf.FontRoleBodyBold, pdf.FontRoleBodyItalic, pdf.FontRoleBodyBoldItalic} {
		if err := doc.BindFontRole(role, "Body"); err != nil {
			log.Fatal(err)
		}
	}
	if err := doc.SetEmojiProvider(emojiProvider{target: targetImage()}); err != nil {
		log.Fatal(err)
	}

	c := pdf.NewContext(&doc)
	flow, err := pdf.NewFlow(c, spec, decoratePage)
	if err != nil {
		log.Fatal(err)
	}
	dark := color.RGBA{R: 39, G: 49, B: 63, A: 255}
	blue := color.RGBA{R: 38, G: 101, B: 160, A: 255}
	muted := color.RGBA{R: 91, G: 102, B: 116, A: 255}

	flow.DrawParagraph([]pdf.TextRun{{
		Text:  "Flow layout 6 × 9",
		Style: pdf.TextStyle{FontRole: pdf.FontRoleBodyBold, FontSize: 24, Color: blue},
	}}, pdf.ParagraphStyle{LineHeight: 31, SpaceAfter: 12})
	flow.DrawParagraph([]pdf.TextRun{
		{Text: "Testo Latin, wrapping automatico e stili inline. Il body rispetta margini, binding offset, header e footer; il cursore apre una nuova pagina quando la prossima riga non entra. ", Style: pdf.TextStyle{FontRole: pdf.FontRoleBody, FontSize: 10.5, Color: dark}},
		{Text: "config.OutputPath=/var/lib/ebookgen/documenti/catalogo_edizione_italiana/output.pdf", Style: pdf.TextStyle{FontRole: pdf.FontRoleMono, FontSize: 8.6, Color: blue, Wrap: pdf.WrapCode}},
	}, pdf.ParagraphStyle{LineHeight: 14.5, SpaceAfter: 9})

	paragraph := "Perché l’unità è già pronta — € 12,50. Le virgolette “tipografiche”, l’ellissi… e gli accenti àèéìòù vengono misurati con lo stesso font incorporato usato per il disegno. Una parola eccezionalmente lunga come anticonstituzionalissimamentecontinuaoltreilnormalespazio viene divisa soltanto come fallback, senza corrompere UTF-8."
	for i := 1; i <= 11; i++ {
		flow.DrawParagraph([]pdf.TextRun{
			{Text: fmt.Sprintf("%02d  ", i), Style: pdf.TextStyle{FontRole: pdf.FontRoleMonoBold, FontSize: 8.5, Color: blue}},
			{Text: paragraph, Style: pdf.TextStyle{FontRole: pdf.FontRoleBody, FontSize: 9.7, Color: dark}},
		}, pdf.ParagraphStyle{LineHeight: 13.2, SpaceAfter: 7})
	}

	flow.NewPage()
	flow.DrawParagraph([]pdf.TextRun{{
		Text:  "Page break esplicito",
		Style: pdf.TextStyle{FontRole: pdf.FontRoleBodyBold, FontSize: 21, Color: blue},
	}}, pdf.ParagraphStyle{LineHeight: 28, SpaceAfter: 12})
	flow.DrawParagraph([]pdf.TextRun{
		{Text: "Questa sezione inizia chiamando Flow.NewPage. Lo stesso meccanismo verrà usato dal nodo semantico di ebookgen. Courier resta disponibile sia nei code block sia nel ", Style: pdf.TextStyle{FontRole: pdf.FontRoleBody, FontSize: 11, Color: dark}},
		{Text: "codice_inline()", Style: pdf.TextStyle{FontRole: pdf.FontRoleMono, FontSize: 9.5, Color: blue, Wrap: pdf.WrapCode}},
		{Text: ", mentre una emoji provider-backed rimane atomica ", Style: pdf.TextStyle{FontRole: pdf.FontRoleBody, FontSize: 11, Color: dark}},
		{Text: "🎯", Emoji: true, Style: pdf.TextStyle{FontSize: 12}},
		{Text: " durante il line breaking.", Style: pdf.TextStyle{FontRole: pdf.FontRoleBody, FontSize: 11, Color: dark}},
	}, pdf.ParagraphStyle{LineHeight: 16, SpaceAfter: 16})
	flow.DrawParagraph([]pdf.TextRun{{
		Text:  "Le linee sottili mostrano il rettangolo effettivamente disponibile al flow. Nelle pagine pari il binding offset passa dal lato sinistro al lato destro.",
		Style: pdf.TextStyle{FontRole: pdf.FontRoleBodyItalic, FontSize: 10.5, Color: muted},
	}}, pdf.ParagraphStyle{Align: pdf.AlignCenter, LineHeight: 15})

	if err := flow.Error(); err != nil {
		log.Fatal(err)
	}
	if err := c.SaveFile("_examples/text_flow_6x9/output.pdf"); err != nil {
		log.Fatal(err)
	}
}

func decoratePage(c *pdf.Context, page pdf.PageInfo) error {
	background := color.RGBA{R: 252, G: 250, B: 246, A: 255}
	ink := color.RGBA{R: 91, G: 102, B: 116, A: 255}
	c.SetFillColor(background).DrawRectangle(0, 0, page.Spec.Width, page.Spec.Height).Fill()
	c.SetStrokeColor(color.RGBA{R: 210, G: 216, B: 222, A: 255}).SetLineWidth(.5).
		DrawRectangle(page.Body.X, page.Body.Y, page.Body.Width, page.Body.Height).Stroke()
	c.SetFillColor(ink).UseFontRole(pdf.FontRoleMono, 7.5).
		DrawString("PDF · FLOW LAYOUT", page.Body.X, page.Spec.MarginTop+10).
		DrawStringAnchored(fmt.Sprintf("— %d —", page.Number), page.Spec.Width/2, page.Spec.Height-page.Spec.MarginBottom-5, .5, 0)
	return c.Error()
}

func targetImage() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			dx, dy := x-32, y-32
			d2 := dx*dx + dy*dy
			switch {
			case d2 < 8*8:
				img.SetNRGBA(x, y, color.NRGBA{R: 245, G: 90, B: 80, A: 255})
			case d2 < 18*18:
				img.SetNRGBA(x, y, color.NRGBA{R: 250, G: 250, B: 248, A: 255})
			case d2 < 29*29:
				img.SetNRGBA(x, y, color.NRGBA{R: 38, G: 101, B: 160, A: 255})
			}
		}
	}
	return img
}

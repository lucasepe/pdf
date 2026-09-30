package main

import (
	"fmt"
	"image/color"
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

	c := pdf.NewContext(&doc)
	flow, err := pdf.NewFlow(c, spec, decoratePage)
	if err != nil {
		log.Fatal(err)
	}
	dark := color.RGBA{R: 39, G: 49, B: 63, A: 255}
	blue := color.RGBA{R: 38, G: 101, B: 160, A: 255}
	body := pdf.TextStyle{FontRole: pdf.FontRoleBody, FontSize: 8.4, Color: dark}
	header := pdf.TextStyle{FontRole: pdf.FontRoleBodyBold, FontSize: 8.2, Color: color.White}
	mono := pdf.TextStyle{FontRole: pdf.FontRoleMono, FontSize: 7.5, Color: blue, Wrap: pdf.WrapCode}

	flow.DrawParagraph([]pdf.TextRun{{
		Text:  "Tabella Markdown multipagina",
		Style: pdf.TextStyle{FontRole: pdf.FontRoleBodyBold, FontSize: 20, Color: blue},
	}}, pdf.ParagraphStyle{LineHeight: 27, SpaceAfter: 6})
	flow.DrawParagraph([]pdf.TextRun{{
		Text:  "Le righe restano atomiche; dopo ogni page break vengono ripetuti header, larghezze e allineamenti.",
		Style: pdf.TextStyle{FontRole: pdf.FontRoleBodyItalic, FontSize: 9.2, Color: dark},
	}}, pdf.ParagraphStyle{LineHeight: 13, SpaceAfter: 10})

	table := pdf.Table{
		Columns: []pdf.TableColumn{
			{Width: 34, Align: pdf.AlignRight},
			{MinWidth: 78, Weight: 1.2},
			{Width: 55, Align: pdf.AlignCenter},
			{MinWidth: 105, Weight: 1.8},
		},
		Header: []pdf.TableCell{
			pdf.TextTableCell("N.", header),
			pdf.TextTableCell("Sezione", header),
			pdf.TextTableCell("Stato", header),
			pdf.TextTableCell("Note di rendering", header),
		},
		Style: pdf.TableStyle{
			PaddingX: 4, PaddingY: 3, LineHeight: 10.8,
			SpaceAfter:  10,
			BorderWidth: .45,
			BorderColor: color.RGBA{R: 174, G: 185, B: 198, A: 255},
			HeaderFill:  color.RGBA{R: 38, G: 101, B: 160, A: 255},
			OddFill:     color.RGBA{R: 255, G: 255, B: 255, A: 255},
			EvenFill:    color.RGBA{R: 239, G: 244, B: 249, A: 255},
		},
	}
	states := []string{"Pronto", "Verifica", "Completo"}
	for i := 1; i <= 27; i++ {
		section := fmt.Sprintf("Capitolo %d — perché l’unità è importante", i)
		note := fmt.Sprintf("La cella %d contiene testo abbastanza lungo da mostrare il wrapping senza oltrepassare la colonna. ", i)
		runs := []pdf.TextRun{{Text: note, Style: body}}
		if i%4 == 0 {
			runs = append(runs,
				pdf.TextRun{Text: "cfg.SectionPath=/book/chapters/", Style: mono},
				pdf.TextRun{Text: fmt.Sprintf("%02d.md", i), Style: mono},
			)
		}
		table.Rows = append(table.Rows, pdf.TableRow{Cells: []pdf.TableCell{
			pdf.TextTableCell(fmt.Sprintf("%02d", i), mono),
			pdf.TextTableCell(section, body),
			pdf.TextTableCell(states[(i-1)%len(states)], body),
			{Runs: runs},
		}})
	}
	flow.DrawTable(table)
	flow.DrawParagraph([]pdf.TextRun{{
		Text:  "Fine della tabella: il testo successivo riprende dal cursore lasciato dall’ultima riga.",
		Style: pdf.TextStyle{FontRole: pdf.FontRoleBodyItalic, FontSize: 9, Color: dark},
	}}, pdf.ParagraphStyle{LineHeight: 13})

	if err := flow.Error(); err != nil {
		log.Fatal(err)
	}
	if err := c.SaveFile("_examples/tables_multipage/output.pdf"); err != nil {
		log.Fatal(err)
	}
}

func decoratePage(c *pdf.Context, page pdf.PageInfo) error {
	background := color.RGBA{R: 252, G: 250, B: 246, A: 255}
	ink := color.RGBA{R: 91, G: 102, B: 116, A: 255}
	c.SetFillColor(background).DrawRectangle(0, 0, page.Spec.Width, page.Spec.Height).Fill()
	c.SetFillColor(ink).UseFontRole(pdf.FontRoleMono, 7.5).
		DrawString("PDF · PAGINATED TABLE", page.Body.X, page.Spec.MarginTop+10).
		DrawStringAnchored(fmt.Sprintf("— %d —", page.Number), page.Spec.Width/2, page.Spec.Height-page.Spec.MarginBottom-5, .5, 0)
	return c.Error()
}

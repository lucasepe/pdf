package main

import (
	_ "embed"
	"image/color"
	"log"

	pdf "github.com/lucasepe/pdf"
)

// The example embeds the same six textual faces used by ebookgen so it is
// reproducible without relying on fonts installed on the host.
//
//go:embed assets/gentium-book/GentiumBook-Regular.ttf
var gentiumRegular []byte

//go:embed assets/gentium-book/GentiumBook-Bold.ttf
var gentiumBold []byte

//go:embed assets/gentium-book/GentiumBook-Italic.ttf
var gentiumItalic []byte

//go:embed assets/gentium-book/GentiumBook-BoldItalic.ttf
var gentiumBoldItalic []byte

//go:embed assets/liberation-mono/LiberationMono-Regular.ttf
var liberationMono []byte

//go:embed assets/liberation-mono/LiberationMono-Bold.ttf
var liberationMonoBold []byte

func main() {
	doc := pdf.NewPDF("A4")
	fonts := []struct {
		name string
		role pdf.FontRole
		data []byte
	}{
		{"Gentium Book Regular", pdf.FontRoleBody, gentiumRegular},
		{"Gentium Book Bold", pdf.FontRoleBodyBold, gentiumBold},
		{"Gentium Book Italic", pdf.FontRoleBodyItalic, gentiumItalic},
		{"Gentium Book Bold Italic", pdf.FontRoleBodyBoldItalic, gentiumBoldItalic},
		{"Liberation Mono Regular", pdf.FontRoleMono, liberationMono},
		{"Liberation Mono Bold", pdf.FontRoleMonoBold, liberationMonoBold},
	}
	for _, font := range fonts {
		if err := doc.RegisterFont(font.name, font.data); err != nil {
			log.Fatal(err)
		}
		if err := doc.BindFontRole(font.role, font.name); err != nil {
			log.Fatal(err)
		}
	}

	c := pdf.NewContext(&doc)
	ink := color.RGBA{R: 31, G: 41, B: 55, A: 255}
	muted := color.RGBA{R: 91, G: 102, B: 116, A: 255}
	blue := color.RGBA{R: 37, G: 99, B: 235, A: 255}

	c.SetFillColor(blue).UseFontRole(pdf.FontRoleBodyBold, 25).
		DrawString("Embedded font sample", 48, 62)
	c.SetFillColor(muted).UseFontRole(pdf.FontRoleBody, 10).
		DrawString("Gentium Book 7.000 + Liberation Mono 2.1.5", 48, 84)

	samples := []struct {
		role  pdf.FontRole
		label string
		text  string
	}{
		{pdf.FontRoleBody, "Gentium Book Regular", "Perché l’unità è già pronta — € 12,50"},
		{pdf.FontRoleBodyBold, "Gentium Book Bold", "ÀÈÉÌÒÙ · façade · cœur · español"},
		{pdf.FontRoleBodyItalic, "Gentium Book Italic", "Virgolette “tipografiche” ed ellissi…"},
		{pdf.FontRoleBodyBoldItalic, "Gentium Book Bold Italic", "Sphinx of black quartz, judge my vow."},
		{pdf.FontRoleMono, "Liberation Mono Regular", `cfg.OutputPath = "/tmp/ebook.pdf"`},
		{pdf.FontRoleMonoBold, "Liberation Mono Bold", "func render(book Book) error"},
	}
	y := 130.0
	for _, sample := range samples {
		c.SetFillColor(muted).UseFontRole(pdf.FontRoleBody, 9).
			DrawString(sample.label, 50, y)
		c.SetFillColor(ink).UseFontRole(sample.role, 16).
			DrawString(sample.text, 50, y+25)
		c.SetStrokeColor(color.RGBA{R: 220, G: 225, B: 232, A: 255}).
			SetLineWidth(.7).MoveTo(50, y+42).LineTo(545, y+42).Stroke()
		y += 92
	}

	if err := c.SaveFile("_examples/font_sample/font-sample.pdf"); err != nil {
		log.Fatal(err)
	}
}

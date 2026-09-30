package pdf

import (
	"fmt"

	"testing"
)

// Test_PDF_FontName_ is the unit test for
func Test_PDF_FontName_(t *testing.T) {

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.FontName(), "Helvetica")
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.FontName(), "Helvetica")
	}()

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.SetFontName("Courier").FontName(), "Courier")
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.SetFontName("Courier").FontName(), "Courier")
	}()

	func() {
		doc := NewPDF("A4")
		doc.SetCompression(false).
			SetUnits("cm").
			SetXY(1, 1).
			SetFont("Helvetica", 10).
			SetFontName("TimesRoman").
			DrawText("Hello World!")
		const want = `
		%PDF-1.4
		1 0 obj <</Type/Catalog/Pages 2 0 R>>
		endobj
		2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 595 841]/Kids[3 0 R]>>
		endobj
		3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R
		/Resources <</Font <</FNT1 5 0 R>> >> >>
		endobj
		4 0 obj <</Length 95>> stream
		BT /FNT1 10 Tf ET
		0.000 0.000 0.000 rg
		0.000 0.000 0.000 RG
		BT 28 813 Td (Hello World!) Tj ET
		endstream
		endobj
		5 0 obj <</Type/Font/Subtype/Type1/Name/FNT1
		/BaseFont/Times-Roman
		/Encoding/StandardEncoding>>
		endobj
		xref
		0 6
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000228 00000 n
		0000000373 00000 n
		trailer
		<</Size 6/Root 1 0 R>>
		startxref
		476
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
	}()
}

// Test_PDF_FontSize_ is the unit test for PDF.FontSize() and SetFontSize()
func Test_PDF_FontSize_(t *testing.T) {

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.FontSize(), 10)
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.FontSize(), 10)
	}()

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.SetFontSize(15).FontSize(), 15)
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.SetFontSize(20).FontSize(), 20)
	}()

	func() {
		doc := NewPDF("A4")
		doc.SetCompression(false).SetUnits("cm")
		pt10 := doc.ToUnits(10)
		const w = 10
		doc.SetFont("Times-Bold", 20).SetXY(1, 1).DrawText("Font Sizes")
		for i, size := range []float64{5, 6, 7, 8, 9, 10, 15, 20, 25, 30} {
			y := 2 + float64(i)*3*pt10
			doc.SetLineWidth(0.1).
				SetXY(1, y+0.5).SetColor("Gray").
				DrawBox(1, y+0*pt10, w, pt10).
				DrawBox(1, y+1*pt10, w, pt10).
				DrawBox(1, y+2*pt10, w, pt10).
				SetXY(1, y+0.5).SetColor("Black").SetFont("Helvetica", size).
				DrawBox(1, y, w, 3*pt10).
				DrawTextInBox(1, y, w, 3*pt10, "TC",
					fmt.Sprintf("Helvetica %1.0f", size))
		}
		const want = `
		%PDF-1.4
		1 0 obj <</Type/Catalog/Pages 2 0 R>>
		endobj
		2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 595 841]/Kids[3 0 R]>>
		endobj
		3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R
		/Resources <</Font <<
		/FNT1 5 0 R
		/FNT2 6 0 R>> >> >>
		endobj
		4 0 obj <</Length 2440>> stream
		BT /FNT1 20 Tf ET
		0.000 0.000 0.000 rg
		0.000 0.000 0.000 RG
		BT 28 813 Td (Font Sizes) Tj ET
		0.745 0.745 0.745 RG
		0.100 w
		28.346 775.197 283.465 10.000 re S
		28.346 765.197 283.465 10.000 re S
		28.346 755.197 283.465 10.000 re S
		0.000 0.000 0.000 RG
		28.346 755.197 283.465 30.000 re S
		BT /FNT2 5 Tf ET
		BT 157 780 Td (Helvetica 5) Tj ET
		0.745 0.745 0.745 RG
		28.346 745.197 283.465 10.000 re S
		28.346 735.197 283.465 10.000 re S
		28.346 725.197 283.465 10.000 re S
		0.000 0.000 0.000 RG
		28.346 725.197 283.465 30.000 re S
		BT /FNT2 6 Tf ET
		BT 155 749 Td (Helvetica 6) Tj ET
		0.745 0.745 0.745 RG
		28.346 715.197 283.465 10.000 re S
		28.346 705.197 283.465 10.000 re S
		28.346 695.197 283.465 10.000 re S
		0.000 0.000 0.000 RG
		28.346 695.197 283.465 30.000 re S
		BT /FNT2 7 Tf ET
		BT 152 718 Td (Helvetica 7) Tj ET
		0.745 0.745 0.745 RG
		28.346 685.197 283.465 10.000 re S
		28.346 675.197 283.465 10.000 re S
		28.346 665.197 283.465 10.000 re S
		0.000 0.000 0.000 RG
		28.346 665.197 283.465 30.000 re S
		BT /FNT2 8 Tf ET
		BT 150 687 Td (Helvetica 8) Tj ET
		0.745 0.745 0.745 RG
		28.346 655.197 283.465 10.000 re S
		28.346 645.197 283.465 10.000 re S
		28.346 635.197 283.465 10.000 re S
		0.000 0.000 0.000 RG
		28.346 635.197 283.465 30.000 re S
		BT /FNT2 9 Tf ET
		BT 147 656 Td (Helvetica 9) Tj ET
		0.745 0.745 0.745 RG
		28.346 625.197 283.465 10.000 re S
		28.346 615.197 283.465 10.000 re S
		28.346 605.197 283.465 10.000 re S
		0.000 0.000 0.000 RG
		28.346 605.197 283.465 30.000 re S
		BT /FNT2 10 Tf ET
		BT 142 625 Td (Helvetica 10) Tj ET
		0.745 0.745 0.745 RG
		28.346 595.197 283.465 10.000 re S
		28.346 585.197 283.465 10.000 re S
		28.346 575.197 283.465 10.000 re S
		0.000 0.000 0.000 RG
		28.346 575.197 283.465 30.000 re S
		BT /FNT2 15 Tf ET
		BT 128 590 Td (Helvetica 15) Tj ET
		0.745 0.745 0.745 RG
		28.346 565.197 283.465 10.000 re S
		28.346 555.197 283.465 10.000 re S
		28.346 545.197 283.465 10.000 re S
		0.000 0.000 0.000 RG
		28.346 545.197 283.465 30.000 re S
		BT /FNT2 20 Tf ET
		BT 115 555 Td (Helvetica 20) Tj ET
		0.745 0.745 0.745 RG
		28.346 535.197 283.465 10.000 re S
		28.346 525.197 283.465 10.000 re S
		28.346 515.197 283.465 10.000 re S
		0.000 0.000 0.000 RG
		28.346 515.197 283.465 30.000 re S
		BT /FNT2 25 Tf ET
		BT 101 520 Td (Helvetica 25) Tj ET
		0.745 0.745 0.745 RG
		28.346 505.197 283.465 10.000 re S
		28.346 495.197 283.465 10.000 re S
		28.346 485.197 283.465 10.000 re S
		0.000 0.000 0.000 RG
		28.346 485.197 283.465 30.000 re S
		BT /FNT2 30 Tf ET
		BT 87 485 Td (Helvetica 30) Tj ET
		endstream
		endobj
		5 0 obj <</Type/Font/Subtype/Type1/Name/FNT1
		/BaseFont/Times-Bold
		/Encoding/StandardEncoding>>
		endobj
		6 0 obj <</Type/Font/Subtype/Type1/Name/FNT2
		/BaseFont/Helvetica
		/Encoding/StandardEncoding>>
		endobj
		xref
		0 7
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000241 00000 n
		0000002733 00000 n
		0000002835 00000 n
		trailer
		<</Size 7/Root 1 0 R>>
		startxref
		2936
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
	}()
}

// Test_PDF_HorizontalScaling_ is the unit test for PDF.HorizontalScaling()
func Test_PDF_HorizontalScaling_(t *testing.T) {

	func() {
		var doc PDF
		tEqual(t, doc.HorizontalScaling(), 100)
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.HorizontalScaling(), 100)
	}()

	func() {
		var doc PDF
		doc.SetHorizontalScaling(149)
		tEqual(t, doc.HorizontalScaling(), 149)
	}()
	func() {
		doc := NewPDF("A4")
		doc.SetHorizontalScaling(149)
		tEqual(t, doc.HorizontalScaling(), 149)
	}()

	func() {
		doc := NewPDF("A4-L")
		{
			doc.SetCompression(false).
				SetUnits("cm").
				SetFont("Times-Bold", 20).
				SetXY(1, 1).
				DrawText("Horizontal Scaling Property")
		}
		for i, hscaling := range []int{50, 100, 150, 200, 250} {
			y := 2.5 + float64(i)*2.5
			doc.SetXY(1, y).
				SetFont("Helvetica", 10).
				SetHorizontalScaling(100).
				DrawText(fmt.Sprintf("Horizontal Scaling = %d", hscaling)).
				SetXY(1, y+0.7).
				SetHorizontalScaling(uint16(hscaling)).
				SetFontSize(20).
				DrawText("Five hexing wizard bots jump quickly")
		}
		const want = `
		%PDF-1.4
		1 0 obj <</Type/Catalog/Pages 2 0 R>>
		endobj
		2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 841 595]/Kids[3 0 R]>>
		endobj
		3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R
		/Resources <</Font <<
		/FNT1 5 0 R
		/FNT2 6 0 R>> >> >>
		endobj
		4 0 obj <</Length 899>> stream
		BT /FNT1 20 Tf ET
		0.000 0.000 0.000 rg
		0.000 0.000 0.000 RG
		BT 28 566 Td (Horizontal Scaling Property) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 524 Td (Horizontal Scaling = 50) Tj ET
		BT /FNT2 20 Tf ET
		BT 50 Tz ET
		BT 28 504 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 100 Tz ET
		BT 28 453 Td (Horizontal Scaling = 100) Tj ET
		BT /FNT2 20 Tf ET
		BT 28 433 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 382 Td (Horizontal Scaling = 150) Tj ET
		BT /FNT2 20 Tf ET
		BT 150 Tz ET
		BT 28 362 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 100 Tz ET
		BT 28 311 Td (Horizontal Scaling = 200) Tj ET
		BT /FNT2 20 Tf ET
		BT 200 Tz ET
		BT 28 291 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 100 Tz ET
		BT 28 240 Td (Horizontal Scaling = 250) Tj ET
		BT /FNT2 20 Tf ET
		BT 250 Tz ET
		BT 28 221 Td (Five hexing wizard bots jump quickly) Tj ET
		endstream
		endobj
		5 0 obj <</Type/Font/Subtype/Type1/Name/FNT1
		/BaseFont/Times-Bold
		/Encoding/StandardEncoding>>
		endobj
		6 0 obj <</Type/Font/Subtype/Type1/Name/FNT2
		/BaseFont/Helvetica
		/Encoding/StandardEncoding>>
		endobj
		xref
		0 7
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000241 00000 n
		0000001191 00000 n
		0000001293 00000 n
		trailer
		<</Size 7/Root 1 0 R>>
		startxref
		1394
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
	}()
}

// Test_PDF_SetFont_ is the unit test for PDF.SetFont()
func Test_PDF_SetFont_(t *testing.T) {

	for _, tc := range []struct {
		size    float64
		inName  string
		expName string
	}{
		{size: 10, inName: "Courier", expName: "Courier"},
		{size: 20, inName: "Zapf-Dingbats", expName: "Zapf-Dingbats"},
		{size: 30, inName: "ZapfDingbats", expName: "ZapfDingbats"},
		{size: 40, inName: "YeOldeScript", expName: "YeOldeScript"},
	} {
		var doc PDF // uninitialized PDF
		doc.SetFont(tc.inName, tc.size)
		tEqual(t, doc.FontName(), tc.expName)
		tEqual(t, doc.FontSize(), tc.size)
	}

	func() {
		doc := NewPDF("A4")
		doc.SetCompression(false).
			SetUnits("cm").
			SetFont("Times-Bold", 20).
			SetXY(1, 1).
			DrawText("Built-in PDF Fonts")
		for i, font := range []string{
			"Courier",
			"Courier-Bold",
			"Courier-BoldOblique",
			"Courier-Oblique",
			"Helvetica",
			"Helvetica-Bold",
			"Helvetica-BoldOblique",
			"Helvetica-Oblique",
			"Symbol",
			"Times-Bold",
			"Times-BoldItalic",
			"Times-Italic",
			"Times-Roman",
			"ZapfDingbats",
		} {
			y := 2.5 + float64(i)*1.8
			doc.SetXY(1, y).
				SetFont("Helvetica", 10).
				DrawText(font).
				SetXY(1, y+0.7).
				SetFont(font, 20).
				DrawText("Five hexing wizard bots jump quickly")

		}
		const want = `
		%PDF-1.4
		1 0 obj <</Type/Catalog/Pages 2 0 R>>
		endobj
		2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 595 841]/Kids[3 0 R]>>
		endobj
		3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R
		/Resources <</Font <<
		/FNT1 5 0 R
		/FNT2 6 0 R
		/FNT3 7 0 R
		/FNT4 8 0 R
		/FNT5 9 0 R
		/FNT6 10 0 R
		/FNT7 11 0 R
		/FNT8 12 0 R
		/FNT9 13 0 R
		/FNT10 14 0 R
		/FNT11 15 0 R
		/FNT12 16 0 R
		/FNT13 17 0 R
		/FNT14 18 0 R>> >> >>
		endobj
		4 0 obj <</Length 1910>> stream
		BT /FNT1 20 Tf ET
		0.000 0.000 0.000 rg
		0.000 0.000 0.000 RG
		BT 28 813 Td (Built-in PDF Fonts) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 771 Td (Courier) Tj ET
		BT /FNT3 20 Tf ET
		BT 28 751 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 720 Td (Courier-Bold) Tj ET
		BT /FNT4 20 Tf ET
		BT 28 700 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 668 Td (Courier-BoldOblique) Tj ET
		BT /FNT5 20 Tf ET
		BT 28 649 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 617 Td (Courier-Oblique) Tj ET
		BT /FNT6 20 Tf ET
		BT 28 598 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 566 Td (Helvetica) Tj ET
		BT /FNT2 20 Tf ET
		BT 28 547 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 515 Td (Helvetica-Bold) Tj ET
		BT /FNT7 20 Tf ET
		BT 28 496 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 464 Td (Helvetica-BoldOblique) Tj ET
		BT /FNT8 20 Tf ET
		BT 28 445 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 413 Td (Helvetica-Oblique) Tj ET
		BT /FNT9 20 Tf ET
		BT 28 394 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 362 Td (Symbol) Tj ET
		BT /FNT10 20 Tf ET
		BT 28 342 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 311 Td (Times-Bold) Tj ET
		BT /FNT1 20 Tf ET
		BT 28 291 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 260 Td (Times-BoldItalic) Tj ET
		BT /FNT11 20 Tf ET
		BT 28 240 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 209 Td (Times-Italic) Tj ET
		BT /FNT12 20 Tf ET
		BT 28 189 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 158 Td (Times-Roman) Tj ET
		BT /FNT13 20 Tf ET
		BT 28 138 Td (Five hexing wizard bots jump quickly) Tj ET
		BT /FNT2 10 Tf ET
		BT 28 107 Td (ZapfDingbats) Tj ET
		BT /FNT14 20 Tf ET
		BT 28 87 Td (Five hexing wizard bots jump quickly) Tj ET
		endstream
		endobj
		5 0 obj <</Type/Font/Subtype/Type1/Name/FNT1
		/BaseFont/Times-Bold
		/Encoding/StandardEncoding>>
		endobj
		6 0 obj <</Type/Font/Subtype/Type1/Name/FNT2
		/BaseFont/Helvetica
		/Encoding/StandardEncoding>>
		endobj
		7 0 obj <</Type/Font/Subtype/Type1/Name/FNT3
		/BaseFont/Courier
		/Encoding/StandardEncoding>>
		endobj
		8 0 obj <</Type/Font/Subtype/Type1/Name/FNT4
		/BaseFont/Courier-Bold
		/Encoding/StandardEncoding>>
		endobj
		9 0 obj <</Type/Font/Subtype/Type1/Name/FNT5
		/BaseFont/Courier-BoldOblique
		/Encoding/StandardEncoding>>
		endobj
		10 0 obj <</Type/Font/Subtype/Type1/Name/FNT6
		/BaseFont/Courier-Oblique
		/Encoding/StandardEncoding>>
		endobj
		11 0 obj <</Type/Font/Subtype/Type1/Name/FNT7
		/BaseFont/Helvetica-Bold
		/Encoding/StandardEncoding>>
		endobj
		12 0 obj <</Type/Font/Subtype/Type1/Name/FNT8
		/BaseFont/Helvetica-BoldOblique
		/Encoding/StandardEncoding>>
		endobj
		13 0 obj <</Type/Font/Subtype/Type1/Name/FNT9
		/BaseFont/Helvetica-Oblique
		/Encoding/StandardEncoding>>
		endobj
		14 0 obj <</Type/Font/Subtype/Type1/Name/FNT10
		/BaseFont/Symbol
		/Encoding/StandardEncoding>>
		endobj
		15 0 obj <</Type/Font/Subtype/Type1/Name/FNT11
		/BaseFont/Times-BoldItalic
		/Encoding/StandardEncoding>>
		endobj
		16 0 obj <</Type/Font/Subtype/Type1/Name/FNT12
		/BaseFont/Times-Italic
		/Encoding/StandardEncoding>>
		endobj
		17 0 obj <</Type/Font/Subtype/Type1/Name/FNT13
		/BaseFont/Times-Roman
		/Encoding/StandardEncoding>>
		endobj
		18 0 obj <</Type/Font/Subtype/Type1/Name/FNT14
		/BaseFont/ZapfDingbats
		/Encoding/StandardEncoding>>
		endobj
		xref
		0 19
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000399 00000 n
		0000002361 00000 n
		0000002463 00000 n
		0000002564 00000 n
		0000002663 00000 n
		0000002767 00000 n
		0000002878 00000 n
		0000002986 00000 n
		0000003093 00000 n
		0000003207 00000 n
		0000003317 00000 n
		0000003417 00000 n
		0000003527 00000 n
		0000003633 00000 n
		0000003738 00000 n
		trailer
		<</Size 19/Root 1 0 R>>
		startxref
		3844
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
	}()
}

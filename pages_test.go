package pdf

import (
	"strings"
	"testing"
)

// Test_PDF_CurrentPage_ tests PDF.CurrentPage()
func Test_PDF_CurrentPage_(t *testing.T) {
	func() {
		var doc PDF // uninitialized PDF

		tEqual(t, doc.CurrentPage(), 1)

		doc.AddPage()
		tEqual(t, doc.CurrentPage(), 1)

		doc.AddPage()
		tEqual(t, doc.CurrentPage(), 2)

		doc.AddPage()
		tEqual(t, doc.CurrentPage(), 3)

		doc.AddPage()
		tEqual(t, doc.CurrentPage(), 4)
	}()
	func() {
		doc := NewPDF("LETTER")

		tEqual(t, doc.CurrentPage(), 1)

		doc.AddPage()
		tEqual(t, doc.CurrentPage(), 1)

		doc.AddPage()
		tEqual(t, doc.CurrentPage(), 2)

		doc.AddPage()
		tEqual(t, doc.CurrentPage(), 3)

		doc.AddPage()
		tEqual(t, doc.CurrentPage(), 4)
	}()
}

// Test_NewPDF_ is the unit test for PDF.NewPDF
func Test_NewPDF_(t *testing.T) {
	const want = `
	%PDF-1.4
	1 0 obj <</Type/Catalog/Pages 2 0 R>>
	endobj
	2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 595 841]/Kids[3 0 R]>>
	endobj
	3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R>>
	endobj
	4 0 obj <</Length 0>> stream
	endstream
	endobj
	xref
	0 5
	0000000000 65535 f
	0000000010 00000 n
	0000000056 00000 n
	0000000130 00000 n
	0000000189 00000 n
	trailer
	<</Size 5/Root 1 0 R>>
	startxref
	238
	%%EOF
	`

	func() {
		doc := NewPDF("A4")
		got := doc.SetCompression(false).AddPage().Bytes()
		pdfCompare(t, got, want)
	}()

	func() {
		doc := NewPDF("A4")
		got := doc.SetCompression(false).Bytes()
		pdfCompare(t, got, want)
	}()
}

// Test_PDF_PageCount_ tests PDF.PageCount()
func Test_PDF_PageCount_(t *testing.T) {

	func() {
		var doc PDF
		tEqual(t, doc.PageCount(), 1)
	}()

	func() {
		var doc PDF
		doc.AddPage()
		tEqual(t, doc.PageCount(), 1)
	}()

	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.PageCount(), 1)
	}()

	func() {
		doc := NewPDF("A4")
		doc.AddPage()
		tEqual(t, doc.PageCount(), 1)
	}()

	func() {
		doc := NewPDF("LETTER")
		doc.SetXY(1, 1)
		tEqual(t, doc.PageCount(), 1)
	}()

	func() {
		var doc PDF //                                     uninitialized PDF
		doc.SetXY(1, 1)
		doc.AddPage()
		tEqual(t, doc.PageCount(), 2)
	}()
	func() {
		doc := NewPDF("LETTER")
		doc.SetXY(1, 1)
		doc.AddPage()
		tEqual(t, doc.PageCount(), 2)
	}()

	func() {
		var doc PDF //                                     uninitialized PDF
		for i := 0; i < 10; i++ {
			doc.AddPage()
		}
		tEqual(t, doc.PageCount(), 10)
	}()
	func() {
		doc := NewPDF("LETTER")
		for i := 0; i < 10; i++ {
			doc.AddPage()
		}
		tEqual(t, doc.PageCount(), 10)
	}()
}

// Test_PDF_PageHeight_ tests PDF.PageHeight()
func Test_PDF_PageHeight_(t *testing.T) {
	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.PageHeight(), 841.889764)
	}()

	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.PageHeight(), 841.889764)

	}()
	func() {
		doc := NewPDF("LETTER")
		tEqual(t, doc.PageHeight(), 790.866142)

	}()
}

// Test_PDF_PageWidth_ tests PDF.PageWidth()
func Test_PDF_PageWidth_(t *testing.T) {
	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.PageWidth(), 595.275591)
	}()

	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.PageWidth(), 595.275591)

	}()
	func() {
		doc := NewPDF("LETTER")
		tEqual(t, doc.PageWidth(), 612.283465)

	}()
}

// Test_PDF_Reset_ tests PDF.Reset()
func Test_PDF_Reset_(t *testing.T) {

	doc := NewPDF("A4")
	{
		doc.SetCompression(false).
			SetUnits("cm").
			SetColumnWidths(1, 4, 9).
			SetColor("#006B3C CadmiumGreen").
			SetFont("Helvetica-Bold", 10).
			SetX(5).
			SetY(5).
			DrawText("FIRST").
			DrawText("SECOND").
			DrawText("THIRD")
		const want = `
		%PDF-1.4
		1 0 obj <</Type/Catalog/Pages 2 0 R>>
		endobj
		2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 595 841]/Kids[3 0 R]>>
		endobj
		3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R
		/Resources <</Font <</FNT1 5 0 R>> >> >>
		endobj
		4 0 obj <</Length 143>> stream
		BT /FNT1 10 Tf ET
		0.000 0.420 0.235 rg
		0.000 0.420 0.235 RG
		BT 0 700 Td (FIRST) Tj ET
		BT 28 700 Td (SECOND) Tj ET
		BT 141 700 Td (THIRD) Tj ET
		endstream
		endobj
		5 0 obj <</Type/Font/Subtype/Type1/Name/FNT1
		/BaseFont/Helvetica-Bold
		/Encoding/StandardEncoding>>
		endobj
		xref
		0 6
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000228 00000 n
		0000000422 00000 n
		trailer
		<</Size 6/Root 1 0 R>>
		startxref
		528
		%%EOF
		`
		got := doc.Bytes()
		pdfCompare(t, got, want)
	}
	{
		doc.Reset()
		//
		// after calling Reset(), the PDF should just be a blank page:
		const want = `
		%PDF-1.4
		1 0 obj <</Type/Catalog/Pages 2 0 R>>
		endobj
		2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 595 841]/Kids[3 0 R]>>
		endobj
		3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R>>
		endobj
		4 0 obj <</Length 0>> stream
		endstream
		endobj
		xref
		0 5
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000189 00000 n
		trailer
		<</Size 5/Root 1 0 R>>
		startxref
		238
		%%EOF
		`
		got := doc.SetCompression(false).Bytes()
		pdfCompare(t, got, want)
	}

}

// Test_PDF_SetXY_ is the unit test for PDF.SetXY()
func Test_PDF_SetXY_(t *testing.T) {

	func() {
		var doc PDF
		doc.SetXY(123, 456)
		tEqual(t, doc.X(), 123)
		tEqual(t, doc.Y(), 456)
	}()
	func() {
		doc := NewPDF("A4")
		doc.SetXY(123, 456)
		tEqual(t, doc.X(), 123)
		tEqual(t, doc.Y(), 456)
	}()

	func() {
		doc := NewPDF("A4")
		doc.SetCompression(false).SetUnits("cm").SetFont("Helvetica", 10).
			SetXY(1, 3).DrawText("X=1cm Y=3cm").
			SetXY(3, 1).DrawText("X=3cm Y=1cm").
			SetXY(10, 5).DrawText("X=10cm Y=5cm").
			SetXY(5, 10).DrawText("X=5cm Y=10cm")
		const want = `
		%PDF-1.4
		1 0 obj <</Type/Catalog/Pages 2 0 R>>
		endobj
		2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 595 841]/Kids[3 0 R]>>
		endobj
		3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R
		/Resources <</Font <</FNT1 5 0 R>> >> >>
		endobj
		4 0 obj <</Length 197>> stream
		BT /FNT1 10 Tf ET
		0.000 0.000 0.000 rg
		0.000 0.000 0.000 RG
		BT 28 756 Td (X=1cm Y=3cm) Tj ET
		BT 85 813 Td (X=3cm Y=1cm) Tj ET
		BT 283 700 Td (X=10cm Y=5cm) Tj ET
		BT 141 558 Td (X=5cm Y=10cm) Tj ET
		endstream
		endobj
		5 0 obj <</Type/Font/Subtype/Type1/Name/FNT1
		/BaseFont/Helvetica
		/Encoding/StandardEncoding>>
		endobj
		xref
		0 6
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000228 00000 n
		0000000476 00000 n
		trailer
		<</Size 6/Root 1 0 R>>
		startxref
		577
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
	}()
}

// Test_PDF_X_ is the unit test for PDF.X()
func Test_PDF_X_(t *testing.T) {

	func() {
		var doc PDF
		tEqual(t, doc.X(), -1)
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.X(), -1)
	}()

	func() {
		var doc PDF
		doc.SetX(123)
		tEqual(t, doc.X(), 123)
	}()
	func() {
		doc := NewPDF("A4")
		doc.SetX(456)
		tEqual(t, doc.X(), 456)
	}()

	func() {
		doc := NewPDF("A4")
		doc.SetCompression(false).
			SetUnits("cm").
			SetXY(10, 1).
			SetFont("Times-Bold", 20).
			DrawText("X=10 Y=1")
		const want = `
		%PDF-1.4
		1 0 obj <</Type/Catalog/Pages 2 0 R>>
		endobj
		2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 595 841]/Kids[3 0 R]>>
		endobj
		3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R
		/Resources <</Font <</FNT1 5 0 R>> >> >>
		endobj
		4 0 obj <</Length 92>> stream
		BT /FNT1 20 Tf ET
		0.000 0.000 0.000 rg
		0.000 0.000 0.000 RG
		BT 283 813 Td (X=10 Y=1) Tj ET
		endstream
		endobj
		5 0 obj <</Type/Font/Subtype/Type1/Name/FNT1
		/BaseFont/Times-Bold
		/Encoding/StandardEncoding>>
		endobj
		xref
		0 6
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000228 00000 n
		0000000370 00000 n
		trailer
		<</Size 6/Root 1 0 R>>
		startxref
		472
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
	}()
}

// Test_PDF_Y_ is the unit test for PDF.Y()
func Test_PDF_Y_(t *testing.T) {

	func() {
		var doc PDF
		tEqual(t, doc.Y(), -1)
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.Y(), -1)
	}()

	func() {
		var doc PDF
		doc.SetY(321)
		tEqual(t, doc.Y(), 321)
	}()
	func() {
		doc := NewPDF("A4")
		doc.SetY(654)
		tEqual(t, doc.Y(), 654)
	}()

	func() {
		doc := NewPDF("A4")
		doc.SetCompression(false).
			SetUnits("cm").
			SetXY(1, 10).
			SetFont("Times-Bold", 20).
			DrawText("X=1 Y=10")
		const want = `
		%PDF-1.4
		1 0 obj <</Type/Catalog/Pages 2 0 R>>
		endobj
		2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 595 841]/Kids[3 0 R]>>
		endobj
		3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R
		/Resources <</Font <</FNT1 5 0 R>> >> >>
		endobj
		4 0 obj <</Length 91>> stream
		BT /FNT1 20 Tf ET
		0.000 0.000 0.000 rg
		0.000 0.000 0.000 RG
		BT 28 558 Td (X=1 Y=10) Tj ET
		endstream
		endobj
		5 0 obj <</Type/Font/Subtype/Type1/Name/FNT1
		/BaseFont/Times-Bold
		/Encoding/StandardEncoding>>
		endobj
		xref
		0 6
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000228 00000 n
		0000000369 00000 n
		trailer
		<</Size 6/Root 1 0 R>>
		startxref
		471
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
	}()
}

// go test --run Test_getPapreSize_
func Test_getPapreSize_(t *testing.T) {

	subtest := func(paperSize, permuted string, w, h float64, err error) {
		var doc PDF
		doc.SetUnits("mm")
		got, gotErr := doc.getPaperSize(permuted)
		if gotErr != err {
			t.Errorf("'error' mismatch: expected: %v returned %v", err, gotErr)
			t.Fail()
		}
		if got.name != paperSize {
			mismatch(t, permuted+" 'name'", paperSize, got.name)
		}
		if floatStr(got.widthPt) != floatStr(w) {
			mismatch(t, permuted+" 'widthPt'", w, got.widthPt)
		}
		if floatStr(got.heightPt) != floatStr(h) {
			mismatch(t, permuted+" 'heightPt'", h, got.heightPt)
		}
	}

	test := func(paperSize string, w, h float64, err error) {
		const PTperMM = 2.83464566929134
		spaces := []string{"", " ", "  ", "   ", "\r", "\n", "\t"}
		for _, orient := range []string{"", "-l", "-L"} {
			permuted := permuteStrings(
				spaces,
				[]string{
					strings.ToLower(paperSize),
					strings.ToUpper(paperSize),
				},
				spaces,
				[]string{orient},
				spaces,
			)
			for _, s := range permuted {
				size := paperSize + strings.ToUpper(orient)
				if orient == "-l" || orient == "-L" {
					subtest(size, s, h*PTperMM, w*PTperMM, err)
				} else {
					subtest(size, s, w*PTperMM, h*PTperMM, err)
				}
			}
		}
	}
	test("A4", 210, 297, nil)
	test("A0", 841, 1189, nil)
	test("A1", 594, 841, nil)
	test("A2", 420, 594, nil)
	test("A3", 297, 420, nil)
	test("A4", 210, 297, nil)
	test("A5", 148, 210, nil)
	test("A6", 105, 148, nil)
	test("A7", 74, 105, nil)
	test("A8", 52, 74, nil)
	test("A9", 37, 52, nil)
	test("A10", 26, 37, nil)
	test("B0", 1000, 1414, nil)
	test("B1", 707, 1000, nil)
	test("B2", 500, 707, nil)
	test("B3", 353, 500, nil)
	test("B4", 250, 353, nil)
	test("B5", 176, 250, nil)
	test("B6", 125, 176, nil)
	test("B7", 88, 125, nil)
	test("B8", 62, 88, nil)
	test("B9", 44, 62, nil)
	test("B10", 31, 44, nil)
	test("C0", 917, 1297, nil)
	test("C1", 648, 917, nil)
	test("C2", 458, 648, nil)
	test("C3", 324, 458, nil)
	test("C4", 229, 324, nil)
	test("C5", 162, 229, nil)
	test("C6", 114, 162, nil)
	test("C7", 81, 114, nil)
	test("C8", 57, 81, nil)
	test("C9", 40, 57, nil)
	test("C10", 28, 40, nil)
	test("LEDGER", 432, 279, nil)
	test("LEGAL", 216, 356, nil)
	test("LETTER", 216, 279, nil)
	test("TABLOID", 279, 432, nil)
}

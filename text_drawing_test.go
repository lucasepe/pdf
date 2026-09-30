package pdf

import (
	"fmt"

	"testing"
)

// Test_PDF_DrawText_ is the unit test for
// DrawText(s string) *PDF
func Test_PDF_DrawText_(t *testing.T) {
	func() {
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
		}
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
		pdfCompare(t, doc.Bytes(), want)
	}()
	func() {
		doc := NewPDF("A4")
		{
			doc.SetCompression(false).
				SetUnits("cm").
				SetFont("Ye-Olde-Scriptte", 10).
				SetXY(5, 5).
				SetHorizontalScaling(150).
				DrawText("Ye-Olde-Scriptte")
		}
		const want = `
		%PDF-1.4
		1 0 obj <</Type/Catalog/Pages 2 0 R>>
		endobj
		2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 595 841]/Kids[3 0 R]>>
		endobj
		3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R
		/Resources <</Font <</FNT1 5 0 R>> >> >>
		endobj
		4 0 obj <</Length 113>> stream
		BT /FNT1 10 Tf ET
		BT 150 Tz ET
		0.000 0.000 0.000 rg
		0.000 0.000 0.000 RG
		BT 141 700 Td (Ye-Olde-Scriptte) Tj ET
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
		0000000392 00000 n
		trailer
		<</Size 6/Root 1 0 R>>
		startxref
		493
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
		tEqual(t, len(doc.Errors()), 1)
		tEqual(t, doc.PullError(),
			fmt.Errorf(`Invalid font "Ye-Olde-Scriptte" @DrawText`))
	}()
}

// Test_PDF_DrawTextAt_ is the unit test for
// DrawTextAt(x, y float64, text string) *PDF
func Test_PDF_DrawTextAt_(t *testing.T) {
	doc := NewPDF("A4")
	{
		doc.SetCompression(false).
			SetUnits("cm").
			SetColor("#36454F Charcoal").
			SetFont("Helvetica-Bold", 20).
			DrawTextAt(5, 5, "").
			DrawTextAt(5, 5, "(5,5)").
			DrawTextAt(10, 10, "").
			DrawTextAt(10, 10, "(10,10)").
			DrawTextAt(15, 15, "").
			DrawTextAt(15, 15, "(15,15)").
			SetColor("#E03C31 CGRed").
			FillBox(5, 5, 0.1, 0.1).
			FillBox(10, 10, 0.1, 0.1).
			FillBox(15, 15, 0.1, 0.1)
	}
	const want = `
	%PDF-1.4
	1 0 obj <</Type/Catalog/Pages 2 0 R>>
	endobj
	2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 595 841]/Kids[3 0 R]>>
	endobj
	3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R
	/Resources <</Font <</FNT1 5 0 R>> >> >>
	endobj
	4 0 obj <</Length 297>> stream
	BT /FNT1 20 Tf ET
	0.212 0.271 0.310 rg
	0.212 0.271 0.310 RG
	BT 141 700 Td (\(5,5\)) Tj ET
	BT 283 558 Td (\(10,10\)) Tj ET
	BT 425 416 Td (\(15,15\)) Tj ET
	0.878 0.235 0.192 rg
	0.878 0.235 0.192 RG
	141.732 697.323 2.835 2.835 re f
	283.465 555.591 2.835 2.835 re f
	425.197 413.858 2.835 2.835 re f
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
	0000000576 00000 n
	trailer
	<</Size 6/Root 1 0 R>>
	startxref
	682
	%%EOF
	`
	pdfCompare(t, doc.Bytes(), want)
}

// Test_PDF_DrawTextInBox_ is the unit test for
// DrawTextInBox(
//
//	x, y, width, height float64, align, text string) *PDF
func Test_PDF_DrawTextInBox_(t *testing.T) {
	const (
		lorem = "Lorem ipsum dolor sit amet," +
			" consectetur adipiscing elit," +
			" sed do eiusmod tempor incididunt ut" +
			" labore et dolore magna aliqua." +
			" Ut enim ad minim veniam," +
			" quis nostrud exercitation ullamco laboris" +
			" nisi ut aliquip ex ea commodo consequat." +
			" Duis aute irure dolor in reprehenderit in voluptate velit" +
			" esse cillum dolore eu fugiat nulla pariatur." +
			" Excepteur sint occaecat cupidatat non proident," +
			" sunt in culpa qui officia deserunt mollit anim id est laborum."
	)
	func() {
		doc := NewPDF("A4")
		{
			doc.SetCompression(false).
				SetUnits("cm").
				SetFont("Helvetica", 10).
				SetColor("Light Gray").
				FillBox(5, 5, 3, 15).
				SetColor("Black").
				DrawTextInBox(5, 5, 3, 15, "C", "").
				DrawTextInBox(5, 5, 3, 15, "C", lorem)
		}
		const want = `
		%PDF-1.4
		1 0 obj <</Type/Catalog/Pages 2 0 R>>
		endobj
		2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 595 841]/Kids[3 0 R]>>
		endobj
		3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R
		/Resources <</Font <</FNT1 5 0 R>> >> >>
		endobj
		4 0 obj <</Length 1252>> stream
		0.827 0.827 0.827 rg
		0.827 0.827 0.827 RG
		141.732 274.961 85.039 425.197 re f
		BT /FNT1 10 Tf ET
		0.000 0.000 0.000 rg
		0.000 0.000 0.000 RG
		BT 153 624 Td (Lorem ipsum ) Tj ET
		BT 151 614 Td (dolor sit amet, ) Tj ET
		BT 157 604 Td (consectetur ) Tj ET
		BT 142 594 Td (adipiscing elit, sed ) Tj ET
		BT 157 584 Td (do eiusmod ) Tj ET
		BT 144 574 Td (tempor incididunt ) Tj ET
		BT 142 564 Td (ut labore et dolore ) Tj ET
		BT 145 554 Td (magna aliqua. Ut ) Tj ET
		BT 150 544 Td (enim ad minim ) Tj ET
		BT 154 534 Td (veniam, quis ) Tj ET
		BT 166 524 Td (nostrud ) Tj ET
		BT 157 514 Td (exercitation ) Tj ET
		BT 149 504 Td (ullamco laboris ) Tj ET
		BT 147 494 Td (nisi ut aliquip ex ) Tj ET
		BT 153 484 Td (ea commodo ) Tj ET
		BT 147 474 Td (consequat. Duis ) Tj ET
		BT 143 464 Td (aute irure dolor in ) Tj ET
		BT 147 454 Td (reprehenderit in ) Tj ET
		BT 152 444 Td (voluptate velit ) Tj ET
		BT 142 434 Td (esse cillum dolore ) Tj ET
		BT 151 424 Td (eu fugiat nulla ) Tj ET
		BT 164 414 Td (pariatur. ) Tj ET
		BT 151 404 Td (Excepteur sint ) Tj ET
		BT 162 394 Td (occaecat ) Tj ET
		BT 152 384 Td (cupidatat non ) Tj ET
		BT 147 374 Td (proident, sunt in ) Tj ET
		BT 148 364 Td (culpa qui officia ) Tj ET
		BT 150 354 Td (deserunt mollit ) Tj ET
		BT 139 344 Td (anim id est laborum.) Tj ET
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
		0000001532 00000 n
		trailer
		<</Size 6/Root 1 0 R>>
		startxref
		1633
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
	}()
	func() {
		doc := NewPDF("A4")
		doc.SetCompression(false).
			SetUnits("cm").
			SetFont("Courier", 10).
			SetColor("Light Gray").
			FillBox(5, 5, 3, 15).
			SetColor("Black").
			DrawTextInBox(5, 5, 3, 15, "C", "").
			DrawTextInBox(5, 5, 3, 15, "C", lorem)
		const want = `
		%PDF-1.4
		1 0 obj <</Type/Catalog/Pages 2 0 R>>
		endobj
		2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 595 841]/Kids[3 0 R]>>
		endobj
		3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R
		/Resources <</Font <</FNT1 5 0 R>> >> >>
		endobj
		4 0 obj <</Length 1505>> stream
		0.827 0.827 0.827 rg
		0.827 0.827 0.827 RG
		141.732 274.961 85.039 425.197 re f
		BT /FNT1 10 Tf ET
		0.000 0.000 0.000 rg
		0.000 0.000 0.000 RG
		BT 148 679 Td (Lorem ipsum ) Tj ET
		BT 154 669 Td (dolor sit ) Tj ET
		BT 166 659 Td (amet, ) Tj ET
		BT 148 649 Td (consectetur ) Tj ET
		BT 151 639 Td (adipiscing ) Tj ET
		BT 145 629 Td (elit, sed do ) Tj ET
		BT 160 619 Td (eiusmod ) Tj ET
		BT 163 609 Td (tempor ) Tj ET
		BT 142 599 Td (incididunt ut ) Tj ET
		BT 154 589 Td (labore et ) Tj ET
		BT 145 579 Td (dolore magna ) Tj ET
		BT 151 569 Td (aliqua. Ut ) Tj ET
		BT 142 559 Td (enim ad minim ) Tj ET
		BT 145 549 Td (veniam, quis ) Tj ET
		BT 160 539 Td (nostrud ) Tj ET
		BT 145 529 Td (exercitation ) Tj ET
		BT 160 519 Td (ullamco ) Tj ET
		BT 145 509 Td (laboris nisi ) Tj ET
		BT 142 499 Td (ut aliquip ex ) Tj ET
		BT 151 489 Td (ea commodo ) Tj ET
		BT 151 479 Td (consequat. ) Tj ET
		BT 154 469 Td (Duis aute ) Tj ET
		BT 148 459 Td (irure dolor ) Tj ET
		BT 175 449 Td (in ) Tj ET
		BT 142 439 Td (reprehenderit ) Tj ET
		BT 145 429 Td (in voluptate ) Tj ET
		BT 151 419 Td (velit esse ) Tj ET
		BT 142 409 Td (cillum dolore ) Tj ET
		BT 154 399 Td (eu fugiat ) Tj ET
		BT 166 389 Td (nulla ) Tj ET
		BT 154 379 Td (pariatur. ) Tj ET
		BT 154 369 Td (Excepteur ) Tj ET
		BT 142 359 Td (sint occaecat ) Tj ET
		BT 142 349 Td (cupidatat non ) Tj ET
		BT 154 339 Td (proident, ) Tj ET
		BT 142 329 Td (sunt in culpa ) Tj ET
		BT 148 319 Td (qui officia ) Tj ET
		BT 157 309 Td (deserunt ) Tj ET
		BT 148 299 Td (mollit anim ) Tj ET
		BT 139 289 Td (id est laborum.) Tj ET
		endstream
		endobj
		5 0 obj <</Type/Font/Subtype/Type1/Name/FNT1
		/BaseFont/Courier
		/Encoding/StandardEncoding>>
		endobj
		xref
		0 6
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000228 00000 n
		0000001785 00000 n
		trailer
		<</Size 6/Root 1 0 R>>
		startxref
		1884
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
	}()
}

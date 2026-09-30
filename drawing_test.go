package pdf

import (
	"fmt"

	"strconv"

	"testing"
)

// Test_PDF_DrawBox_ is the unit test for
// PDF.DrawBox(x, y, width, height float64, fill ...bool) *PDF
//
// Runs the test by drawing three rectangles and one filled rectangle
func Test_PDF_DrawBox_(t *testing.T) {
	var (
		doc = NewPDF("18cm x 18cm")
		x   = 1.0
		y   = 1.0
	)
	{
		doc.SetCompression(false).
			SetUnits("cm").
			SetLineWidth(5).
			SetColor("Black").DrawBox(x, y, 1, 1, true).
			SetColor("Red").DrawBox(x, y, 4, 4).
			SetColor("DarkGreen").DrawBox(x, y, 9, 9).
			SetColor("Blue").DrawBox(x, y, 16, 16)
	}
	const want = `
	%PDF-1.4
	1 0 obj <</Type/Catalog/Pages 2 0 R>>
	endobj
	2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 510 510]/Kids[3 0 R]>>
	endobj
	3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R>>
	endobj
	4 0 obj <</Length 255>> stream
	0.000 0.000 0.000 rg
	0.000 0.000 0.000 RG
	5.000 w
	28.346 453.543 28.346 28.346 re f
	1.000 0.000 0.000 RG
	28.346 368.504 113.386 113.386 re S
	0.000 0.392 0.000 RG
	28.346 226.772 255.118 255.118 re S
	0.000 0.000 1.000 RG
	28.346 28.346 453.543 453.543 re S
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
	495
	%%EOF
	`
	pdfCompare(t, doc.Bytes(), want)
}

// Test_PDF_DrawCircle_ is the unit test for
// PDF.DrawCircle(x, y, radius float64, fill ...bool) *PDF
//
// Runs the test by drawing three concentric
// circles and one small filled circle
func Test_PDF_DrawCircle_(t *testing.T) {
	var (
		doc  = NewPDF("20cm x 20cm")
		x, y = 10.0, 10.0 // center of page
	)
	{
		doc.SetCompression(false).
			SetUnits("cm").
			SetLineWidth(5).
			SetColor("Black").DrawCircle(x, y, 0.5, true).
			SetColor("Red").DrawCircle(x, y, 2).
			SetColor("DarkGreen").DrawCircle(x, y, 4.5).
			SetColor("Blue").DrawCircle(x, y, 8.5)
	}
	const want = `
	%PDF-1.4
	1 0 obj <</Type/Catalog/Pages 2 0 R>>
	endobj
	2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 566 566]/Kids[3 0 R]>>
	endobj
	3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R>>
	endobj
	4 0 obj <</Length 1003>> stream
	0.000 0.000 0.000 rg
	0.000 0.000 0.000 RG
	5.000 w
	269.291 283.465 m
	269.291 291.292 275.637 297.638 283.465 297.638 c
	291.292 297.638 297.638 291.292 297.638 283.465 c
	297.638 275.637 291.292 269.291 283.465 269.291 c
	275.637 269.291 269.291 275.637 269.291 283.465 c
	f
	1.000 0.000 0.000 RG
	226.772 283.465 m
	226.772 314.775 252.154 340.157 283.465 340.157 c
	314.775 340.157 340.157 314.775 340.157 283.465 c
	340.157 252.154 314.775 226.772 283.465 226.772 c
	252.154 226.772 226.772 252.154 226.772 283.465 c
	S
	0.000 0.392 0.000 RG
	155.906 283.465 m
	155.906 353.913 213.016 411.024 283.465 411.024 c
	353.913 411.024 411.024 353.913 411.024 283.465 c
	411.024 213.016 353.913 155.906 283.465 155.906 c
	213.016 155.906 155.906 213.016 155.906 283.465 c
	S
	0.000 0.000 1.000 RG
	42.520 283.465 m
	42.520 416.535 150.394 524.409 283.465 524.409 c
	416.535 524.409 524.409 416.535 524.409 283.465 c
	524.409 150.394 416.535 42.520 283.465 42.520 c
	150.394 42.520 42.520 150.394 42.520 283.465 c
	S
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
	1244
	%%EOF
	`
	pdfCompare(t, doc.Bytes(), want)
}

// Test_PDF_FillBox_ is the unit test for
// PDF.FillBox(x, y, width, height float64) *PDF
//
// Runs the test by filling the shape of a Monolith from 2001 Space Odyssey
func Test_PDF_FillBox_(t *testing.T) {
	var (
		doc    = NewPDF("A4")
		x      = 6.5
		y      = 6.0
		width  = 8.0
		height = 18.0
	)
	doc.SetCompression(false).
		SetUnits("cm").
		SetColor("#1B1B1B EerieBlack").
		FillBox(x, y, width, height)
	const want = `
	%PDF-1.4
	1 0 obj <</Type/Catalog/Pages 2 0 R>>
	endobj
	2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 595 841]/Kids[3 0 R]>>
	endobj
	3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R>>
	endobj
	4 0 obj <</Length 80>> stream
	0.106 0.106 0.106 rg
	0.106 0.106 0.106 RG
	184.252 161.575 226.772 510.236 re f
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
	319
	%%EOF
	`
	pdfCompare(t, doc.Bytes(), want)
}

// Test_PDF_FillCircle_ is the unit test for
// PDF.FillCircle(x, y, radius float64, fill ...bool) *PDF
//
// Runs the test by drawing the flag of Japan using correct proportions
func Test_PDF_FillCircle_(t *testing.T) {
	var (
		doc    = NewPDF("30cm x 20cm")
		x, y   = 15.0, 10.0         // center of page
		radius = (20.0 * 3 / 5) / 2 // diameter = 3/5 of height
	)
	doc.SetCompression(false).
		SetUnits("cm").
		SetColor("#BC002D (close to #BE0032 CrimsonGlory)").
		FillCircle(x, y, radius)
	const want = `
	%PDF-1.4
	1 0 obj <</Type/Catalog/Pages 2 0 R>>
	endobj
	2 0 obj <</Type/Pages/Count 1/MediaBox[0 0 850 566]/Kids[3 0 R]>>
	endobj
	3 0 obj <</Type/Page/Parent 2 0 R/Contents 4 0 R>>
	endobj
	4 0 obj <</Length 267>> stream
	0.737 0.000 0.176 rg
	0.737 0.000 0.176 RG
	255.118 283.465 m
	255.118 377.396 331.265 453.543 425.197 453.543 c
	519.129 453.543 595.276 377.396 595.276 283.465 c
	595.276 189.533 519.129 113.386 425.197 113.386 c
	331.265 113.386 255.118 189.533 255.118 283.465 c
	f
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
	507
	%%EOF
	`
	pdfCompare(t, doc.Bytes(), want)
}

// Test_PDF_LineWidth_ is the unit test for PDF.LineWidth()
// go test --run Test_PDF_LineWidth_
func Test_PDF_LineWidth_(t *testing.T) {

	func() {
		var doc PDF
		tEqual(t, doc.LineWidth(), 1)
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.LineWidth(), 1)
	}()

	func() {
		var doc PDF
		doc.SetLineWidth(42)
		tEqual(t, doc.LineWidth(), 42)
	}()
	func() {
		doc := NewPDF("A4")
		doc.SetLineWidth(7)
		tEqual(t, doc.LineWidth(), 7)
	}()

	func() {
		doc := NewPDF("A4")
		doc.SetCompression(false).
			SetUnits("cm").
			SetXY(1, 1.5).SetColor("Indigo").
			SetFont("Helvetica", 16).
			DrawText("Test PDF.LineWidth()").
			SetFont("Helvetica", 9)
		y := 2.0
		for _, w := range []float64{0.1, 0.2, 0.5, 1, 5, 10, 15, 20, 25} {
			doc.SetColor("Dark Gray").
				SetXY(1.0, y+0.3).DrawText(" y = "+strconv.Itoa(int(y))).
				SetXY(2.3, y+0.3).DrawText(" w = "+fmt.Sprintf("%0.1f", w)).
				SetColor("Gray").SetLineWidth(0.1).DrawLine(1, y, 20, y).
				SetColor("Indigo").SetLineWidth(w).DrawLine(4, y, 15, y)
			y += 1
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
		4 0 obj <</Length 2617>> stream
		BT /FNT1 16 Tf ET
		0.294 0.000 0.510 rg
		0.294 0.000 0.510 RG
		BT 28 799 Td (Test PDF.LineWidth\(\)) Tj ET
		BT /FNT1 9 Tf ET
		0.663 0.663 0.663 rg
		0.663 0.663 0.663 RG
		BT 28 776 Td ( y = 2) Tj ET
		BT 65 776 Td ( w = 0.1) Tj ET
		0.745 0.745 0.745 rg
		0.745 0.745 0.745 RG
		0.100 w
		28.346 785.197 m 566.929 785.197 l S
		0.294 0.000 0.510 rg
		0.294 0.000 0.510 RG
		113.386 785.197 m 425.197 785.197 l S
		0.663 0.663 0.663 rg
		0.663 0.663 0.663 RG
		BT 28 748 Td ( y = 3) Tj ET
		BT 65 748 Td ( w = 0.2) Tj ET
		0.745 0.745 0.745 rg
		0.745 0.745 0.745 RG
		28.346 756.850 m 566.929 756.850 l S
		0.294 0.000 0.510 rg
		0.294 0.000 0.510 RG
		0.200 w
		113.386 756.850 m 425.197 756.850 l S
		0.663 0.663 0.663 rg
		0.663 0.663 0.663 RG
		BT 28 720 Td ( y = 4) Tj ET
		BT 65 720 Td ( w = 0.5) Tj ET
		0.745 0.745 0.745 rg
		0.745 0.745 0.745 RG
		0.100 w
		28.346 728.504 m 566.929 728.504 l S
		0.294 0.000 0.510 rg
		0.294 0.000 0.510 RG
		0.500 w
		113.386 728.504 m 425.197 728.504 l S
		0.663 0.663 0.663 rg
		0.663 0.663 0.663 RG
		BT 28 691 Td ( y = 5) Tj ET
		BT 65 691 Td ( w = 1.0) Tj ET
		0.745 0.745 0.745 rg
		0.745 0.745 0.745 RG
		0.100 w
		28.346 700.157 m 566.929 700.157 l S
		0.294 0.000 0.510 rg
		0.294 0.000 0.510 RG
		1.000 w
		113.386 700.157 m 425.197 700.157 l S
		0.663 0.663 0.663 rg
		0.663 0.663 0.663 RG
		BT 28 663 Td ( y = 6) Tj ET
		BT 65 663 Td ( w = 5.0) Tj ET
		0.745 0.745 0.745 rg
		0.745 0.745 0.745 RG
		0.100 w
		28.346 671.811 m 566.929 671.811 l S
		0.294 0.000 0.510 rg
		0.294 0.000 0.510 RG
		5.000 w
		113.386 671.811 m 425.197 671.811 l S
		0.663 0.663 0.663 rg
		0.663 0.663 0.663 RG
		BT 28 634 Td ( y = 7) Tj ET
		BT 65 634 Td ( w = 10.0) Tj ET
		0.745 0.745 0.745 rg
		0.745 0.745 0.745 RG
		0.100 w
		28.346 643.465 m 566.929 643.465 l S
		0.294 0.000 0.510 rg
		0.294 0.000 0.510 RG
		10.000 w
		113.386 643.465 m 425.197 643.465 l S
		0.663 0.663 0.663 rg
		0.663 0.663 0.663 RG
		BT 28 606 Td ( y = 8) Tj ET
		BT 65 606 Td ( w = 15.0) Tj ET
		0.745 0.745 0.745 rg
		0.745 0.745 0.745 RG
		0.100 w
		28.346 615.118 m 566.929 615.118 l S
		0.294 0.000 0.510 rg
		0.294 0.000 0.510 RG
		15.000 w
		113.386 615.118 m 425.197 615.118 l S
		0.663 0.663 0.663 rg
		0.663 0.663 0.663 RG
		BT 28 578 Td ( y = 9) Tj ET
		BT 65 578 Td ( w = 20.0) Tj ET
		0.745 0.745 0.745 rg
		0.745 0.745 0.745 RG
		0.100 w
		28.346 586.772 m 566.929 586.772 l S
		0.294 0.000 0.510 rg
		0.294 0.000 0.510 RG
		20.000 w
		113.386 586.772 m 425.197 586.772 l S
		0.663 0.663 0.663 rg
		0.663 0.663 0.663 RG
		BT 28 549 Td ( y = 10) Tj ET
		BT 65 549 Td ( w = 25.0) Tj ET
		0.745 0.745 0.745 rg
		0.745 0.745 0.745 RG
		0.100 w
		28.346 558.425 m 566.929 558.425 l S
		0.294 0.000 0.510 rg
		0.294 0.000 0.510 RG
		25.000 w
		113.386 558.425 m 425.197 558.425 l S
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
		0000002897 00000 n
		trailer
		<</Size 6/Root 1 0 R>>
		startxref
		2998
		%%EOF
        `
		pdfCompare(t, doc.Bytes(), want)
	}()
}

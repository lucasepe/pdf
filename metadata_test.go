package pdf

import "testing"

// Test_PDF_DocAuthor_ is the unit test for
func Test_PDF_DocAuthor_(t *testing.T) {

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.DocAuthor(), "")
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.DocAuthor(), "")
	}()

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.SetDocAuthor("Abcdefg").DocAuthor(), "Abcdefg")
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.SetDocAuthor("Abcdefg").DocAuthor(), "Abcdefg")
	}()

	func() {
		doc := NewPDF("A4")
		doc.SetCompression(false).SetDocAuthor("'Author' metadata entry")
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
		5 0 obj <</Type/Info/Author ('Author' metadata entry)>>
		endobj
		xref
		0 6
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000189 00000 n
		0000000238 00000 n
		trailer
		<</Size 6/Root 1 0 R/Info 5 0 R>>
		startxref
		302
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
	}()
}

// Test_PDF_DocCreator_ is the unit test for
func Test_PDF_DocCreator_(t *testing.T) {

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.DocCreator(), "")
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.DocCreator(), "")
	}()

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.SetDocCreator("Abcdefg").DocCreator(), "Abcdefg")
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.SetDocCreator("Abcdefg").DocCreator(), "Abcdefg")
	}()

	func() {
		doc := NewPDF("A4")
		doc.SetCompression(false).SetDocCreator("'Creator' metadata entry")
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
		5 0 obj <</Type/Info/Creator ('Creator' metadata entry)>>
		endobj
		xref
		0 6
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000189 00000 n
		0000000238 00000 n
		trailer
		<</Size 6/Root 1 0 R/Info 5 0 R>>
		startxref
		304
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
	}()
}

// Test_PDF_DocKeywords_ is the unit test for
func Test_PDF_DocKeywords_(t *testing.T) {

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.DocKeywords(), "")
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.DocKeywords(), "")
	}()

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.SetDocKeywords("Abcdefg").DocKeywords(), "Abcdefg")
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.SetDocKeywords("Abcdefg").DocKeywords(), "Abcdefg")
	}()

	func() {
		doc := NewPDF("A4")
		doc.SetCompression(false).SetDocKeywords("'Keywords' metadata entry")
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
		5 0 obj <</Type/Info/Keywords ('Keywords' metadata entry)>>
		endobj
		xref
		0 6
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000189 00000 n
		0000000238 00000 n
		trailer
		<</Size 6/Root 1 0 R/Info 5 0 R>>
		startxref
		306
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
	}()
}

// Test_PDF_DocSubject_ is the unit test for
func Test_PDF_DocSubject_(t *testing.T) {

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.DocSubject(), "")
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.DocSubject(), "")
	}()

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.SetDocSubject("Abcdefg").DocSubject(), "Abcdefg")
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.SetDocSubject("Abcdefg").DocSubject(), "Abcdefg")
	}()

	func() {
		doc := NewPDF("A4")
		doc.SetCompression(false).SetDocSubject("'Subject' metadata entry")
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
		5 0 obj <</Type/Info/Subject ('Subject' metadata entry)>>
		endobj
		xref
		0 6
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000189 00000 n
		0000000238 00000 n
		trailer
		<</Size 6/Root 1 0 R/Info 5 0 R>>
		startxref
		304
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
	}()
}

// Test_PDF_DocTitle_ is the unit test for
func Test_PDF_DocTitle_(t *testing.T) {

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.DocTitle(), "")
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.DocTitle(), "")
	}()

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.SetDocTitle("Abcdefg").DocTitle(), "Abcdefg")
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.SetDocTitle("Abcdefg").DocTitle(), "Abcdefg")
	}()

	func() {
		doc := NewPDF("A4")
		doc.SetCompression(false).SetDocTitle("'Title' metadata entry")
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
		5 0 obj <</Type/Info/Title ('Title' metadata entry)>>
		endobj
		xref
		0 6
		0000000000 65535 f
		0000000010 00000 n
		0000000056 00000 n
		0000000130 00000 n
		0000000189 00000 n
		0000000238 00000 n
		trailer
		<</Size 6/Root 1 0 R/Info 5 0 R>>
		startxref
		300
		%%EOF
		`
		pdfCompare(t, doc.Bytes(), want)
	}()
}

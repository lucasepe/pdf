package pdf

import (
	"fmt"

	"testing"
)

// Test_PDF_Clean_ is the unit test for PDF.Clean()
func Test_PDF_Clean_(t *testing.T) {

	func() {
		var doc PDF // uninitialized PDF
		doc.Clean().Clean().Clean()

		tEqual(t, len(doc.Errors()), 0)
		tEqual(t, doc.Errors(), []error{})
	}()

	func() {
		doc := NewPDF("A4")
		doc.Clean()
		doc.Clean()
		doc.Clean()

		tEqual(t, len(doc.Errors()), 0)
		tEqual(t, doc.Errors(), []error{})
	}()

	func() {
		doc := NewPDF("Parchment")

		tEqual(t, len(doc.Errors()), 1)
		doc.Clean()

		tEqual(t, len(doc.Errors()), 0)
		tEqual(t, doc.Errors(), []error{})
	}()

}

// Test_PDF_Errors_ tests PDF.Errors()
func Test_PDF_Errors_(t *testing.T) {

	func() {
		var doc PDF // uninitialized PDF

		tEqual(t, len(doc.Errors()), 0)
		tEqual(t, doc.Errors(), []error{})
	}()

	func() {
		doc := NewPDF("A4")
		tEqual(t, len(doc.Errors()), 0)
		tEqual(t, doc.Errors(), []error{})
	}()
}

// Test_PDF_PullError_ is the unit test for PullError() error
func Test_PDF_PullError_(t *testing.T) {
	func() {
		doc := NewPDF("Papyrus")

		tEqual(t, len(doc.Errors()), 1)

		err := doc.PullError()
		tEqual(t, err, fmt.Errorf(`Unknown paper size "Papyrus" @NewPDF`))

		tEqual(t, len(doc.Errors()), 0)

		err = doc.PullError()
		tEqual(t, len(doc.Errors()), 0)
		tEqual(t, err, nil)
	}()
}

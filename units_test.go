package pdf

import (
	"fmt"

	"testing"
)

// Test_PDF_ToPoints_ is the unit test for PDF.ToPoints()
func Test_PDF_ToPoints_(t *testing.T) {

	test := func(wantVal float64, wantErr error, inputParts ...[]string) {
		for _, s := range permuteStrings(inputParts...) {
			var doc PDF // uninitialized PDF
			gotVal, gotErr := doc.ToPoints(s)
			tEqual(t,
				fmt.Sprintf("%0.03f", gotVal),
				fmt.Sprintf("%0.03f", wantVal),
			)
			tEqual(t, gotErr, wantErr)
		}
	}
	var (
		cm     = []string{"CM", "Cm", "cM", "cm"}
		inches = []string{
			"IN", "INCH", "INCHES",
			"In", "Inch", "Inches",
			"in", "inch", "inches",
			`"`,
		}
		mm     = []string{"MM", "mm", "Mm", "mM"}
		points = []string{
			"PT", "POINT", "POINTS",
			"Pt", "Point", "Points",
			"pt", "point", "points",
		}
		twips = []string{
			"TW", "TWIP", "TWIPS",
			"Tw", "Twip", "Twips",
			"tw", "twip", "twips",
		}
		spc = []string{
			"", " ", "  ", "\t",
		}
	)

	test(123, nil, spc, []string{"123"}, spc)

	one := []string{"1"}
	test(72, nil, spc, one, spc, inches, spc)
	test(2.835, nil, spc, one, spc, mm, spc)
	test(28.346, nil, spc, one, spc, cm, spc)
	test(0.050, nil, spc, one, spc, twips, spc)
	test(1, nil, spc, one, spc, points, spc)

	negative := []string{"-12.345"}
	test(-888.840, nil, spc, negative, spc, inches, spc)
	test(-34.994, nil, spc, negative, spc, mm, spc)
	test(-349.937, nil, spc, negative, spc, cm, spc)
	test(-0.617, nil, spc, negative, spc, twips, spc)
	test(-12.345, nil, spc, negative, spc, points, spc)

	test(1, nil, spc, []string{"20"}, spc, twips, spc)
	test(-1, nil, spc, []string{"-20"}, spc, twips, spc)

	test(0, fmt.Errorf(`Unknown measurement units "km"`), []string{"1km"})
	test(0, fmt.Errorf(`Invalid number "1.0.1"`), []string{"1.0.1mm"})
}

// Test_PDF_ToUnits_ is the unit test for
// ToUnits(points float64) float64
func Test_PDF_ToUnits_(t *testing.T) {
	func() {
		var doc PDF
		tEqual(t, doc.ToUnits(1), 1)

		doc.SetUnits("cm")
		tEqual(t, doc.ToUnits(1), 0.035278)
		tEqual(t, doc.ToUnits(28.3464566929134), 1)

		doc.SetUnits("in")
		tEqual(t, doc.ToUnits(1), 0.0138888888888889)
		tEqual(t, doc.ToUnits(72), 1)

		doc.SetUnits("mm")
		tEqual(t, doc.ToUnits(1), 0.3527777777777776)
		tEqual(t, doc.ToUnits(2.83464566929134), 1)

		doc.SetUnits("point")
		tEqual(t, doc.ToUnits(1), 1)

		doc.SetUnits("twip")
		tEqual(t, doc.ToUnits(1), 20)
		tEqual(t, doc.ToUnits(0.05), 1)
	}()
}

// Test_PDF_Units_ tests PDF.Units() and SetUnits()
func Test_PDF_Units_(t *testing.T) {

	func() {
		var doc PDF // uninitialized PDF
		tEqual(t, doc.Units(), "POINT")
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, doc.Units(), "POINT")
	}()

	func() {
		doc := NewPDF("A4")
		tEqual(t, len(doc.Errors()), 0)
		doc.SetUnits("cm")
		tEqual(t, len(doc.Errors()), 0)
		tEqual(t, doc.Units(), "CM")
	}()
	func() {
		doc := NewPDF("A4")
		tEqual(t, len(doc.Errors()), 0)
		doc.SetUnits("fathoms")
		tEqual(t, len(doc.Errors()), 1)

		if len(doc.Errors()) == 1 {
			tEqual(t,
				doc.Errors()[0],
				fmt.Errorf(`Unknown measurement units "fathoms" @SetUnits`))
		}
		tEqual(t, doc.Units(), "POINT")
	}()
}

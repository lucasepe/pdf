package pdf

import (
	"strconv"
	"unicode"
)

// Units returns the currently selected measurement units.
// E.g.: mm cm " in inch inches tw twip twips pt point points
func (p *PDF) Units() string { p.init(); return p.units }

// SetUnits changes the current measurement units:
// mm cm " in inch inches tw twip twips pt point points (can be in any case)
func (p *PDF) SetUnits(units string) *PDF {
	ppu, err := p.init().getPointsPerUnit(units)
	if err, isT := err.(pdfError); isT {
		return p.putError(0xEB4AAA, err.msg, units)
	}
	p.ptPerUnit, p.units = ppu, p.toUpperLettersDigits(units, "")
	return p
}

// ToPoints converts a string composed of a number and unit to points.
// For example '1 cm' or '1cm' becomes 28.346 points.
// Recognised units: mm cm " in inch inches tw twip twips pt point points
func (p *PDF) ToPoints(numberAndUnit string) (float64, error) {
	var num, unit string //                              extract number and unit
	for _, r := range numberAndUnit {
		switch {
		case r == '-', r == '.', unicode.IsDigit(r):
			num += string(r)
		case r == '"', unicode.IsLetter(r):
			unit += string(r)
		}
	}
	ppu := 1.0
	if unit != "" {
		var err error
		ppu, err = p.getPointsPerUnit(unit)
		if err, isT := err.(pdfError); isT {
			return 0, err
		}
	}
	n, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0, pdfError{id: 0xE154AC, msg: "Invalid number", val: num}
	}
	return n * ppu, nil
}

// ToUnits converts points to the currently selected unit of measurement.
func (p *PDF) ToUnits(points float64) float64 {
	if int(p.ptPerUnit*100) == 0 {
		return points
	}
	return points / p.ptPerUnit
}

// getPointsPerUnit returns number of points per named measurement unit
func (p *PDF) getPointsPerUnit(units string) (ret float64, err error) {
	switch p.toUpperLettersDigits(units, `"`) {
	case "CM":
		ret = 28.3464566929134
	case "IN", "INCH", "INCHES", `"`:
		ret = 72.0
	case "MM":
		ret = 2.83464566929134
	case "PT", "POINT", "POINTS":
		ret = 1.0
	case "TW", "TWIP", "TWIPS":
		ret = 0.05
	default:
		err = pdfError{id: 0xEE34DA, msg: "Unknown measurement units",
			val: units}
	}
	return ret, err
}

package pdf

import (
	"bytes"
	"strings"
	"unicode"
)

// pdfPaperSize represents a page size name and its dimensions in points
type pdfPaperSize struct {
	name              string  // paper size: e.g. 'Letter', 'A4', etc.
	widthPt, heightPt float64 // width and height in points
} //                                                                pdfPaperSize

// toUpperLettersDigits returns letters and digits from s, in upper case
func (*PDF) toUpperLettersDigits(s, extras string) string {
	buf := bytes.NewBuffer(make([]byte, 0, len(s)))
	for _, r := range strings.ToUpper(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) ||
			strings.ContainsRune(extras, r) {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// getPaperSize returns a pdfPaperSize based on the specified paper name.
// Specify custom paper sizes using "width x height", e.g. "9cm x 20cm"
// If the paper size is not found, returns a zero-initialized structure
func (p *PDF) getPaperSize(name string) (pdfPaperSize, error) {
	s := strings.ToUpper(name)
	if strings.Contains(s, " X ") {
		wh := strings.Split(s, " X ")
		w, err := p.ToPoints(wh[0])
		if err, isT := err.(pdfError); isT {
			return pdfPaperSize{}, err
		}
		var h float64
		h, err = p.ToPoints(wh[1])
		if err, isT := err.(pdfError); isT {
			return pdfPaperSize{}, err
		}
		return pdfPaperSize{s, w, h}, nil
	}
	s = p.toUpperLettersDigits(s, "-")
	landscape := strings.HasSuffix(s, "-L")
	s = p.toUpperLettersDigits(s, "")
	if landscape {
		s = s[:len(s)-1]
	}
	wh, found := pdfStandardPaperSizes[s]
	if !found {
		return pdfPaperSize{},
			pdfError{id: 0xEE42FB, msg: "Unknown paper size", val: name}
	}

	w, h := float64(wh[0])/25.4*72, float64(wh[1])/25.4*72
	if landscape {
		return pdfPaperSize{s + "-L", h, w}, nil
	}
	return pdfPaperSize{s, w, h}, nil
}

// pdfStandardPaperSizes contains standard paper sizes in mm (width x height)
var pdfStandardPaperSizes = map[string][2]int{
	"A0": {841, 1189}, "B0": {1000, 1414}, "C0": {917, 1297},
	"A1": {594, 841}, "B1": {707, 1000}, "C1": {648, 917},
	"A2": {420, 594}, "B2": {500, 707}, "C2": {458, 648},
	"A3": {297, 420}, "B3": {353, 500}, "C3": {324, 458},
	"A4": {210, 297}, "B4": {250, 353}, "C4": {229, 324},
	"A5": {148, 210}, "B5": {176, 250}, "C5": {162, 229},
	"A6": {105, 148}, "B6": {125, 176}, "C6": {114, 162},
	"A7": {74, 105}, "B7": {88, 125}, "C7": {81, 114},
	"A8": {52, 74}, "B8": {62, 88}, "C8": {57, 81},
	"A9": {37, 52}, "B9": {44, 62}, "C9": {40, 57},
	"A10": {26, 37}, "B10": {31, 44}, "C10": {28, 40},
	"LEGAL": {216, 356}, "TABLOID": {279, 432},
	"LETTER": {216, 279}, "LEDGER": {432, 279},
} //                                                       pdfStandardPaperSizes

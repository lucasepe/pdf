// Package pdf provides a compact pure-Go PDF drawing and layout library.
package pdf

import (
	"bytes"
	"image/color"
	"io"
)

// PDF is the main structure representing a PDF document.
type PDF struct {
	paperSize       pdfPaperSize          // paper size used in this PDF
	pageNo          int                   // current page number
	page            *pdfPage              // pointer to the current page
	pages           []pdfPage             // all the pages added to this PDF
	fonts           []pdfFont             // all the fonts used in this PDF
	registeredFonts map[string]*pdfTTFont // caller-supplied embedded fonts
	fontRoles       map[FontRole]string   // semantic document font roles
	emojiProvider   EmojiProvider         // inline raster emoji resolver
	destinations    map[string]pdfDestination
	images          []pdfImage   // all the images used in this PDF
	columnWidths    []float64    // user-set column widths (like tab stops)
	columnNo        int          // number of the current column
	units           string       // name of active measurement unit
	ptPerUnit       float64      // number of points per measurement unit
	color           color.RGBA   // current drawing color
	lineWidth       float64      // current line width (in points)
	font            *pdfFont     // currently selected font
	fontName        string       // current font's name
	fontSizePt      float64      // current font's size (in points)
	horzScaling     uint16       // horizontal scaling factor (in %)
	compression     bool         // enable stream compression?
	content         bytes.Buffer // content buffer where PDF is written
	writer          io.Writer    // writer to PDF buffer or current page's buffer
	objOffsets      []int        // object offsets used by Bytes() and write..()
	writeErr        error        // first serialization error
	errors          []error      // errors that occurred during method calls
	isInit          bool         // has the PDF been initialized?
	//
	// document metadata fields
	docAuthor, docCreator, docKeywords, docSubject, docTitle string
} //                                                                         PDF

// NewPDF creates and initializes a new PDF object. Specify paperSize as:
// A, B, C series (e.g. "A4") or "LETTER", "LEGAL", "LEDGER", or "TABLOID"
// To specify a landscape orientation, add "-L" suffix e.g. "A4-L".
// You can also specify custom paper sizes using "width unit x height unit",
// for example "20 cm x 20 cm" or even "15cm x 10inch", etc.
func NewPDF(paperSize string) PDF {
	var p PDF
	size, err := p.init().getPaperSize(paperSize)
	if err, isT := err.(pdfError); isT {
		p.putError(0xE52F92, err.msg, paperSize)
		p.paperSize, _ = p.getPaperSize("A4")
		return p
	}
	p.paperSize = size
	return p
}

// pdfFontHandler provides measurement, drawing, and serialization for an
// already parsed embedded font.
type pdfFontHandler interface {
	textWidthPt(s string) (float64, error)
	writeText(s string) error
	objectCount() int
	writeFontObjects(objectIDs []int)
} //                                                              pdfFontHandler

// Reset releases all resources and resets all variables, except paper size.
func (p *PDF) Reset() *PDF {
	p.page, p.writer = nil, nil
	*p = NewPDF(p.paperSize.name)
	return p
}

// init initializes the PDF object, if not initialized already
func (p *PDF) init() *PDF {
	if p.isInit {
		return p
	}
	p.units = "POINT"
	p.paperSize, _ = p.getPaperSize("A4")
	p.ptPerUnit, _ = p.getPointsPerUnit(p.units)
	p.color, p.lineWidth = pdfBlack, 1
	p.fontName, p.fontSizePt = "Helvetica", 10
	p.horzScaling, p.compression = 100, true
	p.fontRoles = map[FontRole]string{
		FontRoleMono: "Courier", FontRoleMonoBold: "Courier-Bold",
	}
	p.destinations = make(map[string]pdfDestination)
	p.isInit = true
	return p
}

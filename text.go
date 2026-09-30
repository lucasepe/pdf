package pdf

import (
	"strings"
	"unicode"
)

// FontName returns the name of the currently-active typeface.
func (p *PDF) FontName() string { p.init(); return p.fontName }

// SetFontName changes the current font, while using the
// same font size as the previous font. Use one of the
// standard font names, such as 'Helvetica'.
func (p *PDF) SetFontName(name string) *PDF {
	p.init()
	p.fontName = name
	return p
}

// FontSize returns the current font size in points.
func (p *PDF) FontSize() float64 { p.init(); return p.fontSizePt }

// SetFontSize changes the current font size in points,
// without changing the currently-selected font typeface.
func (p *PDF) SetFontSize(points float64) *PDF {
	p.init()
	p.fontSizePt = points
	return p
}

// SetFont changes the current font name and size in points.
// For the font name, use one of the standard font names, e.g. 'Helvetica'.
// This font will be used for subsequent text drawing.
func (p *PDF) SetFont(name string, points float64) *PDF {
	return p.SetFontName(name).SetFontSize(points)
}

// HorizontalScaling returns the current horizontal scaling in percent.
func (p *PDF) HorizontalScaling() uint16 { p.init(); return p.horzScaling }

// SetHorizontalScaling changes the horizontal scaling in percent.
// For example, 200 will stretch text to double its normal width.
func (p *PDF) SetHorizontalScaling(percent uint16) *PDF {
	p.init()
	p.horzScaling = percent
	return p
}

// DrawText draws a text string at the current position (X, Y).
func (p *PDF) DrawText(s string) *PDF {
	if len(p.columnWidths) == 0 {
		return p.drawTextLine(s)
	}
	x := 0.0
	for i := 0; i < p.columnNo; i, x = i+1, x+p.columnWidths[i] {
	}
	p.SetX(x).drawTextLine(s)
	if p.columnNo < len(p.columnWidths)-1 {
		p.columnNo++
		return p
	}
	return p.NextLine()
}

// DrawTextAlignedToBox draws 'text' within a rectangle specified
// by 'x', 'y', 'width' and 'height'. If 'align' is blank, the
// text is center-aligned both vertically and horizontally.
// Specify 'L' or 'R' to align the text left or right, and 'T' or
// 'B' to align the text to the top or bottom of the box.
func (p *PDF) DrawTextAlignedToBox(
	x, y, width, height float64, align, text string) *PDF {
	return p.drawTextBox(x, y, width, height, false, align, text)
}

// DrawTextAt draws text at the specified point (x, y).
func (p *PDF) DrawTextAt(x, y float64, text string) *PDF {
	return p.SetXY(x, y).DrawText(text)
}

// DrawTextInBox draws word-wrapped text within a rectangle
// specified by 'x', 'y', 'width' and 'height'. If 'align' is blank,
// the text is center-aligned both vertically and horizontally.
// Specify 'L' or 'R' to align the text left or right, and 'T' or
// 'B' to align the text to the top or bottom of the box.
func (p *PDF) DrawTextInBox(
	x, y, width, height float64, align, text string) *PDF {
	return p.drawTextBox(x, y, width, height, true, align, text)
}

// NextLine advances the text writing position to the next line.
// I.e. the Y increases by the height of the font and
// the X-coordinate is reset to zero.
func (p *PDF) NextLine() *PDF {
	p.init().reservePage()
	x, y := 0.0, p.Y()+p.ToUnits(p.FontSize())
	if len(p.columnWidths) > 0 {
		x = p.columnWidths[0]
	}
	if y > p.PageHeight() {
		p.AddPage()
		y = 0
	}
	p.columnNo = 0
	return p.SetXY(x, y)
}

// SetColumnWidths creates column positions (tab stops) along the X-axis.
// To remove all column positions, call this method without any argument.
func (p *PDF) SetColumnWidths(widths ...float64) *PDF {
	p.init()
	p.columnWidths = widths
	return p
}

// TextWidth returns the width of the text in current units.
func (p *PDF) TextWidth(s string) float64 {
	p.init().reservePage()
	return p.ToUnits(p.textWidthPt(s))
}

// WrapTextLines splits a string into multiple lines so that the text
// fits in the specified width. The text is wrapped on word boundaries.
// Newline characters ("\r" and "\n") also cause text to be split.
// You can find out the number of lines needed to wrap some
// text by checking the length of the returned array.
func (p *PDF) WrapTextLines(width float64, text string) (ret []string) {
	fit := func(s []rune, step, n int, width float64) int {
		for max := len(s); n > 0 && n <= max; {
			w := p.TextWidth(string(s[:n]))
			switch step {
			case 1, 3:
				if w <= width {
					return n
				}
				n--
				if step == 1 {
					n /= 2
				}
			case 2:
				if w > width {
					return n
				}
				n = 1 + int((float64(n) * 1.2))
			}
		}
		return 0
	}

	for _, sourceLine := range p.splitLines(text) {
		line := []rune(sourceLine)
		for p.TextWidth(string(line)) > width {
			n := len(line)
			for i := 1; i <= 3; i++ {
				n = fit(line, i, n, width)
			}

			found, max := false, n
			for n > 0 {
				if unicode.IsSpace(line[n-1]) {
					found = true
					break
				}
				n--
			}
			if !found {
				n = max
			}
			if n <= 0 {
				break
			}
			ret = append(ret, string(line[:n]))
			line = line[n:]
		}
		ret = append(ret, string(line))
	}
	return ret
}

// pdfFont represents a font name and its appearance
type pdfFont struct {
	id               int
	name             string
	builtInIndex     int
	isBold, isItalic bool
	handler          pdfFontHandler
} //                                                                     pdfFont

// applyFont writes a font change command, provided the font has
// been changed since the last operation that uses fonts.
//
// This should be called just before a font needs to be used.
// This way, if a font is picked with SetFontName() property, but
// never used to draw text, no font selection command is output.
//
// Before calling this method, the font name must be already
// set by SetFontName(), which is stored in p.font.fontName
//
// What this method does:
//   - Validates the current font name and determines if it is a
//     standard (built-in) font like Helvetica or a TrueType font.
//   - Fills the document-wide list of fonts (p.fonts).
//   - Adds items to the list of font ID's used on the current page.
func (p *PDF) applyFont() (handler pdfFontHandler, err error) {
	var (
		font  pdfFont
		name  = p.toUpperLettersDigits(p.fontName, "")
		valid = name != ""
	)
	if valid {
		valid = false
		for i, fname := range pdfFontNames {
			fname = p.toUpperLettersDigits(fname, "")
			if fname != name {
				continue
			}
			has := strings.Contains
			font = pdfFont{
				name:         pdfFontNames[i],
				builtInIndex: i,
				isBold:       has(fname, "BOLD"),
				isItalic:     has(fname, "OBLIQUE") || has(fname, "ITALIC"),
			}
			valid = true
			break
		}
	}
	if !valid {
		embedded := p.registeredFonts[name]
		if embedded != nil {
			handler = embedded
			font = pdfFont{name: name, handler: embedded}
			valid = true
		}
	}

	if !valid {
		err = pdfError{id: 0xE86819, msg: "Invalid font", val: p.fontName}
		p.fontName = "Helvetica"
		p.applyFont()
		return nil, err
	}

	for _, it := range p.fonts {
		if font.name == it.name {
			font.id = it.id
			break
		}
	}
	if font.id == 0 {
		font.id = 1 + len(p.fonts)
		p.fonts = append(p.fonts, font)
	}
	if p.page.fontID == font.id && p.page.fontSizePt == p.fontSizePt {
		return handler, err
	}
	// add the font ID to the current page, if not already referenced
	var alreadyUsedOnPage bool
	for _, id := range p.page.fontIDs {
		if id == font.id {
			alreadyUsedOnPage = true
			break
		}
	}
	if !alreadyUsedOnPage {
		p.page.fontIDs = append(p.page.fontIDs, 0)
		p.page.fontIDs[len(p.page.fontIDs)-1] = font.id
	}
	p.page.fontID = font.id
	p.page.fontSizePt = p.fontSizePt
	p.write("BT /FNT", p.page.fontID, " ", pdfNumber(p.page.fontSizePt),
		" Tf ET\n")

	return handler, err
}

// drawTextLine writes a line of text at the current coordinates to the
// current page's content stream, using a sequence of raw PDF commands
func (p *PDF) drawTextLine(s string) *PDF {
	if s == "" {
		return p
	}
	p.init().reservePage()

	handler, err := p.applyFont()
	if err, isT := err.(pdfError); isT {
		p.putError(0xEAEAC4, err.msg, err.val)
	}
	font := p.fonts[p.page.fontID-1]
	if p.page.horzScaling != p.horzScaling {
		p.page.horzScaling = p.horzScaling
		p.write("BT ", p.page.horzScaling, " Tz ET\n")

	}
	p.writeMode(true)
	if handler == nil {
		encoded, encodeErr := encodeTextForFont(s, font)
		if encodeErr != nil {
			p.errors = append(p.errors, encodeErr)
			return p
		}
		p.write("BT ", int(p.page.x), " ", int(p.page.y),
			" Td (", escapePDFBytes(encoded), ") Tj ET\n")

		p.page.x += p.textWidthBytes(encoded, font.builtInIndex)
	} else {
		if err := handler.writeText(s); err != nil {
			p.errors = append(p.errors, err)
			return p
		}
		width, err := handler.textWidthPt(s)
		if err != nil {
			p.errors = append(p.errors, err)
			return p
		}
		p.page.x += width
	}
	return p
}

// drawTextBox draws a line of text, or a word-wrapped block of text.
// align: specify up to 2 flags: L R T B to align left, right, top or bottom
// the default (blank) is C center, both vertically and horizontally
func (p *PDF) drawTextBox(x, y, width, height float64,
	wrapText bool, align, text string) *PDF {
	if text == "" {
		return p
	}
	p.reservePage()
	_, err := p.applyFont()
	if err, isT := err.(pdfError); isT {
		p.putError(0xE0737C, err.msg, err.val)
	}
	var lines []string
	if wrapText {
		lines = p.WrapTextLines(width, text)
	} else {
		lines = []string{text}
	}
	align = strings.ToUpper(align)
	lineHeight := p.FontSize()
	allLinesHeight := lineHeight * float64(len(lines))

	y, height = y*p.ptPerUnit+p.fontSizePt, height*p.ptPerUnit
	if strings.Contains(align, "B") {
		y += height - allLinesHeight - 4
	} else if !strings.Contains(align, "T") {
		y += height/2 - allLinesHeight/2 - p.fontSizePt*0.15
	}
	y = p.paperSize.heightPt - y

	x, width = x*p.ptPerUnit, width*p.ptPerUnit
	for _, line := range lines {
		off := 0.0
		if strings.Contains(align, "L") {
			off = p.fontSizePt / 6
		} else if strings.Contains(align, "R") {
			off = width - p.textWidthPt(line) - p.fontSizePt/6
		} else {
			off = width/2 - p.textWidthPt(line)/2
		}
		p.page.x, p.page.y = x+off, y
		p.drawTextLine(line)
		y -= lineHeight
	}
	return p
}

// textWidthPt returns the width of text in points
func (p *PDF) textWidthPt(s string) float64 {
	if s == "" {
		return 0
	}
	name := p.toUpperLettersDigits(p.fontName, "")
	if embedded := p.registeredFonts[name]; embedded != nil {
		width, err := embedded.textWidthPt(s)
		if err != nil {
			p.errors = append(p.errors, err)
			return 0
		}
		return width
	}
	font, ok := p.currentBuiltInFont()
	if !ok {
		p.putError(0xE86819, "Invalid font", p.fontName)
		font = pdfFont{name: "Helvetica", builtInIndex: 0}
	}
	encoded, err := encodeTextForFont(s, font)
	if err != nil {
		p.errors = append(p.errors, err)
		return 0
	}
	return p.textWidthBytes(encoded, font.builtInIndex)
}

func encodeTextForFont(s string, font pdfFont) ([]byte, error) {
	if font.name == "Symbol" || font.name == "ZapfDingbats" {
		return []byte(s), nil
	}
	encoded, err := encodeWinAnsi(s)
	if encodingErr, ok := err.(TextEncodingError); ok {
		encodingErr.Font = font.name
		err = encodingErr
	}
	return encoded, err
}

func (p *PDF) currentBuiltInFont() (pdfFont, bool) {
	return p.currentBuiltInFontNamed(p.fontName)
}

func (p *PDF) currentBuiltInFontNamed(fontName string) (pdfFont, bool) {
	name := p.toUpperLettersDigits(fontName, "")
	for i, candidate := range pdfFontNames {
		if p.toUpperLettersDigits(candidate, "") == name {
			return pdfFont{name: candidate, builtInIndex: i}, true
		}
	}
	return pdfFont{}, false
}

func (p *PDF) textWidthBytes(encoded []byte, fontIndex int) float64 {
	w := 0.0
	for _, b := range encoded {
		if fontIndex >= 0 && fontIndex <= 9 {
			w += float64(pdfFontWidths[b][fontIndex])
		} else {
			w += 600
		}
	}
	return w * p.fontSizePt / 1000.0 * float64(p.horzScaling) / 100.0
}

// isWhiteSpace returns true if all the chars. in 's' are white-spaces
func (*PDF) isWhiteSpace(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return len(s) > 0
}

// splitLines splits 's' into several lines using line breaks in 's'
func (*PDF) splitLines(s string) []string {
	split := func(lines []string, sep string) (ret []string) {
		for _, line := range lines {
			if strings.Contains(line, sep) {
				ret = append(ret, strings.Split(line, sep)...)
				continue
			}
			ret = append(ret, line)
		}
		return ret
	}
	return split(split(split([]string{s}, "\r\n"), "\r"), "\n")
}

package pdf

import (
	"bytes"
	"fmt"
	"image/color"
)

// PageCount returns the total number of pages in the document.
func (p *PDF) PageCount() int { p.reservePage(); return len(p.pages) }

// PageHeight returns the height of the current page in selected units.
func (p *PDF) PageHeight() float64 { return p.init().ToUnits(p.paperSize.heightPt) }

// PageWidth returns the width of the current page in selected units.
func (p *PDF) PageWidth() float64 { return p.init().ToUnits(p.paperSize.widthPt) }

// CurrentPage returns the current page's number, starting from 1.
func (p *PDF) CurrentPage() int { return p.pageNo + 1 }

// SetCurrentPage opens the specified page. Page numbers start from 1.
func (p *PDF) SetCurrentPage(pageNo int) *PDF {
	if pageNo < 1 || pageNo > len(p.pages) {
		p.putError(0xE65AF0, "pageNo out of range",
			fmt.Sprint("pageNo:", pageNo, " range:1..", len(p.pages)))
		return p
	}
	p.pageNo = pageNo - 1
	p.page = &p.pages[p.pageNo]
	p.writer = &p.page.content
	return p
}

// X returns the X-coordinate of the current drawing position.
func (p *PDF) X() float64 { return p.reservePage().ToUnits(p.page.x) }

// SetX changes the X-coordinate of the current drawing position.
func (p *PDF) SetX(x float64) *PDF {
	p.init().reservePage()
	p.page.x = x * p.ptPerUnit
	return p
}

// Y returns the Y-coordinate of the current drawing position.
func (p *PDF) Y() float64 {
	return p.reservePage().ToUnits(p.paperSize.heightPt - p.page.y)
}

// SetY changes the Y-coordinate of the current drawing position.
func (p *PDF) SetY(y float64) *PDF {
	p.init().reservePage()
	p.page.y = p.paperSize.heightPt - y*p.ptPerUnit
	return p
}

// SetXY changes both X- and Y-coordinates of the current drawing position.
func (p *PDF) SetXY(x, y float64) *PDF { return p.SetX(x).SetY(y) }

// AddPage appends a new blank page to the PDF and makes it the current page.
func (p *PDF) AddPage() *PDF {
	p.init()
	COLOR := color.RGBA{1, 0, 1, 0x01}
	p.pages = append(p.pages, pdfPage{
		x: -1, y: p.paperSize.heightPt + 1, lineWidth: 1,
		strokeColor: COLOR, nonStrokeColor: COLOR,
		fontSizePt: 10, horzScaling: 100,
	})
	p.pageNo = len(p.pages) - 1
	p.page = &p.pages[p.pageNo]
	p.writer = &p.page.content
	return p
}

// pdfPage holds references, state and the stream buffer for each page
type pdfPage struct {
	fontIDs, imageIDs           []int        // references to fonts and images
	x, y, lineWidth, fontSizePt float64      // current drawing state
	strokeColor, nonStrokeColor color.RGBA   // "
	fontID                      int          // "
	horzScaling                 uint16       // "
	content                     bytes.Buffer // write..() calls send output here
	links                       []pdfLink
} //                                                                     pdfPage

// reservePage ensures there is at least one page in the PDF
func (p *PDF) reservePage() *PDF {
	if len(p.pages) == 0 {
		p.AddPage()
	}
	return p
}

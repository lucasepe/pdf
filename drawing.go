package pdf

import (
	"image/color"
	"strconv"
)

// Color returns the current color, which is used for text, lines and fills.
func (p *PDF) Color() color.RGBA { p.init(); return p.color }

// SetColor sets the current color using a web/X11 color name
// (e.g. "HONEY DEW") or HTML color value such as "#191970"
// for midnight blue (#RRGGBB). The current color is used
// for subsequent text and line drawing and fills.
// If the name is unknown or invalid, sets color to black.
func (p *PDF) SetColor(nameOrHTMLColor string) *PDF {
	color, err := p.init().ToColor(nameOrHTMLColor)
	if err, isT := err.(pdfError); isT {
		p.putError(0xE5B3A5, err.msg, nameOrHTMLColor)
	}
	p.color = color
	return p
}

// SetColorRGB sets the current color using red, green and blue values.
// The current color is used for subsequent text/line drawing and fills.
func (p *PDF) SetColorRGB(r, g, b byte) *PDF {
	p.init()
	p.color = color.RGBA{r, g, b, 255}
	return p
}

// LineWidth returns the current line width in points.
func (p *PDF) LineWidth() float64 { p.init(); return p.lineWidth }

// SetLineWidth changes the line width in points.
func (p *PDF) SetLineWidth(points float64) *PDF {
	p.init()
	p.lineWidth = points
	return p
}

// DrawBox draws a rectangle of the specified width and height,
// with the top-left corner starting at point (x, y).
// To fill the rectangle, pass true in the optional optFill.
func (p *PDF) DrawBox(x, y, width, height float64, optFill ...bool) *PDF {
	p.init().reservePage()
	width, height = width*p.ptPerUnit, height*p.ptPerUnit
	x, y = x*p.ptPerUnit, p.paperSize.heightPt-y*p.ptPerUnit-height
	mode := p.writeMode(optFill...)
	return p.write(x, " ", y, " ", width, " ", height, " re ", mode, "\n")

}

// DrawCircle draws a circle of radius r centered on (x, y),
// by drawing 4 Bézier curves (PDF has no circle primitive)
// To fill the circle, pass true in the optional optFill.
func (p *PDF) DrawCircle(x, y, radius float64, optFill ...bool) *PDF {
	return p.DrawEllipse(x, y, radius, radius, optFill...)
}

// DrawEllipse draws an ellipse centered on (x, y),
// with horizontal radius xRadius and vertical radius yRadius
// by drawing 4 Bézier curves (PDF has no ellipse primitive).
// To fill the ellipse, pass true in the optional optFill.
func (p *PDF) DrawEllipse(x, y, xRadius, yRadius float64,
	optFill ...bool) *PDF {
	p.init().reservePage()
	x, y = x*p.ptPerUnit, p.paperSize.heightPt-y*p.ptPerUnit
	const ratio = 0.552284749830794 // (4/3) * tan(PI/8)
	var (
		r    = xRadius * p.ptPerUnit   // horizontal radius
		v    = yRadius * p.ptPerUnit   // vertical radius
		m, n = r * ratio, v * ratio    // ratios for control points
		mode = p.writeMode(optFill...) // prepare colors/line width
	)
	return p.write(x-r, " ", y, " m\n").
		writeCurve(x-r, y+n, x-m, y+v, x+0, y+v).
		writeCurve(x+m, y+v, x+r, y+n, x+r, y+0).
		writeCurve(x+r, y-n, x+m, y-v, x+0, y-v).
		writeCurve(x-m, y-v, x-r, y-n, x-r, y+0).
		write(mode, "\n")
}

// DrawLine draws a straight line from point (x1, y1) to point (x2, y2).
func (p *PDF) DrawLine(x1, y1, x2, y2 float64) *PDF {
	p.init().reservePage()
	x1, y1 = x1*p.ptPerUnit, p.paperSize.heightPt-y1*p.ptPerUnit
	x2, y2 = x2*p.ptPerUnit, p.paperSize.heightPt-y2*p.ptPerUnit
	p.writeMode(true)
	return p.write(x1, " ", y1, " m ", x2, " ", y2, " l S\n")

}

// DrawUnitGrid draws a light-gray grid demarcated in the
// current measurement unit. The grid fills the entire page.
// It helps with item positioning.
func (p *PDF) DrawUnitGrid() *PDF {
	pw, ph := p.PageWidth(), p.PageHeight()
	p.SetLineWidth(0.1).SetFont("Helvetica", 8)
	for i, x := 0, 0.0; x < pw; i, x = i+1, x+1 {
		p.SetColorRGB(200, 200, 200).DrawLine(x, 0, x, ph).
			SetColor("Indigo").SetXY(x+0.1, 0.3).DrawText(strconv.Itoa(i))
	}
	for i, y := 0, 0.0; y < ph; i, y = i+1, y+1 {
		p.SetColorRGB(200, 200, 200).DrawLine(0, y, pw, y).
			SetColor("Indigo").SetXY(0.1, y+0.3).DrawText(strconv.Itoa(i))
	}
	return p
}

// FillBox fills a rectangle with the current color.
func (p *PDF) FillBox(x, y, width, height float64) *PDF {
	return p.DrawBox(x, y, width, height, true)
}

// FillCircle fills a circle of radius r centered on (x, y),
// by drawing 4 Bézier curves (PDF has no circle primitive)
func (p *PDF) FillCircle(x, y, radius float64) *PDF {
	return p.DrawEllipse(x, y, radius, radius, true)
}

// FillEllipse fills a Ellipse of radius r centered on (x, y),
// by drawing 4 Bézier curves (PDF has no ellipse primitive)
func (p *PDF) FillEllipse(x, y, xRadius, yRadius float64) *PDF {
	return p.DrawEllipse(x, y, xRadius, yRadius, true)
}

// writeCurve writes a Bézier curve using the 'c' PDF primitive.
// The starting point is the current (x, y) position.
// (x1, y1) and (x2, y2) are the two control points, (x3, y3) the end point.
func (p *PDF) writeCurve(x1, y1, x2, y2, x3, y3 float64) *PDF {
	return p.write(" ", x1, " ", y1, " ", x2, " ", y2,
		" ", x3, " ", y3, " c\n")
}

// writeMode sets the stroking or non-stroking color and line width.
// 'fill' arg specifies non-stroking (true) or stroking mode (none/false)
func (p *PDF) writeMode(optFill ...bool) (mode string) {
	p.reservePage()
	mode = "S"
	if len(optFill) > 0 && optFill[0] {
		mode = "f"
		if pv := &p.page.nonStrokeColor; *pv != p.color {
			*pv = p.color
			p.write(" ", float64(pv.R)/255, " ", float64(pv.G)/255, " ",
				float64(pv.B)/255, " rg\n")
		}
	}
	if pv := &p.page.strokeColor; *pv != p.color {
		*pv = p.color
		p.write(float64(pv.R)/255, " ", float64(pv.G)/255,
			" ", float64(pv.B)/255, " RG\n")
	}
	if pv := &p.page.lineWidth; int(*pv*100) != int(p.lineWidth*100) {
		*pv = p.lineWidth
		p.write(float64(*pv), " w\n")
	}
	return mode
}

package pdf

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
)

// Affine is a two-dimensional affine transform using the same six coefficients
// as PDF and common canvas APIs.
type Affine struct{ A, B, C, D, E, F float64 }

// IdentityAffine returns an identity transform.
func IdentityAffine() Affine { return Affine{A: 1, D: 1} }

// Multiply composes m with n and returns m*n.
func (m Affine) Multiply(n Affine) Affine {
	return Affine{
		A: m.A*n.A + m.C*n.B, B: m.B*n.A + m.D*n.B,
		C: m.A*n.C + m.C*n.D, D: m.B*n.C + m.D*n.D,
		E: m.A*n.E + m.C*n.F + m.E, F: m.B*n.E + m.D*n.F + m.F,
	}
}

// TransformPoint applies m to a point.
func (m Affine) TransformPoint(x, y float64) (float64, float64) {
	return m.A*x + m.C*y + m.E, m.B*x + m.D*y + m.F
}

// LineJoin controls how stroked path segments are joined.
type LineJoin int

const (
	JoinMiter LineJoin = iota
	JoinRound
	JoinBevel
)

// LineCap controls the shape of open stroked path endpoints.
type LineCap int

const (
	CapButt LineCap = iota
	CapRound
	CapSquare
)

// ContextError reports invalid Context geometry or state.
type ContextError struct{ Operation, Detail string }

func (e ContextError) Error() string { return fmt.Sprintf("context %s: %s", e.Operation, e.Detail) }

type contextState struct {
	fill, stroke color.RGBA
	lineWidth    float64
	lineJoin     LineJoin
	lineCap      LineCap
	dash         []float64
	dashOffset   float64
	transform    Affine
	fontName     string
	fontSize     float64
}

type pathKind uint8

const (
	pathMove pathKind = iota
	pathLine
	pathQuad
	pathCubic
	pathClose
)

type pathCommand struct {
	kind   pathKind
	values [6]float64
}

// Context is a gg-like, point-based drawing context backed by a PDF document.
// Its public coordinate system starts at the top-left and grows downward.
type Context struct {
	doc               *PDF
	state             contextState
	stack             []contextState
	path              []pathCommand
	err               error
	allowedStackDepth int
}

// NewContext creates a drawing context for doc.
func NewContext(doc *PDF) *Context {
	c := &Context{doc: doc}
	c.state = contextState{fill: color.RGBA{A: 255}, stroke: color.RGBA{A: 255}, lineWidth: 1, transform: IdentityAffine(), fontName: "Helvetica", fontSize: 10}
	if doc == nil {
		c.err = ContextError{Operation: "new", Detail: "nil document"}
		return c
	}
	doc.init()
	c.state.fontName, c.state.fontSize = doc.FontName(), doc.FontSize()
	return c
}

// Document returns the underlying PDF document.
func (c *Context) Document() *PDF { return c.doc }

// Error returns the first Context error, or an unbalanced-state error.
func (c *Context) Error() error {
	if c.err != nil {
		return c.err
	}
	if len(c.stack) > c.allowedStackDepth {
		return ContextError{Operation: "state", Detail: fmt.Sprintf("%d unmatched Push calls", len(c.stack)-c.allowedStackDepth)}
	}
	return nil
}

// ClearError clears the current Context error.
func (c *Context) ClearError() { c.err = nil }

func (c *Context) fail(op, detail string) *Context {
	if c.err == nil {
		c.err = ContextError{Operation: op, Detail: detail}
	}
	return c
}
func finite(values ...float64) bool {
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// AddPage appends and selects a new page. Graphics-state stacks may not cross pages.
func (c *Context) AddPage() *Context {
	if c.err != nil {
		return c
	}
	if len(c.stack) != 0 {
		return c.fail("add page", "graphics-state stack is not empty")
	}
	c.doc.AddPage()
	c.path = nil
	return c
}

// Push saves the current graphics state and emits PDF q.
func (c *Context) Push() *Context {
	if c.err != nil {
		return c
	}
	saved := c.state
	saved.dash = append([]float64(nil), c.state.dash...)
	c.stack = append(c.stack, saved)
	c.doc.reservePage().write("q\n")
	return c
}

// Pop restores the latest graphics state and emits PDF Q.
func (c *Context) Pop() *Context {
	if len(c.stack) == 0 {
		return c.fail("pop", "graphics-state stack underflow")
	}
	c.doc.write("Q\n")
	c.state = c.stack[len(c.stack)-1]
	c.stack = c.stack[:len(c.stack)-1]
	c.invalidatePDFState()
	return c
}

func (c *Context) invalidatePDFState() {
	if c.doc == nil || c.doc.page == nil {
		return
	}
	c.doc.page.strokeColor = color.RGBA{1, 2, 3, 4}
	c.doc.page.nonStrokeColor = color.RGBA{4, 3, 2, 1}
	c.doc.page.lineWidth = -1
	c.doc.page.fontID = 0
}

// Identity resets the current transform.
func (c *Context) Identity() *Context { c.state.transform = IdentityAffine(); return c }

// SetTransform replaces the current affine transform.
func (c *Context) SetTransform(m Affine) *Context {
	if !finite(m.A, m.B, m.C, m.D, m.E, m.F) {
		return c.fail("transform", "non-finite coefficient")
	}
	c.state.transform = m
	return c
}

// Translate appends a translation.
func (c *Context) Translate(x, y float64) *Context {
	if !finite(x, y) {
		return c.fail("translate", "non-finite value")
	}
	c.state.transform = c.state.transform.Multiply(Affine{A: 1, D: 1, E: x, F: y})
	return c
}

// Scale appends a scale.
func (c *Context) Scale(x, y float64) *Context {
	if !finite(x, y) || x == 0 || y == 0 {
		return c.fail("scale", "scale must be finite and non-zero")
	}
	c.state.transform = c.state.transform.Multiply(Affine{A: x, D: y})
	return c
}

// Rotate appends a clockwise rotation in the top-left coordinate system.
func (c *Context) Rotate(radians float64) *Context {
	if !finite(radians) {
		return c.fail("rotate", "non-finite angle")
	}
	s, co := math.Sin(radians), math.Cos(radians)
	c.state.transform = c.state.transform.Multiply(Affine{A: co, B: s, C: -s, D: co})
	return c
}

// RotateAbout rotates around a point.
func (c *Context) RotateAbout(radians, x, y float64) *Context {
	return c.Translate(x, y).Rotate(radians).Translate(-x, -y)
}

// NewPath clears the current path.
func (c *Context) NewPath() *Context { c.path = nil; return c }

// MoveTo starts a new subpath.
func (c *Context) MoveTo(x, y float64) *Context {
	if !finite(x, y) {
		return c.fail("move to", "non-finite point")
	}
	var v [6]float64
	v[0], v[1] = x, y
	c.path = append(c.path, pathCommand{pathMove, v})
	return c
}

// LineTo adds a line segment.
func (c *Context) LineTo(x, y float64) *Context {
	if !finite(x, y) {
		return c.fail("line to", "non-finite point")
	}
	var v [6]float64
	v[0], v[1] = x, y
	c.path = append(c.path, pathCommand{pathLine, v})
	return c
}

// QuadraticTo adds a quadratic Bézier segment.
func (c *Context) QuadraticTo(cx, cy, x, y float64) *Context {
	if !finite(cx, cy, x, y) {
		return c.fail("quadratic to", "non-finite point")
	}
	var v [6]float64
	v[0], v[1], v[2], v[3] = cx, cy, x, y
	c.path = append(c.path, pathCommand{pathQuad, v})
	return c
}

// CubicTo adds a cubic Bézier segment.
func (c *Context) CubicTo(c1x, c1y, c2x, c2y, x, y float64) *Context {
	if !finite(c1x, c1y, c2x, c2y, x, y) {
		return c.fail("cubic to", "non-finite point")
	}
	c.path = append(c.path, pathCommand{pathCubic, [6]float64{c1x, c1y, c2x, c2y, x, y}})
	return c
}

// ClosePath closes the current subpath.
func (c *Context) ClosePath() *Context {
	c.path = append(c.path, pathCommand{kind: pathClose})
	return c
}

// DrawRectangle appends a rectangle.
func (c *Context) DrawRectangle(x, y, w, h float64) *Context {
	if !finite(x, y, w, h) || w < 0 || h < 0 {
		return c.fail("rectangle", "invalid geometry")
	}
	return c.MoveTo(x, y).LineTo(x+w, y).LineTo(x+w, y+h).LineTo(x, y+h).ClosePath()
}

// DrawRoundedRectangle appends a rounded rectangle.
func (c *Context) DrawRoundedRectangle(x, y, w, h, r float64) *Context {
	if !finite(x, y, w, h, r) || w < 0 || h < 0 || r < 0 {
		return c.fail("rounded rectangle", "invalid geometry")
	}
	if r > math.Min(w, h)/2 {
		r = math.Min(w, h) / 2
	}
	k := 0.5522847498307936
	return c.MoveTo(x+r, y).LineTo(x+w-r, y).CubicTo(x+w-r+k*r, y, x+w, y+r-k*r, x+w, y+r).LineTo(x+w, y+h-r).CubicTo(x+w, y+h-r+k*r, x+w-r+k*r, y+h, x+w-r, y+h).LineTo(x+r, y+h).CubicTo(x+r-k*r, y+h, x, y+h-r+k*r, x, y+h-r).LineTo(x, y+r).CubicTo(x, y+r-k*r, x+r-k*r, y, x+r, y).ClosePath()
}

// DrawCircle appends a circle.
func (c *Context) DrawCircle(x, y, r float64) *Context { return c.DrawEllipse(x, y, r, r) }

// DrawEllipse appends an ellipse centered at x,y.
func (c *Context) DrawEllipse(x, y, rx, ry float64) *Context {
	if !finite(x, y, rx, ry) || rx < 0 || ry < 0 {
		return c.fail("ellipse", "invalid geometry")
	}
	k := 0.5522847498307936
	return c.MoveTo(x+rx, y).CubicTo(x+rx, y+k*ry, x+k*rx, y+ry, x, y+ry).CubicTo(x-k*rx, y+ry, x-rx, y+k*ry, x-rx, y).CubicTo(x-rx, y-k*ry, x-k*rx, y-ry, x, y-ry).CubicTo(x+k*rx, y-ry, x+rx, y-k*ry, x+rx, y).ClosePath()
}

// DrawArc appends a clockwise arc in the top-left coordinate system.
func (c *Context) DrawArc(x, y, r, start, end float64) *Context {
	if !finite(x, y, r, start, end) || r < 0 {
		return c.fail("arc", "invalid geometry")
	}
	sweep := end - start
	if sweep == 0 {
		return c
	}
	segments := int(math.Ceil(math.Abs(sweep) / (math.Pi / 2)))
	step := sweep / float64(segments)
	a := start
	c.MoveTo(x+r*math.Cos(a), y+r*math.Sin(a))
	for i := 0; i < segments; i++ {
		b := a + step
		t := 4.0 / 3.0 * math.Tan((b-a)/4)
		c.CubicTo(x+r*(math.Cos(a)-t*math.Sin(a)), y+r*(math.Sin(a)+t*math.Cos(a)), x+r*(math.Cos(b)+t*math.Sin(b)), y+r*(math.Sin(b)-t*math.Cos(b)), x+r*math.Cos(b), y+r*math.Sin(b))
		a = b
	}
	return c
}

// SetColor sets both fill and stroke colors.
func (c *Context) SetColor(v color.Color) *Context { return c.SetFillColor(v).SetStrokeColor(v) }

// SetFillColor sets the fill/text color. Vector alpha below 1 is unsupported.
func (c *Context) SetFillColor(v color.Color) *Context {
	rgba, err := contextColor(v)
	if err != nil {
		return c.fail("fill color", err.Error())
	}
	c.state.fill = rgba
	return c
}

// SetStrokeColor sets the stroke color. Vector alpha below 1 is unsupported.
func (c *Context) SetStrokeColor(v color.Color) *Context {
	rgba, err := contextColor(v)
	if err != nil {
		return c.fail("stroke color", err.Error())
	}
	c.state.stroke = rgba
	return c
}
func contextColor(v color.Color) (color.RGBA, error) {
	if v == nil {
		return color.RGBA{}, fmt.Errorf("nil color")
	}
	r, g, b, a := v.RGBA()
	if a != 0xffff {
		return color.RGBA{}, fmt.Errorf("vector alpha is not supported")
	}
	return color.RGBA{byte(r >> 8), byte(g >> 8), byte(b >> 8), 255}, nil
}

// SetLineWidth sets stroke width in points.
func (c *Context) SetLineWidth(v float64) *Context {
	if !finite(v) || v <= 0 {
		return c.fail("line width", "must be finite and positive")
	}
	c.state.lineWidth = v
	return c
}

// SetLineJoin sets stroke joins.
func (c *Context) SetLineJoin(v LineJoin) *Context {
	if v < JoinMiter || v > JoinBevel {
		return c.fail("line join", "invalid value")
	}
	c.state.lineJoin = v
	return c
}

// SetLineCap sets stroke caps.
func (c *Context) SetLineCap(v LineCap) *Context {
	if v < CapButt || v > CapSquare {
		return c.fail("line cap", "invalid value")
	}
	c.state.lineCap = v
	return c
}

// SetDash configures a dash pattern in points.
func (c *Context) SetDash(offset float64, values ...float64) *Context {
	if !finite(offset) || offset < 0 {
		return c.fail("dash", "invalid offset")
	}
	positive := false
	for _, v := range values {
		if !finite(v) || v < 0 {
			return c.fail("dash", "invalid length")
		}
		positive = positive || v > 0
	}
	if len(values) > 0 && !positive {
		return c.fail("dash", "at least one length must be positive")
	}
	c.state.dash = append([]float64(nil), values...)
	c.state.dashOffset = offset
	return c
}

func (c *Context) emitPath() bool {
	if c.err != nil || len(c.path) == 0 {
		return false
	}
	c.doc.reservePage()
	var current, sub [2]float64
	have := false
	point := func(x, y float64) (float64, float64) {
		x, y = c.state.transform.TransformPoint(x, y)
		return x, c.doc.paperSize.heightPt - y
	}
	for _, cmd := range c.path {
		switch cmd.kind {
		case pathMove:
			x, y := point(cmd.values[0], cmd.values[1])
			c.doc.write(x, " ", y, " m\n")
			current = [2]float64{cmd.values[0], cmd.values[1]}
			sub = current
			have = true
		case pathLine:
			if !have {
				c.fail("path", "LineTo without MoveTo")
				return false
			}
			x, y := point(cmd.values[0], cmd.values[1])
			c.doc.write(x, " ", y, " l\n")
			current = [2]float64{cmd.values[0], cmd.values[1]}
		case pathQuad:
			if !have {
				c.fail("path", "QuadraticTo without MoveTo")
				return false
			}
			cx, cy, x, y := cmd.values[0], cmd.values[1], cmd.values[2], cmd.values[3]
			c1x, c1y := current[0]+2*(cx-current[0])/3, current[1]+2*(cy-current[1])/3
			c2x, c2y := x+2*(cx-x)/3, y+2*(cy-y)/3
			a, b := point(c1x, c1y)
			d, e := point(c2x, c2y)
			g, h := point(x, y)
			c.doc.write(a, " ", b, " ", d, " ", e, " ", g, " ", h, " c\n")
			current = [2]float64{x, y}
		case pathCubic:
			if !have {
				c.fail("path", "CubicTo without MoveTo")
				return false
			}
			a, b := point(cmd.values[0], cmd.values[1])
			d, e := point(cmd.values[2], cmd.values[3])
			g, h := point(cmd.values[4], cmd.values[5])
			c.doc.write(a, " ", b, " ", d, " ", e, " ", g, " ", h, " c\n")
			current = [2]float64{cmd.values[4], cmd.values[5]}
			have = true
		case pathClose:
			if !have {
				c.fail("path", "ClosePath without MoveTo")
				return false
			}
			c.doc.write("h\n")
			current = sub
		}
	}
	return c.err == nil
}

func (c *Context) emitFillColor() {
	v := c.state.fill
	c.doc.write(float64(v.R)/255, " ", float64(v.G)/255, " ", float64(v.B)/255, " rg\n")
}
func (c *Context) emitStrokeStyle() {
	v := c.state.stroke
	c.doc.write(float64(v.R)/255, " ", float64(v.G)/255, " ", float64(v.B)/255, " RG\n", c.state.lineWidth, " w\n", int(c.state.lineCap), " J\n", int(c.state.lineJoin), " j\n[")
	for i, d := range c.state.dash {
		if i > 0 {
			c.doc.write(" ")
		}
		c.doc.write(d)
	}
	c.doc.write("] ", c.state.dashOffset, " d\n")
}

// Fill fills and clears the current path.
func (c *Context) Fill() *Context {
	if c.emitPath() {
		c.emitFillColor()
		c.doc.write("f\n")
		c.path = nil
	}
	return c
}

// FillPreserve fills without clearing the Context path.
func (c *Context) FillPreserve() *Context {
	if c.emitPath() {
		c.emitFillColor()
		c.doc.write("f\n")
	}
	return c
}

// Stroke strokes and clears the current path.
func (c *Context) Stroke() *Context {
	if c.emitPath() {
		c.emitStrokeStyle()
		c.doc.write("S\n")
		c.path = nil
	}
	return c
}

// StrokePreserve strokes without clearing the Context path.
func (c *Context) StrokePreserve() *Context {
	if c.emitPath() {
		c.emitStrokeStyle()
		c.doc.write("S\n")
	}
	return c
}

// Clip intersects the clipping region with the current path and clears it.
func (c *Context) Clip() *Context {
	if c.emitPath() {
		c.doc.write("W n\n")
		c.path = nil
	}
	return c
}

// SetFont selects a registered or base-14 font and size in points.
func (c *Context) SetFont(name string, size float64) *Context {
	if c.err != nil {
		return c
	}
	if !finite(size) || size <= 0 {
		return c.fail("font", "size must be finite and positive")
	}
	key := c.doc.toUpperLettersDigits(name, "")
	if _, ok := c.doc.currentBuiltInFontNamed(name); !ok && c.doc.registeredFonts[key] == nil {
		return c.fail("font", fmt.Sprintf("%q is not registered", name))
	}
	c.state.fontName, c.state.fontSize = name, size
	c.doc.SetFont(name, size)
	return c
}

// UseFontRole selects a semantic document font role.
func (c *Context) UseFontRole(role FontRole, size float64) *Context {
	if c.err != nil {
		return c
	}
	if err := c.doc.UseFontRole(role, size); err != nil {
		return c.fail("font role", err.Error())
	}
	c.state.fontName, c.state.fontSize = c.doc.FontName(), c.doc.FontSize()
	return c
}

// MeasureString returns approximate width and line height in points.
func (c *Context) MeasureString(s string) (float64, float64) {
	if c.err != nil {
		return 0, 0
	}
	c.doc.SetFont(c.state.fontName, c.state.fontSize)
	before := len(c.doc.errors)
	w := c.doc.textWidthPt(s)
	if len(c.doc.errors) > before {
		c.err = c.doc.errors[len(c.doc.errors)-1]
		return 0, 0
	}
	return w, c.state.fontSize
}

// DrawString draws text with x,y interpreted as its baseline.
func (c *Context) DrawString(s string, x, y float64) *Context {
	if c.err != nil {
		return c
	}
	if !finite(x, y) {
		return c.fail("draw string", "non-finite point")
	}
	c.doc.reservePage()
	c.doc.write("q\n")
	m := c.pdfTransform()
	c.doc.write(m.A, " ", m.B, " ", m.C, " ", m.D, " ", m.E, " ", m.F, " cm\n")
	c.doc.SetColorRGB(c.state.fill.R, c.state.fill.G, c.state.fill.B).SetFont(c.state.fontName, c.state.fontSize).SetXY(x, y)
	before := len(c.doc.errors)
	c.doc.drawTextLine(s)
	c.doc.write("Q\n")
	c.invalidatePDFState()
	if len(c.doc.errors) > before {
		c.err = c.doc.errors[len(c.doc.errors)-1]
	}
	return c
}

// DrawStringAnchored draws text relative to an approximate one-em text box.
func (c *Context) DrawStringAnchored(s string, x, y, anchorX, anchorY float64) *Context {
	w, h := c.MeasureString(s)
	if c.err != nil {
		return c
	}
	return c.DrawString(s, x-anchorX*w, y+(0.8-anchorY)*h)
}

func (c *Context) pdfTransform() Affine {
	h := c.doc.paperSize.heightPt
	m := c.state.transform
	return Affine{A: m.A, B: -m.B, C: -m.C, D: m.D, E: m.C*h + m.E, F: h*(1-m.D) - m.F}
}

// DrawImage draws an image at native pixel dimensions interpreted as points.
func (c *Context) DrawImage(img image.Image, x, y float64) *Context {
	return c.DrawImageAnchored(img, x, y, 0, 0)
}

// DrawImageAnchored draws an image at native pixel dimensions with anchors in [0,1].
func (c *Context) DrawImageAnchored(img image.Image, x, y, anchorX, anchorY float64) *Context {
	if isNilImage(img) {
		return c.fail("draw image", "nil image")
	}
	b := img.Bounds()
	return c.DrawImageScaledAnchored(img, x, y, float64(b.Dx()), float64(b.Dy()), anchorX, anchorY)
}

// DrawImageScaledAnchored draws a transformed image with explicit point dimensions.
func (c *Context) DrawImageScaledAnchored(img image.Image, x, y, w, h, anchorX, anchorY float64) *Context {
	if c.err != nil {
		return c
	}
	if isNilImage(img) || !finite(x, y, w, h, anchorX, anchorY) || w <= 0 || h <= 0 {
		return c.fail("draw image", "invalid image or geometry")
	}
	idx, err := c.doc.loadDirectImage(img)
	if err != nil {
		c.err = err
		return c
	}
	return c.drawImageIndexScaledAnchored(idx, x, y, w, h, anchorX, anchorY)
}

// DrawImageResource draws a registered resource at native pixel dimensions
// interpreted as points.
func (c *Context) DrawImageResource(resource *ImageResource, x, y float64) *Context {
	if resource == nil {
		return c.fail("draw image resource", "nil resource")
	}
	return c.DrawImageResourceScaledAnchored(resource, x, y, float64(resource.width), float64(resource.height), 0, 0)
}

// DrawImageResourceScaledAnchored draws a registered resource with explicit
// point dimensions and anchors in [0,1].
func (c *Context) DrawImageResourceScaledAnchored(resource *ImageResource, x, y, w, h, anchorX, anchorY float64) *Context {
	if c.err != nil {
		return c
	}
	if resource == nil || resource.doc != c.doc || resource.index < 0 || resource.index >= len(c.doc.images) {
		return c.fail("draw image resource", "resource belongs to another document or is invalid")
	}
	return c.drawImageIndexScaledAnchored(resource.index, x, y, w, h, anchorX, anchorY)
}

func (c *Context) drawImageIndexScaledAnchored(idx int, x, y, w, h, anchorX, anchorY float64) *Context {
	if !finite(x, y, w, h, anchorX, anchorY) || w <= 0 || h <= 0 {
		return c.fail("draw image", "invalid image geometry")
	}
	c.doc.reservePage()
	c.doc.addPageImage(idx)
	left, top := x-anchorX*w, y-anchorY*h
	m := c.state.transform
	tlx, tly := m.TransformPoint(left, top)
	trx, try := m.TransformPoint(left+w, top)
	blx, bly := m.TransformPoint(left, top+h)
	height := c.doc.paperSize.heightPt
	pblx, pbly := blx, height-bly
	pbrx, pbry := m.TransformPoint(left+w, top+h)
	pbry = height - pbry
	ptlx, ptly := tlx, height-tly
	_ = trx
	_ = try
	c.doc.write("q\n", pbrx-pblx, " ", pbry-pbly, " ", ptlx-pblx, " ", ptly-pbly, " ", pblx, " ", pbly, " cm\n/IMG", idx, " Do\nQ\n")
	return c
}

// DrawEmojiAt draws a provider-backed emoji at a text baseline.
func (c *Context) DrawEmojiAt(sequence string, x, y, fontSize float64) *Context {
	if c.err != nil {
		return c
	}
	asset, err := c.doc.resolveEmoji(sequence, fontSize)
	if err != nil {
		c.err = err
		return c
	}
	b := asset.Image.Bounds()
	h := asset.HeightEm * fontSize
	w := float64(b.Dx()) / float64(b.Dy()) * h
	top := y - asset.BaselineEm*fontSize - h
	return c.DrawImageScaledAnchored(asset.Image, x, top, w, h, 0, 0)
}

// SaveFile validates Context state and saves the underlying PDF.
func (c *Context) SaveFile(filename string) error {
	if err := c.Error(); err != nil {
		return err
	}
	return c.doc.SaveFile(filename)
}

// WriteTo validates Context state and writes the underlying PDF.
func (c *Context) WriteTo(w io.Writer) (int64, error) {
	if err := c.Error(); err != nil {
		return 0, err
	}
	return c.doc.WriteTo(w)
}

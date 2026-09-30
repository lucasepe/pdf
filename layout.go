package pdf

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"unicode"
)

// Inches converts inches to PDF points.
func Inches(value float64) float64 { return value * 72 }

// Millimeters converts millimetres to PDF points.
func Millimeters(value float64) float64 { return value * 72 / 25.4 }

// Rect is a rectangle in the Context top-left coordinate system.
type Rect struct {
	X, Y, Width, Height float64
}

// PageSpec describes the fixed page and the usable flow area in points.
// HeaderHeight and FooterHeight are reserved in addition to the outer margins
// and are not used by paragraph content.
type PageSpec struct {
	Width, Height              float64
	MarginTop, MarginRight     float64
	MarginBottom, MarginLeft   float64
	BindingOffset              float64
	HeaderHeight, FooterHeight float64
}

// PaperSize returns a custom paper-size string accepted by NewPDF.
func (s PageSpec) PaperSize() string {
	return fmt.Sprintf("%.6fpt x %.6fpt", s.Width, s.Height)
}

// Body returns the usable content rectangle for a one-based page number. The
// binding offset enlarges the inner margin: left on odd pages, right on even.
func (s PageSpec) Body(pageNumber int) Rect {
	left, right := s.MarginLeft, s.MarginRight
	if pageNumber%2 == 0 {
		right += s.BindingOffset
	} else {
		left += s.BindingOffset
	}
	return Rect{
		X:      left,
		Y:      s.MarginTop + s.HeaderHeight,
		Width:  s.Width - left - right,
		Height: s.Height - s.MarginTop - s.MarginBottom - s.HeaderHeight - s.FooterHeight,
	}
}

func (s PageSpec) validate() error {
	values := []float64{s.Width, s.Height, s.MarginTop, s.MarginRight, s.MarginBottom, s.MarginLeft, s.BindingOffset, s.HeaderHeight, s.FooterHeight}
	if !finite(values...) || s.Width <= 0 || s.Height <= 0 {
		return fmt.Errorf("page dimensions must be finite and positive")
	}
	for _, value := range values[2:] {
		if value < 0 {
			return fmt.Errorf("page margins and reserved areas cannot be negative")
		}
	}
	for _, page := range []int{1, 2} {
		body := s.Body(page)
		if body.Width <= 0 || body.Height <= 0 {
			return fmt.Errorf("page %d has no usable body", page)
		}
	}
	return nil
}

// PageInfo describes the page currently being decorated.
type PageInfo struct {
	Number int
	Spec   PageSpec
	Body   Rect
}

// PageDecorator draws page furniture such as backgrounds, headers, footers,
// and page numbers. It is isolated inside a balanced graphics-state scope.
type PageDecorator func(*Context, PageInfo) error

// WrapMode controls line-break opportunities within a text run.
type WrapMode uint8

const (
	// WrapText breaks at whitespace and hyphens.
	WrapText WrapMode = iota
	// WrapCode also breaks after common code and path separators.
	WrapCode
)

// TextStyle selects the appearance of a run. FontName and FontRole are
// mutually exclusive. Empty fields inherit the Context state captured when the
// Flow is created; a zero FontSize also inherits that size.
type TextStyle struct {
	FontName string
	FontRole FontRole
	FontSize float64
	Color    color.Color
	Wrap     WrapMode
}

// TextRun is an inline text or provider-backed emoji item. An emoji run is
// atomic: Text is passed to the configured EmojiProvider as one sequence. URI
// and LinkDestination are mutually exclusive and create clickable annotations
// that follow line wrapping.
type TextRun struct {
	Text            string
	Style           TextStyle
	Emoji           bool
	URI             string
	LinkDestination string
}

// TextAlign controls horizontal paragraph alignment.
type TextAlign uint8

const (
	AlignLeft TextAlign = iota
	AlignCenter
	AlignRight
)

// ParagraphStyle controls paragraph geometry. LineHeight is an absolute point
// value; zero chooses 1.3 times the largest font on each line.
type ParagraphStyle struct {
	Align                   TextAlign
	LineHeight              float64
	SpaceBefore, SpaceAfter float64
	FirstLineIndent         float64
}

// FlowError reports page or paragraph layout failures.
type FlowError struct {
	Operation, Detail string
}

func (e FlowError) Error() string { return fmt.Sprintf("flow %s: %s", e.Operation, e.Detail) }

// Flow lays out paragraphs within a PageSpec and opens pages as needed. It is
// deliberately independent of Markdown and ebookgen document types.
type Flow struct {
	context   *Context
	spec      PageSpec
	decorate  PageDecorator
	page      int
	body      Rect
	y         float64
	err       error
	baseStyle resolvedTextStyle
}

type resolvedTextStyle struct {
	fontName string
	fontRole FontRole
	fontSize float64
	color    color.Color
	wrap     WrapMode
}

type flowToken struct {
	text           string
	style          resolvedTextStyle
	emoji          bool
	space, newline bool
	width          float64
	uri, target    string
}

type flowLine struct {
	tokens  []flowToken
	width   float64
	height  float64
	maxSize float64
}

// NewFlow creates a flow on the document's current page and decorates that
// page immediately. The PageSpec dimensions must match the PDF document.
func NewFlow(context *Context, spec PageSpec, decorator PageDecorator) (*Flow, error) {
	if context == nil || context.doc == nil {
		return nil, FlowError{Operation: "new", Detail: "nil context or document"}
	}
	if err := context.Error(); err != nil {
		return nil, FlowError{Operation: "new", Detail: err.Error()}
	}
	if err := spec.validate(); err != nil {
		return nil, FlowError{Operation: "new", Detail: err.Error()}
	}
	if math.Abs(context.doc.paperSize.widthPt-spec.Width) > 0.01 || math.Abs(context.doc.paperSize.heightPt-spec.Height) > 0.01 {
		return nil, FlowError{Operation: "new", Detail: fmt.Sprintf("PageSpec %.3fx%.3f does not match document %.3fx%.3f", spec.Width, spec.Height, context.doc.paperSize.widthPt, context.doc.paperSize.heightPt)}
	}
	context.doc.reservePage()
	f := &Flow{
		context:  context,
		spec:     spec,
		decorate: decorator,
		page:     len(context.doc.pages),
		baseStyle: resolvedTextStyle{
			fontName: context.state.fontName,
			fontSize: context.state.fontSize,
			color:    context.state.fill,
		},
	}
	f.body = spec.Body(f.page)
	f.y = f.body.Y
	f.decoratePage()
	if f.err != nil {
		return nil, f.err
	}
	return f, nil
}

// Error returns the first layout or drawing error.
func (f *Flow) Error() error {
	if f == nil {
		return FlowError{Operation: "state", Detail: "nil flow"}
	}
	if f.err != nil {
		return f.err
	}
	if err := f.context.Error(); err != nil {
		return err
	}
	return nil
}

func (f *Flow) fail(operation, detail string) *Flow {
	if f.err == nil {
		f.err = FlowError{Operation: operation, Detail: detail}
	}
	return f
}

// PageNumber returns the current one-based page number.
func (f *Flow) PageNumber() int { return f.page }

// Body returns the current page's usable content rectangle.
func (f *Flow) Body() Rect { return f.body }

// CursorY returns the current top edge of the next flow item.
func (f *Flow) CursorY() float64 { return f.y }

// RemainingHeight returns the usable height below the cursor.
func (f *Flow) RemainingHeight() float64 {
	remaining := f.body.Y + f.body.Height - f.y
	if remaining < 0 {
		return 0
	}
	return remaining
}

// NewPage performs an explicit page break and resets the cursor to the new body.
func (f *Flow) NewPage() *Flow {
	if f.err != nil {
		return f
	}
	f.context.AddPage()
	if err := f.context.Error(); err != nil {
		return f.fail("new page", err.Error())
	}
	f.page++
	f.body = f.spec.Body(f.page)
	f.y = f.body.Y
	f.decoratePage()
	return f
}

// PageBreak is a semantic alias for NewPage.
func (f *Flow) PageBreak() *Flow { return f.NewPage() }

// EnsureSpace opens a new page if height does not fit below the cursor. It is
// useful to higher-level block layout such as the future table renderer.
func (f *Flow) EnsureSpace(height float64) *Flow {
	if f.err != nil {
		return f
	}
	if !finite(height) || height < 0 {
		return f.fail("ensure space", "height must be finite and non-negative")
	}
	if height > f.body.Height+0.001 {
		return f.fail("ensure space", fmt.Sprintf("%.3fpt item exceeds %.3fpt body height", height, f.body.Height))
	}
	if height > f.RemainingHeight()+0.001 {
		f.NewPage()
	}
	return f
}

// Advance moves the flow cursor by height without drawing.
func (f *Flow) Advance(height float64) *Flow {
	if f.err != nil {
		return f
	}
	if !finite(height) || height < 0 {
		return f.fail("advance", "height must be finite and non-negative")
	}
	f.EnsureSpace(height)
	if f.err == nil {
		f.y += height
	}
	return f
}

func (f *Flow) decoratePage() {
	if f.err != nil || f.decorate == nil {
		return
	}
	stackDepth := len(f.context.stack)
	oldAllowedDepth := f.context.allowedStackDepth
	f.context.Push()
	f.context.allowedStackDepth = len(f.context.stack)
	err := f.decorate(f.context, PageInfo{Number: f.page, Spec: f.spec, Body: f.body})
	f.context.allowedStackDepth = oldAllowedDepth
	if len(f.context.stack) != stackDepth+1 {
		f.fail("decorate page", "decorator changed its enclosing graphics-state scope")
	}
	for len(f.context.stack) > stackDepth {
		f.context.Pop()
	}
	if err != nil {
		f.fail("decorate page", err.Error())
		return
	}
	if err := f.context.Error(); err != nil {
		f.fail("decorate page", err.Error())
	}
}

// DrawParagraph wraps and draws runs, opening pages between complete lines.
func (f *Flow) DrawParagraph(runs []TextRun, style ParagraphStyle) *Flow {
	if f.err != nil {
		return f
	}
	if style.Align > AlignRight || !finite(style.LineHeight, style.SpaceBefore, style.SpaceAfter, style.FirstLineIndent) || style.LineHeight < 0 || style.SpaceBefore < 0 || style.SpaceAfter < 0 || style.FirstLineIndent < 0 || style.FirstLineIndent >= f.body.Width {
		return f.fail("paragraph", "invalid paragraph geometry or alignment")
	}
	lines := f.breakLines(runs, style)
	if f.err != nil {
		return f
	}
	if style.SpaceBefore > 0 && f.y > f.body.Y+0.001 {
		if style.SpaceBefore > f.RemainingHeight() {
			f.NewPage()
		} else {
			f.y += style.SpaceBefore
		}
	}
	for i, line := range lines {
		f.EnsureSpace(line.height)
		if f.err != nil {
			return f
		}
		indent := 0.0
		if i == 0 {
			indent = style.FirstLineIndent
		}
		available := f.body.Width - indent
		x := f.body.X + indent
		switch style.Align {
		case AlignCenter:
			x += (available - line.width) / 2
		case AlignRight:
			x += available - line.width
		}
		baseline := f.y + line.maxSize*.82
		for _, token := range line.tokens {
			tokenX := x
			f.applyStyle(token.style)
			if f.err != nil {
				return f
			}
			if token.emoji {
				f.context.DrawEmojiAt(token.text, x, baseline, token.style.fontSize)
			} else {
				f.context.DrawString(token.text, x, baseline)
			}
			if err := f.context.Error(); err != nil {
				return f.fail("draw paragraph", err.Error())
			}
			x += token.width
			if token.width > 0 && token.text != "" {
				rect := Rect{X: tokenX, Y: f.y, Width: token.width, Height: line.height}
				if token.uri != "" {
					f.context.AddExternalLink(token.uri, rect)
				} else if token.target != "" {
					f.context.AddInternalLink(token.target, rect)
				}
				if err := f.context.Error(); err != nil {
					return f.fail("draw paragraph link", err.Error())
				}
			}
		}
		f.y += line.height
	}
	if style.SpaceAfter > 0 {
		f.y = math.Min(f.body.Y+f.body.Height, f.y+style.SpaceAfter)
	}
	return f
}

func (f *Flow) breakLines(runs []TextRun, paragraph ParagraphStyle) []flowLine {
	return f.breakLinesWidth(runs, paragraph, f.body.Width)
}

func (f *Flow) breakLinesWidth(runs []TextRun, paragraph ParagraphStyle, width float64) []flowLine {
	var tokens []flowToken
	for _, run := range runs {
		if run.URI != "" && run.LinkDestination != "" {
			f.fail("paragraph", "URI and LinkDestination are mutually exclusive")
			return nil
		}
		style := f.resolveStyle(run.Style)
		if f.err != nil {
			return nil
		}
		if run.Emoji {
			if run.Text == "" {
				f.fail("paragraph", "empty emoji run")
				return nil
			}
			width, err := f.context.doc.MeasureEmoji(run.Text, style.fontSize)
			if err != nil {
				f.fail("measure emoji", err.Error())
				return nil
			}
			tokens = append(tokens, flowToken{text: run.Text, style: style, emoji: true, width: width, uri: run.URI, target: run.LinkDestination})
			continue
		}
		for _, token := range tokenizeText(run.Text, style) {
			token.uri, token.target = run.URI, run.LinkDestination
			if !token.newline {
				token.width = f.measure(token.text, style)
				if f.err != nil {
					return nil
				}
			}
			tokens = append(tokens, token)
		}
	}

	defaultHeight := f.baseStyle.fontSize * 1.3
	if paragraph.LineHeight > 0 {
		defaultHeight = paragraph.LineHeight
	}
	var lines []flowLine
	line := flowLine{height: defaultHeight, maxSize: f.baseStyle.fontSize}
	first := true
	flush := func(force bool) {
		line = trimTrailingSpaces(line)
		if len(line.tokens) > 0 || force {
			if paragraph.LineHeight > 0 {
				line.height = paragraph.LineHeight
			} else {
				line.height = line.maxSize * 1.3
			}
			lines = append(lines, line)
		}
		line = flowLine{height: defaultHeight, maxSize: f.baseStyle.fontSize}
		first = false
	}

	for _, token := range tokens {
		if token.newline {
			flush(true)
			continue
		}
		limit := width
		if first {
			limit -= paragraph.FirstLineIndent
		}
		if token.space {
			if len(line.tokens) > 0 || token.style.wrap == WrapCode {
				line.tokens = append(line.tokens, token)
				line.width += token.width
				line.maxSize = math.Max(line.maxSize, token.style.fontSize)
			}
			continue
		}
		if token.width > limit+0.001 {
			if len(trimTrailingSpaces(line).tokens) > 0 {
				flush(false)
				limit = width
			}
			pieces := f.splitToken(token, limit)
			if f.err != nil {
				return nil
			}
			for i, piece := range pieces {
				line.tokens = append(line.tokens, piece)
				line.width += piece.width
				line.maxSize = math.Max(line.maxSize, piece.style.fontSize)
				if i < len(pieces)-1 {
					flush(false)
				}
			}
			continue
		}
		candidate := line.width + token.width
		if candidate > limit+0.001 && len(trimTrailingSpaces(line).tokens) > 0 {
			flush(false)
			limit = width
		}
		line.tokens = append(line.tokens, token)
		line.width += token.width
		line.maxSize = math.Max(line.maxSize, token.style.fontSize)
	}
	if len(line.tokens) > 0 || len(lines) == 0 {
		flush(len(lines) == 0)
	}
	return lines
}

func tokenizeText(text string, style resolvedTextStyle) []flowToken {
	var result []flowToken
	var word strings.Builder
	flushWord := func() {
		if word.Len() > 0 {
			result = append(result, flowToken{text: word.String(), style: style})
			word.Reset()
		}
	}
	for _, r := range text {
		if r == '\n' {
			flushWord()
			result = append(result, flowToken{style: style, newline: true})
			continue
		}
		if unicode.IsSpace(r) {
			flushWord()
			if style.wrap == WrapCode || len(result) == 0 || !result[len(result)-1].space {
				result = append(result, flowToken{text: " ", style: style, space: true})
			}
			continue
		}
		word.WriteRune(r)
		if isWrapSeparator(r, style.wrap) {
			flushWord()
		}
	}
	flushWord()
	return result
}

func isWrapSeparator(r rune, mode WrapMode) bool {
	if r == '-' || r == '‐' || r == '‑' {
		return true
	}
	return mode == WrapCode && strings.ContainsRune("/._,:=;?&#+", r)
}

func (f *Flow) splitToken(token flowToken, limit float64) []flowToken {
	if token.emoji {
		f.fail("line break", fmt.Sprintf("emoji %q is wider than the %.3fpt line", token.text, limit))
		return nil
	}
	var pieces []flowToken
	var current strings.Builder
	currentWidth := 0.0
	for _, r := range token.text {
		part := string(r)
		width := f.measure(part, token.style)
		if f.err != nil {
			return nil
		}
		if width > limit+0.001 {
			f.fail("line break", fmt.Sprintf("rune %q is wider than the %.3fpt line", r, limit))
			return nil
		}
		if current.Len() > 0 && currentWidth+width > limit+0.001 {
			pieces = append(pieces, flowToken{text: current.String(), style: token.style, width: currentWidth, uri: token.uri, target: token.target})
			current.Reset()
			currentWidth = 0
		}
		current.WriteRune(r)
		currentWidth += width
	}
	if current.Len() > 0 {
		pieces = append(pieces, flowToken{text: current.String(), style: token.style, width: currentWidth, uri: token.uri, target: token.target})
	}
	return pieces
}

func trimTrailingSpaces(line flowLine) flowLine {
	for len(line.tokens) > 0 && line.tokens[len(line.tokens)-1].space {
		line.width -= line.tokens[len(line.tokens)-1].width
		line.tokens = line.tokens[:len(line.tokens)-1]
	}
	return line
}

func (f *Flow) resolveStyle(style TextStyle) resolvedTextStyle {
	resolved := f.baseStyle
	if style.FontName != "" && style.FontRole != "" {
		f.fail("text style", "FontName and FontRole are mutually exclusive")
		return resolved
	}
	if style.FontName != "" {
		resolved.fontName, resolved.fontRole = style.FontName, ""
	}
	if style.FontRole != "" {
		resolved.fontName, resolved.fontRole = "", style.FontRole
	}
	if style.FontSize != 0 {
		if !finite(style.FontSize) || style.FontSize <= 0 {
			f.fail("text style", "font size must be finite and positive")
			return resolved
		}
		resolved.fontSize = style.FontSize
	}
	if style.Color != nil {
		resolved.color = style.Color
	}
	if style.Wrap > WrapCode {
		f.fail("text style", "invalid wrap mode")
		return resolved
	}
	resolved.wrap = style.Wrap
	return resolved
}

func (f *Flow) applyStyle(style resolvedTextStyle) {
	if style.fontRole != "" {
		f.context.UseFontRole(style.fontRole, style.fontSize)
	} else {
		f.context.SetFont(style.fontName, style.fontSize)
	}
	f.context.SetFillColor(style.color)
	if err := f.context.Error(); err != nil {
		f.fail("text style", err.Error())
	}
}

func (f *Flow) measure(text string, style resolvedTextStyle) float64 {
	f.applyStyle(style)
	if f.err != nil {
		return 0
	}
	width, _ := f.context.MeasureString(text)
	if err := f.context.Error(); err != nil {
		f.fail("measure text", err.Error())
		return 0
	}
	return width
}

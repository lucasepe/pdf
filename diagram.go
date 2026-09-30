package pdf

import (
	"image/color"
	"math"
	"strings"
	"unicode"
)

const (
	defaultDiagramFontSize         = 7.0
	defaultDiagramMaxWidthFraction = 0.98
	defaultDiagramLineWidth        = 0.55
)

// ASCIIDiagramOptions controls a directly rendered ASCII diagram. Scale is
// applied before the diagram is reduced, if necessary, to fit the page body.
// Zero values select Courier at 7pt, a 1.4em row height, a 0.98 body-width
// limit, centered alignment, and black strokes and text.
type ASCIIDiagramOptions struct {
	Scale, MaxWidthFraction float64
	FontSize, LineHeight    float64
	LineWidth, CornerRadius float64
	Align                   ImageAlign
	SpaceBefore, SpaceAfter float64
	StrokeColor, TextColor  color.Color
}

// DiagramLayoutError reports invalid source, geometry, or drawing state.
type DiagramLayoutError struct{ Detail string }

func (e DiagramLayoutError) Error() string { return "ASCII diagram: " + e.Detail }

type diagramDirection uint8

const (
	dirN diagramDirection = iota
	dirNE
	dirE
	dirSE
	dirS
	dirSW
	dirW
	dirNW
)

var diagramDeltas = [...]struct{ row, col int }{
	{-1, 0}, {-1, 1}, {0, 1}, {1, 1},
	{1, 0}, {1, -1}, {0, -1}, {-1, -1},
}

type diagramCell struct {
	text        string
	connections uint8
	geometry    bool
}

type asciiDiagram struct {
	cells         [][]diagramCell
	rows, columns int
}

// DrawASCIIDiagram draws source as one atomic Flow block. Connected ASCII
// punctuation becomes vector geometry; other cells become Courier labels.
// Emoji sequences are passed intact to the document EmojiProvider.
func (f *Flow) DrawASCIIDiagram(source string, options ASCIIDiagramOptions) *Flow {
	if f.err != nil {
		return f
	}
	if options.Align > ImageAlignRight ||
		!finite(options.Scale, options.MaxWidthFraction, options.FontSize, options.LineHeight,
			options.LineWidth, options.CornerRadius, options.SpaceBefore, options.SpaceAfter) ||
		options.Scale < 0 || options.MaxWidthFraction < 0 || options.FontSize < 0 ||
		options.LineHeight < 0 || options.LineWidth < 0 || options.CornerRadius < 0 ||
		options.SpaceBefore < 0 || options.SpaceAfter < 0 {
		return f.failDiagram("invalid diagram options")
	}

	diagram, err := parseASCIIDiagram(source)
	if err != nil {
		return f.failDiagram(err.Error())
	}
	fontSize := options.FontSize
	if fontSize == 0 {
		fontSize = defaultDiagramFontSize
	}
	lineHeight := options.LineHeight
	if lineHeight == 0 {
		lineHeight = fontSize * 1.4
	}
	if lineHeight < fontSize {
		return f.failDiagram("line height must not be smaller than font size")
	}
	lineWidth := options.LineWidth
	if lineWidth == 0 {
		lineWidth = defaultDiagramLineWidth
	}
	maxWidthFraction := options.MaxWidthFraction
	if maxWidthFraction == 0 {
		maxWidthFraction = defaultDiagramMaxWidthFraction
	}
	if maxWidthFraction <= 0 || maxWidthFraction > 1 {
		return f.failDiagram("maximum width fraction must be in (0,1]")
	}
	scale := options.Scale
	if scale == 0 {
		scale = 1
	}

	// Courier's base-14 metrics are fixed at 0.6em for every character.
	cellWidth := fontSize * 0.6
	naturalWidth := float64(diagram.columns) * cellWidth
	naturalHeight := float64(diagram.rows) * lineHeight
	fitScale := math.Min(scale, math.Min(f.body.Width*maxWidthFraction/naturalWidth, f.body.Height/naturalHeight))
	if !finite(fitScale) || fitScale <= 0 {
		return f.failDiagram("diagram has no drawable area")
	}
	width, height := naturalWidth*fitScale, naturalHeight*fitScale

	f.diagramSpaceBefore(options.SpaceBefore)
	f.EnsureSpace(height)
	if f.err != nil {
		return f
	}
	x := f.body.X
	switch options.Align {
	case ImageAlignCenter:
		x += (f.body.Width - width) / 2
	case ImageAlignRight:
		x += f.body.Width - width
	}

	stroke := options.StrokeColor
	if stroke == nil {
		stroke = color.Black
	}
	textColor := options.TextColor
	if textColor == nil {
		textColor = color.Black
	}
	if err := drawASCIIDiagram(f.context, diagram, x, f.y, cellWidth*fitScale,
		lineHeight*fitScale, fontSize*fitScale, lineWidth*fitScale,
		options.CornerRadius*fitScale, stroke, textColor); err != nil {
		return f.failDiagram(err.Error())
	}
	f.y += height
	f.diagramSpaceAfter(options.SpaceAfter)
	return f
}

func (f *Flow) failDiagram(detail string) *Flow {
	if f.err == nil {
		f.err = DiagramLayoutError{Detail: detail}
	}
	return f
}

func (f *Flow) diagramSpaceBefore(space float64) {
	if space == 0 || f.err != nil || f.y <= f.body.Y+0.001 {
		return
	}
	if space > f.RemainingHeight()+0.001 {
		f.NewPage()
	} else {
		f.y += space
	}
}

func (f *Flow) diagramSpaceAfter(space float64) {
	if space == 0 || f.err != nil {
		return
	}
	remaining := f.RemainingHeight()
	if space <= remaining+0.001 {
		f.y += math.Min(space, remaining)
	}
}

func parseASCIIDiagram(source string) (asciiDiagram, error) {
	source = strings.Trim(source, "\r\n")
	if source == "" {
		return asciiDiagram{}, DiagramLayoutError{Detail: "empty source"}
	}
	if strings.ContainsRune(source, '\t') {
		return asciiDiagram{}, DiagramLayoutError{Detail: "tabs are not supported; expand them to spaces"}
	}
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	grid := make([][]string, len(lines))
	columns := 0
	for row, line := range lines {
		grid[row] = diagramGraphemes(line)
		if len(grid[row]) > columns {
			columns = len(grid[row])
		}
	}
	if columns == 0 {
		return asciiDiagram{}, DiagramLayoutError{Detail: "source contains no cells"}
	}
	d := asciiDiagram{rows: len(grid), columns: columns, cells: make([][]diagramCell, len(grid))}
	for row := range d.cells {
		d.cells[row] = make([]diagramCell, columns)
		for col := range d.cells[row] {
			if col < len(grid[row]) {
				d.cells[row][col].text = grid[row][col]
			} else {
				d.cells[row][col].text = " "
			}
		}
	}
	for row := range d.cells {
		for col := range d.cells[row] {
			ports := diagramPorts(d.cells[row][col].text)
			if ports == 0 {
				continue
			}
			for direction, delta := range diagramDeltas {
				neighborRow, neighborCol := row+delta.row, col+delta.col
				if neighborRow < 0 || neighborRow >= d.rows || neighborCol < 0 || neighborCol >= d.columns {
					continue
				}
				dir := diagramDirection(direction)
				if ports&(1<<dir) != 0 && diagramPorts(d.cells[neighborRow][neighborCol].text)&(1<<oppositeDiagramDirection(dir)) != 0 {
					d.cells[row][col].connections |= 1 << dir
				}
			}
			d.cells[row][col].geometry = d.cells[row][col].connections != 0
		}
	}
	return d, nil
}

func diagramGraphemes(line string) []string {
	var cells []string
	joinNext := false
	for _, r := range line {
		combining := unicode.Is(unicode.Mn, r) || r == '\ufe0f' || r == '\ufe0e' || (r >= '\U0001f3fb' && r <= '\U0001f3ff')
		if len(cells) > 0 && (combining || joinNext || r == '\u200d') {
			cells[len(cells)-1] += string(r)
			joinNext = r == '\u200d'
			continue
		}
		cells = append(cells, string(r))
		joinNext = r == '\u200d'
	}
	return cells
}

func diagramPorts(cell string) uint8 {
	switch cell {
	case "-", "=":
		return 1<<dirE | 1<<dirW
	case "|", ":":
		return 1<<dirN | 1<<dirS
	case "/":
		return 1<<dirNE | 1<<dirSW
	case "\\":
		return 1<<dirNW | 1<<dirSE
	case "+", "#", ".", "'":
		return 1<<dirN | 1<<dirE | 1<<dirS | 1<<dirW
	case "<":
		return 1 << dirE
	case ">":
		return 1 << dirW
	case "^":
		return 1<<dirS | 1<<dirSE | 1<<dirSW
	case "v", "V":
		return 1<<dirN | 1<<dirNE | 1<<dirNW
	default:
		return 0
	}
}

func oppositeDiagramDirection(direction diagramDirection) diagramDirection {
	return (direction + 4) % 8
}

func drawASCIIDiagram(c *Context, diagram asciiDiagram, left, top, cellWidth, lineHeight,
	fontSize, lineWidth, cornerRadius float64, stroke, textColor color.Color) error {
	stackDepth := len(c.stack)
	oldAllowedDepth := c.allowedStackDepth
	c.Push()
	c.allowedStackDepth = len(c.stack)
	defer func() {
		c.allowedStackDepth = oldAllowedDepth
		for len(c.stack) > stackDepth {
			c.Pop()
		}
	}()
	c.SetStrokeColor(stroke).SetFillColor(stroke).SetLineWidth(lineWidth).SetLineCap(CapRound).SetLineJoin(JoinRound)
	if cornerRadius == 0 {
		cornerRadius = math.Min(cellWidth, lineHeight) * 0.42
	}
	cornerRadius = math.Min(cornerRadius, math.Min(cellWidth, lineHeight)*0.48)

	// Draw each undirected edge once. Endpoints at rounded corner cells stop
	// short of the center; the corner's quadratic arc joins them below.
	for row := range diagram.cells {
		for col := range diagram.cells[row] {
			cell := diagram.cells[row][col]
			if !cell.geometry {
				continue
			}
			for _, direction := range []diagramDirection{dirE, dirSE, dirS, dirSW} {
				if cell.connections&(1<<direction) == 0 {
					continue
				}
				delta := diagramDeltas[direction]
				otherRow, otherCol := row+delta.row, col+delta.col
				x1, y1 := diagramCellCenter(left, top, cellWidth, lineHeight, row, col)
				x2, y2 := diagramCellCenter(left, top, cellWidth, lineHeight, otherRow, otherCol)
				if diagramRoundedCorner(cell) {
					x1 += float64(delta.col) * cornerRadius
					y1 += float64(delta.row) * cornerRadius
				}
				other := diagram.cells[otherRow][otherCol]
				if diagramRoundedCorner(other) {
					x2 -= float64(delta.col) * cornerRadius
					y2 -= float64(delta.row) * cornerRadius
				}
				if diagramDotted(cell.text) || diagramDotted(other.text) {
					c.SetDash(0, lineWidth*1.2, lineWidth*2.5)
				} else {
					c.SetDash(0)
				}
				c.NewPath().MoveTo(x1, y1).LineTo(x2, y2).Stroke()
			}
		}
	}

	// True rounded corners, rather than merely round line joins.
	c.SetDash(0)
	for row := range diagram.cells {
		for col := range diagram.cells[row] {
			cell := diagram.cells[row][col]
			if !diagramRoundedCorner(cell) {
				continue
			}
			directions := diagramConnectedCardinals(cell.connections)
			if len(directions) != 2 || oppositeDiagramDirection(directions[0]) == directions[1] {
				continue
			}
			x, y := diagramCellCenter(left, top, cellWidth, lineHeight, row, col)
			a, b := directions[0], directions[1]
			ax := x + float64(diagramDeltas[a].col)*cornerRadius
			ay := y + float64(diagramDeltas[a].row)*cornerRadius
			bx := x + float64(diagramDeltas[b].col)*cornerRadius
			by := y + float64(diagramDeltas[b].row)*cornerRadius
			if diagramDotted(cell.text) {
				c.SetDash(0, lineWidth*1.2, lineWidth*2.5)
			}
			c.NewPath().MoveTo(ax, ay).QuadraticTo(x, y, bx, by).Stroke().SetDash(0)
		}
	}

	// Arrowheads are filled triangles centered on their glyph cell.
	for row := range diagram.cells {
		for col := range diagram.cells[row] {
			cell := diagram.cells[row][col]
			if !cell.geometry || !strings.Contains("<>^vV", cell.text) {
				continue
			}
			x, y := diagramCellCenter(left, top, cellWidth, lineHeight, row, col)
			drawDiagramArrow(c, diagramArrowDirection(cell), x, y, math.Min(cellWidth*.48, lineHeight*.34))
		}
	}

	// Draw every maximal non-geometry run. Runs containing emoji are split by
	// cells so a complete grapheme sequence reaches the provider unchanged.
	c.SetFillColor(textColor).SetFont("Courier", fontSize)
	for row := range diagram.cells {
		for col := 0; col < diagram.columns; {
			if diagram.cells[row][col].geometry {
				col++
				continue
			}
			start := col
			for col < diagram.columns && !diagram.cells[row][col].geometry {
				col++
			}
			end := col
			for start < end && diagram.cells[row][start].text == " " {
				start++
			}
			for end > start && diagram.cells[row][end-1].text == " " {
				end--
			}
			if start == end {
				continue
			}
			baseline := top + (float64(row)+.5)*lineHeight + fontSize*.32
			for i := start; i < end; {
				x := left + float64(i)*cellWidth
				if diagramEmojiCell(diagram.cells[row][i].text) {
					c.DrawEmojiAt(diagram.cells[row][i].text, x, baseline, fontSize)
					i++
					continue
				}
				plainStart := i
				for i < end && !diagramEmojiCell(diagram.cells[row][i].text) {
					i++
				}
				var text strings.Builder
				for j := plainStart; j < i; j++ {
					text.WriteString(diagram.cells[row][j].text)
				}
				c.DrawString(text.String(), x, baseline)
			}
		}
	}
	if c.err != nil {
		return c.err
	}
	return nil
}

func diagramCellCenter(left, top, cellWidth, lineHeight float64, row, col int) (float64, float64) {
	return left + (float64(col)+.5)*cellWidth, top + (float64(row)+.5)*lineHeight
}

func diagramRoundedCorner(cell diagramCell) bool {
	return (cell.text == "." || cell.text == "'") && cell.geometry
}

func diagramDotted(cell string) bool { return cell == "=" || cell == ":" }

func diagramConnectedCardinals(connections uint8) []diagramDirection {
	var result []diagramDirection
	for _, direction := range []diagramDirection{dirN, dirE, dirS, dirW} {
		if connections&(1<<direction) != 0 {
			result = append(result, direction)
		}
	}
	return result
}

func diagramArrowDirection(cell diagramCell) diagramDirection {
	var incoming []diagramDirection
	switch cell.text {
	case "<":
		incoming = []diagramDirection{dirE}
	case ">":
		incoming = []diagramDirection{dirW}
	case "^":
		incoming = []diagramDirection{dirS, dirSE, dirSW}
	default: // v and V
		incoming = []diagramDirection{dirN, dirNE, dirNW}
	}
	for _, direction := range incoming {
		if cell.connections&(1<<direction) != 0 {
			return oppositeDiagramDirection(direction)
		}
	}
	return oppositeDiagramDirection(incoming[0])
}

func drawDiagramArrow(c *Context, direction diagramDirection, x, y, radius float64) {
	delta := diagramDeltas[direction]
	length := math.Hypot(float64(delta.col), float64(delta.row))
	ux, uy := float64(delta.col)/length, float64(delta.row)/length
	px, py := -uy, ux
	tipX, tipY := x+ux*radius, y+uy*radius
	baseX, baseY := x-ux*radius*.65, y-uy*radius*.65
	aX, aY := baseX+px*radius*.75, baseY+py*radius*.75
	bX, bY := baseX-px*radius*.75, baseY-py*radius*.75
	c.NewPath().MoveTo(tipX, tipY).LineTo(aX, aY).LineTo(bX, bY).ClosePath().Fill()
}

func diagramEmojiCell(cell string) bool {
	for _, r := range cell {
		if r == '\ufe0f' || r == '\u200d' || r >= '\U0001f000' {
			return true
		}
	}
	return false
}

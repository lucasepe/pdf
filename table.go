package pdf

import (
	"fmt"
	"image/color"
	"math"
)

// TableColumn controls one table column. Width fixes its outer width in
// points. Otherwise the two-pass auto layout uses measured content, MinWidth,
// and Weight when distributing spare width.
type TableColumn struct {
	Width, MinWidth, Weight float64
	Align                   TextAlign
}

// TableCell contains the inline runs rendered in one cell.
type TableCell struct {
	Runs []TextRun
}

// TextTableCell constructs a single-run table cell.
func TextTableCell(text string, style TextStyle) TableCell {
	return TableCell{Runs: []TextRun{{Text: text, Style: style}}}
}

// TableRow is one atomic table row.
type TableRow struct {
	Cells []TableCell
}

// TableStyle controls table spacing and paint. Zero padding and border width
// select compact defaults; set NoBorder to suppress the grid.
type TableStyle struct {
	PaddingX, PaddingY      float64
	LineHeight              float64
	SpaceBefore, SpaceAfter float64
	BorderWidth             float64
	NoBorder                bool
	BorderColor             color.Color
	HeaderFill              color.Color
	OddFill, EvenFill       color.Color
}

// Table is a backend-neutral table. Header is optional. Width zero uses the
// current page body width; a non-zero width is left-aligned within the body.
type Table struct {
	Columns []TableColumn
	Header  []TableCell
	Rows    []TableRow
	Width   float64
	Style   TableStyle
}

// TableError reports invalid schemas, impossible geometry, or an oversized
// atomic row. Row is one-based for body rows and zero for the header/schema.
type TableError struct {
	Row, Column int
	Detail      string
}

func (e TableError) Error() string {
	where := "table"
	if e.Row > 0 {
		where = fmt.Sprintf("table row %d", e.Row)
	} else if e.Row == 0 && e.Column > 0 {
		where = fmt.Sprintf("table column %d", e.Column)
	} else if e.Row < 0 {
		where = "table header"
	}
	if e.Column > 0 && e.Row != 0 {
		where += fmt.Sprintf(" column %d", e.Column)
	}
	return where + ": " + e.Detail
}

type resolvedTableStyle struct {
	paddingX, paddingY      float64
	lineHeight              float64
	spaceBefore, spaceAfter float64
	borderWidth             float64
	noBorder                bool
	borderColor             color.Color
	headerFill              color.Color
	oddFill, evenFill       color.Color
}

type tableCellLayout struct {
	lines []flowLine
}

type tableRowLayout struct {
	cells  []tableCellLayout
	height float64
}

// DrawTable measures and renders a table, opening pages and repeating Header
// as necessary. Rows are never split between pages.
func (f *Flow) DrawTable(table Table) *Flow {
	if f.err != nil {
		return f
	}
	style, ok := f.resolveTableStyle(table.Style)
	if !ok {
		return f
	}
	columnCount, ok := f.tableColumnCount(table)
	if !ok {
		return f
	}
	columns := table.Columns
	if len(columns) == 0 {
		columns = make([]TableColumn, columnCount)
	}
	if len(columns) != columnCount {
		return f.failWithTableError(TableError{Detail: fmt.Sprintf("has %d columns but schema defines %d", columnCount, len(columns))})
	}
	if !f.validateTableRows(table, columnCount) {
		return f
	}
	tableWidth := table.Width
	if tableWidth == 0 {
		tableWidth = f.body.Width
	}
	if !finite(tableWidth) || tableWidth <= 0 || tableWidth > f.body.Width+0.001 {
		return f.failWithTableError(TableError{Detail: fmt.Sprintf("width %.3fpt does not fit %.3fpt body", tableWidth, f.body.Width)})
	}
	widths := f.tableColumnWidths(table, columns, tableWidth, style)
	if f.err != nil {
		return f
	}

	var header *tableRowLayout
	if len(table.Header) > 0 {
		layout := f.layoutTableRow(table.Header, columns, widths, style)
		if f.err != nil {
			return f
		}
		header = &layout
		if header.height > f.body.Height+0.001 {
			return f.failWithTableError(TableError{Row: -1, Detail: fmt.Sprintf("%.3fpt height exceeds %.3fpt page body", header.height, f.body.Height)})
		}
	}
	rows := make([]tableRowLayout, len(table.Rows))
	for i, row := range table.Rows {
		rows[i] = f.layoutTableRow(row.Cells, columns, widths, style)
		if f.err != nil {
			return f
		}
		available := f.body.Height
		if header != nil {
			available -= header.height
		}
		if rows[i].height > available+0.001 {
			return f.failWithTableError(TableError{Row: i + 1, Detail: fmt.Sprintf("%.3fpt height cannot fit with the repeated header in %.3fpt", rows[i].height, f.body.Height)})
		}
	}

	if style.spaceBefore > 0 && f.y > f.body.Y+0.001 {
		if style.spaceBefore > f.RemainingHeight() {
			f.NewPage()
		} else {
			f.y += style.spaceBefore
		}
	}
	firstHeight := 0.0
	if header != nil {
		firstHeight += header.height
	}
	if len(rows) > 0 {
		firstHeight += rows[0].height
	}
	if firstHeight > 0 {
		f.EnsureSpace(firstHeight)
		if f.err != nil {
			return f
		}
	}
	if header != nil {
		f.drawTableRow(*header, widths, columns, style, true, 0)
	}
	for i, row := range rows {
		if row.height > f.RemainingHeight()+0.001 {
			f.NewPage()
			if f.err != nil {
				return f
			}
			if header != nil {
				f.drawTableRow(*header, widths, columns, style, true, 0)
			}
		}
		f.drawTableRow(row, widths, columns, style, false, i)
		if f.err != nil {
			return f
		}
	}
	if style.spaceAfter > 0 {
		f.y = math.Min(f.body.Y+f.body.Height, f.y+style.spaceAfter)
	}
	return f
}

func (f *Flow) failWithTableError(err TableError) *Flow {
	if f.err == nil {
		f.err = err
	}
	return f
}

func (f *Flow) resolveTableStyle(style TableStyle) (resolvedTableStyle, bool) {
	values := []float64{style.PaddingX, style.PaddingY, style.LineHeight, style.SpaceBefore, style.SpaceAfter, style.BorderWidth}
	if !finite(values...) {
		f.failWithTableError(TableError{Detail: "style contains a non-finite value"})
		return resolvedTableStyle{}, false
	}
	for _, value := range values {
		if value < 0 {
			f.failWithTableError(TableError{Detail: "style dimensions cannot be negative"})
			return resolvedTableStyle{}, false
		}
	}
	resolved := resolvedTableStyle{
		paddingX: style.PaddingX, paddingY: style.PaddingY,
		lineHeight: style.LineHeight, spaceBefore: style.SpaceBefore, spaceAfter: style.SpaceAfter,
		borderWidth: style.BorderWidth, noBorder: style.NoBorder,
		borderColor: style.BorderColor, headerFill: style.HeaderFill,
		oddFill: style.OddFill, evenFill: style.EvenFill,
	}
	if resolved.paddingX == 0 {
		resolved.paddingX = 5
	}
	if resolved.paddingY == 0 {
		resolved.paddingY = 4
	}
	if resolved.borderWidth == 0 {
		resolved.borderWidth = .5
	}
	if resolved.borderColor == nil {
		resolved.borderColor = color.RGBA{R: 178, G: 186, B: 196, A: 255}
	}
	if resolved.headerFill == nil {
		resolved.headerFill = color.RGBA{R: 225, G: 232, B: 240, A: 255}
	}
	if resolved.oddFill == nil {
		resolved.oddFill = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	if resolved.evenFill == nil {
		resolved.evenFill = color.RGBA{R: 246, G: 248, B: 250, A: 255}
	}
	return resolved, true
}

func (f *Flow) tableColumnCount(table Table) (int, bool) {
	count := len(table.Columns)
	if count == 0 {
		count = len(table.Header)
	}
	if count == 0 && len(table.Rows) > 0 {
		count = len(table.Rows[0].Cells)
	}
	if count == 0 {
		f.failWithTableError(TableError{Detail: "has no columns"})
		return 0, false
	}
	return count, true
}

func (f *Flow) validateTableRows(table Table, count int) bool {
	if len(table.Header) > 0 && len(table.Header) != count {
		f.failWithTableError(TableError{Row: -1, Detail: fmt.Sprintf("has %d cells, want %d", len(table.Header), count)})
		return false
	}
	for i, column := range table.Columns {
		if !finite(column.Width, column.MinWidth, column.Weight) || column.Width < 0 || column.MinWidth < 0 || column.Weight < 0 || column.Align > AlignRight {
			f.failWithTableError(TableError{Row: 0, Column: i + 1, Detail: "invalid width, minimum, weight, or alignment"})
			return false
		}
	}
	for i, row := range table.Rows {
		if len(row.Cells) != count {
			f.failWithTableError(TableError{Row: i + 1, Detail: fmt.Sprintf("has %d cells, want %d", len(row.Cells), count)})
			return false
		}
	}
	return true
}

func (f *Flow) tableColumnWidths(table Table, columns []TableColumn, width float64, style resolvedTableStyle) []float64 {
	count := len(columns)
	desired := make([]float64, count)
	minimum := make([]float64, count)
	allRows := make([][]TableCell, 0, len(table.Rows)+1)
	if len(table.Header) > 0 {
		allRows = append(allRows, table.Header)
	}
	for _, row := range table.Rows {
		allRows = append(allRows, row.Cells)
	}
	for _, cells := range allRows {
		for i, cell := range cells {
			preferred, min := f.measureTableCell(cell, style.paddingX)
			if f.err != nil {
				return nil
			}
			desired[i] = math.Max(desired[i], preferred)
			minimum[i] = math.Max(minimum[i], min)
		}
	}

	widths := make([]float64, count)
	fixedTotal := 0.0
	var auto []int
	for i, column := range columns {
		minimum[i] = math.Max(minimum[i], column.MinWidth)
		desired[i] = math.Max(desired[i], minimum[i])
		if column.Width > 0 {
			if column.Width+0.001 < minimum[i] {
				f.failWithTableError(TableError{Row: 0, Column: i + 1, Detail: fmt.Sprintf("fixed %.3fpt width is below %.3fpt content minimum", column.Width, minimum[i])})
				return nil
			}
			widths[i] = column.Width
			fixedTotal += column.Width
		} else {
			auto = append(auto, i)
		}
	}
	remaining := width - fixedTotal
	if remaining < -0.001 {
		f.failWithTableError(TableError{Detail: fmt.Sprintf("fixed columns exceed %.3fpt table width", width)})
		return nil
	}
	if len(auto) == 0 {
		if math.Abs(remaining) > 0.001 {
			f.failWithTableError(TableError{Detail: fmt.Sprintf("fixed columns leave %.3fpt undistributed", remaining)})
			return nil
		}
		return widths
	}
	sumMin, sumDesired := 0.0, 0.0
	for _, i := range auto {
		sumMin += minimum[i]
		sumDesired += desired[i]
	}
	if sumMin > remaining+0.001 {
		f.failWithTableError(TableError{Detail: fmt.Sprintf("minimum column widths %.3fpt exceed %.3fpt available", sumMin, remaining)})
		return nil
	}
	if sumDesired > remaining+0.001 {
		capacity := sumDesired - sumMin
		ratio := 0.0
		if capacity > 0 {
			ratio = (remaining - sumMin) / capacity
		}
		for _, i := range auto {
			widths[i] = minimum[i] + (desired[i]-minimum[i])*ratio
		}
	} else {
		extra := remaining - sumDesired
		weightTotal := 0.0
		for _, i := range auto {
			weight := columns[i].Weight
			if weight == 0 {
				weight = 1
			}
			weightTotal += weight
		}
		for _, i := range auto {
			weight := columns[i].Weight
			if weight == 0 {
				weight = 1
			}
			widths[i] = desired[i] + extra*weight/weightTotal
		}
	}
	// Absorb floating point residue in the last automatic column.
	total := 0.0
	for _, value := range widths {
		total += value
	}
	widths[auto[len(auto)-1]] += width - total
	return widths
}

func (f *Flow) measureTableCell(cell TableCell, paddingX float64) (preferred, minimum float64) {
	lineWidth := 0.0
	for _, run := range cell.Runs {
		style := f.resolveStyle(run.Style)
		if f.err != nil {
			return 0, 0
		}
		if run.Emoji {
			width, err := f.context.doc.MeasureEmoji(run.Text, style.fontSize)
			if err != nil {
				f.failWithTableError(TableError{Detail: err.Error()})
				return 0, 0
			}
			lineWidth += width
			minimum = math.Max(minimum, width)
			continue
		}
		for _, r := range run.Text {
			if r == '\n' {
				preferred = math.Max(preferred, lineWidth)
				lineWidth = 0
				continue
			}
			width := f.measure(string(r), style)
			if f.err != nil {
				return 0, 0
			}
			lineWidth += width
			minimum = math.Max(minimum, width)
		}
	}
	preferred = math.Max(preferred, lineWidth) + 2*paddingX
	minimum += 2 * paddingX
	return preferred, minimum
}

func (f *Flow) layoutTableRow(cells []TableCell, columns []TableColumn, widths []float64, style resolvedTableStyle) tableRowLayout {
	row := tableRowLayout{cells: make([]tableCellLayout, len(cells))}
	for i, cell := range cells {
		contentWidth := widths[i] - 2*style.paddingX
		if contentWidth <= 0 {
			f.failWithTableError(TableError{Row: 0, Column: i + 1, Detail: "padding leaves no cell content width"})
			return tableRowLayout{}
		}
		lines := f.breakLinesWidth(cell.Runs, ParagraphStyle{Align: columns[i].Align, LineHeight: style.lineHeight}, contentWidth)
		if f.err != nil {
			return tableRowLayout{}
		}
		height := 2 * style.paddingY
		for _, line := range lines {
			height += line.height
		}
		row.cells[i] = tableCellLayout{lines: lines}
		row.height = math.Max(row.height, height)
	}
	return row
}

func (f *Flow) drawTableRow(row tableRowLayout, widths []float64, columns []TableColumn, style resolvedTableStyle, header bool, logicalIndex int) {
	if f.err != nil {
		return
	}
	top := f.y
	fill := style.headerFill
	if !header {
		if logicalIndex%2 == 0 {
			fill = style.oddFill
		} else {
			fill = style.evenFill
		}
	}
	if fill != nil {
		f.context.SetFillColor(fill).DrawRectangle(f.body.X, top, sumFloat64(widths), row.height).Fill()
	}
	x := f.body.X
	for i, cell := range row.cells {
		contentWidth := widths[i] - 2*style.paddingX
		lineY := top + style.paddingY
		for _, line := range cell.lines {
			lineX := x + style.paddingX
			switch columns[i].Align {
			case AlignCenter:
				lineX += (contentWidth - line.width) / 2
			case AlignRight:
				lineX += contentWidth - line.width
			}
			baseline := lineY + line.maxSize*.82
			for _, token := range line.tokens {
				f.applyStyle(token.style)
				if f.err != nil {
					return
				}
				if token.emoji {
					f.context.DrawEmojiAt(token.text, lineX, baseline, token.style.fontSize)
				} else {
					f.context.DrawString(token.text, lineX, baseline)
				}
				if err := f.context.Error(); err != nil {
					f.fail("draw table", err.Error())
					return
				}
				lineX += token.width
			}
			lineY += line.height
		}
		x += widths[i]
	}
	if !style.noBorder {
		f.context.SetStrokeColor(style.borderColor).SetLineWidth(style.borderWidth).
			DrawRectangle(f.body.X, top, sumFloat64(widths), row.height).Stroke()
		x = f.body.X
		for i := 0; i < len(widths)-1; i++ {
			x += widths[i]
			f.context.MoveTo(x, top).LineTo(x, top+row.height).Stroke()
		}
	}
	if err := f.context.Error(); err != nil {
		f.fail("draw table", err.Error())
		return
	}
	f.y += row.height
}

func sumFloat64(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total
}

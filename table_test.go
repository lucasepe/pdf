package pdf

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTableColumnWidthsMixFixedAndMeasuredColumns(t *testing.T) {
	doc := NewPDF("220pt x 240pt")
	flow, err := NewFlow(NewContext(&doc), PageSpec{Width: 220, Height: 240, MarginTop: 20, MarginRight: 20, MarginBottom: 20, MarginLeft: 20}, nil)
	if err != nil {
		t.Fatal(err)
	}
	text := TextStyle{FontRole: FontRoleMono, FontSize: 8}
	table := Table{
		Columns: []TableColumn{{Width: 36}, {}, {Weight: 2}},
		Header:  []TableCell{TextTableCell("ID", text), TextTableCell("Nome", text), TextTableCell("Descrizione", text)},
		Rows: []TableRow{{Cells: []TableCell{
			TextTableCell("01", text), TextTableCell("Unità", text), TextTableCell("Descrizione decisamente più lunga", text),
		}}},
	}
	style, ok := flow.resolveTableStyle(TableStyle{PaddingX: 3, PaddingY: 2})
	if !ok {
		t.Fatal(flow.Error())
	}
	widths := flow.tableColumnWidths(table, table.Columns, 180, style)
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	if got := sumFloat64(widths); math.Abs(got-180) > 0.001 {
		t.Fatalf("column total = %.6f, want 180", got)
	}
	if widths[0] != 36 {
		t.Fatalf("fixed column = %.3f, want 36", widths[0])
	}
	if widths[2] <= widths[1] {
		t.Fatalf("measured columns = %v, want description wider than name", widths)
	}
}

func TestTablePaginationRepeatsHeaderKeepsRowsAtomicAndZebraStable(t *testing.T) {
	doc := NewPDF("240pt x 170pt")
	doc.SetCompression(false)
	flow, err := NewFlow(NewContext(&doc), PageSpec{Width: 240, Height: 170, MarginTop: 15, MarginRight: 15, MarginBottom: 15, MarginLeft: 15}, nil)
	if err != nil {
		t.Fatal(err)
	}
	text := TextStyle{FontRole: FontRoleMono, FontSize: 8}
	header := TextStyle{FontRole: FontRoleMonoBold, FontSize: 8}
	table := Table{
		Columns: []TableColumn{{Width: 42, Align: AlignRight}, {}},
		Header:  []TableCell{TextTableCell("COD", header), TextTableCell("HEADER_NAME", header)},
		Style: TableStyle{
			PaddingX: 3, PaddingY: 2, LineHeight: 10,
			OddFill: color.RGBA{R: 255, A: 255}, EvenFill: color.RGBA{B: 255, A: 255},
		},
	}
	for i := 1; i <= 9; i++ {
		table.Rows = append(table.Rows, TableRow{Cells: []TableCell{
			TextTableCell(fmt.Sprintf("%02d", i), text),
			TextTableCell(fmt.Sprintf("ROW_%02d descrizione estesa che deve andare a capo nella cella", i), text),
		}})
	}
	flow.DrawTable(table)
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	if doc.PageCount() < 2 {
		t.Fatalf("PageCount() = %d, want multipage table", doc.PageCount())
	}
	all := ""
	for pageIndex, page := range doc.pages {
		content := page.content.String()
		all += content
		if got := strings.Count(content, "(HEADER_NAME) Tj"); got != 1 {
			t.Fatalf("page %d header count = %d\n%s", pageIndex+1, got, content)
		}
	}
	for i := 1; i <= 9; i++ {
		marker := fmt.Sprintf("(ROW_%02d) Tj", i)
		pages := 0
		for _, page := range doc.pages {
			if strings.Contains(page.content.String(), marker) {
				pages++
			}
		}
		if pages != 1 {
			t.Fatalf("%s appears on %d pages", marker, pages)
		}
	}
	if got := strings.Count(all, "1.000 0.000 0.000 rg"); got != 5 {
		t.Fatalf("odd zebra fill count = %d, want 5", got)
	}
	if got := strings.Count(all, "0.000 0.000 1.000 rg"); got != 4 {
		t.Fatalf("even zebra fill count = %d, want 4", got)
	}
}

func TestTableRejectsOversizedAtomicRowAndIrregularSchema(t *testing.T) {
	newFlow := func(t *testing.T) *Flow {
		t.Helper()
		doc := NewPDF("120pt x 100pt")
		flow, err := NewFlow(NewContext(&doc), PageSpec{Width: 120, Height: 100, MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return flow
	}
	text := TextStyle{FontRole: FontRoleMono, FontSize: 8}
	flow := newFlow(t)
	flow.DrawTable(Table{
		Columns: []TableColumn{{}},
		Rows:    []TableRow{{Cells: []TableCell{TextTableCell(strings.Repeat("una parola ", 80), text)}}},
		Style:   TableStyle{PaddingX: 3, PaddingY: 2, LineHeight: 10},
	})
	var tableErr TableError
	if !errors.As(flow.Error(), &tableErr) || tableErr.Row != 1 || !strings.Contains(tableErr.Detail, "cannot fit") {
		t.Fatalf("oversized row error = %#v", flow.Error())
	}

	flow = newFlow(t)
	flow.DrawTable(Table{
		Columns: []TableColumn{{}, {}},
		Rows:    []TableRow{{Cells: []TableCell{TextTableCell("only one", text)}}},
	})
	if !errors.As(flow.Error(), &tableErr) || tableErr.Row != 1 || !strings.Contains(tableErr.Detail, "want 2") {
		t.Fatalf("irregular schema error = %#v", flow.Error())
	}
}

func TestTableGeneratedPDFPassesQPDF(t *testing.T) {
	qpdf, err := exec.LookPath("qpdf")
	if err != nil {
		t.Skip("qpdf not installed")
	}
	doc := NewPDF("240pt x 180pt")
	flow, err := NewFlow(NewContext(&doc), PageSpec{Width: 240, Height: 180, MarginTop: 15, MarginRight: 15, MarginBottom: 15, MarginLeft: 15}, nil)
	if err != nil {
		t.Fatal(err)
	}
	text := TextStyle{FontRole: FontRoleMono, FontSize: 8}
	table := Table{
		Columns: []TableColumn{{Width: 45}, {}},
		Header:  []TableCell{TextTableCell("ID", text), TextTableCell("Descrizione", text)},
		Style:   TableStyle{LineHeight: 10},
	}
	for i := 0; i < 12; i++ {
		table.Rows = append(table.Rows, TableRow{Cells: []TableCell{TextTableCell("01", text), TextTableCell("Perché l’unità è pronta — € 12,50", text)}})
	}
	flow.DrawTable(table)
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "table.pdf")
	if err := flow.context.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(qpdf, "--check", path).CombinedOutput(); err != nil {
		t.Fatalf("qpdf --check: %v\n%s", err, output)
	}
}

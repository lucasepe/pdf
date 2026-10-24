package pdf

import (
	"bytes"
	"strings"
	"testing"
)

func TestTableLinksAcrossWrappedRowsAndRepeatedHeaders(t *testing.T) {
	for _, align := range []TextAlign{AlignLeft, AlignCenter, AlignRight} {
		doc := NewPDF("200pt x 160pt")
		ctx := NewContext(&doc)
		flow, err := NewFlow(ctx, PageSpec{Width: 200, Height: 160, MarginTop: 10, MarginBottom: 10, MarginLeft: 10, MarginRight: 10}, nil)
		if err != nil {
			t.Fatal(err)
		}
		style := TextStyle{FontRole: FontRoleMono, FontSize: 8}
		table := Table{Columns: []TableColumn{{Width: 90, Align: align}, {Width: 90, Align: align}}, Header: []TableCell{
			{Runs: []TextRun{{Text: "Website", Style: style, URI: "https://example.com"}}},
			TextTableCell("Plain", style),
		}}
		for i := 0; i < 12; i++ {
			table.Rows = append(table.Rows, TableRow{Cells: []TableCell{
				{Runs: []TextRun{{Text: "multiple words that wrap to another line", Style: style, LinkDestination: "target"}}},
				TextTableCell("No link", style),
			}})
		}
		flow.DrawTable(table)
		if err := flow.Error(); err != nil {
			t.Fatal(err)
		}
		ctx.AddDestination("target", 10, flow.CursorY())
		if doc.PageCount() < 2 {
			t.Fatal("table did not span pages")
		}
		for n, page := range doc.pages {
			external, internal := 0, 0
			for _, link := range page.links {
				if link.uri != "" {
					external++
				}
				if link.target != "" {
					internal++
				}
				if link.rect.X < 10 || link.rect.X+link.rect.Width > 100.001 || link.rect.Y < 10 || link.rect.Y+link.rect.Height > 150.001 {
					t.Fatalf("page %d link outside cell/page: %+v", n, link)
				}
			}
			if external == 0 || internal == 0 {
				t.Fatalf("page %d lacks header/body links", n)
			}
		}
		var out bytes.Buffer
		if _, err := ctx.WriteTo(&out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "/S/URI") || !strings.Contains(out.String(), "/S/GoTo") {
			t.Fatal("missing PDF annotations")
		}
	}
}

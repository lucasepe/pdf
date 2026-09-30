package pdf

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type diagramEmojiProvider struct{ image image.Image }

func (p diagramEmojiProvider) LookupEmoji(sequence string) (EmojiAsset, bool, error) {
	if sequence != "☝️" {
		return EmojiAsset{}, false, nil
	}
	return EmojiAsset{Image: p.image, HeightEm: 1.05, BaselineEm: -.12, AdvanceEm: 1}, true, nil
}

func TestParseASCIIDiagramConnectivityAndUnicodeCells(t *testing.T) {
	diagram, err := parseASCIIDiagram(".----->\n| café |\n'-----'")
	if err != nil {
		t.Fatal(err)
	}
	if diagram.rows != 3 || diagram.columns != 8 {
		t.Fatalf("dimensions = %dx%d", diagram.columns, diagram.rows)
	}
	if got := diagram.cells[0][0].connections; got&(1<<dirE) == 0 || got&(1<<dirS) == 0 {
		t.Fatalf("rounded corner connections = %08b", got)
	}
	if !diagram.cells[0][6].geometry || diagram.cells[1][4].geometry {
		t.Fatal("arrow was not geometry or label punctuation became geometry")
	}
	emoji := diagramGraphemes("A☝️B")
	if len(emoji) != 3 || emoji[1] != "☝️" {
		t.Fatalf("emoji cells = %#v", emoji)
	}
}

func TestDiagramArrowDirectionFollowsDiagonalConnection(t *testing.T) {
	tests := []struct {
		source   string
		row, col int
		want     diagramDirection
	}{
		{"  ^\n /", 0, 2, dirNE},
		{"^\n \\", 0, 0, dirNW},
		{" /\nv", 1, 0, dirSW},
		{"\\\n v", 1, 1, dirSE},
		{"^\n|", 0, 0, dirN},
		{"|\nv", 1, 0, dirS},
	}
	for _, test := range tests {
		diagram, err := parseASCIIDiagram(test.source)
		if err != nil {
			t.Fatal(err)
		}
		cell := diagram.cells[test.row][test.col]
		if got := diagramArrowDirection(cell); got != test.want {
			t.Errorf("arrow direction for %q = %d, want %d (connections %08b)", test.source, got, test.want, cell.connections)
		}
	}
}

func TestDrawASCIIDiagramEmitsCurvesDashesArrowsAndEmoji(t *testing.T) {
	spec := PageSpec{Width: 220, Height: 180, MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10}
	doc := NewPDF(spec.PaperSize())
	doc.SetCompression(false)
	if err := doc.SetEmojiProvider(diagramEmojiProvider{image: testDiagramEmoji()}); err != nil {
		t.Fatal(err)
	}
	c := NewContext(&doc)
	flow, err := NewFlow(c, spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow.DrawASCIIDiagram("☝️\n.====.\n: go :---->\n'===='", ASCIIDiagramOptions{})
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	content := doc.pages[0].content.String()
	for _, operator := range []string{" c\n", "] 0.000 d\n", "h\n", "/IMG0 Do"} {
		if !strings.Contains(content, operator) {
			t.Fatalf("content missing %q:\n%s", operator, content)
		}
	}
	if len(doc.images) != 1 {
		t.Fatalf("embedded images = %d", len(doc.images))
	}
}

func TestDrawASCIIDiagramFitsAndPaginatesAtomically(t *testing.T) {
	spec := PageSpec{Width: 120, Height: 100, MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10}
	doc := NewPDF(spec.PaperSize())
	c := NewContext(&doc)
	flow, err := NewFlow(c, spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow.Advance(60).DrawASCIIDiagram(".----------------.\n| fitted diagram |\n'----------------'", ASCIIDiagramOptions{})
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	if flow.PageNumber() != 2 {
		t.Fatalf("page = %d, want 2", flow.PageNumber())
	}
	if flow.CursorY() > flow.Body().Y+flow.Body().Height+.001 {
		t.Fatalf("cursor %.3f exceeds body", flow.CursorY())
	}
}

func TestDrawASCIIDiagramValidation(t *testing.T) {
	spec := PageSpec{Width: 100, Height: 100, MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10}
	doc := NewPDF(spec.PaperSize())
	flow, err := NewFlow(NewContext(&doc), spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow.DrawASCIIDiagram("", ASCIIDiagramOptions{})
	if err := flow.Error(); err == nil || !strings.Contains(err.Error(), "empty source") {
		t.Fatalf("empty source error = %v", err)
	}
}

func TestAllEbookgenDiagramFixturesRenderOnNinePages(t *testing.T) {
	names, err := filepath.Glob("_examples/ascii_diagrams/testdata/*.txt")
	if err != nil || len(names) != 9 {
		t.Fatalf("fixture corpus = %d files, %v", len(names), err)
	}
	spec := PageSpec{Width: Inches(6), Height: Inches(9), MarginTop: 42, MarginRight: 24, MarginBottom: 42, MarginLeft: 24}
	doc := NewPDF(spec.PaperSize())
	if err := doc.SetEmojiProvider(diagramEmojiProvider{image: testDiagramEmoji()}); err != nil {
		t.Fatal(err)
	}
	flow, err := NewFlow(NewContext(&doc), spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range names {
		if i > 0 {
			flow.NewPage()
		}
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		flow.DrawASCIIDiagram(string(source), ASCIIDiagramOptions{})
	}
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	if got := doc.PageCount(); got != 9 {
		t.Fatalf("pages = %d, want 9", got)
	}
}

func TestASCIIDiagramPDFPassesQPDF(t *testing.T) {
	qpdf, err := exec.LookPath("qpdf")
	if err != nil {
		t.Skip("qpdf not installed")
	}
	spec := PageSpec{Width: Inches(6), Height: Inches(9), MarginTop: 24, MarginRight: 24, MarginBottom: 24, MarginLeft: 24}
	doc := NewPDF(spec.PaperSize())
	c := NewContext(&doc)
	flow, err := NewFlow(c, spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow.DrawASCIIDiagram(".--------.\n| direct |---->\n'--------'", ASCIIDiagramOptions{})
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if _, err := c.WriteTo(&output); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "diagram.pdf")
	if err := c.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	if check, err := exec.Command(qpdf, "--check", path).CombinedOutput(); err != nil {
		t.Fatalf("qpdf --check: %v\n%s", err, check)
	}
}

func testDiagramEmoji() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 24, 24))
	for y := 3; y < 21; y++ {
		for x := 8; x < 16; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 244, G: 184, B: 116, A: 255})
		}
	}
	return img
}

package pdf

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPageSpecBodyUsesMirroredBindingOffset(t *testing.T) {
	spec := PageSpec{
		Width: 432, Height: 648,
		MarginTop: 18, MarginRight: 18, MarginBottom: 18, MarginLeft: 18,
		BindingOffset: 9, HeaderHeight: 20, FooterHeight: 24,
	}
	odd, even := spec.Body(1), spec.Body(2)
	if odd != (Rect{X: 27, Y: 38, Width: 387, Height: 568}) {
		t.Fatalf("odd body = %+v", odd)
	}
	if even != (Rect{X: 18, Y: 38, Width: 387, Height: 568}) {
		t.Fatalf("even body = %+v", even)
	}
	if got := spec.PaperSize(); got != "432.000000pt x 648.000000pt" {
		t.Fatalf("PaperSize() = %q", got)
	}
}

func TestFlowBreaksMixedRunsAndLongTokensWithinBody(t *testing.T) {
	doc := NewPDF("180pt x 240pt")
	if err := doc.RegisterFont("Body", loadVera(t)); err != nil {
		t.Fatal(err)
	}
	if err := doc.BindFontRole(FontRoleBody, "Body"); err != nil {
		t.Fatal(err)
	}
	c := NewContext(&doc)
	spec := PageSpec{Width: 180, Height: 240, MarginTop: 20, MarginRight: 20, MarginBottom: 20, MarginLeft: 20}
	flow, err := NewFlow(c, spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	runs := []TextRun{
		{Text: "Perché una riga con accenti deve andare a capo senza oltrepassare i margini. ", Style: TextStyle{FontRole: FontRoleBody, FontSize: 11}},
		{Text: "config.Path=/directory_estremamente_lunga/senza/spazi/output.pdf", Style: TextStyle{FontRole: FontRoleMono, FontSize: 9, Wrap: WrapCode}},
		{Text: " Supercalifragilistichespiralidosissimo", Style: TextStyle{FontRole: FontRoleBody, FontSize: 11}},
	}
	lines := flow.breakLines(runs, ParagraphStyle{FirstLineIndent: 12})
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	if len(lines) < 5 {
		t.Fatalf("got %d lines, want wrapping", len(lines))
	}
	for i, line := range lines {
		limit := flow.body.Width
		if i == 0 {
			limit -= 12
		}
		if line.width > limit+0.001 {
			t.Fatalf("line %d width %.3f exceeds %.3f", i, line.width, limit)
		}
		if len(line.tokens) > 0 && line.tokens[len(line.tokens)-1].space {
			t.Fatalf("line %d retains trailing whitespace", i)
		}
	}
}

func TestFlowHangingIndentWrapsContinuationLinesInsideIndentedWidth(t *testing.T) {
	doc := NewPDF("180pt x 240pt")
	if err := doc.RegisterFont("Body", loadVera(t)); err != nil {
		t.Fatal(err)
	}
	if err := doc.BindFontRole(FontRoleBody, "Body"); err != nil {
		t.Fatal(err)
	}
	doc.SetCompression(false)
	flow, err := NewFlow(NewContext(&doc), PageSpec{
		Width: 180, Height: 240,
		MarginTop: 20, MarginRight: 20, MarginBottom: 20, MarginLeft: 20,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	style := ParagraphStyle{LineHeight: 13, LeftIndent: 18, FirstLineIndent: -18}
	runs := []TextRun{{
		Text:  "10. A numbered item long enough to wrap over several continuation lines without returning below its marker.",
		Style: TextStyle{FontRole: FontRoleBody, FontSize: 10},
	}}
	lines := flow.breakLines(runs, style)
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	if len(lines) < 2 {
		t.Fatalf("got %d lines, want wrapping", len(lines))
	}
	for i, line := range lines {
		limit := flow.body.Width - style.LeftIndent
		if i == 0 {
			limit -= style.FirstLineIndent
		}
		if line.width > limit+0.001 {
			t.Fatalf("line %d width %.3f exceeds %.3f", i, line.width, limit)
		}
	}

	flow.DrawParagraph(runs, style)
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	content := doc.pages[0].content.String()
	if !strings.Contains(content, "BT 20 ") {
		t.Fatalf("first line does not start at the body edge:\n%s", content)
	}
	if !strings.Contains(content, "BT 38 ") {
		t.Fatalf("continuation line does not start at the hanging indent:\n%s", content)
	}
}

func TestFlowRejectsInvalidParagraphIndents(t *testing.T) {
	tests := []ParagraphStyle{
		{LeftIndent: -1},
		{LeftIndent: 80},
		{LeftIndent: 10, FirstLineIndent: -11},
		{LeftIndent: 70, FirstLineIndent: 10},
	}
	for _, style := range tests {
		doc := NewPDF("100pt x 100pt")
		flow, err := NewFlow(NewContext(&doc), PageSpec{
			Width: 100, Height: 100,
			MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10,
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
		flow.DrawParagraph([]TextRun{{Text: "invalid"}}, style)
		if err := flow.Error(); err == nil || !strings.Contains(err.Error(), "invalid paragraph geometry") {
			t.Fatalf("style %+v error = %v", style, err)
		}
	}
}

func TestFlowAutomaticAndExplicitPaginationDecoratesEveryPage(t *testing.T) {
	doc := NewPDF("200pt x 150pt")
	doc.SetCompression(false)
	c := NewContext(&doc)
	spec := PageSpec{
		Width: 200, Height: 150,
		MarginTop: 10, MarginRight: 20, MarginBottom: 10, MarginLeft: 20,
		HeaderHeight: 15, FooterHeight: 15,
	}
	var decorated []int
	flow, err := NewFlow(c, spec, func(c *Context, page PageInfo) error {
		decorated = append(decorated, page.Number)
		c.UseFontRole(FontRoleMono, 7).DrawString(fmt.Sprintf("page-%d", page.Number), page.Body.X, 15)
		return c.Error()
	})
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("Una riga abbastanza lunga deve andare a capo entro il body. ", 14)
	flow.DrawParagraph([]TextRun{{Text: text, Style: TextStyle{FontRole: FontRoleMono, FontSize: 9}}}, ParagraphStyle{LineHeight: 12, SpaceAfter: 4})
	automaticPages := flow.PageNumber()
	if automaticPages < 2 {
		t.Fatalf("automatic page count = %d", automaticPages)
	}
	flow.NewPage().DrawParagraph([]TextRun{{Text: "Dopo il page break esplicito.", Style: TextStyle{FontRole: FontRoleMono, FontSize: 9}}}, ParagraphStyle{})
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	if got, want := doc.PageCount(), automaticPages+1; got != want {
		t.Fatalf("PageCount() = %d, want %d", got, want)
	}
	if len(decorated) != doc.PageCount() {
		t.Fatalf("decorated %v for %d pages", decorated, doc.PageCount())
	}
	for i, page := range doc.pages {
		if !strings.Contains(page.content.String(), fmt.Sprintf("(page-%d) Tj", i+1)) {
			t.Fatalf("page %d lacks decorator:\n%s", i+1, page.content.String())
		}
	}
}

func TestFlowNewlineAlignmentAndParagraphSpacing(t *testing.T) {
	doc := NewPDF("200pt x 150pt")
	doc.SetCompression(false)
	flow, err := NewFlow(NewContext(&doc), PageSpec{Width: 200, Height: 150, MarginTop: 20, MarginRight: 20, MarginBottom: 20, MarginLeft: 20}, nil)
	if err != nil {
		t.Fatal(err)
	}
	style := TextStyle{FontRole: FontRoleMono, FontSize: 10}
	flow.DrawParagraph([]TextRun{{Text: "A\nBB", Style: style}}, ParagraphStyle{Align: AlignRight, LineHeight: 12})
	flow.DrawParagraph([]TextRun{{Text: "C", Style: style}}, ParagraphStyle{LineHeight: 12, SpaceBefore: 5, SpaceAfter: 3})
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	if got, want := flow.CursorY(), 64.0; got != want {
		t.Fatalf("CursorY() = %.3f, want %.3f", got, want)
	}
	content := doc.pages[0].content.String()
	for _, want := range []string{"BT 174 121 Td (A) Tj", "BT 168 109 Td (BB) Tj", "BT 20 92 Td (C) Tj"} {
		if !strings.Contains(content, want) {
			t.Fatalf("content missing %q:\n%s", want, content)
		}
	}
}

func TestFlowDecoratorCannotChangePage(t *testing.T) {
	doc := NewPDF("100pt x 100pt")
	_, err := NewFlow(NewContext(&doc), PageSpec{Width: 100, Height: 100, MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10}, func(c *Context, _ PageInfo) error {
		c.AddPage()
		return c.Error()
	})
	if err == nil || !strings.Contains(err.Error(), "graphics-state stack") {
		t.Fatalf("decorator page change error = %v", err)
	}
	if got := doc.PageCount(); got != 1 {
		t.Fatalf("decorator changed page count to %d", got)
	}
}

func TestFlowEmojiIsAtomicAndOversizeIsExplicit(t *testing.T) {
	doc := NewPDF("100pt x 100pt")
	if err := doc.SetEmojiProvider(pngEmojiProvider{
		assets:  map[string][]byte{"🎯": transparentEmojiPNG(t)},
		metrics: EmojiAsset{HeightEm: 1, AdvanceEm: 10},
	}); err != nil {
		t.Fatal(err)
	}
	flow, err := NewFlow(NewContext(&doc), PageSpec{Width: 100, Height: 100, MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10}, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow.DrawParagraph([]TextRun{{Text: "🎯", Emoji: true, Style: TextStyle{FontSize: 10}}}, ParagraphStyle{})
	if err := flow.Error(); err == nil || !strings.Contains(err.Error(), "emoji") {
		t.Fatalf("oversize emoji error = %v", err)
	}
}

func TestFlowGeneratedPDFPassesQPDF(t *testing.T) {
	qpdf, err := exec.LookPath("qpdf")
	if err != nil {
		t.Skip("qpdf not installed")
	}
	spec := PageSpec{Width: Inches(6), Height: Inches(9), MarginTop: 24, MarginRight: 24, MarginBottom: 24, MarginLeft: 24}
	doc := NewPDF(spec.PaperSize())
	doc.SetCompression(false)
	c := NewContext(&doc)
	flow, err := NewFlow(c, spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow.DrawParagraph([]TextRun{{Text: strings.Repeat("Latin wrapping valido — perché sì. ", 120), Style: TextStyle{FontRole: FontRoleMono, FontSize: 10}}}, ParagraphStyle{})
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if _, err := c.WriteTo(&output); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "flow.pdf")
	if err := c.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	if check, err := exec.Command(qpdf, "--check", path).CombinedOutput(); err != nil {
		t.Fatalf("qpdf --check: %v\n%s", err, check)
	}
}

func TestWrapCodePreservesLeadingAndRepeatedSpaces(t *testing.T) {
	doc := NewPDF("200pt x 100pt")
	doc.SetCompression(false)
	ctx := NewContext(&doc)
	flow, err := NewFlow(ctx, PageSpec{Width: 200, Height: 100, MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10}, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow.DrawParagraph([]TextRun{{Text: "    return  two", Style: TextStyle{FontRole: FontRoleMono, FontSize: 9, Wrap: WrapCode}}}, ParagraphStyle{})
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	content := doc.pages[0].content.String()
	if !strings.Contains(content, "( ) Tj") || strings.Count(content, "( ) Tj") < 5 {
		t.Fatalf("code whitespace was not preserved:\n%s", content)
	}
}

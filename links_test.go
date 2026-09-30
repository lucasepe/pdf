package pdf

import (
	"bytes"
	"strings"
	"testing"
)

func TestLinkAnnotationsSerializeURIAndGoToActions(t *testing.T) {
	doc := NewPDF("200 pt x 200 pt")
	doc.SetCompression(false)
	ctx := NewContext(&doc)
	ctx.AddDestination("intro", 12, 18).
		AddExternalLink("https://example.com/a_(b)", Rect{X: 10, Y: 30, Width: 80, Height: 12})
	ctx.AddPage().AddInternalLink("intro", Rect{X: 20, Y: 40, Width: 60, Height: 14})

	var output bytes.Buffer
	if _, err := ctx.WriteTo(&output); err != nil {
		t.Fatal(err)
	}
	pdf := output.String()
	for _, want := range []string{
		"/Annots[", "/S/URI/URI(https://example.com/a_\\(b\\))",
		"/S/GoTo/D[3 0 R/XYZ 12 182 null]", "/Rect[10 158 90 170]",
	} {
		if !strings.Contains(pdf, want) {
			t.Fatalf("generated PDF does not contain %q", want)
		}
	}
}

func TestUnresolvedInternalLinkFailsSerialization(t *testing.T) {
	doc := NewPDF("200 pt x 200 pt")
	ctx := NewContext(&doc)
	ctx.AddInternalLink("missing", Rect{X: 10, Y: 10, Width: 20, Height: 10})
	var output bytes.Buffer
	if _, err := ctx.WriteTo(&output); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected unresolved destination error, got %v", err)
	}
}

func TestFlowCreatesLinkAnnotationsAfterWrapping(t *testing.T) {
	doc := NewPDF("200 pt x 200 pt")
	doc.SetCompression(false)
	ctx := NewContext(&doc)
	flow, err := NewFlow(ctx, PageSpec{Width: 200, Height: 200, MarginTop: 10, MarginRight: 130, MarginBottom: 10, MarginLeft: 10}, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow.DrawParagraph([]TextRun{{Text: "alpha beta gamma", URI: "https://example.com", Style: TextStyle{FontSize: 10}}}, ParagraphStyle{LineHeight: 12})
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	if got := len(doc.pages[0].links); got < 3 {
		t.Fatalf("expected wrapped linked tokens, got %d annotations", got)
	}
}

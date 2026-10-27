package pdf

import (
	"math"
	"testing"
)

func TestRightIndentWrapAndAlignment(t *testing.T) {
	for _, align := range []TextAlign{AlignLeft, AlignCenter, AlignRight} {
		doc := NewPDF("220pt x 200pt")
		flow, err := NewFlow(NewContext(&doc), PageSpec{Width: 220, Height: 200, MarginLeft: 10, MarginRight: 10, MarginTop: 10, MarginBottom: 10}, nil)
		if err != nil {
			t.Fatal(err)
		}
		flow.DrawParagraph([]TextRun{{Text: "word word word word word word word word word word word word", Style: TextStyle{FontRole: FontRoleMono, FontSize: 10}, URI: "https://example.com"}}, ParagraphStyle{LeftIndent: 20, RightIndent: 30, FirstLineIndent: -10, Align: align})
		if err := flow.Error(); err != nil {
			t.Fatal(err)
		}
		links := doc.pages[0].links
		if len(links) < 2 {
			t.Fatal("missing text annotations")
		}
		for _, link := range links {
			if link.rect.X < 20-.001 || link.rect.X+link.rect.Width > 180+.001 {
				t.Fatalf("text outside indented area: %+v", link.rect)
			}
		}
	}
}

func TestRightIndentValidation(t *testing.T) {
	for _, value := range []float64{-1, 100, math.NaN(), math.Inf(1)} {
		if validParagraphStyle(ParagraphStyle{RightIndent: value}, 100) {
			t.Fatalf("accepted right indent %v", value)
		}
	}
	if validParagraphStyle(ParagraphStyle{LeftIndent: 20, RightIndent: 40, FirstLineIndent: 40}, 100) {
		t.Fatal("accepted zero first-line width")
	}
	if got := paragraphLineWidth(100, ParagraphStyle{LeftIndent: 20, RightIndent: 30, FirstLineIndent: -10}, true); got != 60 {
		t.Fatalf("first width=%v", got)
	}
}

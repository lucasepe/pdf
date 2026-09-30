package pdf

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextPathStateClipAndTransforms(t *testing.T) {
	doc := NewPDF("200pt x 100pt")
	doc.SetCompression(false)
	c := NewContext(&doc)
	c.SetFillColor(color.RGBA{R: 240, G: 100, B: 20, A: 255}).
		SetStrokeColor(color.RGBA{R: 10, G: 20, B: 30, A: 255}).
		SetLineWidth(2).SetLineCap(CapRound).SetLineJoin(JoinBevel).
		SetDash(1, 4, 2).
		MoveTo(10, 20).LineTo(30, 40).QuadraticTo(40, 50, 60, 20).
		ClosePath().FillPreserve().Stroke()
	if err := c.Error(); err != nil {
		t.Fatal(err)
	}
	content := doc.pages[0].content.String()
	for _, want := range []string{
		"10.000 80.000 m", "30.000 60.000 l", "f\n", "2.000 w", "1 J", "2 j", "[4.000 2.000] 1.000 d", "S\n",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("content missing %q:\n%s", want, content)
		}
	}

	c.Push().Translate(5, 10).DrawRectangle(0, 0, 20, 20).Clip().
		DrawCircle(10, 10, 8).Fill().Pop()
	if err := c.Error(); err != nil {
		t.Fatal(err)
	}
	content = doc.pages[0].content.String()
	if !strings.Contains(content, "5.000 90.000 m") || !strings.Contains(content, "W n\n") {
		t.Fatalf("transformed clip missing:\n%s", content)
	}
	if strings.Count(content, "q\n") != strings.Count(content, "Q\n") {
		t.Fatalf("unbalanced q/Q:\n%s", content)
	}
}

func TestContextTextRolesAnchoringAndTransform(t *testing.T) {
	doc := NewPDF("A4")
	doc.SetCompression(false)
	c := NewContext(&doc)
	c.UseFontRole(FontRoleMono, 10)
	w, h := c.MeasureString("code")
	if w != 24 || h != 10 {
		t.Fatalf("MeasureString(code) = %v,%v", w, h)
	}
	c.SetFillColor(color.RGBA{R: 30, G: 40, B: 50, A: 255}).
		Push().RotateAbout(math.Pi/12, 100, 100).
		DrawStringAnchored("code", 100, 100, .5, .5).Pop()
	if err := c.Error(); err != nil {
		t.Fatal(err)
	}
	content := doc.pages[0].content.String()
	if !strings.Contains(content, "/FNT1 10 Tf") || !strings.Contains(content, " cm\n") || !strings.Contains(content, "(code) Tj") {
		t.Fatalf("transformed text missing:\n%s", content)
	}
}

func TestContextImageAlphaAnchoringAndDeduplication(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 100})
	doc := NewPDF("A4")
	doc.SetCompression(false)
	c := NewContext(&doc)
	c.DrawImageScaledAnchored(img, 50, 60, 20, 20, .5, .5).
		DrawImageScaledAnchored(img, 100, 60, 20, 20, .5, .5)
	if err := c.Error(); err != nil {
		t.Fatal(err)
	}
	if len(doc.images) != 1 || len(doc.images[0].alpha) == 0 {
		t.Fatalf("image resources = %+v", doc.images)
	}
	out := doc.Bytes()
	if bytes.Count(out, []byte("/SMask ")) != 1 {
		t.Fatalf("soft-mask count = %d", bytes.Count(out, []byte("/SMask ")))
	}
}

func TestContextErrorsAreExplicitAndStateCanBeBalanced(t *testing.T) {
	doc := NewPDF("A4")
	c := NewContext(&doc)
	c.Push().SetLineWidth(-1).Pop()
	if c.Error() == nil {
		t.Fatal("invalid line width was not reported")
	}
	if len(c.stack) != 0 {
		t.Fatal("Pop did not balance state after an error")
	}
	c.ClearError()
	c.Pop()
	if c.Error() == nil {
		t.Fatal("Pop underflow was not reported")
	}

	doc2 := NewPDF("A4")
	c2 := NewContext(&doc2).Push()
	if err := c2.SaveFile(filepath.Join(t.TempDir(), "unbalanced.pdf")); err == nil {
		t.Fatal("unbalanced graphics state was saved")
	}
}

func TestContextMixedDocumentPassesQPDF(t *testing.T) {
	qpdf, err := exec.LookPath("qpdf")
	if err != nil {
		t.Skip("qpdf not installed")
	}
	doc := NewPDF("A4")
	c := NewContext(&doc)
	c.SetFillColor(color.RGBA{R: 230, G: 240, B: 250, A: 255}).DrawRoundedRectangle(20, 20, 200, 80, 12).Fill().
		SetStrokeColor(color.RGBA{R: 20, G: 60, B: 100, A: 255}).SetLineWidth(3).
		DrawArc(120, 160, 60, 0, math.Pi*1.5).Stroke().
		UseFontRole(FontRoleMono, 12).DrawString("context_ok()", 40, 240)
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	img.SetNRGBA(2, 2, color.NRGBA{R: 255, A: 180})
	c.DrawImageScaledAnchored(img, 300, 100, 48, 48, .5, .5)
	if err := c.Error(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "context.pdf")
	if err := c.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(qpdf, "--check", path).CombinedOutput(); err != nil {
		t.Fatalf("qpdf --check: %v\n%s", err, output)
	}
}

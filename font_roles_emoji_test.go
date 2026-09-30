package pdf

import (
	"bytes"
	"compress/zlib"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestFontRolesUseEmbeddedBodyAndCourierCode(t *testing.T) {
	doc := NewPDF("A4")
	if got, err := doc.FontForRole(FontRoleMono); err != nil || got != "Courier" {
		t.Fatalf("mono role = %q, %v", got, err)
	}
	if got, err := doc.FontForRole(FontRoleMonoBold); err != nil || got != "Courier-Bold" {
		t.Fatalf("mono-bold role = %q, %v", got, err)
	}
	if err := doc.BindFontRole(FontRoleBody, "Gentium"); err == nil {
		t.Fatal("unregistered body font was accepted")
	}
	if err := doc.BindFontRole(FontRoleBody, "Helvetica"); err == nil {
		t.Fatal("base-14 body font was accepted instead of an embedded face")
	}
	if err := doc.RegisterFont("Gentium", loadVera(t)); err != nil {
		t.Fatal(err)
	}
	if err := doc.BindFontRole(FontRoleBody, "Gentium"); err != nil {
		t.Fatal(err)
	}
	if err := doc.UseFontRole(FontRoleBody, 11); err != nil {
		t.Fatal(err)
	}
	doc.DrawText("corpo")

	if err := doc.UseFontRole(FontRoleMono, 10); err != nil {
		t.Fatal(err)
	}
	inlineWidth := doc.TextWidth("inline_code()")
	if err := doc.UseFontRole(FontRoleMono, 10); err != nil {
		t.Fatal(err)
	}
	blockWidth := doc.TextWidth("inline_code()")
	if inlineWidth != blockWidth {
		t.Fatalf("inline width %v != block width %v", inlineWidth, blockWidth)
	}
	doc.DrawText("inline_code()")
	if err := doc.UseFontRole(FontRoleMonoBold, 10); err != nil {
		t.Fatal(err)
	}
	doc.DrawText("bold_code()")
	out := doc.Bytes()
	for _, name := range []string{"/BaseFont/Courier", "/BaseFont/Courier-Bold"} {
		if !strings.Contains(string(out), name) {
			t.Fatalf("missing %s", name)
		}
	}
}

func TestFontRoleErrorsDoNotChangeSelection(t *testing.T) {
	doc := NewPDF("A4")
	beforeName, beforeSize := doc.FontName(), doc.FontSize()
	for _, err := range []error{
		doc.UseFontRole(FontRole("missing"), 10),
		doc.UseFontRole(FontRoleBody, 10),
		doc.UseFontRole(FontRoleMono, 0),
	} {
		var target FontRoleError
		if !errors.As(err, &target) {
			t.Fatalf("error %v is not FontRoleError", err)
		}
	}
	if doc.FontName() != beforeName || doc.FontSize() != beforeSize {
		t.Fatalf("failed role selection changed font to %s %v", doc.FontName(), doc.FontSize())
	}
}

type pngEmojiProvider struct {
	assets  map[string][]byte
	metrics EmojiAsset
	err     error
}

func (p pngEmojiProvider) LookupEmoji(sequence string) (EmojiAsset, bool, error) {
	if p.err != nil {
		return EmojiAsset{}, false, p.err
	}
	data, ok := p.assets[sequence]
	if !ok {
		return EmojiAsset{}, false, nil
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return EmojiAsset{}, false, err
	}
	asset := p.metrics
	asset.Image = img
	return asset, true, nil
}

func TestTransparentEmojiGeometrySoftMaskAndDeduplication(t *testing.T) {
	pngData := transparentEmojiPNG(t)
	provider := pngEmojiProvider{assets: map[string][]byte{"🎯": pngData}, metrics: EmojiAsset{HeightEm: 1.15, BaselineEm: -0.18, AdvanceEm: 1.30}}
	doc := NewPDF("A4")
	if err := doc.SetEmojiProvider(provider); err != nil {
		t.Fatal(err)
	}
	advance, err := doc.MeasureEmoji("🎯", 10)
	if err != nil || advance != 13 {
		t.Fatalf("MeasureEmoji() = %v, %v", advance, err)
	}
	doc.SetXY(10, 20)
	if err := doc.DrawEmoji("🎯", 10); err != nil {
		t.Fatal(err)
	}
	if got := doc.X(); got != 23 {
		t.Fatalf("X() after emoji = %v, want 23", got)
	}
	content := doc.pages[0].content.String()
	if !strings.Contains(content, "23.000 0 0 11.500 10.000 820.090") {
		t.Fatalf("unexpected inline emoji matrix: %q", content)
	}
	doc.AddPage().SetXY(10, 20)
	if err := doc.DrawEmoji("🎯", 10); err != nil {
		t.Fatal(err)
	}
	if len(doc.images) != 1 || len(doc.images[0].alpha) != 2 {
		t.Fatalf("emoji resources = %d, alpha bytes = %d", len(doc.images), len(doc.images[0].alpha))
	}
	out := doc.Bytes()
	if bytes.Count(out, []byte("/SMask ")) != 1 || bytes.Count(out, []byte("/Subtype/Image")) != 2 {
		t.Fatalf("unexpected image/mask objects: SMask=%d images=%d", bytes.Count(out, []byte("/SMask ")), bytes.Count(out, []byte("/Subtype/Image")))
	}
	streams := flateStreams(t, out)
	if !containsBytes(streams, []byte{255, 0, 0, 0, 0, 255}) {
		t.Fatalf("RGB samples not found in %x", streams)
	}
	if !containsBytes(streams, []byte{128, 255}) {
		t.Fatalf("alpha samples not found in %x", streams)
	}
	validateXRefOffsets(t, out)
}

func TestOpaqueEmojiDoesNotAllocateSoftMask(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	doc := NewPDF("A4")
	if err := doc.SetEmojiProvider(staticEmojiProvider{asset: EmojiAsset{Image: img, HeightEm: 1, AdvanceEm: 1}}); err != nil {
		t.Fatal(err)
	}
	doc.SetXY(1, 1)
	if err := doc.DrawEmoji("x", 10); err != nil {
		t.Fatal(err)
	}
	out := doc.Bytes()
	if bytes.Contains(out, []byte("/SMask")) || bytes.Count(out, []byte("/Subtype/Image")) != 1 {
		t.Fatalf("opaque image unexpectedly has a soft mask")
	}
}

type staticEmojiProvider struct{ asset EmojiAsset }

func (p staticEmojiProvider) LookupEmoji(string) (EmojiAsset, bool, error) { return p.asset, true, nil }

func TestEmojiErrorsDoNotDrawPartialContent(t *testing.T) {
	doc := NewPDF("A4")
	if _, err := doc.MeasureEmoji("🎯", 10); err == nil {
		t.Fatal("missing provider accepted")
	}
	provider := pngEmojiProvider{assets: map[string][]byte{}, metrics: EmojiAsset{HeightEm: 1, AdvanceEm: 1}}
	if err := doc.SetEmojiProvider(provider); err != nil {
		t.Fatal(err)
	}
	if err := doc.DrawEmoji("missing", 10); err == nil {
		t.Fatal("missing sequence accepted")
	}
	bad := staticEmojiProvider{asset: EmojiAsset{Image: image.NewRGBA(image.Rect(0, 0, 1, 1)), HeightEm: -1, AdvanceEm: 1}}
	if err := doc.SetEmojiProvider(bad); err != nil {
		t.Fatal(err)
	}
	if err := doc.DrawEmoji("bad", 10); err == nil {
		t.Fatal("invalid metrics accepted")
	}
	failing := pngEmojiProvider{err: errors.New("lookup failed")}
	if err := doc.SetEmojiProvider(failing); err != nil {
		t.Fatal(err)
	}
	if err := doc.DrawEmoji("bad", 10); err == nil || !strings.Contains(err.Error(), "lookup failed") {
		t.Fatalf("provider error = %v", err)
	}
	if err := doc.DrawEmoji("bad", -1); err == nil {
		t.Fatal("invalid emoji size accepted")
	}
	if len(doc.pages) != 0 || len(doc.images) != 0 {
		t.Fatalf("failed emoji draw changed document resources")
	}
}

func TestTransparentEmojiPDFPassesQPDF(t *testing.T) {
	qpdf, err := exec.LookPath("qpdf")
	if err != nil {
		t.Skip("qpdf not installed")
	}
	provider := pngEmojiProvider{assets: map[string][]byte{"🎯": transparentEmojiPNG(t)}, metrics: EmojiAsset{HeightEm: 1.15, BaselineEm: -0.18, AdvanceEm: 1.3}}
	doc := NewPDF("A4")
	if err := doc.SetEmojiProvider(provider); err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	validate := func(name string) {
		path := filepath.Join(tempDir, name+".pdf")
		if err := doc.SaveFile(path); err != nil {
			t.Fatal(err)
		}
		if output, err := exec.Command(qpdf, "--check", path).CombinedOutput(); err != nil {
			t.Fatalf("qpdf --check: %v\n%s", err, output)
		}
	}
	doc.SetXY(20, 20)
	if err := doc.DrawEmoji("🎯", 12); err != nil {
		t.Fatal(err)
	}
	validate("one-page")
	doc.AddPage().SetXY(20, 20)
	if err := doc.DrawEmoji("🎯", 12); err != nil {
		t.Fatal(err)
	}
	validate("two-pages")
}

func TestManualEmojiPNG(t *testing.T) {
	assetPath := os.Getenv("PDF_TEST_EMOJI")
	if assetPath == "" {
		t.Skip("set PDF_TEST_EMOJI for a manual PNG check")
	}
	data, err := os.ReadFile(assetPath)
	if err != nil {
		t.Fatal(err)
	}
	provider := pngEmojiProvider{assets: map[string][]byte{"🎯": data}, metrics: EmojiAsset{HeightEm: 1.15, BaselineEm: -0.18, AdvanceEm: 1.15}}
	doc := NewPDF("A4")
	if err := doc.SetEmojiProvider(provider); err != nil {
		t.Fatal(err)
	}
	doc.SetXY(20, 20)
	if err := doc.DrawEmoji("🎯", 12); err != nil {
		t.Fatal(err)
	}
	if len(doc.images) != 1 || len(doc.images[0].alpha) == 0 {
		t.Fatal("manual PNG did not produce a transparent image resource")
	}
	path := filepath.Join(t.TempDir(), "manual-emoji.pdf")
	if err := doc.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	if qpdf, err := exec.LookPath("qpdf"); err == nil {
		if output, err := exec.Command(qpdf, "--check", path).CombinedOutput(); err != nil {
			t.Fatalf("qpdf --check: %v\n%s", err, output)
		}
	}
}

func transparentEmojiPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 128})
	img.SetNRGBA(1, 0, color.NRGBA{B: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func flateStreams(t *testing.T, pdf []byte) [][]byte {
	t.Helper()
	re := regexp.MustCompile(`/Filter/FlateDecode/Length ([0-9]+)>> stream\n`)
	matches := re.FindAllSubmatchIndex(pdf, -1)
	out := make([][]byte, 0, len(matches))
	for _, match := range matches {
		length, err := strconv.Atoi(string(pdf[match[2]:match[3]]))
		if err != nil {
			t.Fatal(err)
		}
		start := match[1]
		if start+length > len(pdf) {
			t.Fatalf("stream length exceeds document")
		}
		zr, err := zlib.NewReader(bytes.NewReader(pdf[start : start+length]))
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := io.ReadAll(zr)
		if closeErr := zr.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, decoded)
	}
	return out
}

func containsBytes(haystack [][]byte, needle []byte) bool {
	for _, item := range haystack {
		if bytes.Equal(item, needle) {
			return true
		}
	}
	return false
}

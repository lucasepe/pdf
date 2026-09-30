package pdf

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestInvalidPaperFallsBackToA4(t *testing.T) {
	doc := NewPDF("not-a-paper-size")
	if got, want := doc.PageWidth(), 595.2755905511812; math.Abs(got-want) > 1e-9 {
		t.Fatalf("PageWidth() = %v, want A4 width %v", got, want)
	}
	if got, want := doc.PageHeight(), 841.8897637795276; math.Abs(got-want) > 1e-9 {
		t.Fatalf("PageHeight() = %v, want A4 height %v", got, want)
	}
	if len(doc.Errors()) == 0 {
		t.Fatal("invalid paper size did not report an error")
	}
}

func TestSetCurrentPageChangesContentDestination(t *testing.T) {
	doc := NewPDF("A4")
	doc.SetCompression(false).DrawText("page-one")
	doc.AddPage().DrawText("page-two")
	doc.SetCurrentPage(1).DrawText("back-on-one")

	first := doc.pages[0].content.String()
	second := doc.pages[1].content.String()
	if !strings.Contains(first, "page-one") || !strings.Contains(first, "back-on-one") {
		t.Fatalf("first page content is wrong: %q", first)
	}
	if strings.Contains(second, "back-on-one") || !strings.Contains(second, "page-two") {
		t.Fatalf("second page content is wrong: %q", second)
	}
}

func TestNextLineUsesSelectedUnits(t *testing.T) {
	for _, tc := range []struct {
		units string
		want  float64
	}{
		{"pt", 12},
		{"in", 12.0 / 72},
		{"cm", 12.0 / 28.3464566929134},
		{"mm", 12.0 / 2.83464566929134},
	} {
		t.Run(tc.units, func(t *testing.T) {
			doc := NewPDF("A4")
			doc.SetUnits(tc.units).SetFontSize(12).SetY(1).NextLine()
			if got := doc.Y(); math.Abs(got-(1+tc.want)) > 1e-9 {
				t.Fatalf("Y() = %v, want %v", got, 1+tc.want)
			}
		})
	}
}

func TestFractionalDimensionsAndFontSizeArePreserved(t *testing.T) {
	doc := NewPDF("10.5pt x 20.25pt")
	doc.SetCompression(false).SetFont("Helvetica", 10.5).DrawText("x")
	out := string(doc.Bytes())
	if !strings.Contains(out, "/MediaBox[0 0 10.5 20.25]") {
		t.Fatalf("fractional MediaBox missing from PDF:\n%s", out)
	}
	if !strings.Contains(out, "/FNT1 10.5 Tf") {
		t.Fatalf("fractional font size missing from PDF:\n%s", out)
	}
}

func TestLatinTextUsesWinAnsiBytes(t *testing.T) {
	const text = "Perché l’unità è già pronta — € 12,50"
	want := []byte("Perch\xe9 l\x92unit\xe0 \xe8 gi\xe0 pronta \x97 \x80 12,50")
	doc := NewPDF("A4")
	doc.SetCompression(false).DrawText(text)
	out := doc.Bytes()
	if !bytes.Contains(out, append(append([]byte("("), want...), []byte(") Tj")...)) {
		t.Fatalf("WinAnsi text bytes not found in output: %x", out)
	}
	if bytes.Contains(out, []byte(text)) {
		t.Fatal("UTF-8 source bytes leaked into a WinAnsi content stream")
	}
	if !bytes.Contains(out, []byte("/Encoding/WinAnsiEncoding")) {
		t.Fatal("font object does not declare WinAnsiEncoding")
	}
}

func TestUnsupportedTextReportsTypedEncodingError(t *testing.T) {
	doc := NewPDF("A4")
	doc.DrawText("Latin then Ж")
	var target TextEncodingError
	for _, err := range doc.Errors() {
		if errors.As(err, &target) {
			if target.Rune != 'Ж' || target.Encoding != "WinAnsiEncoding" || target.Font != "Helvetica" {
				t.Fatalf("unexpected encoding error: %+v", target)
			}
			return
		}
	}
	t.Fatalf("typed TextEncodingError not found in %v", doc.Errors())
}

func TestWinAnsiLatinRepertoireMappings(t *testing.T) {
	for r := rune(0xa0); r <= 0xff; r++ {
		encoded, err := encodeWinAnsi(string(r))
		if err != nil || len(encoded) != 1 || encoded[0] != byte(r) {
			t.Fatalf("encodeWinAnsi(%U) = %x, %v", r, encoded, err)
		}
	}
	for r, want := range winAnsiSpecial {
		encoded, err := encodeWinAnsi(string(r))
		if err != nil || len(encoded) != 1 || encoded[0] != want {
			t.Fatalf("encodeWinAnsi(%U) = %x, %v; want %02x", r, encoded, err, want)
		}
	}
	if _, err := encodeWinAnsi("e\u0301"); err == nil {
		t.Fatal("decomposed combining acute accent was silently accepted")
	}
}

func TestMetadataUsesPDFDocEncoding(t *testing.T) {
	doc := NewPDF("A4")
	doc.SetDocTitle("Costo € — l’unità")
	out := doc.Bytes()
	want := []byte("/Title (Costo \xa0 \x84 l\x90unit\xe0)")
	if !bytes.Contains(out, want) {
		t.Fatalf("PDFDocEncoding metadata not found: %x", out)
	}
}

func TestSymbolFontsDoNotDeclareWinAnsi(t *testing.T) {
	for _, font := range []string{"Symbol", "ZapfDingbats"} {
		t.Run(font, func(t *testing.T) {
			doc := NewPDF("A4")
			doc.SetFont(font, 12).DrawText("a")
			out := doc.Bytes()
			if bytes.Contains(out, []byte("/Encoding/WinAnsiEncoding")) {
				t.Fatalf("%s incorrectly declares WinAnsiEncoding", font)
			}
		})
	}
}

func TestWriteToReportsWriterFailures(t *testing.T) {
	doc := NewPDF("A4")
	doc.DrawText("hello")
	if _, err := doc.WriteTo(shortWriter{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("WriteTo(short writer) error = %v, want io.ErrShortWrite", err)
	}
	want := errors.New("write failed")
	if _, err := doc.WriteTo(errorWriter{want}); !errors.Is(err, want) {
		t.Fatalf("WriteTo(error writer) error = %v, want %v", err, want)
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) / 2, nil }

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

func TestMakeImageSupportsNonZeroBounds(t *testing.T) {
	src := image.NewRGBA(image.Rect(5, 7, 7, 9))
	src.Set(5, 7, color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff})
	w, h, gray, data := makeImage(src, color.RGBA{})
	if w != 2 || h != 2 || gray {
		t.Fatalf("makeImage dimensions/model = %dx%d gray=%v", w, h, gray)
	}
	if got, want := data[:3], []byte{0x12, 0x34, 0x56}; !bytes.Equal(got, want) {
		t.Fatalf("first pixel = %x, want %x", got, want)
	}
}

func TestFillDoesNotStroke(t *testing.T) {
	doc := NewPDF("A4")
	doc.SetCompression(false).FillBox(1, 2, 3, 4)
	content := doc.pages[0].content.String()
	if !strings.Contains(content, " re f\n") || strings.Contains(content, " re b\n") {
		t.Fatalf("unexpected fill operator in %q", content)
	}
}

func TestGeneratedXRefOffsetsPointAtObjects(t *testing.T) {
	doc := NewPDF("A4")
	doc.DrawText("first")
	doc.AddPage().SetFont("Times-Roman", 11).DrawText("second")
	out := doc.Bytes()
	re := regexp.MustCompile(`(?m)^(\d{10}) 00000 n $`)
	matches := re.FindAllSubmatch(out, -1)
	if len(matches) == 0 {
		t.Fatal("xref entries not found")
	}
	for i, match := range matches {
		offset, err := strconv.Atoi(string(match[1]))
		if err != nil || offset >= len(out) {
			t.Fatalf("invalid xref offset %q", match[1])
		}
		want := fmt.Sprintf("%d 0 obj", i+1)
		if !bytes.HasPrefix(out[offset:], []byte(want)) {
			t.Fatalf("xref object %d offset %d does not point to %q", i+1, offset, want)
		}
	}
}

func TestCompressedPageStreamContainsExpectedCommands(t *testing.T) {
	doc := NewPDF("A4")
	doc.SetCompression(true).SetXY(12, 34).DrawText("stream-marker")
	out := doc.Bytes()
	re := regexp.MustCompile(`/Filter/FlateDecode/Length ([0-9]+)>> stream\n`)
	match := re.FindSubmatchIndex(out)
	if match == nil {
		t.Fatal("compressed stream not found")
	}
	length, err := strconv.Atoi(string(out[match[2]:match[3]]))
	if err != nil {
		t.Fatal(err)
	}
	start := match[1]
	if start+length > len(out) {
		t.Fatalf("stream length %d exceeds document", length)
	}
	zr, err := zlib.NewReader(bytes.NewReader(out[start : start+length]))
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
	if !bytes.Contains(decoded, []byte("(stream-marker) Tj")) {
		t.Fatalf("unexpected decoded stream: %q", decoded)
	}
}

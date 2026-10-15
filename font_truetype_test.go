package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const embeddedLatinSample = "Perché l’unità è già pronta — € 12,50"

var regexpXRef = regexp.MustCompile(`(?m)^(\d{10}) 00000 n $`)

func loadVera(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/fonts/Vera.ttf")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestRegisterFontFromBytesAndReader(t *testing.T) {
	data := loadVera(t)
	for _, source := range []any{data, bytes.NewReader(data)} {
		doc := NewPDF("A4")
		if err := doc.RegisterFont("Body", source); err != nil {
			t.Fatal(err)
		}
		doc.SetFont("Body", 12)
		width := doc.TextWidth(embeddedLatinSample)
		if width <= 0 {
			t.Fatalf("TextWidth() = %v", width)
		}
		doc.SetXY(40, 60).DrawText(embeddedLatinSample)
		if got := doc.X(); math.Abs(got-(40+width)) > 1e-9 {
			t.Fatalf("drawing advance = %v, measured width = %v", got-40, width)
		}
		if errs := doc.Errors(); len(errs) != 0 {
			t.Fatalf("drawing errors: %v", errs)
		}
		out := doc.Bytes()
		for _, marker := range [][]byte{[]byte("/Subtype/Type0"), []byte("/Subtype/CIDFontType2"), []byte("/FontFile2"), []byte("/CIDToGIDMap"), []byte("/ToUnicode")} {
			if !bytes.Contains(out, marker) {
				t.Fatalf("missing %q", marker)
			}
		}
	}
}

func TestEmbeddedFontReusedAcrossPages(t *testing.T) {
	doc := NewPDF("A4")
	if err := doc.RegisterFont("Body", loadVera(t)); err != nil {
		t.Fatal(err)
	}
	doc.SetFont("Body", 12).DrawText("first")
	doc.AddPage().SetFont("Body", 12).DrawText("second")
	out := doc.Bytes()
	if got := bytes.Count(out, []byte("/Subtype/Type0")); got != 1 {
		t.Fatalf("Type0 font count = %d, want 1", got)
	}
	validateXRefOffsets(t, out)
}

func TestEmbeddedFontSubsetsUsedGlyphs(t *testing.T) {
	data := loadVera(t)
	doc := NewPDF("A4")
	if err := doc.RegisterFont("Body", data); err != nil {
		t.Fatal(err)
	}
	doc.SetFont("Body", 12).DrawText("Subset é")
	font := doc.registeredFonts["BODY"]
	subset, name, err := font.subsetProgram()
	if err != nil {
		t.Fatal(err)
	}
	if len(subset) >= len(data)/2 {
		t.Fatalf("subset size = %d; original = %d", len(subset), len(data))
	}
	if len(name) < 8 || name[6] != '+' {
		t.Fatalf("invalid subset font name %q", name)
	}
	if got := sfntChecksum(subset); got != 0xB1B0AFBA {
		t.Fatalf("subset sfnt checksum = %08x", got)
	}
	parsed, err := parseTrueType(subset)
	if err != nil {
		t.Fatalf("subset is not a valid TrueType font: %v", err)
	}
	if parsed.numGlyphs != font.numGlyphs {
		t.Fatalf("glyph count changed from %d to %d", font.numGlyphs, parsed.numGlyphs)
	}
	originalLoca, _ := font.table("loca", 0)
	originalGlyf, _ := font.table("glyf", 0)
	subsetLoca, _ := parsed.table("loca", 0)
	subsetGlyf, _ := parsed.table("glyf", 0)
	originalLong := i16(data[int(font.tables["head"].offset):], 50) == 1
	for _, gid := range font.cidToGID {
		start, end := glyphOffsets(originalLoca, gid, originalLong)
		subStart, subEnd := glyphOffsets(subsetLoca, gid, true)
		if subEnd-subStart < end-start || !bytes.Equal(originalGlyf[start:end], subsetGlyf[subStart:subStart+end-start]) {
			t.Fatalf("used glyph %d was not preserved", gid)
		}
	}
}

func TestEmbeddedFontSubsetIncludesCompositeComponents(t *testing.T) {
	font, err := parseTrueType(loadVera(t))
	if err != nil {
		t.Fatal(err)
	}
	loca, _ := font.table("loca", 0)
	glyf, _ := font.table("glyf", 0)
	head, _ := font.table("head", 54)
	long := i16(head, 50) == 1
	var target rune
	var component uint16
	for r, gid := range font.cmap {
		if _, ok := winAnsiByte(r); !ok {
			continue
		}
		start, end := glyphOffsets(loca, gid, long)
		if end-start >= 14 && i16(glyf[start:], 0) < 0 {
			target, component = r, u16(glyf[start:], 12)
			break
		}
	}
	if target == 0 {
		t.Skip("fixture has no WinAnsi composite glyph")
	}
	if _, err := font.encodeRun(string(target)); err != nil {
		t.Fatal(err)
	}
	subset, _, err := font.subsetProgram()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parseTrueType(subset)
	if err != nil {
		t.Fatal(err)
	}
	subLoca, _ := parsed.table("loca", 0)
	subGlyf, _ := parsed.table("glyf", 0)
	start, end := glyphOffsets(subLoca, component, true)
	if start == end {
		t.Fatalf("component glyph %d of %q was omitted", component, target)
	}
	originalStart, originalEnd := glyphOffsets(loca, component, long)
	if !bytes.Equal(subGlyf[start:start+originalEnd-originalStart], glyf[originalStart:originalEnd]) {
		t.Fatalf("component glyph %d of %q was changed", component, target)
	}
}

func TestEmbeddedFontRespectsNoSubsettingPermission(t *testing.T) {
	data := loadVera(t)
	tables, err := fontDirectory(data)
	if err != nil {
		t.Fatal(err)
	}
	os2 := tables["OS/2"]
	data[int(os2.offset)+8], data[int(os2.offset)+9] = 0x01, 0x00
	doc := NewPDF("A4")
	if err := doc.RegisterFont("NoSubset", data); err != nil {
		t.Fatal(err)
	}
	doc.SetFont("NoSubset", 12).DrawText("Hello")
	output := doc.Bytes()
	if !bytes.Contains(output, []byte("/BaseFont/BitstreamVeraSans-Roman")) {
		t.Fatal("no-subsetting font was given a subset name")
	}
	if !bytes.Contains(output, []byte(fmt.Sprintf("/Length1 %d", len(data)))) {
		t.Fatal("no-subsetting font program was shortened")
	}
}

func TestEmbeddedFontRejectsUnsupportedText(t *testing.T) {
	doc := NewPDF("A4")
	if err := doc.RegisterFont("Body", loadVera(t)); err != nil {
		t.Fatal(err)
	}
	doc.SetFont("Body", 12).DrawText("Latin Ж")
	var target TextEncodingError
	for _, err := range doc.Errors() {
		if errors.As(err, &target) {
			if target.Rune != 'Ж' || target.Font != "Body" {
				t.Fatalf("unexpected encoding error: %+v", target)
			}
			return
		}
	}
	t.Fatalf("TextEncodingError not found in %v", doc.Errors())
}

func TestRegisterFontRejectsUnsupportedAndRestrictedFonts(t *testing.T) {
	doc := NewPDF("A4")
	if err := doc.RegisterFont("CFF", append([]byte("OTTO"), make([]byte, 8)...)); err == nil || !strings.Contains(err.Error(), "CFF") {
		t.Fatalf("CFF error = %v", err)
	}
	if err := doc.RegisterFont("Broken", []byte{0, 1, 0, 0}); err == nil {
		t.Fatal("truncated font accepted")
	}
	badOffset := loadVera(t)
	for i := 20; i < 24; i++ {
		badOffset[i] = 0xff
	}
	if err := doc.RegisterFont("BadOffset", badOffset); err == nil || !strings.Contains(err.Error(), "exceeds file") {
		t.Fatalf("bad table offset error = %v", err)
	}

	restricted := loadVera(t)
	tables, err := fontDirectory(restricted)
	if err != nil {
		t.Fatal(err)
	}
	os2 := tables["OS/2"]
	restricted[int(os2.offset)+8] = 0
	restricted[int(os2.offset)+9] = 2
	if err := doc.RegisterFont("Restricted", restricted); err == nil || !strings.Contains(err.Error(), "embedding prohibited") {
		t.Fatalf("restricted embedding error = %v", err)
	}
}

func TestEmbeddedFontReportsMissingGlyph(t *testing.T) {
	doc := NewPDF("A4")
	if err := doc.RegisterFont("Body", loadVera(t)); err != nil {
		t.Fatal(err)
	}
	delete(doc.registeredFonts["BODY"].cmap, 'é')
	doc.SetFont("Body", 12).DrawText("é")
	var target TextEncodingError
	if len(doc.Errors()) == 0 || !errors.As(doc.Errors()[0], &target) || target.Encoding != "font cmap" {
		t.Fatalf("missing-glyph error = %v", doc.Errors())
	}
}

func TestEmbeddedFontExternalValidationAndExtraction(t *testing.T) {
	qpdf, err := exec.LookPath("qpdf")
	if err != nil {
		t.Skip("qpdf not installed")
	}
	mutool, err := exec.LookPath("mutool")
	if err != nil {
		t.Skip("mutool not installed")
	}
	doc := NewPDF("A4")
	if err := doc.RegisterFont("Body", loadVera(t)); err != nil {
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
		output, err := exec.Command(mutool, "draw", "-F", "txt", "-o", "-", path).CombinedOutput()
		if err != nil {
			t.Fatalf("mutool extraction: %v\n%s", err, output)
		}
		if !strings.Contains(string(output), embeddedLatinSample) {
			t.Fatalf("extracted text %q does not contain %q", output, embeddedLatinSample)
		}
	}
	doc.SetFont("Body", 14).SetXY(40, 60).DrawText(embeddedLatinSample)
	validate("one-page")
	doc.AddPage().SetFont("Body", 14).SetXY(40, 60).DrawText("second page")
	validate("two-pages")
}

func TestManualExternalTrueType(t *testing.T) {
	fontPath := os.Getenv("PDF_TEST_FONT")
	if fontPath == "" {
		t.Skip("set PDF_TEST_FONT for a manual font check")
	}
	data, err := os.ReadFile(fontPath)
	if err != nil {
		t.Fatal(err)
	}
	doc := NewPDF("A4")
	if err := doc.RegisterFont("Manual", data); err != nil {
		t.Fatal(err)
	}
	doc.SetFont("Manual", 14).SetXY(40, 60).DrawText(embeddedLatinSample)
	path := filepath.Join(t.TempDir(), "manual-font.pdf")
	if err := doc.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	if qpdf, err := exec.LookPath("qpdf"); err == nil {
		if output, err := exec.Command(qpdf, "--check", path).CombinedOutput(); err != nil {
			t.Fatalf("qpdf --check: %v\n%s", err, output)
		}
	}
	if mutool, err := exec.LookPath("mutool"); err == nil {
		output, err := exec.Command(mutool, "draw", "-F", "txt", "-o", "-", path).CombinedOutput()
		if err != nil || !strings.Contains(string(output), embeddedLatinSample) {
			t.Fatalf("mutool extraction: %v\n%s", err, output)
		}
	}
}

func fontDirectory(data []byte) (map[string]fontTable, error) {
	font, err := parseTrueType(data)
	if err != nil {
		return nil, err
	}
	return font.tables, nil
}

func validateXRefOffsets(t *testing.T, out []byte) {
	t.Helper()
	matches := regexpXRef.FindAllSubmatch(out, -1)
	for i, match := range matches {
		offset, err := strconv.Atoi(string(match[1]))
		if err != nil || offset >= len(out) {
			t.Fatalf("invalid xref offset %q", match[1])
		}
		want := fmt.Sprintf("%d 0 obj", i+1)
		if !bytes.HasPrefix(out[offset:], []byte(want)) {
			t.Fatalf("xref %d does not point to %q", i+1, want)
		}
	}
}

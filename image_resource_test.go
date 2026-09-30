package pdf

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegisterJPEGPreservesDCTBytesAndDeduplicates(t *testing.T) {
	data := testJPEG(t, 80, 40)
	doc := NewPDF("200pt x 120pt")
	first, err := doc.RegisterImage(data)
	if err != nil {
		t.Fatal(err)
	}
	second, err := doc.RegisterImage(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if first.Format() != "jpeg" || first.PixelWidth() != 80 || first.PixelHeight() != 40 {
		t.Fatalf("resource = format %q, %dx%d", first.Format(), first.PixelWidth(), first.PixelHeight())
	}
	if first.index != second.index || len(doc.images) != 1 {
		t.Fatalf("JPEG resources = indexes %d,%d images=%d", first.index, second.index, len(doc.images))
	}
	if got := first.EffectiveDPI(72); got != 80 {
		t.Fatalf("EffectiveDPI(72) = %v", got)
	}
	c := NewContext(&doc)
	c.DrawImageResourceScaledAnchored(first, 10, 10, 80, 40, 0, 0)
	out := doc.Bytes()
	if !bytes.Contains(out, []byte("/Filter/DCTDecode")) {
		t.Fatal("JPEG image does not use DCTDecode")
	}
	if !bytes.Contains(out, data) {
		t.Fatal("original JPEG bytes are not embedded verbatim")
	}
}

func TestRegisterGrayJPEGAndNilReader(t *testing.T) {
	gray := image.NewGray(image.Rect(0, 0, 12, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 12; x++ {
			gray.SetGray(x, y, color.Gray{Y: uint8(x*15 + y)})
		}
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, gray, nil); err != nil {
		t.Fatal(err)
	}
	doc := NewPDF("100pt x 100pt")
	resource, err := doc.RegisterImage(encoded.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if resource.Format() != "jpeg" || !doc.images[resource.index].isGray || doc.images[resource.index].filter != "DCTDecode" {
		t.Fatalf("gray JPEG resource = %+v, internal = %+v", resource, doc.images[resource.index])
	}
	var reader *bytes.Reader
	if _, err := doc.RegisterImage(reader); err == nil || !strings.Contains(err.Error(), "nil reader") {
		t.Fatalf("nil reader error = %v", err)
	}
}

func TestRegisterTransparentPNGKeepsSoftMask(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	img.SetNRGBA(1, 0, color.NRGBA{R: 240, G: 30, B: 20, A: 120})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	doc := NewPDF("100pt x 100pt")
	resource, err := doc.RegisterImage(encoded.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if resource.Format() != "raster" || len(doc.images[resource.index].alpha) == 0 {
		t.Fatalf("PNG resource = format %q alpha=%d", resource.Format(), len(doc.images[resource.index].alpha))
	}
	NewContext(&doc).DrawImageResource(resource, 10, 10)
	if out := doc.Bytes(); !bytes.Contains(out, []byte("/SMask ")) {
		t.Fatal("transparent PNG has no SMask")
	}
}

func TestImageFlowScaleGroupWrappingAndPagination(t *testing.T) {
	doc := NewPDF("240pt x 300pt")
	resource, err := doc.RegisterImage(testRaster(100, 50))
	if err != nil {
		t.Fatal(err)
	}
	flow, err := NewFlow(NewContext(&doc), PageSpec{Width: 240, Height: 300, MarginTop: 20, MarginRight: 20, MarginBottom: 20, MarginLeft: 20}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defaultImage, ok := flow.layoutImage(resource, 0, 0, 1, .98)
	if !ok || defaultImage.width != 156 || defaultImage.height != 78 {
		t.Fatalf("default image = %+v, ok=%v", defaultImage, ok)
	}
	largeImage, ok := flow.layoutImage(resource, 0, 0, 2, .98)
	if !ok || largeImage.width != 196 || largeImage.height != 98 {
		t.Fatalf("clamped image = %+v, ok=%v", largeImage, ok)
	}
	flow.DrawImageGroup([]ImageItem{{Image: resource, Scale: .4}, {Image: resource, Scale: .4}}, ImageGroupOptions{})
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	if got, want := flow.CursorY(), 51.2; math.Abs(got-want) > 0.001 {
		t.Fatalf("one-row CursorY() = %.3f, want %.3f", got, want)
	}

	doc2 := NewPDF("240pt x 300pt")
	resource2, err := doc2.RegisterImage(testRaster(100, 50))
	if err != nil {
		t.Fatal(err)
	}
	flow2, err := NewFlow(NewContext(&doc2), PageSpec{Width: 240, Height: 300, MarginTop: 20, MarginRight: 20, MarginBottom: 20, MarginLeft: 20}, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow2.DrawImageGroup([]ImageItem{{Image: resource2, Scale: .8}, {Image: resource2, Scale: .8}}, ImageGroupOptions{})
	if err := flow2.Error(); err != nil {
		t.Fatal(err)
	}
	if got, want := flow2.CursorY(), 150.8; math.Abs(got-want) > 0.001 {
		t.Fatalf("wrapped CursorY() = %.3f, want %.3f", got, want)
	}

	doc3 := NewPDF("200pt x 120pt")
	resource3, err := doc3.RegisterImage(testRaster(100, 50))
	if err != nil {
		t.Fatal(err)
	}
	flow3, err := NewFlow(NewContext(&doc3), PageSpec{Width: 200, Height: 120, MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10}, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow3.Advance(70).DrawImage(resource3, ImageOptions{Width: 80})
	if err := flow3.Error(); err != nil {
		t.Fatal(err)
	}
	if flow3.PageNumber() != 2 || len(doc3.pages[0].imageIDs) != 0 || len(doc3.pages[1].imageIDs) != 1 {
		t.Fatalf("pagination = page %d, image IDs %v / %v", flow3.PageNumber(), doc3.pages[0].imageIDs, doc3.pages[1].imageIDs)
	}
}

func TestImageFlowRejectsOversizeAndForeignResource(t *testing.T) {
	doc := NewPDF("100pt x 100pt")
	resource, err := doc.RegisterImage(testRaster(10, 100))
	if err != nil {
		t.Fatal(err)
	}
	flow, err := NewFlow(NewContext(&doc), PageSpec{Width: 100, Height: 100, MarginTop: 10, MarginRight: 10, MarginBottom: 10, MarginLeft: 10}, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow.DrawImage(resource, ImageOptions{Width: 40})
	var layoutErr ImageLayoutError
	if !errors.As(flow.Error(), &layoutErr) || !strings.Contains(layoutErr.Detail, "height exceeds") {
		t.Fatalf("oversize error = %v", flow.Error())
	}

	other := NewPDF("100pt x 100pt")
	foreign, err := other.RegisterImage(testRaster(10, 10))
	if err != nil {
		t.Fatal(err)
	}
	context := NewContext(&doc)
	context.DrawImageResource(foreign, 0, 0)
	if err := context.Error(); err == nil || !strings.Contains(err.Error(), "another document") {
		t.Fatalf("foreign resource error = %v", err)
	}
}

func TestImageResourcePDFPassesQPDF(t *testing.T) {
	qpdf, err := exec.LookPath("qpdf")
	if err != nil {
		t.Skip("qpdf not installed")
	}
	doc := NewPDF("240pt x 180pt")
	photo, err := doc.RegisterImage(testJPEG(t, 120, 60))
	if err != nil {
		t.Fatal(err)
	}
	transparent, err := doc.RegisterImage(transparentEmojiPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	flow, err := NewFlow(NewContext(&doc), PageSpec{Width: 240, Height: 180, MarginTop: 15, MarginRight: 15, MarginBottom: 15, MarginLeft: 15}, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow.DrawImageGroup([]ImageItem{{Image: photo, Scale: .45}, {Image: transparent, Scale: .45}}, ImageGroupOptions{})
	if err := flow.Error(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "images.pdf")
	if err := flow.context.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(qpdf, "--check", path).CombinedOutput(); err != nil {
		t.Fatalf("qpdf --check: %v\n%s", err, output)
	}
}

func testJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x * 255 / width), G: uint8(y * 255 / height), B: 150, A: 255})
		}
	}
	var output bytes.Buffer
	if err := jpeg.Encode(&output, img, &jpeg.Options{Quality: 82}); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func testRaster(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 180, A: 255})
		}
	}
	return img
}

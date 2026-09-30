package pdf

import (
	"bytes"
	"crypto/sha512"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png" // init image decoders
	"os"
	"reflect"
)

// DrawImage draws a PNG image. x, y, height specify the position and height
// of the image. Width is scaled to match the image's aspect ratio.
// fileNameOrBytes is either a string specifying a file name,
// or a byte slice with PNG image data.
func (p *PDF) DrawImage(x, y, height float64, fileNameOrBytes any,
	backColor ...string) *PDF {

	back := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if len(backColor) > 0 {
		back, _ = p.ToColor(backColor[0])
	}

	p.reservePage()
	img, idx, err := p.loadImage(fileNameOrBytes, back)
	if err, isT := err.(pdfError); isT {
		return p.putError(0xE8F375, err.msg, err.val)
	}
	var found bool
	for _, id := range p.page.imageIDs {
		if id == idx {
			found = true
			break
		}
	}
	if !found {
		p.page.imageIDs = append(p.page.imageIDs, idx)
	}

	h := height * p.ptPerUnit
	w := float64(img.widthPx) / float64(img.heightPx) * h
	x, y = x*p.ptPerUnit, p.paperSize.heightPt-y*p.ptPerUnit-h
	return p.write("q\n", w, " 0 0 ", h, " ", x, " ", y, " cm\n"+
		"/IMG", idx, " Do\n"+"Q\n")

}

// pdfImage represents an image
type pdfImage struct {
	filename          string     // name of file from which image was read
	widthPx, heightPx int        // width and height in pixels
	data              []byte     // image data
	alpha             []byte     // optional PDF soft-mask samples
	hash              [64]byte   // hash of data (used to compare images)
	backColor         color.RGBA // background color (used to compare images)
	isGray            bool       // image is grayscale? (if false, color image)
	isDirect          bool       // image.Image resource rather than legacy input
	filter            string     // pre-encoded PDF stream filter, e.g. DCTDecode
} //                                                                    pdfImage

// loadImage reads an image from a file or byte array, stores its data in
// the PDF's images array, and returns a pdfImage and its reference index
func (p *PDF) loadImage(fileNameOrBytes any, back color.RGBA,
) (img pdfImage, idx int, err error) {
	var buf *bytes.Buffer
	switch val := fileNameOrBytes.(type) {
	case string:
		for i, it := range p.images {
			if it.filename == val && it.backColor == back {
				return it, i, nil
			}
		}
		img.filename = val
		data, err := os.ReadFile(val)
		if err != nil {
			return pdfImage{}, -1, pdfError{id: 0xE9F387,
				msg: "Failed reading file", val: err.Error()}
		}
		buf = bytes.NewBuffer(data)
		img.hash = sha512.Sum512(data)
	case []byte:
		buf = bytes.NewBuffer(val)
		img.hash = sha512.Sum512(val)
	default:
		return pdfImage{}, -1,
			pdfError{id: 0xEE3E42, msg: "Invalid type in fileNameOrBytes",
				val: fmt.Sprintf("%s = %v",
					reflect.TypeOf(fileNameOrBytes), fileNameOrBytes)}
	}
	for i, it := range p.images {
		if bytes.Equal(it.hash[:], img.hash[:]) && it.backColor == back {
			return it, i, nil
		}
	}
	decoded, _, err2 := image.Decode(buf)
	if err2 != nil {
		return pdfImage{}, -1,
			pdfError{id: 0xE64335, msg: "Image not decoded", val: err2.Error()}
	}
	img.backColor = back
	img.widthPx, img.heightPx, img.isGray, img.data = makeImage(decoded, back)
	p.images = append(p.images, img)
	return img, len(p.images) - 1, nil
}

// makeImage encodes the source image in a PDF image data stream
func makeImage(source image.Image, back color.RGBA,
) (widthPx, heightPx int, isGray bool, ar []byte) {

	blend := func(color, alpha uint32, back byte) byte {
		c, a := float64(color), 65535-float64(alpha)
		return byte((c + (float64(back)*255-c)/65536*a) / 65536 * 255)
	}
	bounds := source.Bounds()
	widthPx, heightPx = bounds.Dx(), bounds.Dy()
	model := source.ColorModel()
	isGray = model == color.GrayModel || model == color.Gray16Model
	for y := 0; y < heightPx; y++ {
		for x := 0; x < widthPx; x++ {
			r, g, b, a := source.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			switch {
			case isGray:
				ar = append(ar, byte(r>>8))
			case a == 65535:
				ar = append(ar, byte(r>>8), byte(g>>8), byte(b>>8))
			case a == 0:
				ar = append(ar, back.R, back.G, back.B)
			default:
				ar = append(ar,
					blend(r, a, back.R),
					blend(g, a, back.G),
					blend(b, a, back.B))
			}
		}
	}
	return widthPx, heightPx, isGray, ar
}

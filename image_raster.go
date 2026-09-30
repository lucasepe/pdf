package pdf

import (
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"image"
	"reflect"
)

const maxRasterPixels = 16 << 20

func isNilImage(img image.Image) bool {
	if img == nil {
		return true
	}
	value := reflect.ValueOf(img)
	return (value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface) && value.IsNil()
}

func (p *PDF) loadDirectImage(source image.Image) (int, error) {
	if isNilImage(source) || source.Bounds().Empty() {
		return -1, ImageError{Detail: "invalid image"}
	}
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if uint64(width)*uint64(height) > maxRasterPixels {
		return -1, ImageError{Detail: fmt.Sprintf("image exceeds %d pixels", maxRasterPixels)}
	}
	rgb := make([]byte, 0, width*height*3)
	alpha := make([]byte, 0, width*height)
	opaque := true
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := source.At(x, y).RGBA()
			if a > 0 && a < 0xffff {
				r = minUint32(0xffff, r*0xffff/a)
				g = minUint32(0xffff, g*0xffff/a)
				b = minUint32(0xffff, b*0xffff/a)
			}
			rgb = append(rgb, byte(r>>8), byte(g>>8), byte(b>>8))
			alpha = append(alpha, byte(a>>8))
			opaque = opaque && a == 0xffff
		}
	}
	if opaque {
		alpha = nil
	}
	hasher := sha512.New()
	var dimensions [8]byte
	binary.BigEndian.PutUint32(dimensions[:4], uint32(width))
	binary.BigEndian.PutUint32(dimensions[4:], uint32(height))
	_, _ = hasher.Write(dimensions[:])
	_, _ = hasher.Write(rgb)
	_, _ = hasher.Write(alpha)
	var hash [64]byte
	copy(hash[:], hasher.Sum(nil))
	for i, candidate := range p.images {
		if candidate.isDirect && candidate.filter == "" && candidate.hash == hash {
			return i, nil
		}
	}
	p.images = append(p.images, pdfImage{widthPx: width, heightPx: height,
		data: rgb, alpha: alpha, hash: hash, isDirect: true})
	return len(p.images) - 1, nil
}

func minUint32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func (p *PDF) addPageImage(idx int) {
	for _, id := range p.page.imageIDs {
		if id == idx {
			return
		}
	}
	p.page.imageIDs = append(p.page.imageIDs, idx)
}

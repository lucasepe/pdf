package pdf

import (
	"bytes"
	"crypto/sha512"
	"fmt"
	"image"
	"io"
	"reflect"
)

const maxEncodedImageBytes = 64 << 20

// ImageError reports invalid, unsupported, or oversized image resources.
type ImageError struct{ Detail string }

func (e ImageError) Error() string { return "image: " + e.Detail }

// ImageResource is an immutable image registered with one PDF document.
// Encoded RGB/Gray JPEG resources retain their original DCT stream.
type ImageResource struct {
	doc           *PDF
	index         int
	width, height int
	format        string
}

// PixelWidth returns the source width in pixels.
func (r *ImageResource) PixelWidth() int {
	if r == nil {
		return 0
	}
	return r.width
}

// PixelHeight returns the source height in pixels.
func (r *ImageResource) PixelHeight() int {
	if r == nil {
		return 0
	}
	return r.height
}

// Format reports "jpeg" for passthrough JPEG data and "raster" for decoded
// resources.
func (r *ImageResource) Format() string {
	if r == nil {
		return ""
	}
	return r.format
}

// EffectiveDPI returns the horizontal source resolution when drawn at
// widthPoints. Invalid inputs return zero.
func (r *ImageResource) EffectiveDPI(widthPoints float64) float64 {
	if r == nil || widthPoints <= 0 || !finite(widthPoints) {
		return 0
	}
	return float64(r.width) * 72 / widthPoints
}

// RegisterImage registers []byte, io.Reader, or image.Image input and returns
// a reusable document-owned resource. Encoded input is copied before return.
func (p *PDF) RegisterImage(source any) (*ImageResource, error) {
	if p == nil {
		return nil, ImageError{Detail: "nil document"}
	}
	p.init()
	switch value := source.(type) {
	case []byte:
		return p.registerEncodedImage(value)
	case io.Reader:
		valueOf := reflect.ValueOf(value)
		if (valueOf.Kind() == reflect.Ptr || valueOf.Kind() == reflect.Interface) && valueOf.IsNil() {
			return nil, ImageError{Detail: "nil reader"}
		}
		data, err := io.ReadAll(io.LimitReader(value, maxEncodedImageBytes+1))
		if err != nil {
			return nil, ImageError{Detail: err.Error()}
		}
		if len(data) > maxEncodedImageBytes {
			return nil, ImageError{Detail: fmt.Sprintf("encoded data exceeds %d bytes", maxEncodedImageBytes)}
		}
		return p.registerEncodedImage(data)
	case image.Image:
		index, err := p.loadDirectImage(value)
		if err != nil {
			return nil, err
		}
		img := p.images[index]
		return &ImageResource{doc: p, index: index, width: img.widthPx, height: img.heightPx, format: "raster"}, nil
	default:
		return nil, ImageError{Detail: fmt.Sprintf("unsupported source type %T", source)}
	}
}

func (p *PDF) registerEncodedImage(source []byte) (*ImageResource, error) {
	if len(source) == 0 {
		return nil, ImageError{Detail: "empty encoded data"}
	}
	if len(source) > maxEncodedImageBytes {
		return nil, ImageError{Detail: fmt.Sprintf("encoded data exceeds %d bytes", maxEncodedImageBytes)}
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(source))
	if err != nil {
		return nil, ImageError{Detail: fmt.Sprintf("decode config: %v", err)}
	}
	if config.Width <= 0 || config.Height <= 0 || uint64(config.Width)*uint64(config.Height) > maxRasterPixels {
		return nil, ImageError{Detail: fmt.Sprintf("image exceeds %d pixels or has invalid dimensions", maxRasterPixels)}
	}
	if format == "jpeg" {
		components, ok := jpegComponentCount(source)
		if ok && (components == 1 || components == 3) {
			hash := sha512.Sum512(source)
			for i, candidate := range p.images {
				if candidate.filter == "DCTDecode" && candidate.hash == hash {
					return &ImageResource{doc: p, index: i, width: candidate.widthPx, height: candidate.heightPx, format: "jpeg"}, nil
				}
			}
			data := append([]byte(nil), source...)
			p.images = append(p.images, pdfImage{
				widthPx: config.Width, heightPx: config.Height, data: data,
				hash: hash, isGray: components == 1, isDirect: true, filter: "DCTDecode",
			})
			index := len(p.images) - 1
			return &ImageResource{doc: p, index: index, width: config.Width, height: config.Height, format: "jpeg"}, nil
		}
	}
	decoded, _, err := image.Decode(bytes.NewReader(source))
	if err != nil {
		return nil, ImageError{Detail: fmt.Sprintf("decode: %v", err)}
	}
	index, err := p.loadDirectImage(decoded)
	if err != nil {
		return nil, err
	}
	img := p.images[index]
	return &ImageResource{doc: p, index: index, width: img.widthPx, height: img.heightPx, format: "raster"}, nil
}

func jpegComponentCount(data []byte) (int, bool) {
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return 0, false
	}
	for i := 2; i+1 < len(data); {
		for i < len(data) && data[i] != 0xff {
			i++
		}
		for i < len(data) && data[i] == 0xff {
			i++
		}
		if i >= len(data) {
			break
		}
		marker := data[i]
		i++
		if marker == 0xd9 || marker == 0xda {
			break
		}
		if marker == 0x01 || marker >= 0xd0 && marker <= 0xd7 {
			continue
		}
		if i+2 > len(data) {
			return 0, false
		}
		length := int(data[i])<<8 | int(data[i+1])
		if length < 2 || i+length > len(data) {
			return 0, false
		}
		if isJPEGStartOfFrame(marker) {
			if length < 8 {
				return 0, false
			}
			return int(data[i+7]), true
		}
		i += length
	}
	return 0, false
}

func isJPEGStartOfFrame(marker byte) bool {
	switch marker {
	case 0xc0, 0xc1, 0xc2, 0xc3, 0xc5, 0xc6, 0xc7, 0xc9, 0xca, 0xcb, 0xcd, 0xce, 0xcf:
		return true
	default:
		return false
	}
}

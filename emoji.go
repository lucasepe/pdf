package pdf

import (
	"fmt"
	"image"
	"math"
	"reflect"
)

// EmojiAsset describes raster artwork and its inline geometry in em units.
// BaselineEm is added to the text baseline; a negative value lowers the image.
type EmojiAsset struct {
	Image      image.Image
	HeightEm   float64
	BaselineEm float64
	AdvanceEm  float64
}

// EmojiProvider resolves a complete Unicode emoji sequence.
type EmojiProvider interface {
	LookupEmoji(sequence string) (asset EmojiAsset, found bool, err error)
}

// EmojiError reports provider, lookup, artwork, or metric failures.
type EmojiError struct {
	Sequence string
	Detail   string
}

func (e EmojiError) Error() string {
	if e.Sequence == "" {
		return "emoji: " + e.Detail
	}
	return fmt.Sprintf("emoji %q: %s", e.Sequence, e.Detail)
}

// SetEmojiProvider sets the provider used by MeasureEmoji and DrawEmoji. A nil
// provider clears the current provider.
func (p *PDF) SetEmojiProvider(provider EmojiProvider) error {
	if provider != nil {
		value := reflect.ValueOf(provider)
		if (value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface) && value.IsNil() {
			return EmojiError{Detail: "nil provider"}
		}
	}
	p.emojiProvider = provider
	return nil
}

// MeasureEmoji returns the inline advance for sequence in the selected document
// units. fontSize is expressed in points.
func (p *PDF) MeasureEmoji(sequence string, fontSize float64) (float64, error) {
	asset, err := p.resolveEmoji(sequence, fontSize)
	if err != nil {
		return 0, err
	}
	return p.ToUnits(asset.AdvanceEm * fontSize), nil
}

// DrawEmoji draws sequence at the current X position and text baseline, then
// advances X by the asset's configured advance. fontSize is in points.
func (p *PDF) DrawEmoji(sequence string, fontSize float64) error {
	asset, err := p.resolveEmoji(sequence, fontSize)
	if err != nil {
		return err
	}
	idx, err := p.loadDirectImage(asset.Image)
	if err != nil {
		return err
	}
	p.reservePage()
	p.addPageImage(idx)
	height := asset.HeightEm * fontSize
	width := float64(p.images[idx].widthPx) / float64(p.images[idx].heightPx) * height
	bottom := p.page.y + asset.BaselineEm*fontSize
	p.write("q\n", width, " 0 0 ", height, " ", p.page.x, " ", bottom,
		" cm\n/IMG", idx, " Do\nQ\n")
	p.page.x += asset.AdvanceEm * fontSize
	return nil
}

func (p *PDF) resolveEmoji(sequence string, fontSize float64) (EmojiAsset, error) {
	p.init()
	if p.emojiProvider == nil {
		return EmojiAsset{}, EmojiError{Sequence: sequence, Detail: "no provider configured"}
	}
	if sequence == "" {
		return EmojiAsset{}, EmojiError{Detail: "empty sequence"}
	}
	if fontSize <= 0 || math.IsNaN(fontSize) || math.IsInf(fontSize, 0) {
		return EmojiAsset{}, EmojiError{Sequence: sequence, Detail: "font size must be finite and positive"}
	}
	asset, found, err := p.emojiProvider.LookupEmoji(sequence)
	if err != nil {
		return EmojiAsset{}, EmojiError{Sequence: sequence, Detail: err.Error()}
	}
	if !found {
		return EmojiAsset{}, EmojiError{Sequence: sequence, Detail: "not found"}
	}
	if isNilImage(asset.Image) {
		return EmojiAsset{}, EmojiError{Sequence: sequence, Detail: "provider returned a nil image"}
	}
	bounds := asset.Image.Bounds()
	if bounds.Empty() {
		return EmojiAsset{}, EmojiError{Sequence: sequence, Detail: "provider returned an empty image"}
	}
	if asset.HeightEm <= 0 || asset.AdvanceEm <= 0 ||
		math.IsNaN(asset.HeightEm) || math.IsNaN(asset.BaselineEm) || math.IsNaN(asset.AdvanceEm) ||
		math.IsInf(asset.HeightEm, 0) || math.IsInf(asset.BaselineEm, 0) || math.IsInf(asset.AdvanceEm, 0) {
		return EmojiAsset{}, EmojiError{Sequence: sequence, Detail: "provider returned invalid metrics"}
	}
	return asset, nil
}

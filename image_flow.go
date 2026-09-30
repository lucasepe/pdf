package pdf

import (
	"fmt"
	"math"
)

const (
	defaultImageWidthFraction = 0.78
	defaultImageMaxFraction   = 0.98
	defaultImageGapFraction   = 0.02
)

// ImageLayoutError reports invalid image sizing or an image block that cannot
// fit the page body.
type ImageLayoutError struct{ Detail string }

func (e ImageLayoutError) Error() string { return "image layout: " + e.Detail }

// ImageAlign controls horizontal placement of image blocks. The zero value is
// centered to match ebookgen's standalone-image behavior.
type ImageAlign uint8

const (
	ImageAlignCenter ImageAlign = iota
	ImageAlignLeft
	ImageAlignRight
)

// ImageOptions controls a standalone image block. Width, when positive, is the
// final point width and overrides WidthFraction, Scale, and MaxWidthFraction.
// Otherwise zero values select ebookgen-compatible defaults: width 0.78,
// scale 1, and maximum width 0.98 of the body.
type ImageOptions struct {
	Width, WidthFraction, MaxWidthFraction float64
	Scale                                  float64
	Align                                  ImageAlign
	SpaceBefore, SpaceAfter                float64
}

// ImageItem is one member of an image flow group. Width overrides fractional
// sizing; otherwise zero values use width fraction 0.78 and scale 1.
type ImageItem struct {
	Image                *ImageResource
	Width, WidthFraction float64
	Scale                float64
}

// ImageGroupOptions controls a row-wrapping image group. Zero values use a
// maximum row width of 0.98 of the body, horizontal gap 0.02 of the body, and
// a 6pt vertical row gap.
type ImageGroupOptions struct {
	MaxWidthFraction        float64
	GapFraction, RowGap     float64
	Align                   ImageAlign
	SpaceBefore, SpaceAfter float64
}

type laidOutImage struct {
	resource      *ImageResource
	width, height float64
}

type imageRow struct {
	items         []laidOutImage
	width, height float64
}

// DrawImage lays out a centered, left-aligned, or right-aligned atomic image
// block and opens a new page first when necessary.
func (f *Flow) DrawImage(resource *ImageResource, options ImageOptions) *Flow {
	if f.err != nil {
		return f
	}
	if options.Align > ImageAlignRight || !finite(options.Width, options.WidthFraction, options.MaxWidthFraction, options.Scale, options.SpaceBefore, options.SpaceAfter) || options.Width < 0 || options.WidthFraction < 0 || options.MaxWidthFraction < 0 || options.Scale < 0 || options.SpaceBefore < 0 || options.SpaceAfter < 0 {
		return f.failImage("invalid standalone image options")
	}
	maxFraction := options.MaxWidthFraction
	if maxFraction == 0 {
		maxFraction = defaultImageMaxFraction
	}
	image, ok := f.layoutImage(resource, options.Width, options.WidthFraction, options.Scale, maxFraction)
	if !ok {
		return f
	}
	if image.height > f.body.Height+0.001 {
		return f.failImage(fmt.Sprintf("%.3fpt image height exceeds %.3fpt page body", image.height, f.body.Height))
	}
	f.imageSpaceBefore(options.SpaceBefore)
	f.EnsureSpace(image.height)
	if f.err != nil {
		return f
	}
	x := f.body.X
	switch options.Align {
	case ImageAlignCenter:
		x += (f.body.Width - image.width) / 2
	case ImageAlignRight:
		x += f.body.Width - image.width
	}
	f.context.DrawImageResourceScaledAnchored(resource, x, f.y, image.width, image.height, 0, 0)
	if err := f.context.Error(); err != nil {
		return f.failImage(err.Error())
	}
	f.y += image.height
	f.imageSpaceAfter(options.SpaceAfter)
	return f
}

// DrawImageGroup lays out resources in centered rows, wrapping before the row
// exceeds MaxWidthFraction. Each row is atomic for pagination.
func (f *Flow) DrawImageGroup(items []ImageItem, options ImageGroupOptions) *Flow {
	if f.err != nil {
		return f
	}
	if len(items) == 0 {
		return f.failImage("empty image group")
	}
	if options.Align > ImageAlignRight || !finite(options.MaxWidthFraction, options.GapFraction, options.RowGap, options.SpaceBefore, options.SpaceAfter) || options.MaxWidthFraction < 0 || options.GapFraction < 0 || options.RowGap < 0 || options.SpaceBefore < 0 || options.SpaceAfter < 0 {
		return f.failImage("invalid image group options")
	}
	maxFraction := options.MaxWidthFraction
	if maxFraction == 0 {
		maxFraction = defaultImageMaxFraction
	}
	if maxFraction <= 0 || maxFraction > 1 {
		return f.failImage("maximum width fraction must be in (0,1]")
	}
	gapFraction := options.GapFraction
	if gapFraction == 0 {
		gapFraction = defaultImageGapFraction
	}
	if gapFraction > 1 {
		return f.failImage("gap fraction must not exceed 1")
	}
	rowGap := options.RowGap
	if rowGap == 0 {
		rowGap = 6
	}
	maxWidth := f.body.Width * maxFraction
	gap := f.body.Width * gapFraction
	var rows []imageRow
	row := imageRow{}
	for i, item := range items {
		if !finite(item.Width, item.WidthFraction, item.Scale) || item.Width < 0 || item.WidthFraction < 0 || item.Scale < 0 {
			return f.failImage(fmt.Sprintf("item %d has invalid sizing", i+1))
		}
		image, ok := f.layoutImage(item.Image, item.Width, item.WidthFraction, item.Scale, maxFraction)
		if !ok {
			return f
		}
		if image.width > maxWidth+0.001 {
			return f.failImage(fmt.Sprintf("item %d width %.3fpt exceeds %.3fpt group row", i+1, image.width, maxWidth))
		}
		if image.height > f.body.Height+0.001 {
			return f.failImage(fmt.Sprintf("item %d height %.3fpt exceeds %.3fpt page body", i+1, image.height, f.body.Height))
		}
		additional := image.width
		if len(row.items) > 0 {
			additional += gap
		}
		if len(row.items) > 0 && row.width+additional > maxWidth+0.001 {
			rows = append(rows, row)
			row = imageRow{}
			additional = image.width
		}
		row.items = append(row.items, image)
		row.width += additional
		row.height = math.Max(row.height, image.height)
	}
	if len(row.items) > 0 {
		rows = append(rows, row)
	}

	f.imageSpaceBefore(options.SpaceBefore)
	for i, row := range rows {
		if i > 0 {
			if rowGap+row.height > f.RemainingHeight()+0.001 {
				f.NewPage()
			} else {
				f.y += rowGap
			}
		} else {
			f.EnsureSpace(row.height)
		}
		if f.err != nil {
			return f
		}
		x := f.body.X
		switch options.Align {
		case ImageAlignCenter:
			x += (f.body.Width - row.width) / 2
		case ImageAlignRight:
			x += f.body.Width - row.width
		}
		for j, image := range row.items {
			if j > 0 {
				x += gap
			}
			y := f.y + row.height - image.height
			f.context.DrawImageResourceScaledAnchored(image.resource, x, y, image.width, image.height, 0, 0)
			if err := f.context.Error(); err != nil {
				return f.failImage(err.Error())
			}
			x += image.width
		}
		f.y += row.height
	}
	f.imageSpaceAfter(options.SpaceAfter)
	return f
}

func (f *Flow) layoutImage(resource *ImageResource, width, fraction, scale, maxFraction float64) (laidOutImage, bool) {
	if resource == nil || resource.doc != f.context.doc || resource.width <= 0 || resource.height <= 0 {
		f.failImage("resource belongs to another document or is invalid")
		return laidOutImage{}, false
	}
	if maxFraction <= 0 || maxFraction > 1 {
		f.failImage("maximum width fraction must be in (0,1]")
		return laidOutImage{}, false
	}
	if width == 0 {
		if fraction == 0 {
			fraction = defaultImageWidthFraction
		}
		if fraction <= 0 {
			f.failImage("width fraction must be positive")
			return laidOutImage{}, false
		}
		if scale == 0 {
			scale = 1
		}
		width = f.body.Width * math.Min(fraction*scale, maxFraction)
	}
	if width <= 0 || width > f.body.Width+0.001 {
		f.failImage(fmt.Sprintf("%.3fpt image width does not fit %.3fpt body", width, f.body.Width))
		return laidOutImage{}, false
	}
	height := width * float64(resource.height) / float64(resource.width)
	return laidOutImage{resource: resource, width: width, height: height}, true
}

func (f *Flow) failImage(detail string) *Flow {
	if f.err == nil {
		f.err = ImageLayoutError{Detail: detail}
	}
	return f
}

func (f *Flow) imageSpaceBefore(space float64) {
	if f.err != nil || space == 0 || f.y <= f.body.Y+0.001 {
		return
	}
	if space > f.RemainingHeight() {
		f.NewPage()
	} else {
		f.y += space
	}
}

func (f *Flow) imageSpaceAfter(space float64) {
	if f.err == nil && space > 0 {
		f.y = math.Min(f.body.Y+f.body.Height, f.y+space)
	}
}

package ui

import (
	"image"

	"github.com/diamondburned/gotk4/pkg/cairo"
)

func imageSurface(img *image.NRGBA) *cairo.Surface {
	if img == nil {
		return nil
	}
	width, height := img.Rect.Dx(), img.Rect.Dy()
	if width < 1 || height < 1 {
		return nil
	}
	surf := cairo.CreateImageSurface(cairo.FormatARGB32, width, height)
	dst := surf.Data()
	src := img.Pix
	srcStride := img.Stride
	dstStride := surf.Stride()
	for y := 0; y < height; y++ {
		srow := src[y*srcStride:]
		drow := dst[y*dstStride:]
		for x := 0; x < width; x++ {
			si := x * 4
			di := x * 4
			a := srow[si+3]
			if a == 255 {
				drow[di+0] = srow[si+2]
				drow[di+1] = srow[si+1]
				drow[di+2] = srow[si+0]
				drow[di+3] = 255
				continue
			}
			if a == 0 {
				drow[di+0] = 0
				drow[di+1] = 0
				drow[di+2] = 0
				drow[di+3] = 0
				continue
			}
			ia := uint16(a)
			drow[di+0] = uint8(uint16(srow[si+2]) * ia / 255)
			drow[di+1] = uint8(uint16(srow[si+1]) * ia / 255)
			drow[di+2] = uint8(uint16(srow[si+0]) * ia / 255)
			drow[di+3] = a
		}
	}
	surf.MarkDirty()
	return surf
}

func paintSurface(cr *cairo.Context, surface *cairo.Surface, filter cairo.Filter) {
	if surface == nil {
		return
	}
	pattern, err := cairo.NewPatternForSurface(surface)
	if err != nil {
		cr.SetSourceSurface(surface, 0, 0)
		cr.Paint()
		return
	}
	pattern.PatternSetFilter(filter)
	cr.SetSource(pattern)
	cr.Paint()
}

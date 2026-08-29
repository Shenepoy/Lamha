package grab

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"os"
)

// PNGSize reads the pixel size of a capture without decoding pixels.
func PNGSize(path string) (image.Point, error) {
	file, err := os.Open(path)
	if err != nil {
		return image.Point{}, err
	}
	defer file.Close()
	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		return image.Point{}, fmt.Errorf("read capture size: %w", err)
	}
	return image.Pt(cfg.Width, cfg.Height), nil
}

// TooSmall reports a grab that is not a usable desktop or window image.
func TooSmall(path string) bool {
	size, err := PNGSize(path)
	return err != nil || size.X < 64 || size.Y < 64
}

// CropPNG replaces path with the intersection of the image and region.
func CropPNG(path string, region image.Rectangle) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	img, err := png.Decode(file)
	file.Close()
	if err != nil {
		return fmt.Errorf("decode capture for crop: %w", err)
	}

	region = region.Intersect(img.Bounds())
	if region.Empty() || region.Eq(img.Bounds()) {
		return nil
	}

	dst := image.NewNRGBA(image.Rect(0, 0, region.Dx(), region.Dy()))
	draw.Draw(dst, dst.Bounds(), img, region.Min, draw.Src)

	tmp := path + ".crop"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := png.Encode(out, dst); err != nil {
		out.Close()
		os.Remove(tmp)
		return fmt.Errorf("encode cropped capture: %w", err)
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

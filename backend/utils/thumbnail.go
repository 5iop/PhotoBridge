package utils

import (
	"bytes"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

const (
	ThumbSmallWidth  = 400
	ThumbLargeWidth  = 1600
	JpegQualitySmall = 75
	JpegQualityLarge = 85

	// For very large images, pre-shrink to reduce peak memory and resize cost.
	preShrinkMaxLongSide = ThumbLargeWidth * 2
)

// ThumbnailResult contains generated thumbnails and source dimensions.
type ThumbnailResult struct {
	Small       []byte
	Large       []byte
	Width       int
	Height      int
	SmallWidth  int
	SmallHeight int
}

// GenerateThumbnails creates small and large JPEG thumbnails from an image file.
func GenerateThumbnails(imagePath string) (*ThumbnailResult, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	result := &ThumbnailResult{
		Width:  cfg.Width,
		Height: cfg.Height,
	}

	working := img
	longSide := cfg.Width
	if cfg.Height > longSide {
		longSide = cfg.Height
	}
	if longSide > preShrinkMaxLongSide {
		// Pre-shrink huge images across all formats to lower memory/CPU in later stages.
		if cfg.Width >= cfg.Height {
			working = imaging.Resize(img, preShrinkMaxLongSide, 0, imaging.Box)
		} else {
			working = imaging.Resize(img, 0, preShrinkMaxLongSide, imaging.Box)
		}
		img = nil
	}

	largeWidth := ThumbLargeWidth
	if cfg.Width < largeWidth {
		largeWidth = cfg.Width
	}
	largeImg := imaging.Resize(working, largeWidth, 0, imaging.CatmullRom)

	smallImg := imaging.Resize(largeImg, ThumbSmallWidth, 0, imaging.Box)
	smallBounds := smallImg.Bounds()
	result.SmallWidth = smallBounds.Dx()
	result.SmallHeight = smallBounds.Dy()

	var smallBuf bytes.Buffer
	if err := jpeg.Encode(&smallBuf, smallImg, &jpeg.Options{Quality: JpegQualitySmall}); err != nil {
		return nil, err
	}
	result.Small = smallBuf.Bytes()
	smallImg = nil

	var largeBuf bytes.Buffer
	if err := jpeg.Encode(&largeBuf, largeImg, &jpeg.Options{Quality: JpegQualityLarge}); err != nil {
		return nil, err
	}
	result.Large = largeBuf.Bytes()

	return result, nil
}

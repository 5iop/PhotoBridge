package utils

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"

	"github.com/disintegration/imaging"
)

const (
	ThumbSmallWidth = 400  // 列表缩略图宽度
	ThumbLargeWidth = 1600 // 预览缩略图宽度
	JpegQuality     = 85   // JPEG压缩质量
)

// ThumbnailResult 缩略图生成结果
type ThumbnailResult struct {
	Small       []byte
	Large       []byte
	Width       int
	Height      int
	SmallWidth  int
	SmallHeight int
}

// GenerateThumbnails 从图片文件生成两种尺寸的缩略图
func GenerateThumbnails(imagePath string) (*ThumbnailResult, error) {
	// 打开原图
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 解码图片
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()

	result := &ThumbnailResult{
		Width:  originalWidth,
		Height: originalHeight,
	}

	// 生成小缩略图 (用于列表)
	smallImg := imaging.Resize(img, ThumbSmallWidth, 0, imaging.Lanczos)
	smallBounds := smallImg.Bounds()
	result.SmallWidth = smallBounds.Dx()
	result.SmallHeight = smallBounds.Dy()

	var smallBuf bytes.Buffer
	if err := jpeg.Encode(&smallBuf, smallImg, &jpeg.Options{Quality: JpegQuality}); err != nil {
		return nil, err
	}
	result.Small = smallBuf.Bytes()

	// 生成大缩略图 (用于预览)
	// 如果原图小于预览尺寸，则使用原图尺寸
	largeWidth := ThumbLargeWidth
	if originalWidth < largeWidth {
		largeWidth = originalWidth
	}

	largeImg := imaging.Resize(img, largeWidth, 0, imaging.Lanczos)
	var largeBuf bytes.Buffer
	if err := jpeg.Encode(&largeBuf, largeImg, &jpeg.Options{Quality: JpegQuality}); err != nil {
		return nil, err
	}
	result.Large = largeBuf.Bytes()

	return result, nil
}

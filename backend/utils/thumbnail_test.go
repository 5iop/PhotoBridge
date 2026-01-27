package utils

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func createTestImage(t *testing.T, path string, width, height int, format string) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a test color
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	defer file.Close()

	switch format {
	case "jpeg", "jpg":
		if err := jpeg.Encode(file, img, &jpeg.Options{Quality: 90}); err != nil {
			t.Fatalf("Failed to encode JPEG: %v", err)
		}
	case "png":
		if err := png.Encode(file, img); err != nil {
			t.Fatalf("Failed to encode PNG: %v", err)
		}
	default:
		t.Fatalf("Unsupported format: %s", format)
	}
}

func TestGenerateThumbnailsJPEG(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "thumbtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test JPEG image
	imagePath := filepath.Join(tempDir, "test.jpg")
	createTestImage(t, imagePath, 2000, 1500, "jpeg")

	// Generate thumbnails
	result, err := GenerateThumbnails(imagePath)
	if err != nil {
		t.Fatalf("GenerateThumbnails failed: %v", err)
	}

	// Verify result
	if result.Width != 2000 {
		t.Errorf("Expected width 2000, got %d", result.Width)
	}
	if result.Height != 1500 {
		t.Errorf("Expected height 1500, got %d", result.Height)
	}

	// Check small thumbnail
	if result.Small == nil || len(result.Small) == 0 {
		t.Error("Small thumbnail should not be empty")
	}
	if result.SmallWidth != ThumbSmallWidth {
		t.Errorf("Expected small width %d, got %d", ThumbSmallWidth, result.SmallWidth)
	}

	// Check large thumbnail
	if result.Large == nil || len(result.Large) == 0 {
		t.Error("Large thumbnail should not be empty")
	}

	// Verify thumbnails are valid JPEG
	if len(result.Small) < 2 || result.Small[0] != 0xFF || result.Small[1] != 0xD8 {
		t.Error("Small thumbnail is not a valid JPEG")
	}
	if len(result.Large) < 2 || result.Large[0] != 0xFF || result.Large[1] != 0xD8 {
		t.Error("Large thumbnail is not a valid JPEG")
	}
}

func TestGenerateThumbnailsPNG(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "thumbtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test PNG image
	imagePath := filepath.Join(tempDir, "test.png")
	createTestImage(t, imagePath, 2000, 1500, "png")

	result, err := GenerateThumbnails(imagePath)
	if err != nil {
		t.Fatalf("GenerateThumbnails failed: %v", err)
	}

	if result.Width != 2000 || result.Height != 1500 {
		t.Errorf("Dimensions mismatch: %dx%d", result.Width, result.Height)
	}

	if len(result.Small) == 0 {
		t.Error("Small thumbnail should not be empty")
	}
	if len(result.Large) == 0 {
		t.Error("Large thumbnail should not be empty")
	}
}

func TestGenerateThumbnailsSmallImage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "thumbtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a small image (smaller than ThumbLargeWidth)
	imagePath := filepath.Join(tempDir, "small.jpg")
	createTestImage(t, imagePath, 800, 600, "jpeg")

	result, err := GenerateThumbnails(imagePath)
	if err != nil {
		t.Fatalf("GenerateThumbnails failed: %v", err)
	}

	if result.Width != 800 {
		t.Errorf("Expected width 800, got %d", result.Width)
	}
	if result.Height != 600 {
		t.Errorf("Expected height 600, got %d", result.Height)
	}

	// Both thumbnails should still be generated
	if len(result.Small) == 0 {
		t.Error("Small thumbnail should not be empty")
	}
	if len(result.Large) == 0 {
		t.Error("Large thumbnail should not be empty")
	}
}

func TestGenerateThumbnailsNonExistent(t *testing.T) {
	_, err := GenerateThumbnails("/nonexistent/path/image.jpg")
	if err == nil {
		t.Error("Should return error for non-existent file")
	}
}

func TestGenerateThumbnailsInvalidImage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "thumbtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file that is not a valid image
	invalidPath := filepath.Join(tempDir, "invalid.jpg")
	if err := os.WriteFile(invalidPath, []byte("not an image"), 0644); err != nil {
		t.Fatalf("Failed to create invalid file: %v", err)
	}

	_, err = GenerateThumbnails(invalidPath)
	if err == nil {
		t.Error("Should return error for invalid image")
	}
}

func TestThumbnailConstants(t *testing.T) {
	if ThumbSmallWidth <= 0 {
		t.Error("ThumbSmallWidth should be positive")
	}
	if ThumbLargeWidth <= 0 {
		t.Error("ThumbLargeWidth should be positive")
	}
	if ThumbSmallWidth >= ThumbLargeWidth {
		t.Error("ThumbSmallWidth should be less than ThumbLargeWidth")
	}
	if JpegQualitySmall <= 0 || JpegQualitySmall > 100 {
		t.Error("JpegQualitySmall should be between 1 and 100")
	}
	if JpegQualityLarge <= 0 || JpegQualityLarge > 100 {
		t.Error("JpegQualityLarge should be between 1 and 100")
	}
}

func TestGenerateThumbnailsAspectRatio(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "thumbtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a 4:3 aspect ratio image
	imagePath := filepath.Join(tempDir, "aspect.jpg")
	createTestImage(t, imagePath, 4000, 3000, "jpeg")

	result, err := GenerateThumbnails(imagePath)
	if err != nil {
		t.Fatalf("GenerateThumbnails failed: %v", err)
	}

	// Check aspect ratio is preserved for small thumbnail
	expectedSmallHeight := ThumbSmallWidth * 3 / 4
	if result.SmallHeight < expectedSmallHeight-1 || result.SmallHeight > expectedSmallHeight+1 {
		t.Errorf("Small thumbnail aspect ratio not preserved: got %dx%d, expected ~%dx%d",
			result.SmallWidth, result.SmallHeight, ThumbSmallWidth, expectedSmallHeight)
	}
}

func TestGenerateThumbnailsVerticalImage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "thumbtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a vertical (portrait) image
	imagePath := filepath.Join(tempDir, "vertical.jpg")
	createTestImage(t, imagePath, 1500, 2000, "jpeg")

	result, err := GenerateThumbnails(imagePath)
	if err != nil {
		t.Fatalf("GenerateThumbnails failed: %v", err)
	}

	if result.Width != 1500 {
		t.Errorf("Expected width 1500, got %d", result.Width)
	}
	if result.Height != 2000 {
		t.Errorf("Expected height 2000, got %d", result.Height)
	}

	// Small thumbnail should be scaled by width
	if result.SmallWidth != ThumbSmallWidth {
		t.Errorf("Expected small width %d, got %d", ThumbSmallWidth, result.SmallWidth)
	}
	// Height should be taller than width for vertical image
	if result.SmallHeight <= result.SmallWidth {
		t.Errorf("Vertical image small thumbnail should have height > width: %dx%d",
			result.SmallWidth, result.SmallHeight)
	}
}

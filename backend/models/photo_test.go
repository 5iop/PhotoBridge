package models

import "testing"

func TestIsRawExtension(t *testing.T) {
	tests := []struct {
		name     string
		ext      string
		expected bool
	}{
		// RAW formats
		{"CR2", ".cr2", true},
		{"CR3", ".cr3", true},
		{"NEF", ".nef", true},
		{"ARW", ".arw", true},
		{"DNG", ".dng", true},
		{"ORF", ".orf", true},
		{"RW2", ".rw2", true},
		{"PEF", ".pef", true},
		{"RAF", ".raf", true},
		{"SRW", ".srw", true},
		{"X3F", ".x3f", true},
		{"RAW", ".raw", true},

		// Non-RAW formats
		{"JPG", ".jpg", false},
		{"JPEG", ".jpeg", false},
		{"PNG", ".png", false},
		{"GIF", ".gif", false},
		{"empty", "", false},
		{"uppercase CR2", ".CR2", false}, // Case sensitive
		{"txt", ".txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRawExtension(tt.ext)
			if result != tt.expected {
				t.Errorf("IsRawExtension(%q) = %v, expected %v", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestIsImageExtension(t *testing.T) {
	tests := []struct {
		name     string
		ext      string
		expected bool
	}{
		// Image formats
		{"JPG", ".jpg", true},
		{"JPEG", ".jpeg", true},
		{"PNG", ".png", true},
		{"GIF", ".gif", true},
		{"WEBP", ".webp", true},
		{"BMP", ".bmp", true},
		{"TIFF", ".tiff", true},
		{"TIF", ".tif", true},

		// Non-image formats
		{"CR2", ".cr2", false},
		{"NEF", ".nef", false},
		{"empty", "", false},
		{"uppercase JPG", ".JPG", false}, // Case sensitive
		{"txt", ".txt", false},
		{"pdf", ".pdf", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsImageExtension(tt.ext)
			if result != tt.expected {
				t.Errorf("IsImageExtension(%q) = %v, expected %v", tt.ext, result, tt.expected)
			}
		})
	}
}

package utils

import "testing"

func TestValidatePathComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid cases
		{"simple name", "project1", true},
		{"with underscore", "my_project", true},
		{"with hyphen", "my-project", true},
		{"with space", "my project", true},
		{"with numbers", "project123", true},
		{"chinese name", "婚礼摄影", true},
		{"mixed", "婚礼摄影2024", true},
		{"with extension", "photo.jpg", true},

		// Invalid cases
		{"empty string", "", false},
		{"path traversal dot dot", "..", false},
		{"path traversal with slash", "../etc", false},
		{"forward slash", "path/to/file", false},
		{"backslash", "path\\to\\file", false},
		{"hidden file", ".hidden", false},
		{"null byte", "file\x00name", false},
		{"starts with dot", ".gitignore", false},
		{"double dots in path", "foo/../bar", false},
		{"special chars", "file<>name", false},
		{"only dots", "...", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePathComponent(tt.input)
			if result != tt.expected {
				t.Errorf("ValidatePathComponent(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeProjectName(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedName  string
		expectedValid bool
	}{
		// Valid cases
		{"simple", "project", "project", true},
		{"with spaces to trim", "  project  ", "project", true},
		{"chinese", "婚礼摄影", "婚礼摄影", true},

		// Invalid cases
		{"empty", "", "", false},
		{"path traversal", "../secret", "", false},
		{"hidden", ".hidden", "", false},
		{"with slash", "a/b", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, valid := SanitizeProjectName(tt.input)
			if valid != tt.expectedValid {
				t.Errorf("SanitizeProjectName(%q) valid = %v, expected %v", tt.input, valid, tt.expectedValid)
			}
			if valid && result != tt.expectedName {
				t.Errorf("SanitizeProjectName(%q) = %q, expected %q", tt.input, result, tt.expectedName)
			}
		})
	}
}

func TestValidateFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid cases
		{"simple jpg", "photo.jpg", true},
		{"simple png", "image.png", true},
		{"with underscore", "my_photo.jpg", true},
		{"chinese filename", "照片.jpg", true},

		// Invalid cases
		{"empty", "", false},
		{"path included", "path/to/file.jpg", false},
		{"backslash path", "path\\file.jpg", false},
		{"hidden file", ".hidden.jpg", false},
		{"traversal", "../file.jpg", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateFileName(tt.input)
			if result != tt.expected {
				t.Errorf("ValidateFileName(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

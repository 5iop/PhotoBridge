package utils

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestGenerateETag(t *testing.T) {
	tests := []struct {
		name      string
		photoID   uint
		updatedAt time.Time
		size      string
		wantLen   int
	}{
		{
			name:      "small thumbnail",
			photoID:   1,
			updatedAt: time.Unix(1234567890, 0),
			size:      "small",
			wantLen:   34, // "xxxx..." with quotes
		},
		{
			name:      "large thumbnail",
			photoID:   999,
			updatedAt: time.Unix(1234567890, 0),
			size:      "large",
			wantLen:   34,
		},
		{
			name:      "different photo same time",
			photoID:   2,
			updatedAt: time.Unix(1234567890, 0),
			size:      "small",
			wantLen:   34,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			etag := GenerateETag(tt.photoID, tt.updatedAt, tt.size)

			// Check format: should start and end with quotes
			if etag[0] != '"' || etag[len(etag)-1] != '"' {
				t.Errorf("ETag should be quoted, got: %s", etag)
			}

			// Check length
			if len(etag) != tt.wantLen {
				t.Errorf("ETag length = %d, want %d", len(etag), tt.wantLen)
			}

			// Check hex characters (between quotes)
			for i := 1; i < len(etag)-1; i++ {
				c := etag[i]
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("ETag contains non-hex character: %c", c)
					break
				}
			}
		})
	}
}

func TestGenerateETagConsistency(t *testing.T) {
	// Same input should always produce same output
	photoID := uint(123)
	updatedAt := time.Unix(1234567890, 0)
	size := "small"

	etag1 := GenerateETag(photoID, updatedAt, size)
	etag2 := GenerateETag(photoID, updatedAt, size)

	if etag1 != etag2 {
		t.Errorf("Same input produced different ETags: %s != %s", etag1, etag2)
	}
}

func TestGenerateETagDifferentInputs(t *testing.T) {
	baseTime := time.Unix(1234567890, 0)

	tests := []struct {
		name      string
		photoID1  uint
		photoID2  uint
		time1     time.Time
		time2     time.Time
		size1     string
		size2     string
		shouldDif bool
	}{
		{
			name:      "different photo ID",
			photoID1:  1,
			photoID2:  2,
			time1:     baseTime,
			time2:     baseTime,
			size1:     "small",
			size2:     "small",
			shouldDif: true,
		},
		{
			name:      "different update time",
			photoID1:  1,
			photoID2:  1,
			time1:     baseTime,
			time2:     baseTime.Add(time.Hour),
			size1:     "small",
			size2:     "small",
			shouldDif: true,
		},
		{
			name:      "different size",
			photoID1:  1,
			photoID2:  1,
			time1:     baseTime,
			time2:     baseTime,
			size1:     "small",
			size2:     "large",
			shouldDif: true,
		},
		{
			name:      "same everything",
			photoID1:  1,
			photoID2:  1,
			time1:     baseTime,
			time2:     baseTime,
			size1:     "small",
			size2:     "small",
			shouldDif: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			etag1 := GenerateETag(tt.photoID1, tt.time1, tt.size1)
			etag2 := GenerateETag(tt.photoID2, tt.time2, tt.size2)

			if tt.shouldDif {
				if etag1 == etag2 {
					t.Errorf("Expected different ETags, got same: %s", etag1)
				}
			} else {
				if etag1 != etag2 {
					t.Errorf("Expected same ETags, got different: %s != %s", etag1, etag2)
				}
			}
		})
	}
}

func TestGenerateFileETag(t *testing.T) {
	// Create temporary file
	tempDir, err := os.MkdirTemp("", "etagtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test.jpg")
	content := []byte("test image content")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	etag, err := GenerateFileETag(filePath)
	if err != nil {
		t.Fatalf("GenerateFileETag failed: %v", err)
	}

	// Check format
	if etag[0] != '"' || etag[len(etag)-1] != '"' {
		t.Errorf("ETag should be quoted, got: %s", etag)
	}

	// Check length (MD5 hash is 32 hex chars + 2 quotes = 34)
	if len(etag) != 34 {
		t.Errorf("ETag length = %d, want 34", len(etag))
	}
}

func TestGenerateFileETagNonExistent(t *testing.T) {
	_, err := GenerateFileETag("/nonexistent/file.jpg")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestGenerateFileETagConsistency(t *testing.T) {
	// Create temporary file
	tempDir, err := os.MkdirTemp("", "etagtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test.jpg")
	content := []byte("test content")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Generate ETag twice
	etag1, err := GenerateFileETag(filePath)
	if err != nil {
		t.Fatalf("First GenerateFileETag failed: %v", err)
	}

	etag2, err := GenerateFileETag(filePath)
	if err != nil {
		t.Fatalf("Second GenerateFileETag failed: %v", err)
	}

	if etag1 != etag2 {
		t.Errorf("Same file produced different ETags: %s != %s", etag1, etag2)
	}
}

func TestGenerateFileETagChangedContent(t *testing.T) {
	// Create temporary file
	tempDir, err := os.MkdirTemp("", "etagtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test.txt")

	// Write initial content
	if err := os.WriteFile(filePath, []byte("original"), 0644); err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	etag1, err := GenerateFileETag(filePath)
	if err != nil {
		t.Fatalf("First GenerateFileETag failed: %v", err)
	}

	// Wait enough to ensure different modification time (file system dependent)
	time.Sleep(1100 * time.Millisecond)

	// Change content to different size (ETag uses size + mtime)
	if err := os.WriteFile(filePath, []byte("modified content with different length"), 0644); err != nil {
		t.Fatalf("Failed to write modified file: %v", err)
	}

	etag2, err := GenerateFileETag(filePath)
	if err != nil {
		t.Fatalf("Second GenerateFileETag failed: %v", err)
	}

	// ETags should be different (different size and mtime)
	if etag1 == etag2 {
		t.Errorf("Expected different ETags after file modification, got same: %s", etag1)
	}
}

func TestCheckETag(t *testing.T) {
	tests := []struct {
		name       string
		clientETag string
		serverETag string
		want       bool
	}{
		{
			name:       "matching ETags",
			clientETag: `"abc123"`,
			serverETag: `"abc123"`,
			want:       true,
		},
		{
			name:       "different ETags",
			clientETag: `"abc123"`,
			serverETag: `"def456"`,
			want:       false,
		},
		{
			name:       "empty client ETag",
			clientETag: "",
			serverETag: `"abc123"`,
			want:       false,
		},
		{
			name:       "client has multiple ETags",
			clientETag: `"abc123", "def456"`,
			serverETag: `"abc123"`,
			want:       false, // Our implementation only checks exact match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test context
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set request with If-None-Match header
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.clientETag != "" {
				req.Header.Set("If-None-Match", tt.clientETag)
			}
			c.Request = req

			got := CheckETag(c, tt.serverETag)
			if got != tt.want {
				t.Errorf("CheckETag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckETagNoHeader(t *testing.T) {
	// Create test context without If-None-Match header
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	result := CheckETag(c, `"abc123"`)
	if result {
		t.Error("CheckETag should return false when If-None-Match header is missing")
	}
}

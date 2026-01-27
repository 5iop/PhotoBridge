package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
)

// createMultipartFile creates a multipart.FileHeader for testing
func createMultipartFile(t *testing.T, filename string, content []byte) (*multipart.FileHeader, func()) {
	t.Helper()

	// Create temp file
	tempDir, err := os.MkdirTemp("", "hashtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	filePath := filepath.Join(tempDir, filename)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write temp file: %v", err)
	}

	// Create a buffer to write the multipart form
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create form file: %v", err)
	}

	// Copy file content
	_, err = fw.Write(content)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write file content: %v", err)
	}
	w.Close()

	// Parse the multipart form
	r := multipart.NewReader(&b, w.Boundary())
	form, err := r.ReadForm(10 << 20)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to read form: %v", err)
	}

	files := form.File["file"]
	if len(files) == 0 {
		os.RemoveAll(tempDir)
		t.Fatal("No files in form")
	}

	cleanup := func() {
		form.RemoveAll()
		os.RemoveAll(tempDir)
	}

	return files[0], cleanup
}

func TestCalculateFileHashBasic(t *testing.T) {
	content := []byte("Hello, World!")
	fh, cleanup := createMultipartFile(t, "test.txt", content)
	defer cleanup()

	hash, err := CalculateFileHash(fh)
	if err != nil {
		t.Fatalf("CalculateFileHash failed: %v", err)
	}

	// Calculate expected hash
	hasher := sha256.New()
	hasher.Write(content)
	expected := hex.EncodeToString(hasher.Sum(nil))

	if hash != expected {
		t.Errorf("Hash mismatch: got %s, expected %s", hash, expected)
	}
}

func TestCalculateFileHashEmpty(t *testing.T) {
	content := []byte{}
	fh, cleanup := createMultipartFile(t, "empty.txt", content)
	defer cleanup()

	hash, err := CalculateFileHash(fh)
	if err != nil {
		t.Fatalf("CalculateFileHash failed: %v", err)
	}

	// SHA-256 of empty string
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hash != expected {
		t.Errorf("Hash mismatch for empty file: got %s, expected %s", hash, expected)
	}
}

func TestCalculateFileHashLargeFile(t *testing.T) {
	// Create a 1MB file
	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}

	fh, cleanup := createMultipartFile(t, "large.bin", content)
	defer cleanup()

	hash, err := CalculateFileHash(fh)
	if err != nil {
		t.Fatalf("CalculateFileHash failed: %v", err)
	}

	// Calculate expected hash
	hasher := sha256.New()
	hasher.Write(content)
	expected := hex.EncodeToString(hasher.Sum(nil))

	if hash != expected {
		t.Errorf("Hash mismatch for large file: got %s, expected %s", hash, expected)
	}
}

func TestCalculateFileHashDifferentContent(t *testing.T) {
	content1 := []byte("content one")
	content2 := []byte("content two")

	fh1, cleanup1 := createMultipartFile(t, "file1.txt", content1)
	defer cleanup1()
	fh2, cleanup2 := createMultipartFile(t, "file2.txt", content2)
	defer cleanup2()

	hash1, err := CalculateFileHash(fh1)
	if err != nil {
		t.Fatalf("CalculateFileHash for file1 failed: %v", err)
	}

	hash2, err := CalculateFileHash(fh2)
	if err != nil {
		t.Fatalf("CalculateFileHash for file2 failed: %v", err)
	}

	if hash1 == hash2 {
		t.Error("Different content should produce different hashes")
	}
}

func TestCalculateFileHashSameContent(t *testing.T) {
	content := []byte("same content")

	fh1, cleanup1 := createMultipartFile(t, "file1.txt", content)
	defer cleanup1()
	fh2, cleanup2 := createMultipartFile(t, "file2.txt", content)
	defer cleanup2()

	hash1, err := CalculateFileHash(fh1)
	if err != nil {
		t.Fatalf("CalculateFileHash for file1 failed: %v", err)
	}

	hash2, err := CalculateFileHash(fh2)
	if err != nil {
		t.Fatalf("CalculateFileHash for file2 failed: %v", err)
	}

	if hash1 != hash2 {
		t.Error("Same content should produce same hashes")
	}
}

func TestCalculateFileHashFormat(t *testing.T) {
	content := []byte("test")
	fh, cleanup := createMultipartFile(t, "test.txt", content)
	defer cleanup()

	hash, err := CalculateFileHash(fh)
	if err != nil {
		t.Fatalf("CalculateFileHash failed: %v", err)
	}

	// SHA-256 produces 64 hex characters
	if len(hash) != 64 {
		t.Errorf("Hash should be 64 characters, got %d", len(hash))
	}

	// Should only contain hex characters
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Hash contains non-hex character: %c", c)
		}
	}
}

func TestCalculateFileHashBinaryContent(t *testing.T) {
	// Test with binary content (like an image)
	content := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
	fh, cleanup := createMultipartFile(t, "test.jpg", content)
	defer cleanup()

	hash, err := CalculateFileHash(fh)
	if err != nil {
		t.Fatalf("CalculateFileHash failed: %v", err)
	}

	// Calculate expected hash
	hasher := sha256.New()
	hasher.Write(content)
	expected := hex.EncodeToString(hasher.Sum(nil))

	if hash != expected {
		t.Errorf("Hash mismatch for binary content: got %s, expected %s", hash, expected)
	}
}

// Benchmark hash calculation
func BenchmarkCalculateFileHash(b *testing.B) {
	// Create a 100KB test file
	content := make([]byte, 100*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}

	tempDir, err := os.MkdirTemp("", "hashbench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "bench.bin")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		b.Fatalf("Failed to write file: %v", err)
	}

	// Create multipart form
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", "bench.bin")
	io.Copy(fw, bytes.NewReader(content))
	w.Close()

	r := multipart.NewReader(&buf, w.Boundary())
	form, _ := r.ReadForm(10 << 20)
	fh := form.File["file"][0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateFileHash(fh)
	}
}

package utils

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateZip(t *testing.T) {
	// Create temp directory with test files
	tempDir, err := os.MkdirTemp("", "ziptest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []struct {
		name    string
		content string
	}{
		{"file1.txt", "Hello World"},
		{"file2.txt", "Test Content"},
		{"photo.jpg", "fake image data"},
	}

	var filePaths []string
	for _, tf := range testFiles {
		path := filepath.Join(tempDir, tf.name)
		if err := os.WriteFile(path, []byte(tf.content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", tf.name, err)
		}
		filePaths = append(filePaths, path)
	}

	// Create zip
	var buf bytes.Buffer
	err = CreateZip(&buf, filePaths, tempDir)
	if err != nil {
		t.Fatalf("CreateZip failed: %v", err)
	}

	// Verify zip contents
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Failed to read zip: %v", err)
	}

	if len(zipReader.File) != len(testFiles) {
		t.Errorf("Expected %d files in zip, got %d", len(testFiles), len(zipReader.File))
	}

	// Verify each file
	fileMap := make(map[string]string)
	for _, tf := range testFiles {
		fileMap[tf.name] = tf.content
	}

	for _, f := range zipReader.File {
		expectedContent, ok := fileMap[f.Name]
		if !ok {
			t.Errorf("Unexpected file in zip: %s", f.Name)
			continue
		}

		rc, err := f.Open()
		if err != nil {
			t.Errorf("Failed to open %s in zip: %v", f.Name, err)
			continue
		}

		content := make([]byte, f.UncompressedSize64)
		_, err = rc.Read(content)
		rc.Close()

		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s: got %q, expected %q", f.Name, string(content), expectedContent)
		}
	}
}

func TestCreateZipMaxFilesLimit(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "ziptest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create more files than MaxFilesPerZip
	var filePaths []string
	for i := 0; i <= MaxFilesPerZip; i++ {
		path := filepath.Join(tempDir, "file"+string(rune('0'+i%10))+".txt")
		filePaths = append(filePaths, path)
	}

	var buf bytes.Buffer
	err = CreateZip(&buf, filePaths, tempDir)
	if err == nil {
		t.Error("Expected error for too many files, got nil")
	}
}

func TestCreateZipEmptyList(t *testing.T) {
	var buf bytes.Buffer
	err := CreateZip(&buf, []string{}, ".")
	if err != nil {
		t.Errorf("CreateZip with empty list should succeed, got: %v", err)
	}

	// Verify empty zip is valid
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Failed to read empty zip: %v", err)
	}

	if len(zipReader.File) != 0 {
		t.Errorf("Expected 0 files in empty zip, got %d", len(zipReader.File))
	}
}

func TestCreateZipNonExistentFile(t *testing.T) {
	var buf bytes.Buffer
	err := CreateZip(&buf, []string{"/nonexistent/file.txt"}, "/nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

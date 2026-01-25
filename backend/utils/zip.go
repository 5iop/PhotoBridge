package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MaxFilesPerZip limits the number of files in a single zip download to prevent abuse
const MaxFilesPerZip = 1000


// CreateZip creates a zip archive from a list of files using streaming.
// This implementation is memory-efficient as it uses io.Copy which streams
// file contents through a small buffer (typically 32KB) rather than loading
// entire files into memory.
func CreateZip(writer io.Writer, files []string, basePath string) error {
	if len(files) > MaxFilesPerZip {
		return fmt.Errorf("too many files (%d), maximum allowed is %d", len(files), MaxFilesPerZip)
	}

	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()

	for _, file := range files {
		err := addFileToZip(zipWriter, file, basePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func addFileToZip(zipWriter *zip.Writer, filePath string, basePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// Use relative path in zip
	relPath, err := filepath.Rel(basePath, filePath)
	if err != nil {
		relPath = filepath.Base(filePath)
	}
	header.Name = relPath

	// Always use Store (no compression) - photos are already compressed
	// This reduces CPU and memory usage significantly on limited servers
	header.Method = zip.Store

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

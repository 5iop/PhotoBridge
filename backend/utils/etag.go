package utils

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// GenerateETag generates an ETag based on photo ID, updated time, and size
func GenerateETag(photoID uint, updatedAt time.Time, size string) string {
	// Format: "photoID-timestamp-size"
	data := fmt.Sprintf("%d-%d-%s", photoID, updatedAt.Unix(), size)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf(`"%x"`, hash)
}

// GenerateFileETag generates an ETag for a file based on its path, size, and modification time
func GenerateFileETag(filePath string) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	// Use file name, size, and modification time to generate ETag
	data := fmt.Sprintf("%s-%d-%d", filepath.Base(filePath), info.Size(), info.ModTime().Unix())
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf(`"%x"`, hash), nil
}

// CheckETag checks if the request's If-None-Match header matches the given ETag
// Returns true if ETag matches (client has fresh cache)
func CheckETag(c *gin.Context, etag string) bool {
	clientETag := c.GetHeader("If-None-Match")
	return clientETag != "" && clientETag == etag
}

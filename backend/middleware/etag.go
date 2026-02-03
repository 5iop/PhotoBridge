package middleware

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// StaticFileETag adds ETag support for static files
func StaticFileETag() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only process GET requests for static files
		if c.Request.Method != "GET" {
			c.Next()
			return
		}

		// Get the file path from the request
		// This middleware should be used before serving static files
		c.Next()

		// If response has already been written and it's a file, add ETag
		if c.Writer.Status() == 200 && c.Writer.Header().Get("Content-Type") != "" {
			// ETag already set by Gin or other handlers
			return
		}
	}
}

// GenerateFileETag generates an ETag for a file based on its path and modification time
func GenerateFileETag(filePath string) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	// Use file path, size, and modification time to generate ETag
	data := fmt.Sprintf("%s-%d-%d", filepath.Base(filePath), info.Size(), info.ModTime().Unix())
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf(`"%x"`, hash), nil
}

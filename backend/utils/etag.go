package utils

import (
	"crypto/md5"
	"fmt"
	"time"
)

// GenerateETag generates an ETag based on photo ID, updated time, and size
// Used for thumbnails stored in database (not files)
func GenerateETag(photoID uint, updatedAt time.Time, size string) string {
	// Format: "photoID-timestamp-size"
	data := fmt.Sprintf("%d-%d-%s", photoID, updatedAt.Unix(), size)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf(`"%x"`, hash)
}

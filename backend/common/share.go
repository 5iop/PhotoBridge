package common

import (
	"photobridge/database"
	"photobridge/models"
)

// GetExcludedIDs extracts photo IDs from exclusions
func GetExcludedIDs(exclusions []models.PhotoExclusion) []uint {
	excludedIDs := make([]uint, len(exclusions))
	for i, e := range exclusions {
		excludedIDs[i] = e.PhotoID
	}
	return excludedIDs
}

// IsPhotoExcluded checks if a photo is excluded from a share link
// Returns true if the photo is excluded, false otherwise
func IsPhotoExcluded(linkID uint, photoID uint) bool {
	var exclusionCount int64
	database.DB.Model(&models.PhotoExclusion{}).Where("link_id = ? AND photo_id = ?", linkID, photoID).Count(&exclusionCount)
	return exclusionCount > 0
}

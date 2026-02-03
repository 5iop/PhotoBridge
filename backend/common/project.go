package common

import (
	"photobridge/database"
	"photobridge/models"
)

// CountPhotosInProject returns the number of photos in a project
func CountPhotosInProject(projectID uint) int64 {
	var count int64
	database.DB.Model(&models.Photo{}).Where("project_id = ?", projectID).Count(&count)
	return count
}

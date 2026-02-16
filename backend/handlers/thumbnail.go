package handlers

import (
	"net/http"
	"strconv"

	"photobridge/database"
	"photobridge/models"
	"photobridge/services"
	"photobridge/utils"

	"github.com/gin-gonic/gin"
)

// serveThumb is a unified handler for serving thumbnails
// size: "small" or "large"
func serveThumb(c *gin.Context, photo *models.Photo, size string) {
	if photo.NormalExt == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "raw_only", "message": "Only RAW file exists"})
		return
	}

	var thumbData []byte
	if size == "small" {
		thumbData = photo.ThumbSmall
	} else {
		thumbData = photo.ThumbLarge
	}

	if len(thumbData) == 0 {
		var project models.Project
		if err := database.DB.First(&project, photo.ProjectID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}

		if services.Queue == nil || !services.Queue.IsRunning() {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "queue_unavailable",
				"message": "Thumbnail service unavailable, please retry later",
				"queued":  false,
			})
			return
		}

		enqueued := services.Queue.Enqueue(photo, project.Name)
		if !enqueued && !services.Queue.IsProcessing(photo.ID) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "queue_busy",
				"message": "Thumbnail queue is full, please retry later",
				"queued":  false,
			})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"error":   "generating",
			"message": "Thumbnail is being generated, please retry later",
			"queued":  services.Queue.IsProcessing(photo.ID),
		})
		return
	}

	etag := utils.GenerateETag(photo.ID, photo.UpdatedAt, size)

	c.Header("ETag", etag)
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Header("Vary", "Accept")

	if clientETag := c.GetHeader("If-None-Match"); clientETag != "" && clientETag == etag {
		c.Status(http.StatusNotModified)
		return
	}

	c.Header("Content-Type", "image/jpeg")
	c.Data(http.StatusOK, "image/jpeg", thumbData)
}

// getAdminPhoto retrieves a photo for admin endpoints
func getAdminPhoto(c *gin.Context) (*models.Photo, bool) {
	photoID := c.Param("id")
	var photo models.Photo

	if err := database.DB.First(&photo, photoID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return nil, false
	}

	return &photo, true
}

// getSharePhoto retrieves a photo for share endpoints with validation
func getSharePhoto(c *gin.Context) (*models.Photo, bool) {
	token := c.Param("token")
	photoIDStr := c.Param("photoId")

	photoIDUint, err := strconv.ParseUint(photoIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid photo ID"})
		return nil, false
	}

	var link models.ShareLink
	if err := database.DB.Where("token = ?", token).First(&link).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return nil, false
	}

	var exclusionCount int64
	database.DB.Model(&models.PhotoExclusion{}).Where("link_id = ? AND photo_id = ?", link.ID, photoIDUint).Count(&exclusionCount)
	if exclusionCount > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Photo not accessible"})
		return nil, false
	}

	var photo models.Photo
	if err := database.DB.Where("id = ? AND project_id = ?", photoIDUint, link.ProjectID).First(&photo).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return nil, false
	}

	return &photo, true
}

// GetPhotoThumbSmall returns small thumbnail for list view.
func GetPhotoThumbSmall(c *gin.Context) {
	photo, ok := getAdminPhoto(c)
	if !ok {
		return
	}
	serveThumb(c, photo, "small")
}

// GetPhotoThumbLarge returns large thumbnail for preview.
func GetPhotoThumbLarge(c *gin.Context) {
	photo, ok := getAdminPhoto(c)
	if !ok {
		return
	}
	serveThumb(c, photo, "large")
}

// GetSharePhotoThumbSmall returns small thumbnail for share page.
func GetSharePhotoThumbSmall(c *gin.Context) {
	photo, ok := getSharePhoto(c)
	if !ok {
		return
	}
	serveThumb(c, photo, "small")
}

// GetSharePhotoThumbLarge returns large thumbnail for share page.
func GetSharePhotoThumbLarge(c *gin.Context) {
	photo, ok := getSharePhoto(c)
	if !ok {
		return
	}
	serveThumb(c, photo, "large")
}

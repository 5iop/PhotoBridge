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
	// 如果只有RAW没有普通图片
	if photo.NormalExt == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "raw_only", "message": "只有RAW文件"})
		return
	}

	// 获取对应大小的缩略图数据
	var thumbData []byte
	if size == "small" {
		thumbData = photo.ThumbSmall
	} else {
		thumbData = photo.ThumbLarge
	}

	// 如果没有缩略图，加入队列生成
	if len(thumbData) == 0 {
		var project models.Project
		if err := database.DB.First(&project, photo.ProjectID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}

		// 加入生成队列
		if services.Queue != nil {
			services.Queue.Enqueue(photo, project.Name)
		}

		c.JSON(http.StatusAccepted, gin.H{
			"error":   "generating",
			"message": "缩略图生成中，请稍后重试",
			"queued":  services.Queue != nil && services.Queue.IsProcessing(photo.ID),
		})
		return
	}

	// Generate ETag based on photo ID, update time, and size
	etag := utils.GenerateETag(photo.ID, photo.UpdatedAt, size)

	// Set ETag header
	c.Header("ETag", etag)
	c.Header("Cache-Control", "public, max-age=31536000")

	// Check if client has fresh cache (If-None-Match header)
	if clientETag := c.GetHeader("If-None-Match"); clientETag != "" && clientETag == etag {
		c.Status(http.StatusNotModified)
		return
	}

	// Return thumbnail image
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

	// 验证分享链接（不预加载 Exclusions，按需查询）
	var link models.ShareLink
	if err := database.DB.Where("token = ?", token).First(&link).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return nil, false
	}

	// 只检查这一张照片是否被排除（优化：直接查询而非加载所有排除项）
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

// GetPhotoThumbSmall 获取列表用小缩略图
func GetPhotoThumbSmall(c *gin.Context) {
	photo, ok := getAdminPhoto(c)
	if !ok {
		return
	}
	serveThumb(c, photo, "small")
}

// GetPhotoThumbLarge 获取预览用大缩略图
func GetPhotoThumbLarge(c *gin.Context) {
	photo, ok := getAdminPhoto(c)
	if !ok {
		return
	}
	serveThumb(c, photo, "large")
}

// GetSharePhotoThumbSmall 分享页面获取小缩略图
func GetSharePhotoThumbSmall(c *gin.Context) {
	photo, ok := getSharePhoto(c)
	if !ok {
		return
	}
	serveThumb(c, photo, "small")
}

// GetSharePhotoThumbLarge 分享页面获取大缩略图
func GetSharePhotoThumbLarge(c *gin.Context) {
	photo, ok := getSharePhoto(c)
	if !ok {
		return
	}
	serveThumb(c, photo, "large")
}

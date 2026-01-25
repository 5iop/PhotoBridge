package handlers

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sync"

	"photobridge/config"
	"photobridge/database"
	"photobridge/models"
	"photobridge/utils"

	"github.com/gin-gonic/gin"
)

// 用于防止同一张照片的缩略图同时生成（竞态条件）
var thumbGenerating sync.Map

// 异步生成缩略图（带竞态条件保护）
func generateThumbsAsync(photo *models.Photo, projectName string) {
	photoID := photo.ID

	// 检查是否已经在生成中
	if _, loaded := thumbGenerating.LoadOrStore(photoID, true); loaded {
		log.Printf("Thumbnail generation already in progress for photo %d", photoID)
		return
	}

	go func() {
		// 完成后清除标记
		defer thumbGenerating.Delete(photoID)

		if photo.NormalExt == "" {
			return // 只有RAW，不生成缩略图
		}

		imagePath := filepath.Join(config.AppConfig.UploadDir, projectName, photo.BaseName+photo.NormalExt)
		thumbResult, err := utils.GenerateThumbnails(imagePath)
		if err != nil {
			log.Printf("Async thumbnail generation failed for photo %d: %v", photoID, err)
			return
		}

		// 更新数据库
		if err := database.DB.Model(&models.Photo{}).Where("id = ?", photoID).Updates(map[string]interface{}{
			"thumb_small":  thumbResult.Small,
			"thumb_large":  thumbResult.Large,
			"thumb_width":  thumbResult.Width,
			"thumb_height": thumbResult.Height,
		}).Error; err != nil {
			log.Printf("Failed to save thumbnail for photo %d: %v", photoID, err)
			return
		}
		log.Printf("Async thumbnail generated for photo %d", photoID)
	}()
}

// GetPhotoThumbSmall 获取列表用小缩略图
func GetPhotoThumbSmall(c *gin.Context) {
	photoID := c.Param("id")
	var photo models.Photo

	if err := database.DB.First(&photo, photoID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	// 如果只有RAW没有普通图片
	if photo.NormalExt == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "raw_only", "message": "只有RAW文件"})
		return
	}

	// 如果有普通图片但没有缩略图，异步生成
	if len(photo.ThumbSmall) == 0 {
		var project models.Project
		database.DB.First(&project, photo.ProjectID)
		generateThumbsAsync(&photo, project.Name)
		c.JSON(http.StatusAccepted, gin.H{"error": "generating", "message": "正在生成缩略图"})
		return
	}

	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Data(http.StatusOK, "image/jpeg", photo.ThumbSmall)
}

// GetPhotoThumbLarge 获取预览用大缩略图
func GetPhotoThumbLarge(c *gin.Context) {
	photoID := c.Param("id")
	var photo models.Photo

	if err := database.DB.First(&photo, photoID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	// 如果只有RAW没有普通图片
	if photo.NormalExt == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "raw_only", "message": "只有RAW文件"})
		return
	}

	// 如果有普通图片但没有缩略图，异步生成
	if len(photo.ThumbLarge) == 0 {
		var project models.Project
		database.DB.First(&project, photo.ProjectID)
		generateThumbsAsync(&photo, project.Name)
		c.JSON(http.StatusAccepted, gin.H{"error": "generating", "message": "正在生成缩略图"})
		return
	}

	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Data(http.StatusOK, "image/jpeg", photo.ThumbLarge)
}

// GetSharePhotoThumbSmall 分享页面获取小缩略图
func GetSharePhotoThumbSmall(c *gin.Context) {
	token := c.Param("token")
	photoID := c.Param("photoId")

	// 验证分享链接
	var link models.ShareLink
	if err := database.DB.Preload("Exclusions").Where("token = ?", token).First(&link).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	// 检查照片是否被排除
	for _, exclusion := range link.Exclusions {
		if fmt.Sprintf("%d", exclusion.PhotoID) == photoID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Photo not accessible"})
			return
		}
	}

	var photo models.Photo
	if err := database.DB.Where("id = ? AND project_id = ?", photoID, link.ProjectID).First(&photo).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	// 如果只有RAW没有普通图片
	if photo.NormalExt == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "raw_only", "message": "只有RAW文件"})
		return
	}

	// 如果有普通图片但没有缩略图，异步生成
	if len(photo.ThumbSmall) == 0 {
		var project models.Project
		database.DB.First(&project, photo.ProjectID)
		generateThumbsAsync(&photo, project.Name)
		c.JSON(http.StatusAccepted, gin.H{"error": "generating", "message": "正在生成缩略图"})
		return
	}

	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Data(http.StatusOK, "image/jpeg", photo.ThumbSmall)
}

// GetSharePhotoThumbLarge 分享页面获取大缩略图
func GetSharePhotoThumbLarge(c *gin.Context) {
	token := c.Param("token")
	photoID := c.Param("photoId")

	// 验证分享链接
	var link models.ShareLink
	if err := database.DB.Preload("Exclusions").Where("token = ?", token).First(&link).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	// 检查照片是否被排除
	for _, exclusion := range link.Exclusions {
		if fmt.Sprintf("%d", exclusion.PhotoID) == photoID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Photo not accessible"})
			return
		}
	}

	var photo models.Photo
	if err := database.DB.Where("id = ? AND project_id = ?", photoID, link.ProjectID).First(&photo).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	// 如果只有RAW没有普通图片
	if photo.NormalExt == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "raw_only", "message": "只有RAW文件"})
		return
	}

	// 如果有普通图片但没有缩略图，异步生成
	if len(photo.ThumbLarge) == 0 {
		var project models.Project
		database.DB.First(&project, photo.ProjectID)
		generateThumbsAsync(&photo, project.Name)
		c.JSON(http.StatusAccepted, gin.H{"error": "generating", "message": "正在生成缩略图"})
		return
	}

	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Data(http.StatusOK, "image/jpeg", photo.ThumbLarge)
}

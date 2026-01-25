package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"photobridge/config"
	"photobridge/database"
	"photobridge/models"
	"photobridge/utils"

	"github.com/gin-gonic/gin"
)

func UploadPhotos(c *gin.Context) {
	projectID := c.Param("id")
	var project models.Project

	if err := database.DB.First(&project, projectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files uploaded"})
		return
	}

	// Create project upload directory
	uploadDir := filepath.Join(config.AppConfig.UploadDir, project.Name)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	var uploadedPhotos []models.Photo

	for _, file := range files {
		filename := filepath.Base(file.Filename)
		origExt := filepath.Ext(filename)
		ext := strings.ToLower(origExt)
		baseName := strings.TrimSuffix(filename, origExt)

		// Save file with lowercase extension for consistency
		newFilename := baseName + ext
		dst := filepath.Join(uploadDir, newFilename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			continue
		}

		// Check if photo with same base name exists
		var existingPhoto models.Photo
		result := database.DB.Where("project_id = ? AND base_name = ?", project.ID, baseName).First(&existingPhoto)

		if result.Error == nil {
			// Update existing photo
			if models.IsRawExtension(ext) {
				existingPhoto.RawExt = ext
				existingPhoto.HasRaw = true
			} else if models.IsImageExtension(ext) {
				existingPhoto.NormalExt = ext
				// 清除旧缩略图，浏览时会按需重新生成
				existingPhoto.ThumbSmall = nil
				existingPhoto.ThumbLarge = nil
				existingPhoto.ThumbWidth = 0
				existingPhoto.ThumbHeight = 0
			}
			database.DB.Save(&existingPhoto)
			uploadedPhotos = append(uploadedPhotos, existingPhoto)
		} else {
			// Create new photo (不生成缩略图，浏览时按需生成)
			photo := models.Photo{
				ProjectID: project.ID,
				BaseName:  baseName,
			}
			if models.IsRawExtension(ext) {
				photo.RawExt = ext
				photo.HasRaw = true
			} else if models.IsImageExtension(ext) {
				photo.NormalExt = ext
			}
			database.DB.Create(&photo)
			uploadedPhotos = append(uploadedPhotos, photo)

			// Set first photo as cover if not set
			if project.CoverPhoto == "" {
				project.CoverPhoto = baseName + ext
				database.DB.Save(&project)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Uploaded %d files", len(uploadedPhotos)),
		"photos":  uploadedPhotos,
	})
}

func UploadViaAPI(c *gin.Context) {
	projectName := c.Param("project")

	// 验证项目名称安全性（防止路径遍历攻击）
	sanitizedName, valid := utils.SanitizeProjectName(projectName)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	projectName = sanitizedName

	// Find or create project
	var project models.Project
	result := database.DB.Where("name = ?", projectName).First(&project)
	if result.Error != nil {
		project = models.Project{Name: projectName}
		database.DB.Create(&project)
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files uploaded"})
		return
	}

	// Create project upload directory
	uploadDir := filepath.Join(config.AppConfig.UploadDir, project.Name)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	var uploadedCount int

	for _, file := range files {
		filename := filepath.Base(file.Filename)
		origExt := filepath.Ext(filename)
		ext := strings.ToLower(origExt)
		baseName := strings.TrimSuffix(filename, origExt)

		// Save file with lowercase extension for consistency
		newFilename := baseName + ext
		dst := filepath.Join(uploadDir, newFilename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			continue
		}

		// Check if photo with same base name exists
		var existingPhoto models.Photo
		result := database.DB.Where("project_id = ? AND base_name = ?", project.ID, baseName).First(&existingPhoto)

		if result.Error == nil {
			if models.IsRawExtension(ext) {
				existingPhoto.RawExt = ext
				existingPhoto.HasRaw = true
			} else if models.IsImageExtension(ext) {
				existingPhoto.NormalExt = ext
				// 清除旧缩略图，浏览时会按需重新生成
				existingPhoto.ThumbSmall = nil
				existingPhoto.ThumbLarge = nil
				existingPhoto.ThumbWidth = 0
				existingPhoto.ThumbHeight = 0
			}
			database.DB.Save(&existingPhoto)
		} else {
			// Create new photo (不生成缩略图，浏览时按需生成)
			photo := models.Photo{
				ProjectID: project.ID,
				BaseName:  baseName,
			}
			if models.IsRawExtension(ext) {
				photo.RawExt = ext
				photo.HasRaw = true
			} else if models.IsImageExtension(ext) {
				photo.NormalExt = ext
			}
			database.DB.Create(&photo)

			if project.CoverPhoto == "" {
				project.CoverPhoto = baseName + ext
				database.DB.Save(&project)
			}
		}
		uploadedCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Uploaded %d files to project '%s'", uploadedCount, project.Name),
		"project": project,
	})
}

func GetProjectPhotos(c *gin.Context) {
	projectID := c.Param("id")
	var photos []models.Photo

	result := database.DB.Where("project_id = ?", projectID).Find(&photos)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, photos)
}

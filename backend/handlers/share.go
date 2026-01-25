package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"photobridge/config"
	"photobridge/database"
	"photobridge/models"
	"photobridge/utils"

	"github.com/gin-gonic/gin"
)

type ShareInfoResponse struct {
	ProjectName string `json:"project_name"`
	Description string `json:"description"`
	Alias       string `json:"alias"`
	AllowRaw    bool   `json:"allow_raw"`
	PhotoCount  int    `json:"photo_count"`
}

func GetShareInfo(c *gin.Context) {
	token := c.Param("token")
	var link models.ShareLink

	result := database.DB.Where("token = ?", token).Preload("Exclusions").First(&link)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	var project models.Project
	database.DB.First(&project, link.ProjectID)

	// Get photo count (excluding excluded photos)
	var photoCount int64
	excludedIDs := make([]uint, len(link.Exclusions))
	for i, e := range link.Exclusions {
		excludedIDs[i] = e.PhotoID
	}

	query := database.DB.Model(&models.Photo{}).Where("project_id = ?", link.ProjectID)
	if len(excludedIDs) > 0 {
		query = query.Where("id NOT IN ?", excludedIDs)
	}
	query.Count(&photoCount)

	c.JSON(http.StatusOK, ShareInfoResponse{
		ProjectName: project.Name,
		Description: project.Description,
		Alias:       link.Alias,
		AllowRaw:    link.AllowRaw,
		PhotoCount:  int(photoCount),
	})
}

func GetSharePhotos(c *gin.Context) {
	token := c.Param("token")
	var link models.ShareLink

	result := database.DB.Where("token = ?", token).Preload("Exclusions").First(&link)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	var project models.Project
	database.DB.First(&project, link.ProjectID)

	// Get photos excluding excluded ones
	excludedIDs := make([]uint, len(link.Exclusions))
	for i, e := range link.Exclusions {
		excludedIDs[i] = e.PhotoID
	}

	var photos []models.Photo
	query := database.DB.Where("project_id = ?", link.ProjectID)
	if len(excludedIDs) > 0 {
		query = query.Where("id NOT IN ?", excludedIDs)
	}
	query.Find(&photos)

	// Return photos with URLs
	type PhotoWithURL struct {
		models.Photo
		NormalURL string `json:"normal_url"`
		RawURL    string `json:"raw_url,omitempty"`
	}

	var response []PhotoWithURL
	for _, photo := range photos {
		item := PhotoWithURL{Photo: photo}
		if photo.NormalExt != "" {
			item.NormalURL = fmt.Sprintf("/uploads/%s/%s%s", project.Name, photo.BaseName, photo.NormalExt)
		}
		if photo.HasRaw && link.AllowRaw && photo.RawExt != "" {
			item.RawURL = fmt.Sprintf("/uploads/%s/%s%s", project.Name, photo.BaseName, photo.RawExt)
		}
		response = append(response, item)
	}

	c.JSON(http.StatusOK, response)
}

func GetSharePhoto(c *gin.Context) {
	token := c.Param("token")
	photoID := c.Param("photoId")
	photoType := c.DefaultQuery("type", "normal") // normal or raw

	var link models.ShareLink
	result := database.DB.Where("token = ?", token).Preload("Exclusions").First(&link)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	// Check if photo is excluded
	for _, e := range link.Exclusions {
		if fmt.Sprintf("%d", e.PhotoID) == photoID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Photo not accessible"})
			return
		}
	}

	var photo models.Photo
	if err := database.DB.First(&photo, photoID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	var project models.Project
	database.DB.First(&project, photo.ProjectID)

	var filePath string
	if photoType == "raw" {
		if !link.AllowRaw {
			c.JSON(http.StatusForbidden, gin.H{"error": "RAW download not allowed"})
			return
		}
		filePath = filepath.Join(config.AppConfig.UploadDir, project.Name, photo.BaseName+photo.RawExt)
	} else {
		filePath = filepath.Join(config.AppConfig.UploadDir, project.Name, photo.BaseName+photo.NormalExt)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.File(filePath)
}

func DownloadSharePhotos(c *gin.Context) {
	token := c.Param("token")
	downloadType := c.DefaultQuery("type", "normal") // normal, raw, or all

	var link models.ShareLink
	result := database.DB.Where("token = ?", token).Preload("Exclusions").First(&link)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	var project models.Project
	database.DB.First(&project, link.ProjectID)

	// Get photos excluding excluded ones
	excludedIDs := make([]uint, len(link.Exclusions))
	for i, e := range link.Exclusions {
		excludedIDs[i] = e.PhotoID
	}

	var photos []models.Photo
	query := database.DB.Where("project_id = ?", link.ProjectID)
	if len(excludedIDs) > 0 {
		query = query.Where("id NOT IN ?", excludedIDs)
	}
	query.Find(&photos)

	// Collect files to zip
	uploadDir := filepath.Join(config.AppConfig.UploadDir, project.Name)
	var files []string

	for _, photo := range photos {
		if downloadType == "normal" || downloadType == "all" {
			if photo.NormalExt != "" {
				filePath := filepath.Join(uploadDir, photo.BaseName+photo.NormalExt)
				if _, err := os.Stat(filePath); err == nil {
					files = append(files, filePath)
				}
			}
		}
		if (downloadType == "raw" || downloadType == "all") && link.AllowRaw {
			if photo.HasRaw && photo.RawExt != "" {
				filePath := filepath.Join(uploadDir, photo.BaseName+photo.RawExt)
				if _, err := os.Stat(filePath); err == nil {
					files = append(files, filePath)
				}
			}
		}
	}

	if len(files) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No files to download"})
		return
	}

	// Set headers for zip download
	zipName := fmt.Sprintf("%s-%s.zip", project.Name, downloadType)
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipName))

	// Stream zip
	err := utils.CreateZip(c.Writer, files, uploadDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create zip"})
		return
	}
}

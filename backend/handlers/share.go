package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"photobridge/config"
	"photobridge/database"
	"photobridge/models"
	"photobridge/utils"

	"github.com/gin-gonic/gin"
)

type ShareInfoResponse struct {
	ProjectName  string  `json:"project_name"`
	Description  string  `json:"description"`
	Alias        string  `json:"alias"`
	AllowRaw     bool    `json:"allow_raw"`
	PhotoCount   int     `json:"photo_count"`
	CDNBaseURL   string  `json:"cdn_base_url"`           // CDN base URL for China users, empty if not applicable
	Country      *string `json:"country"`                // Client's country code from CF-IPCountry header, null if not available
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

	// Get country from CF-IPCountry header
	var country *string
	if countryHeader := c.GetHeader("CF-IPCountry"); countryHeader != "" {
		country = &countryHeader
	}

	c.JSON(http.StatusOK, ShareInfoResponse{
		ProjectName: project.Name,
		Description: project.Description,
		Alias:       link.Alias,
		AllowRaw:    link.AllowRaw,
		PhotoCount:  int(photoCount),
		CDNBaseURL:  utils.GetCDNBaseURL(c),
		Country:     country,
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

	// Get CDN base URL based on client's country (CF-IPCountry header)
	cdnBase := utils.GetCDNBaseURL(c)

	// URL编码项目名称，防止特殊字符问题
	encodedProjectName := url.PathEscape(project.Name)

	var response []PhotoWithURL
	for _, photo := range photos {
		item := PhotoWithURL{Photo: photo}
		encodedBaseName := url.PathEscape(photo.BaseName)
		if photo.NormalExt != "" {
			item.NormalURL = fmt.Sprintf("%s/uploads/%s/%s%s", cdnBase, encodedProjectName, encodedBaseName, photo.NormalExt)
		}
		if photo.HasRaw && link.AllowRaw && photo.RawExt != "" {
			item.RawURL = fmt.Sprintf("%s/uploads/%s/%s%s", cdnBase, encodedProjectName, encodedBaseName, photo.RawExt)
		}
		response = append(response, item)
	}

	c.JSON(http.StatusOK, response)
}

func GetSharePhoto(c *gin.Context) {
	token := c.Param("token")
	photoIDStr := c.Param("photoId")
	photoType := c.DefaultQuery("type", "normal") // normal or raw

	photoIDUint, err := strconv.ParseUint(photoIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid photo ID"})
		return
	}

	var link models.ShareLink
	result := database.DB.Where("token = ?", token).First(&link)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	// Check if photo is excluded (optimized: direct query instead of loading all exclusions)
	var exclusionCount int64
	database.DB.Model(&models.PhotoExclusion{}).Where("link_id = ? AND photo_id = ?", link.ID, photoIDUint).Count(&exclusionCount)
	if exclusionCount > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Photo not accessible"})
		return
	}

	var photo models.Photo
	// 验证照片属于该分享链接的项目
	if err := database.DB.Where("id = ? AND project_id = ?", photoIDUint, link.ProjectID).First(&photo).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	var project models.Project
	database.DB.First(&project, photo.ProjectID)

	// 验证项目名称安全性（虽然来自数据库，但做额外验证）
	if _, valid := utils.SanitizeProjectName(project.Name); !valid {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid project configuration"})
		return
	}

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

	// Open file for ServeContent (handles ETag, If-None-Match, 304, Range requests)
	file, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file info"})
		return
	}

	// Set cache headers
	c.Header("Cache-Control", "public, max-age=31536000")

	// ServeContent automatically handles ETag, If-None-Match, 304, and Range requests
	http.ServeContent(c.Writer, c.Request, fileInfo.Name(), fileInfo.ModTime(), file)
}

// DownloadSinglePhoto - download a single photo with all its files (normal + raw) as zip
func DownloadSinglePhoto(c *gin.Context) {
	token := c.Param("token")
	photoIDStr := c.Param("photoId")

	photoIDUint, err := strconv.ParseUint(photoIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid photo ID"})
		return
	}

	var link models.ShareLink
	result := database.DB.Where("token = ?", token).First(&link)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	// Check if photo is excluded (optimized: direct query instead of loading all exclusions)
	var exclusionCount int64
	database.DB.Model(&models.PhotoExclusion{}).Where("link_id = ? AND photo_id = ?", link.ID, photoIDUint).Count(&exclusionCount)
	if exclusionCount > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Photo not accessible"})
		return
	}

	var photo models.Photo
	if err := database.DB.Where("id = ? AND project_id = ?", photoIDUint, link.ProjectID).First(&photo).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	var project models.Project
	database.DB.First(&project, photo.ProjectID)

	uploadDir := filepath.Join(config.AppConfig.UploadDir, project.Name)
	var files []string

	// Add normal photo
	if photo.NormalExt != "" {
		filePath := filepath.Join(uploadDir, photo.BaseName+photo.NormalExt)
		if _, err := os.Stat(filePath); err == nil {
			files = append(files, filePath)
		}
	}

	// Add RAW if allowed
	if photo.HasRaw && photo.RawExt != "" && link.AllowRaw {
		filePath := filepath.Join(uploadDir, photo.BaseName+photo.RawExt)
		if _, err := os.Stat(filePath); err == nil {
			files = append(files, filePath)
		}
	}

	if len(files) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No files to download"})
		return
	}

	// If only one file, send directly without zip
	if len(files) == 1 {
		// Open file for ServeContent (handles ETag, If-None-Match, 304, Range requests)
		file, err := os.Open(files[0])
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file info"})
			return
		}

		// Set cache headers
		c.Header("Cache-Control", "public, max-age=31536000")

		// ServeContent automatically handles ETag, If-None-Match, 304, and Range requests
		http.ServeContent(c.Writer, c.Request, fileInfo.Name(), fileInfo.ModTime(), file)
		return
	}

	// Multiple files - create zip
	zipName := fmt.Sprintf("%s.zip", photo.BaseName)
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipName))

	if err := utils.CreateZip(c.Writer, files, uploadDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create zip"})
		return
	}
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

package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"photobridge/common"
	"photobridge/config"
	"photobridge/database"
	"photobridge/models"
	"photobridge/utils"

	"github.com/gin-gonic/gin"
)

type ShareInfoResponse struct {
	ProjectName string  `json:"project_name"`
	Description string  `json:"description"`
	Alias       string  `json:"alias"`
	AllowRaw    bool    `json:"allow_raw"`
	PhotoCount  int     `json:"photo_count"`
	CDNBaseURL  string  `json:"cdn_base_url"` // CDN base URL for China users, empty if not applicable
	Country     *string `json:"country"`      // Client's country code from CF-IPCountry header, null if not available
}

func GetShareInfo(c *gin.Context) {
	token := c.Param("token")
	var link models.ShareLink

	result := database.DB.Where("token = ?", token).Preload("Exclusions").Preload("Project").First(&link)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	project := link.Project
	// Check if project exists (Preload doesn't fail if foreign key references non-existent record)
	if project.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Get photo count (excluding excluded photos)
	var photoCount int64
	excludedIDs := common.GetExcludedIDs(link.Exclusions)

	query := database.DB.Model(&models.Photo{}).Where("project_id = ?", link.ProjectID)
	if len(excludedIDs) > 0 {
		query = query.Where("id NOT IN ?", excludedIDs)
	}
	query.Count(&photoCount)

	// Get country from CF-IPCountry header
	var country *string
	// In development environment (non-Docker), return "DEV" as country
	if os.Getenv("ENV") != "production" && os.Getenv("DOCKER") != "true" {
		devCountry := "DEV"
		country = &devCountry
	} else if countryHeader := c.GetHeader("CF-IPCountry"); countryHeader != "" {
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

	result := database.DB.Where("token = ?", token).Preload("Exclusions").Preload("Project").First(&link)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	project := link.Project
	// Check if project exists (Preload doesn't fail if foreign key references non-existent record)
	if project.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Get photos excluding excluded ones
	excludedIDs := common.GetExcludedIDs(link.Exclusions)

	var photos []models.Photo
	query := database.DB.Select(photoMetaColumns).Where("project_id = ?", link.ProjectID)
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
	result := database.DB.Where("token = ?", token).Preload("Project").First(&link)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	project := link.Project
	// Check if project exists (Preload doesn't fail if foreign key references non-existent record)
	if project.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Check if photo is excluded (optimized: direct query instead of loading all exclusions)
	if common.IsPhotoExcluded(link.ID, uint(photoIDUint)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Photo not accessible"})
		return
	}

	var photo models.Photo
	// 验证照片属于该分享链接的项目
	if err := database.DB.Select("id, project_id, base_name, normal_ext, raw_ext, has_raw").
		Where("id = ? AND project_id = ?", photoIDUint, link.ProjectID).First(&photo).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	// 验证项目名称安全性（虽然来自数据库，但做额外验证）
	if !utils.ValidatePathComponent(project.Name) {
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

	// Validate file path is secure before opening
	safeFilePath, err := utils.ValidateSecurePath(config.AppConfig.UploadDir, filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid file path"})
		return
	}

	// Open file for ServeContent (handles ETag, If-None-Match, 304, Range requests)
	file, err := os.Open(safeFilePath)
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
	result := database.DB.Where("token = ?", token).Preload("Project").First(&link)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	project := link.Project
	// Check if project exists (Preload doesn't fail if foreign key references non-existent record)
	if project.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Check if photo is excluded (optimized: direct query instead of loading all exclusions)
	if common.IsPhotoExcluded(link.ID, uint(photoIDUint)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Photo not accessible"})
		return
	}

	var photo models.Photo
	if err := database.DB.Select("id, project_id, base_name, normal_ext, raw_ext, has_raw").
		Where("id = ? AND project_id = ?", photoIDUint, link.ProjectID).First(&photo).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	// Validate project name to prevent directory traversal
	if !utils.ValidatePathComponent(project.Name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}

	uploadDir := filepath.Join(config.AppConfig.UploadDir, project.Name)

	// Validate upload directory path is secure
	safeUploadDir, err := utils.ValidateSecurePath(config.AppConfig.UploadDir, uploadDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid directory path"})
		return
	}

	var files []string

	// Add normal photo
	if photo.NormalExt != "" {
		filePath := filepath.Join(safeUploadDir, photo.BaseName+photo.NormalExt)
		if _, err := os.Stat(filePath); err == nil {
			files = append(files, filePath)
		}
	}

	// Add RAW if allowed
	if photo.HasRaw && photo.RawExt != "" && link.AllowRaw {
		filePath := filepath.Join(safeUploadDir, photo.BaseName+photo.RawExt)
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

	// Note: HTTP headers are already sent at this point. If CreateZip fails,
	// the client will receive an incomplete/malformed zip file.
	// This is acceptable as pre-validating all files would be expensive.
	if err := utils.CreateZip(c.Writer, files, safeUploadDir); err != nil {
		// Cannot send error response - headers already sent
		return
	}
}

func DownloadSharePhotos(c *gin.Context) {
	token := c.Param("token")
	downloadType := c.DefaultQuery("type", "normal") // normal, raw, or all

	var link models.ShareLink
	result := database.DB.Where("token = ?", token).Preload("Exclusions").Preload("Project").First(&link)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	project := link.Project
	// Check if project exists (Preload doesn't fail if foreign key references non-existent record)
	if project.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Validate project name to prevent directory traversal
	if !utils.ValidatePathComponent(project.Name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}

	// Get photos excluding excluded ones
	excludedIDs := common.GetExcludedIDs(link.Exclusions)

	var photos []models.Photo
	query := database.DB.Select("base_name, normal_ext, raw_ext, has_raw").Where("project_id = ?", link.ProjectID)
	if len(excludedIDs) > 0 {
		query = query.Where("id NOT IN ?", excludedIDs)
	}
	query.Find(&photos)

	// Collect files to zip
	uploadDir := filepath.Join(config.AppConfig.UploadDir, project.Name)

	// Validate upload directory path is secure
	safeUploadDir, err := utils.ValidateSecurePath(config.AppConfig.UploadDir, uploadDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid directory path"})
		return
	}

	var files []string

	for _, photo := range photos {
		if downloadType == "normal" || downloadType == "all" {
			if photo.NormalExt != "" {
				filePath := filepath.Join(safeUploadDir, photo.BaseName+photo.NormalExt)
				if _, err := os.Stat(filePath); err == nil {
					files = append(files, filePath)
				}
			}
		}
		if (downloadType == "raw" || downloadType == "all") && link.AllowRaw {
			if photo.HasRaw && photo.RawExt != "" {
				filePath := filepath.Join(safeUploadDir, photo.BaseName+photo.RawExt)
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

	// Note: HTTP headers are already sent at this point. If CreateZip fails,
	// the client will receive an incomplete/malformed zip file.
	// This is acceptable as pre-validating all files would be expensive.
	// Stream zip
	err = utils.CreateZip(c.Writer, files, safeUploadDir)
	if err != nil {
		// Cannot send error response - headers already sent
		return
	}
}

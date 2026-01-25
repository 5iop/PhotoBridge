package handlers

import (
	"fmt"
	"mime/multipart"
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

// processUploadedFile handles the common logic for processing an uploaded file
// Returns the photo model and any error
func processUploadedFile(c *gin.Context, file *multipart.FileHeader, project *models.Project, uploadDir string) (*models.Photo, error) {
	filename := filepath.Base(file.Filename)
	origExt := filepath.Ext(filename)
	ext := strings.ToLower(origExt)
	baseName := strings.TrimSuffix(filename, origExt)

	// Save file with lowercase extension for consistency
	newFilename := baseName + ext
	dst := filepath.Join(uploadDir, newFilename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		return nil, err
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
		return &existingPhoto, nil
	}

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

	// Set first photo as cover if not set
	if project.CoverPhoto == "" {
		project.CoverPhoto = baseName + ext
		database.DB.Save(project)
	}

	return &photo, nil
}

// prepareUpload validates and prepares for file upload
// Returns files, uploadDir, and any error
func prepareUpload(c *gin.Context, project *models.Project) ([]*multipart.FileHeader, string, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse form")
	}

	files := form.File["files"]
	if len(files) == 0 {
		return nil, "", fmt.Errorf("no files uploaded")
	}

	// Create project upload directory
	uploadDir := filepath.Join(config.AppConfig.UploadDir, project.Name)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create upload directory")
	}

	return files, uploadDir, nil
}

func UploadPhotos(c *gin.Context) {
	projectID := c.Param("id")
	var project models.Project

	if err := database.DB.First(&project, projectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	files, uploadDir, err := prepareUpload(c, &project)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var uploadedPhotos []models.Photo
	var failedFiles []string

	for _, file := range files {
		photo, err := processUploadedFile(c, file, &project, uploadDir)
		if err != nil {
			failedFiles = append(failedFiles, filepath.Base(file.Filename))
			continue
		}
		uploadedPhotos = append(uploadedPhotos, *photo)
	}

	response := gin.H{
		"message": fmt.Sprintf("Uploaded %d files", len(uploadedPhotos)),
		"photos":  uploadedPhotos,
	}
	if len(failedFiles) > 0 {
		response["failed"] = failedFiles
		response["message"] = fmt.Sprintf("Uploaded %d files, %d failed", len(uploadedPhotos), len(failedFiles))
	}
	c.JSON(http.StatusOK, response)
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

	files, uploadDir, err := prepareUpload(c, &project)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var uploadedCount int
	var failedFiles []string

	for _, file := range files {
		_, err := processUploadedFile(c, file, &project, uploadDir)
		if err != nil {
			failedFiles = append(failedFiles, filepath.Base(file.Filename))
			continue
		}
		uploadedCount++
	}

	response := gin.H{
		"message": fmt.Sprintf("Uploaded %d files to project '%s'", uploadedCount, project.Name),
		"project": project,
	}
	if len(failedFiles) > 0 {
		response["failed"] = failedFiles
		response["message"] = fmt.Sprintf("Uploaded %d files to project '%s', %d failed", uploadedCount, project.Name, len(failedFiles))
	}
	c.JSON(http.StatusOK, response)
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

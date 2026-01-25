package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"photobridge/config"
	"photobridge/database"
	"photobridge/middleware"
	"photobridge/models"
	"photobridge/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// generateShortToken generates a short URL-safe token (8 characters)
func generateShortToken() string {
	b := make([]byte, 6)
	rand.Read(b)
	token := base64.URLEncoding.EncodeToString(b)
	token = strings.TrimRight(token, "=")
	return token
}

// generateUniqueToken generates a unique share token with retry mechanism
func generateUniqueToken() (string, error) {
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		token := generateShortToken()
		// Check if token already exists
		var count int64
		database.DB.Model(&models.ShareLink{}).Where("token = ?", token).Count(&count)
		if count == 0 {
			return token, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique token after %d attempts", maxRetries)
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Username != config.AppConfig.AdminUsername || req.Password != config.AppConfig.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	claims := &middleware.Claims{
		Username: req.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.AppConfig.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{Token: tokenString})
}

// Project handlers
func GetProjects(c *gin.Context) {
	var projects []models.Project
	result := database.DB.Find(&projects)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// Add photo count to response (使用Count而不是Preload，避免加载所有Photo数据)
	type ProjectWithCount struct {
		models.Project
		PhotoCount int64 `json:"photo_count"`
	}

	var response []ProjectWithCount
	for _, p := range projects {
		var count int64
		database.DB.Model(&models.Photo{}).Where("project_id = ?", p.ID).Count(&count)
		response = append(response, ProjectWithCount{
			Project:    p,
			PhotoCount: count,
		})
	}

	c.JSON(http.StatusOK, response)
}

func CreateProject(c *gin.Context) {
	var req models.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project := models.Project{
		Name:        req.Name,
		Description: req.Description,
	}

	result := database.DB.Create(&project)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusCreated, project)
}

func GetProject(c *gin.Context) {
	id := c.Param("id")
	var project models.Project

	result := database.DB.Preload("Photos").Preload("ShareLinks").First(&project, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	c.JSON(http.StatusOK, project)
}

func UpdateProject(c *gin.Context) {
	id := c.Param("id")
	var project models.Project

	if err := database.DB.First(&project, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	var req models.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		// 验证项目名称安全性
		if _, valid := utils.SanitizeProjectName(req.Name); !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
			return
		}
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.CoverPhoto != "" {
		updates["cover_photo"] = req.CoverPhoto
	}

	if err := database.DB.Model(&project).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	// 重新加载更新后的项目
	database.DB.First(&project, id)
	c.JSON(http.StatusOK, project)
}

func DeleteProject(c *gin.Context) {
	id := c.Param("id")
	var project models.Project

	if err := database.DB.First(&project, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// 获取所有关联的分享链接，以便删除其排除规则
	var linkIDs []uint
	database.DB.Model(&models.ShareLink{}).Where("project_id = ?", id).Pluck("id", &linkIDs)
	if len(linkIDs) > 0 {
		database.DB.Where("link_id IN ?", linkIDs).Delete(&models.PhotoExclusion{})
	}

	// Delete associated photos and links
	database.DB.Where("project_id = ?", id).Delete(&models.Photo{})
	database.DB.Where("project_id = ?", id).Delete(&models.ShareLink{})
	database.DB.Delete(&project)

	// 删除项目的物理文件目录
	uploadDir := filepath.Join(config.AppConfig.UploadDir, project.Name)
	if err := os.RemoveAll(uploadDir); err != nil {
		// 日志记录但不影响响应，因为数据库已清理
		// 可以考虑在生产环境中添加日志
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project deleted"})
}

// Share link handlers
func GetShareLinks(c *gin.Context) {
	projectID := c.Param("id")
	var links []models.ShareLink

	result := database.DB.Where("project_id = ?", projectID).Preload("Exclusions").Find(&links)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, links)
}

func CreateShareLink(c *gin.Context) {
	projectID := c.Param("id")
	var project models.Project

	if err := database.DB.First(&project, projectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	var req models.CreateShareLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := generateUniqueToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate unique token"})
		return
	}

	link := models.ShareLink{
		ProjectID: project.ID,
		Token:     token,
		Alias:     req.Alias,
		AllowRaw:  req.AllowRaw,
	}

	result := database.DB.Create(&link)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// Add exclusions
	for _, photoID := range req.Exclusions {
		exclusion := models.PhotoExclusion{
			LinkID:  link.ID,
			PhotoID: photoID,
		}
		database.DB.Create(&exclusion)
	}

	database.DB.Preload("Exclusions").First(&link, link.ID)
	c.JSON(http.StatusCreated, link)
}

func UpdateShareLink(c *gin.Context) {
	linkID := c.Param("id")
	var link models.ShareLink

	if err := database.DB.First(&link, linkID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	var req models.UpdateShareLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Alias != "" {
		updates["alias"] = req.Alias
	}
	if req.AllowRaw != nil {
		updates["allow_raw"] = *req.AllowRaw
	}

	database.DB.Model(&link).Updates(updates)

	// Update exclusions
	if req.Exclusions != nil {
		database.DB.Where("link_id = ?", link.ID).Delete(&models.PhotoExclusion{})
		for _, photoID := range req.Exclusions {
			exclusion := models.PhotoExclusion{
				LinkID:  link.ID,
				PhotoID: photoID,
			}
			database.DB.Create(&exclusion)
		}
	}

	database.DB.Preload("Exclusions").First(&link, link.ID)
	c.JSON(http.StatusOK, link)
}

func DeleteShareLink(c *gin.Context) {
	linkID := c.Param("id")
	var link models.ShareLink

	if err := database.DB.First(&link, linkID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	database.DB.Where("link_id = ?", link.ID).Delete(&models.PhotoExclusion{})
	database.DB.Delete(&link)

	c.JSON(http.StatusOK, gin.H{"message": "Share link deleted"})
}

func DeletePhoto(c *gin.Context) {
	photoID := c.Param("id")
	var photo models.Photo

	if err := database.DB.First(&photo, photoID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	// Delete exclusions
	if err := database.DB.Where("photo_id = ?", photo.ID).Delete(&models.PhotoExclusion{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete photo exclusions"})
		return
	}

	if err := database.DB.Delete(&photo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete photo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Photo deleted"})
}

// GetPhotoFiles returns the list of files for a photo
func GetPhotoFiles(c *gin.Context) {
	photoID := c.Param("id")
	var photo models.Photo

	if err := database.DB.First(&photo, photoID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	var project models.Project
	database.DB.First(&project, photo.ProjectID)

	type FileInfo struct {
		Type     string `json:"type"`
		Filename string `json:"filename"`
		URL      string `json:"url"`
		Ext      string `json:"ext"`
	}

	var files []FileInfo

	// URL编码项目名称和文件名，防止特殊字符问题
	encodedProjectName := url.PathEscape(project.Name)
	encodedBaseName := url.PathEscape(photo.BaseName)

	if photo.NormalExt != "" {
		files = append(files, FileInfo{
			Type:     "normal",
			Filename: photo.BaseName + photo.NormalExt,
			URL:      "/uploads/" + encodedProjectName + "/" + encodedBaseName + photo.NormalExt,
			Ext:      photo.NormalExt,
		})
	}

	if photo.HasRaw && photo.RawExt != "" {
		files = append(files, FileInfo{
			Type:     "raw",
			Filename: photo.BaseName + photo.RawExt,
			URL:      "/uploads/" + encodedProjectName + "/" + encodedBaseName + photo.RawExt,
			Ext:      photo.RawExt,
		})
	}

	c.JSON(http.StatusOK, files)
}

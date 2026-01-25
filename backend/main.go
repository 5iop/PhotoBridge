package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"photobridge/config"
	"photobridge/database"
	"photobridge/handlers"
	"photobridge/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	config.Load()

	// Initialize database
	database.Init()

	// Create Gin router
	r := gin.Default()

	// Configure CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
	}))

	// Serve uploaded files
	r.Static("/uploads", config.AppConfig.UploadDir)

	// API routes
	api := r.Group("/api")
	{
		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Public auth
		api.POST("/admin/login", handlers.Login)

		// Admin routes (require JWT)
		admin := api.Group("/admin")
		admin.Use(middleware.JWTAuth())
		{
			// Projects
			admin.GET("/projects", handlers.GetProjects)
			admin.POST("/projects", handlers.CreateProject)
			admin.GET("/projects/:id", handlers.GetProject)
			admin.PUT("/projects/:id", handlers.UpdateProject)
			admin.DELETE("/projects/:id", handlers.DeleteProject)

			// Photos
			admin.POST("/projects/:id/photos", handlers.UploadPhotos)
			admin.GET("/projects/:id/photos", handlers.GetProjectPhotos)
			admin.DELETE("/photos/:id", handlers.DeletePhoto)

			// Share links
			admin.GET("/projects/:id/links", handlers.GetShareLinks)
			admin.POST("/projects/:id/links", handlers.CreateShareLink)
			admin.PUT("/links/:id", handlers.UpdateShareLink)
			admin.DELETE("/links/:id", handlers.DeleteShareLink)
		}

		// API upload (require API Key)
		upload := api.Group("/upload")
		upload.Use(middleware.APIKeyAuth())
		{
			upload.POST("/:project", handlers.UploadViaAPI)
		}

		// Share routes (public)
		share := api.Group("/share")
		{
			share.GET("/:token", handlers.GetShareInfo)
			share.GET("/:token/photos", handlers.GetSharePhotos)
			share.GET("/:token/photo/:photoId", handlers.GetSharePhoto)
			share.GET("/:token/download", handlers.DownloadSharePhotos)
		}
	}

	// Serve frontend static files (for production)
	frontendDir := "./frontend/dist"
	if _, err := os.Stat(frontendDir); err == nil {
		r.Static("/assets", filepath.Join(frontendDir, "assets"))
		r.StaticFile("/vite.svg", filepath.Join(frontendDir, "vite.svg"))

		// Serve index.html for all non-API routes (SPA support)
		r.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(frontendDir, "index.html"))
		})
	}

	// Start server
	log.Printf("Server starting on port %s", config.AppConfig.Port)
	if err := r.Run(":" + config.AppConfig.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

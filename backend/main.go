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
	"photobridge/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const shortname = "[PhotoBridge]"

func main() {
	log.Printf("%s Starting PhotoBridge", shortname)

	// Load configuration
	config.Load()

	// Initialize database
	database.Init()

	// Initialize thumbnail generation queue
	// Workers: 2 (concurrent thumbnail generators)
	// Queue is unbounded - tasks only store file paths, not image data
	services.InitQueue(2)

	// Create Gin router with custom middleware
	r := gin.New()
	r.Use(gin.Recovery())            // Recover from panics
	r.Use(middleware.Logger())       // Custom logger with real IP and health check filtering

	// Set max memory for multipart forms to 8MB
	// Files larger than this will be stored in temp files on disk
	// This prevents large uploads from consuming too much RAM
	r.MaxMultipartMemory = 8 << 20 // 8 MB

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

		// Swagger UI and OpenAPI spec
		api.GET("/docs", func(c *gin.Context) {
			c.File("./docs/swagger.html")
		})
		api.GET("/docs/openapi.yaml", func(c *gin.Context) {
			c.File("./docs/openapi.yaml")
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
			admin.POST("/projects/:id/photos/check-hashes", handlers.CheckHashes)
			admin.DELETE("/photos/:id", handlers.DeletePhoto)
			admin.GET("/photos/:id/exif", handlers.GetAdminPhotoExif)
			admin.GET("/photos/:id/files", handlers.GetPhotoFiles)
			admin.GET("/photos/:id/thumb/small", handlers.GetPhotoThumbSmall)
			admin.GET("/photos/:id/thumb/large", handlers.GetPhotoThumbLarge)

			// Share links
			admin.GET("/projects/:id/links", handlers.GetShareLinks)
			admin.POST("/projects/:id/links", handlers.CreateShareLink)
			admin.PUT("/links/:id", handlers.UpdateShareLink)
			admin.DELETE("/links/:id", handlers.DeleteShareLink)
		}

		// API routes (require API Key)
		apiKey := api.Group("")
		apiKey.Use(middleware.APIKeyAuth())
		{
			// Upload
			apiKey.POST("/upload/:project", handlers.UploadViaAPI)
			// Projects
			apiKey.GET("/projects", handlers.GetProjectsViaAPI)
			apiKey.POST("/projects", handlers.CreateProjectViaAPI)
			apiKey.DELETE("/projects/:project", handlers.DeleteProjectViaAPI)
			apiKey.GET("/projects/:project/photos", handlers.GetProjectPhotosViaAPI)
		}

		// Share routes (public)
		share := api.Group("/share")
		{
			share.GET("/:token", handlers.GetShareInfo)
			share.GET("/:token/photos", handlers.GetSharePhotos)
			share.GET("/:token/photo/:photoId", handlers.GetSharePhoto)
			share.GET("/:token/photo/:photoId/exif", handlers.GetPhotoExif)
			share.GET("/:token/photo/:photoId/download", handlers.DownloadSinglePhoto)
			share.GET("/:token/photo/:photoId/thumb/small", handlers.GetSharePhotoThumbSmall)
			share.GET("/:token/photo/:photoId/thumb/large", handlers.GetSharePhotoThumbLarge)
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
	log.Printf("%s Server starting on port %s", shortname, config.AppConfig.Port)
	log.Printf("%s Access the application at http://localhost:%s", shortname, config.AppConfig.Port)
	if err := r.Run(":" + config.AppConfig.Port); err != nil {
		log.Fatalf("%s Failed to start server: %v", shortname, err)
	}
}

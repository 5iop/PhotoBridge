package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

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
	// Set log format to include date, time, and short file name (file.go:line)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Printf("%s Starting PhotoBridge", shortname)

	// Load configuration
	config.Load()

	// Initialize database
	database.Init()

	// Initialize thumbnail generation queue
	// Workers and timeout are configurable via environment variables.
	// Queue is unbounded - tasks only store file paths, not image data
	services.InitQueue(
		config.AppConfig.ThumbWorkers,
		time.Duration(config.AppConfig.ThumbJobTimeoutSec)*time.Second,
	)

	// Create Gin router with custom middleware
	r := gin.New()
	r.Use(gin.Recovery())      // Recover from panics
	r.Use(middleware.Logger()) // Custom logger with real IP and health check filtering

	// Set max memory for multipart forms to 8MB
	// Files larger than this will be stored in temp files on disk
	// This prevents large uploads from consuming too much RAM
	r.MaxMultipartMemory = 8 << 20 // 8 MB

	// Configure CORS
	// In production (Docker), restrict CORS to prevent unauthorized access
	// In development, allow all origins for convenience
	var corsConfig cors.Config
	if os.Getenv("ENV") == "production" || os.Getenv("DOCKER") == "true" {
		// Production: Use specific origins if provided, otherwise allow all requests
		if allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); allowedOrigins != "" {
			corsConfig = cors.Config{
				AllowOrigins:     []string{allowedOrigins},
				AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key"},
				ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
				AllowCredentials: true,
			}
			log.Printf("%s CORS restricted to: %v", shortname, []string{allowedOrigins})
		} else {
			// Fallback: Allow any origin (frontend and backend are typically on same domain)
			corsConfig = cors.Config{
				AllowOriginFunc: func(origin string) bool {
					return true // Allow all origins
				},
				AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key"},
				ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
				AllowCredentials: true,
			}
			log.Printf("%s CORS allowing all origins (no CORS_ALLOWED_ORIGINS set)", shortname)
		}
	} else {
		// Development: Allow all origins
		corsConfig = cors.Config{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key"},
			ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
			AllowCredentials: true,
		}
		log.Printf("%s CORS allowing all origins (development mode)", shortname)
	}

	r.Use(cors.New(corsConfig))

	// Serve uploaded files
	r.Static("/uploads", config.AppConfig.UploadDir)

	// Serve frontend static files (must be before wildcard routes)
	frontendDir := "./frontend/dist"
	if _, err := os.Stat(frontendDir); err == nil {
		r.Static("/assets", filepath.Join(frontendDir, "assets"))
		r.StaticFile("/vite.svg", filepath.Join(frontendDir, "vite.svg"))
	}

	// Robots.txt - Block all crawlers
	r.GET("/robots.txt", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, "User-agent: *\nDisallow: /\n")
	})

	// API routes
	api := r.Group("/api")
	{
		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Turnstile verification endpoint (public)
		api.POST("/verify", middleware.VerifyTurnstileHandler)

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

		// Share routes (public, with Turnstile verification)
		// API routes: /api/share/:token for programmatic access
		// Frontend uses /s/:token for short URLs (handled by SPA router)
		share := api.Group("/share")
		share.Use(middleware.RequireTurnstile()) // Require verification for first-time visitors
		{
			// Password verification endpoint (does not require password middleware)
			share.POST("/:token/verify-password", middleware.VerifySharePasswordHandler)

			// Protected routes (require password if enabled)
			shareProtected := share.Group("")
			shareProtected.Use(middleware.RequireSharePassword())
			{
				shareProtected.GET("/:token", handlers.GetShareInfo)
				shareProtected.GET("/:token/photos", handlers.GetSharePhotos)
				shareProtected.GET("/:token/photo/:photoId", handlers.GetSharePhoto)
				shareProtected.GET("/:token/photo/:photoId/exif", handlers.GetPhotoExif)
				shareProtected.GET("/:token/photo/:photoId/download", handlers.DownloadSinglePhoto)
				shareProtected.GET("/:token/photo/:photoId/thumb/small", handlers.GetSharePhotoThumbSmall)
				shareProtected.GET("/:token/photo/:photoId/thumb/large", handlers.GetSharePhotoThumbLarge)
				shareProtected.GET("/:token/download", handlers.DownloadSharePhotos)
			}
		}
	}

	// Serve index.html for all non-API routes (SPA support)
	if _, err := os.Stat(frontendDir); err == nil {
		r.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(frontendDir, "index.html"))
		})
	}

	// Start server
	log.Printf("%s Server starting on 0.0.0.0:%s (all interfaces)", shortname, config.AppConfig.Port)
	log.Printf("%s Local access: http://localhost:%s", shortname, config.AppConfig.Port)
	log.Printf("%s Network access: http://<your-ip>:%s", shortname, config.AppConfig.Port)
	if err := r.Run(":" + config.AppConfig.Port); err != nil {
		log.Fatalf("%s Failed to start server: %v", shortname, err)
	}
}

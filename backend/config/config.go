package config

import (
	"log"
	"os"
)

type Config struct {
	AdminUsername string
	AdminPassword string
	APIKey        string
	JWTSecret     string
	Port          string
	UploadDir     string
	DatabasePath  string
	CNCDNURL      string // China CDN URL (e.g., https://cdn.pb.jangit.me)
}

var AppConfig *Config

const shortname = "[Config]"

func Load() {
	log.Printf("%s Loading configuration", shortname)
	AppConfig = &Config{
		AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		APIKey:        getEnv("API_KEY", "photobridge-api-key"),
		JWTSecret:     getEnv("JWT_SECRET", "photobridge-jwt-secret"),
		Port:          getEnv("PORT", "8060"),
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
		DatabasePath:  getEnv("DATABASE_PATH", "./data/photobridge.db"),
		CNCDNURL:      getEnv("CNCDN_URL", ""), // Optional China CDN URL
	}
	log.Printf("%s Configuration loaded - Port: %s, UploadDir: %s, DatabasePath: %s",
		shortname, AppConfig.Port, AppConfig.UploadDir, AppConfig.DatabasePath)

	// Ensure upload directory exists
	log.Printf("%s Creating upload directory: %s", shortname, AppConfig.UploadDir)
	if err := os.MkdirAll(AppConfig.UploadDir, 0755); err != nil {
		log.Fatalf("%s Failed to create upload directory %s: %v", shortname, AppConfig.UploadDir, err)
	}
	log.Printf("%s Upload directory created/verified: %s", shortname, AppConfig.UploadDir)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

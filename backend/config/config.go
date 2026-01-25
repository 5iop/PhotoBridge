package config

import (
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
}

var AppConfig *Config

func Load() {
	AppConfig = &Config{
		AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		APIKey:        getEnv("API_KEY", "photobridge-api-key"),
		JWTSecret:     getEnv("JWT_SECRET", "photobridge-jwt-secret"),
		Port:          getEnv("PORT", "8080"),
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
		DatabasePath:  getEnv("DATABASE_PATH", "./data/photobridge.db"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue string
		expected     string
	}{
		{"env set", "TEST_CONFIG_VAR", "custom_value", "default", "custom_value"},
		{"env not set", "TEST_CONFIG_UNSET_VAR", "", "default", "default"},
		{"empty default", "TEST_CONFIG_VAR2", "value", "", "value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnv(%q, %q) = %q, expected %q", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestLoadDefaults(t *testing.T) {
	// Clear any existing env vars that might interfere
	envVars := []string{
		"ADMIN_USERNAME", "ADMIN_PASSWORD", "API_KEY",
		"JWT_SECRET", "PORT", "UPLOAD_DIR", "DATABASE_PATH",
	}
	originalValues := make(map[string]string)
	for _, key := range envVars {
		originalValues[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	defer func() {
		for key, val := range originalValues {
			if val != "" {
				os.Setenv(key, val)
			}
		}
	}()

	// Create temp directory for upload dir
	tempDir, err := os.MkdirTemp("", "configtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set upload dir to temp to avoid creating directories
	os.Setenv("UPLOAD_DIR", filepath.Join(tempDir, "uploads"))
	defer os.Unsetenv("UPLOAD_DIR")

	Load()

	if AppConfig == nil {
		t.Fatal("AppConfig should not be nil after Load()")
	}

	// Check defaults
	if AppConfig.AdminUsername != "admin" {
		t.Errorf("Default AdminUsername should be 'admin', got %q", AppConfig.AdminUsername)
	}
	if AppConfig.AdminPassword != "admin123" {
		t.Errorf("Default AdminPassword should be 'admin123', got %q", AppConfig.AdminPassword)
	}
	if AppConfig.APIKey != "photobridge-api-key" {
		t.Errorf("Default APIKey should be 'photobridge-api-key', got %q", AppConfig.APIKey)
	}
	if AppConfig.JWTSecret != "photobridge-jwt-secret" {
		t.Errorf("Default JWTSecret should be 'photobridge-jwt-secret', got %q", AppConfig.JWTSecret)
	}
	if AppConfig.Port != "8060" {
		t.Errorf("Default Port should be '8060', got %q", AppConfig.Port)
	}
	if AppConfig.DatabasePath != "./data/photobridge.db" {
		t.Errorf("Default DatabasePath should be './data/photobridge.db', got %q", AppConfig.DatabasePath)
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "configtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	uploadDir := filepath.Join(tempDir, "custom_uploads")

	// Set custom env values
	testEnv := map[string]string{
		"ADMIN_USERNAME": "custom_admin",
		"ADMIN_PASSWORD": "custom_password",
		"API_KEY":        "custom-api-key",
		"JWT_SECRET":     "custom-jwt-secret",
		"PORT":           "9090",
		"UPLOAD_DIR":     uploadDir,
		"DATABASE_PATH":  filepath.Join(tempDir, "custom.db"),
	}

	// Save and set env vars
	originalValues := make(map[string]string)
	for key, val := range testEnv {
		originalValues[key] = os.Getenv(key)
		os.Setenv(key, val)
	}
	defer func() {
		for key, val := range originalValues {
			if val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	Load()

	if AppConfig.AdminUsername != "custom_admin" {
		t.Errorf("AdminUsername should be 'custom_admin', got %q", AppConfig.AdminUsername)
	}
	if AppConfig.AdminPassword != "custom_password" {
		t.Errorf("AdminPassword should be 'custom_password', got %q", AppConfig.AdminPassword)
	}
	if AppConfig.APIKey != "custom-api-key" {
		t.Errorf("APIKey should be 'custom-api-key', got %q", AppConfig.APIKey)
	}
	if AppConfig.JWTSecret != "custom-jwt-secret" {
		t.Errorf("JWTSecret should be 'custom-jwt-secret', got %q", AppConfig.JWTSecret)
	}
	if AppConfig.Port != "9090" {
		t.Errorf("Port should be '9090', got %q", AppConfig.Port)
	}
	if AppConfig.UploadDir != uploadDir {
		t.Errorf("UploadDir should be %q, got %q", uploadDir, AppConfig.UploadDir)
	}
}

func TestLoadCreatesUploadDir(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "configtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set upload dir to a non-existent path
	uploadDir := filepath.Join(tempDir, "new_uploads")
	os.Setenv("UPLOAD_DIR", uploadDir)
	defer os.Unsetenv("UPLOAD_DIR")

	// Verify it doesn't exist yet
	if _, err := os.Stat(uploadDir); !os.IsNotExist(err) {
		t.Fatal("Upload dir should not exist before Load()")
	}

	Load()

	// Verify it was created
	info, err := os.Stat(uploadDir)
	if err != nil {
		t.Fatalf("Upload dir should be created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Upload dir should be a directory")
	}
}

func TestConfigStructFields(t *testing.T) {
	cfg := Config{
		AdminUsername: "user",
		AdminPassword: "pass",
		APIKey:        "key",
		JWTSecret:     "secret",
		Port:          "8080",
		UploadDir:     "/uploads",
		DatabasePath:  "/data/db.sqlite",
	}

	if cfg.AdminUsername != "user" {
		t.Error("AdminUsername field not set correctly")
	}
	if cfg.AdminPassword != "pass" {
		t.Error("AdminPassword field not set correctly")
	}
	if cfg.APIKey != "key" {
		t.Error("APIKey field not set correctly")
	}
	if cfg.JWTSecret != "secret" {
		t.Error("JWTSecret field not set correctly")
	}
	if cfg.Port != "8080" {
		t.Error("Port field not set correctly")
	}
	if cfg.UploadDir != "/uploads" {
		t.Error("UploadDir field not set correctly")
	}
	if cfg.DatabasePath != "/data/db.sqlite" {
		t.Error("DatabasePath field not set correctly")
	}
}

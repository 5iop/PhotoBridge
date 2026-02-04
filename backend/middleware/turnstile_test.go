package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"photobridge/config"
	"photobridge/utils"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	// Initialize config for all tests
	config.AppConfig = &config.Config{
		TurnstileSiteKey:   "",
		TurnstileSecretKey: "",
		JWTSecret:          "test-jwt-secret",
	}
	config.AppConfig.InitCDNIPSet()
	os.Exit(m.Run())
}

func TestGetRealIP_CloudflareHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create request with CF-Connecting-IP header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("CF-Connecting-IP", "1.2.3.4")
	req.Header.Set("X-Real-IP", "5.6.7.8")
	req.Header.Set("X-Forwarded-For", "9.10.11.12")
	c.Request = req

	ip := GetRealIP(c)
	if ip != "1.2.3.4" {
		t.Errorf("Expected IP from CF-Connecting-IP, got %s", ip)
	}
}

func TestGetRealIP_XRealIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create request with X-Real-IP header (no CF header)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "5.6.7.8")
	req.Header.Set("X-Forwarded-For", "9.10.11.12")
	c.Request = req

	ip := GetRealIP(c)
	if ip != "5.6.7.8" {
		t.Errorf("Expected IP from X-Real-IP, got %s", ip)
	}
}

func TestGetRealIP_XForwardedFor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create request with X-Forwarded-For header (no CF or X-Real-IP)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "9.10.11.12, 13.14.15.16")
	c.Request = req

	ip := GetRealIP(c)
	if ip != "9.10.11.12" {
		t.Errorf("Expected first IP from X-Forwarded-For, got %s", ip)
	}
}

func TestRequireTurnstile_SkipWhenNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Save original config
	originalSiteKey := config.AppConfig.TurnstileSiteKey
	originalSecretKey := config.AppConfig.TurnstileSecretKey
	defer func() {
		config.AppConfig.TurnstileSiteKey = originalSiteKey
		config.AppConfig.TurnstileSecretKey = originalSecretKey
	}()

	// Clear Turnstile keys
	config.AppConfig.TurnstileSiteKey = ""
	config.AppConfig.TurnstileSecretKey = ""

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	// Apply middleware
	middleware := RequireTurnstile()
	middleware(c)

	// Should not abort
	if c.IsAborted() {
		t.Error("Middleware should not abort when Turnstile not configured")
	}
}

func TestRequireTurnstile_SkipForCDNIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Save original config
	originalSiteKey := config.AppConfig.TurnstileSiteKey
	originalSecretKey := config.AppConfig.TurnstileSecretKey
	defer func() {
		config.AppConfig.TurnstileSiteKey = originalSiteKey
		config.AppConfig.TurnstileSecretKey = originalSecretKey
	}()

	// Enable Turnstile
	config.AppConfig.TurnstileSiteKey = "test-site-key"
	config.AppConfig.TurnstileSecretKey = "test-secret-key"

	// Add a test IP to CDN whitelist
	testIP := "1.2.3.4"
	config.AppConfig.AddCDNIP(testIP)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("CF-Connecting-IP", testIP)
	c.Request = req

	// Apply middleware
	middleware := RequireTurnstile()
	middleware(c)

	// Should not abort for CDN IP
	if c.IsAborted() {
		t.Error("Middleware should not abort for CDN IP")
	}
}

func TestRequireTurnstile_SkipWithValidCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Save original config
	originalSiteKey := config.AppConfig.TurnstileSiteKey
	originalSecretKey := config.AppConfig.TurnstileSecretKey
	originalJWTSecret := config.AppConfig.JWTSecret
	defer func() {
		config.AppConfig.TurnstileSiteKey = originalSiteKey
		config.AppConfig.TurnstileSecretKey = originalSecretKey
		config.AppConfig.JWTSecret = originalJWTSecret
	}()

	// Enable Turnstile and set JWT secret for cookie signing
	config.AppConfig.TurnstileSiteKey = "test-site-key"
	config.AppConfig.TurnstileSecretKey = "test-secret-key"
	config.AppConfig.JWTSecret = "test-jwt-secret"

	// Generate a valid signed cookie
	validCookie := utils.GenerateVerificationCookie()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "pb_verified",
		Value: validCookie,
	})
	c.Request = req

	// Apply middleware
	middleware := RequireTurnstile()
	middleware(c)

	// Should not abort with valid cookie
	if c.IsAborted() {
		t.Error("Middleware should not abort with valid verification cookie")
	}
}

func TestRequireTurnstile_InvalidCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Save original config
	originalSiteKey := config.AppConfig.TurnstileSiteKey
	originalSecretKey := config.AppConfig.TurnstileSecretKey
	originalJWTSecret := config.AppConfig.JWTSecret
	defer func() {
		config.AppConfig.TurnstileSiteKey = originalSiteKey
		config.AppConfig.TurnstileSecretKey = originalSecretKey
		config.AppConfig.JWTSecret = originalJWTSecret
	}()

	// Enable Turnstile
	config.AppConfig.TurnstileSiteKey = "test-site-key"
	config.AppConfig.TurnstileSecretKey = "test-secret-key"
	config.AppConfig.JWTSecret = "test-jwt-secret"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/test", nil)
	// Add invalid/tampered cookie
	req.AddCookie(&http.Cookie{
		Name:  "pb_verified",
		Value: "invalid.cookie.signature",
	})
	c.Request = req

	// Apply middleware
	middleware := RequireTurnstile()
	middleware(c)

	// Should return 403 for invalid cookie
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for invalid cookie, got %d", w.Code)
	}
}

func TestRequireTurnstile_Returns403(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Save original config
	originalSiteKey := config.AppConfig.TurnstileSiteKey
	originalSecretKey := config.AppConfig.TurnstileSecretKey
	defer func() {
		config.AppConfig.TurnstileSiteKey = originalSiteKey
		config.AppConfig.TurnstileSecretKey = originalSecretKey
	}()

	// Enable Turnstile
	config.AppConfig.TurnstileSiteKey = "test-site-key"
	config.AppConfig.TurnstileSecretKey = "test-secret-key"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	// Apply middleware
	middleware := RequireTurnstile()
	middleware(c)

	// Should return 403
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["error"] != "verification_required" {
		t.Errorf("Expected error 'verification_required', got %v", response["error"])
	}

	if response["turnstile_key"] != "test-site-key" {
		t.Errorf("Expected turnstile_key in response, got %v", response["turnstile_key"])
	}

	if response["verification_url"] != "/api/verify" {
		t.Errorf("Expected verification_url in response, got %v", response["verification_url"])
	}
}

func TestRequireTurnstile_IPWithPort(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Save original config
	originalSiteKey := config.AppConfig.TurnstileSiteKey
	originalSecretKey := config.AppConfig.TurnstileSecretKey
	defer func() {
		config.AppConfig.TurnstileSiteKey = originalSiteKey
		config.AppConfig.TurnstileSecretKey = originalSecretKey
	}()

	// Enable Turnstile
	config.AppConfig.TurnstileSiteKey = "test-site-key"
	config.AppConfig.TurnstileSecretKey = "test-secret-key"

	// Add IP to whitelist (without port)
	testIP := "1.2.3.4"
	config.AppConfig.AddCDNIP(testIP)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/test", nil)
	// Header contains IP with port
	req.Header.Set("CF-Connecting-IP", "1.2.3.4:12345")
	c.Request = req

	// Apply middleware
	middleware := RequireTurnstile()
	middleware(c)

	// Should not abort (port should be stripped and matched)
	if c.IsAborted() {
		t.Error("Middleware should strip port and match CDN IP")
	}
}

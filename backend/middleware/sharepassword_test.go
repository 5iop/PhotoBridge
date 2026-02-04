package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"photobridge/config"
	"photobridge/database"
	"photobridge/models"
	"photobridge/utils"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory database for testing
func setupTestDB(t *testing.T) {
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Migrate schema
	err = database.DB.AutoMigrate(&models.ShareLink{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}
}

// createTestShareLink creates a share link in the test database
func createTestShareLink(t *testing.T, token string, passwordEnabled bool, password string) *models.ShareLink {
	link := &models.ShareLink{
		ProjectID:       1,
		Token:           token,
		Alias:           "test-alias",
		AllowRaw:        true,
		PasswordEnabled: passwordEnabled,
		Password:        password,
	}

	if err := database.DB.Create(link).Error; err != nil {
		t.Fatalf("Failed to create test share link: %v", err)
	}

	return link
}

func TestRequireSharePassword_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB(t)

	// Create a share link with password disabled
	token := "test-token-no-password"
	createTestShareLink(t, token, false, "")

	// Verify the link was created with password disabled
	var verifyLink models.ShareLink
	if err := database.DB.Where("token = ?", token).First(&verifyLink).Error; err != nil {
		t.Fatalf("Failed to verify created link: %v", err)
	}
	if verifyLink.PasswordEnabled {
		t.Fatalf("Link should have PasswordEnabled=false, got %v (ID=%d)", verifyLink.PasswordEnabled, verifyLink.ID)
	}

	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	// Add a dummy handler to be called after middleware
	called := false
	router.GET("/test/:token", RequireSharePassword(), func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	c.Request = httptest.NewRequest("GET", "/test/"+token, nil)
	router.ServeHTTP(w, c.Request)

	// Should not abort when password is disabled
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if !called {
		t.Error("Handler should have been called when password is disabled")
	}
}

func TestRequireSharePassword_ValidCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB(t)

	// Set up JWT secret
	config.AppConfig = &config.Config{
		JWTSecret: "test-secret",
	}

	// Create a share link with password enabled
	token := "test-token-with-password"
	createTestShareLink(t, token, true, "1234")

	// Generate a valid password cookie for this token
	validCookie := utils.GeneratePasswordCookie(token)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "token", Value: token}}
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "pb_share_verified_" + token,
		Value: validCookie,
	})
	c.Request = req

	// Apply middleware
	middleware := RequireSharePassword()
	middleware(c)

	// Should not abort with valid cookie
	if c.IsAborted() {
		t.Error("Middleware should not abort with valid password cookie")
	}
}

func TestRequireSharePassword_InvalidCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB(t)

	// Set up JWT secret
	config.AppConfig = &config.Config{
		JWTSecret: "test-secret",
	}

	// Create a share link with password enabled
	token := "test-token-with-password"
	createTestShareLink(t, token, true, "1234")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "token", Value: token}}
	req := httptest.NewRequest("GET", "/test", nil)
	// Add invalid cookie
	req.AddCookie(&http.Cookie{
		Name:  "pb_share_verified_" + token,
		Value: "invalid.cookie.signature",
	})
	c.Request = req

	// Apply middleware
	middleware := RequireSharePassword()
	middleware(c)

	// Should return 403 for invalid cookie
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for invalid cookie, got %d", w.Code)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["error"] != "password_required" {
		t.Errorf("Expected error 'password_required', got %v", response["error"])
	}
}

func TestRequireSharePassword_NoCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB(t)

	// Create a share link with password enabled
	token := "test-token-with-password"
	createTestShareLink(t, token, true, "1234")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "token", Value: token}}
	c.Request = httptest.NewRequest("GET", "/test", nil)

	// Apply middleware
	middleware := RequireSharePassword()
	middleware(c)

	// Should return 403 without cookie
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 without cookie, got %d", w.Code)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["error"] != "password_required" {
		t.Errorf("Expected error 'password_required', got %v", response["error"])
	}

	if response["verification_url"] != "/api/share/"+token+"/verify-password" {
		t.Errorf("Expected verification_url in response, got %v", response["verification_url"])
	}
}

func TestRequireSharePassword_LinkNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "token", Value: "non-existent-token"}}
	c.Request = httptest.NewRequest("GET", "/test", nil)

	// Apply middleware
	middleware := RequireSharePassword()
	middleware(c)

	// Should return 404
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent link, got %d", w.Code)
	}
}

func TestVerifySharePasswordHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB(t)

	// Set up JWT secret
	config.AppConfig = &config.Config{
		JWTSecret: "test-secret",
	}

	// Create a share link with password
	token := "test-token"
	password := "1234"
	createTestShareLink(t, token, true, password)

	// Create request body
	reqBody := map[string]string{"password": password}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "token", Value: token}}
	c.Request = httptest.NewRequest("POST", "/verify-password", bytes.NewReader(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	// Call handler
	VerifySharePasswordHandler(c)

	// Should return 200
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["success"] != true {
		t.Errorf("Expected success=true, got %v", response["success"])
	}

	// Check cookie was set
	cookies := w.Result().Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "pb_share_verified_"+token {
			found = true
			if cookie.Value == "" {
				t.Error("Cookie value should not be empty")
			}
			if cookie.MaxAge != 30*24*60*60 {
				t.Errorf("Cookie MaxAge should be 30 days, got %d", cookie.MaxAge)
			}
			if !cookie.HttpOnly {
				t.Error("Cookie should be HttpOnly")
			}
		}
	}
	if !found {
		t.Error("Password verification cookie should be set")
	}
}

func TestVerifySharePasswordHandler_WrongPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB(t)

	// Create a share link with password
	token := "test-token"
	password := "1234"
	createTestShareLink(t, token, true, password)

	// Create request body with wrong password
	reqBody := map[string]string{"password": "9999"}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "token", Value: token}}
	c.Request = httptest.NewRequest("POST", "/verify-password", bytes.NewReader(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	// Call handler
	VerifySharePasswordHandler(c)

	// Should return 403
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for wrong password, got %d", w.Code)
	}

	// Check response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["success"] != false {
		t.Errorf("Expected success=false, got %v", response["success"])
	}
}

func TestVerifySharePasswordHandler_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB(t)

	// Create a share link
	token := "test-token"
	createTestShareLink(t, token, true, "1234")

	// Create request body without password field
	reqBody := map[string]string{"wrong_field": "value"}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "token", Value: token}}
	c.Request = httptest.NewRequest("POST", "/verify-password", bytes.NewReader(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	// Call handler
	VerifySharePasswordHandler(c)

	// Should return 400
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid request, got %d", w.Code)
	}
}

func TestVerifySharePasswordHandler_LinkNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB(t)

	// Create request body
	reqBody := map[string]string{"password": "1234"}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "token", Value: "non-existent-token"}}
	c.Request = httptest.NewRequest("POST", "/verify-password", bytes.NewReader(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	// Call handler
	VerifySharePasswordHandler(c)

	// Should return 404
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent link, got %d", w.Code)
	}
}

func TestRequireSharePassword_CookieTokenBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupTestDB(t)

	// Set up JWT secret
	config.AppConfig = &config.Config{
		JWTSecret: "test-secret",
	}

	// Create two share links
	token1 := "test-token-1"
	token2 := "test-token-2"
	createTestShareLink(t, token1, true, "1234")
	createTestShareLink(t, token2, true, "5678")

	// Generate a valid password cookie for token1
	cookie1 := utils.GeneratePasswordCookie(token1)

	// Try to use cookie1 to access token2 (should fail due to token binding)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "token", Value: token2}}
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "pb_share_verified_" + token2,
		Value: cookie1, // Cookie from token1
	})
	c.Request = req

	// Apply middleware
	middleware := RequireSharePassword()
	middleware(c)

	// Should return 403 (cookie from token1 should not work for token2)
	if w.Code != http.StatusForbidden {
		t.Error("Cookie from one token should not work for a different token")
	}
}

package middleware

import (
	"net/http"
	"time"

	"photobridge/database"
	"photobridge/models"
	"photobridge/utils"

	"github.com/gin-gonic/gin"
)

const (
	passwordCookieName   = "pb_share_verified_"
	passwordCookieMaxAge = 30 * 24 * 60 * 60 // 30 days
)

// RequireSharePassword is a middleware that requires password verification for share links
func RequireSharePassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")

		// Get share link
		var link models.ShareLink
		if err := database.DB.Where("token = ?", token).First(&link).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
			c.Abort()
			return
		}

		// If password is not enabled, allow access
		if !link.PasswordEnabled {
			c.Next()
			return
		}

		// Check if user has valid verification cookie
		cookieName := passwordCookieName + token
		if cookie, err := c.Cookie(cookieName); err == nil && cookie != "" {
			// Verify cookie signature
			if utils.VerifyPasswordCookie(cookie, token) {
				// User is already verified with valid signature
				c.Next()
				return
			}
			// Invalid signature - fall through to require verification
		}

		// User needs password verification
		c.JSON(http.StatusForbidden, gin.H{
			"error":            "password_required",
			"message":          "Please enter the password to access this share link",
			"verification_url": "/api/share/" + token + "/verify-password",
		})
		c.Abort()
	}
}

// VerifySharePasswordHandler handles password verification requests
func VerifySharePasswordHandler(c *gin.Context) {
	token := c.Param("token")

	var req struct {
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get share link
	var link models.ShareLink
	if err := database.DB.Where("token = ?", token).First(&link).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	// Verify password
	if req.Password != link.Password {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Incorrect password",
			"message": "密码错误，请重试",
		})
		return
	}

	// Determine if cookie should be Secure based on request protocol
	// Check TLS or X-Forwarded-Proto header (for reverse proxies)
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"

	// Set verification cookie (30 days)
	cookieName := passwordCookieName + token
	c.SetCookie(
		cookieName,
		utils.GeneratePasswordCookie(token),
		passwordCookieMaxAge,
		"/",
		"",       // domain (empty = current domain)
		isSecure, // secure (HTTPS only when appropriate)
		true,     // httpOnly (not accessible via JavaScript)
	)

	// Add debug header
	c.Header("X-Password-Verification-Time", time.Now().Format(time.RFC3339))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password verified",
	})
}

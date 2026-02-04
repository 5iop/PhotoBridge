package middleware

import (
	"net/http"
	"time"

	"photobridge/config"
	"photobridge/utils"

	"github.com/gin-gonic/gin"
)

const (
	verificationCookieName = "pb_verified"
	cookieMaxAge           = 30 * 24 * 60 * 60 // 30 days
)

// RequireTurnstile is a middleware that requires Turnstile verification for first-time visitors
func RequireTurnstile() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if Turnstile is not configured
		if config.AppConfig.TurnstileSiteKey == "" || config.AppConfig.TurnstileSecretKey == "" {
			c.Next()
			return
		}

		// Get real client IP (considering Cloudflare headers)
		realIP := GetRealIP(c)

		// Skip verification for CDN server IPs (auto-resolved from CNCDN_URL)
		// If CNCDN_URL is set to https://cdn.pb.jangit.me, this will automatically
		// resolve cdn.pb.jangit.me to its IPs and whitelist them
		if config.AppConfig.IsCDNIP(realIP) {
			c.Next()
			return
		}

		// Check if user already has verification cookie
		if cookie, err := c.Cookie(verificationCookieName); err == nil && cookie != "" {
			// Verify cookie signature
			if utils.VerifyVerificationCookie(cookie) {
				// User is already verified with valid signature
				c.Next()
				return
			}
			// Invalid signature - fall through to require verification
		}

		// User needs verification - return 403 with Turnstile site key
		c.JSON(http.StatusForbidden, gin.H{
			"error":            "verification_required",
			"message":          "Please complete the verification challenge",
			"turnstile_key":    config.AppConfig.TurnstileSiteKey,
			"verification_url": "/api/verify",
		})
		c.Abort()
	}
}

// VerifyTurnstileHandler handles Turnstile token verification
func VerifyTurnstileHandler(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get real IP for verification
	realIP := GetRealIP(c)

	// Verify token with Cloudflare
	success, err := utils.VerifyTurnstileToken(req.Token, realIP)
	if err != nil || !success {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Verification failed",
			"message": "Please try again",
		})
		return
	}

	// Determine if cookie should be Secure based on request protocol
	// Check TLS or X-Forwarded-Proto header (for reverse proxies)
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"

	// Set verification cookie (30 days)
	c.SetCookie(
		verificationCookieName,
		utils.GenerateVerificationCookie(),
		cookieMaxAge,
		"/",
		"",        // domain (empty = current domain)
		isSecure,  // secure (HTTPS only when appropriate)
		true,      // httpOnly (not accessible via JavaScript)
	)

	// Add debug header
	c.Header("X-Verification-Time", time.Now().Format(time.RFC3339))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Verification successful",
	})
}

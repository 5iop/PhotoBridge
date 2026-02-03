package utils

import (
	"photobridge/config"

	"github.com/gin-gonic/gin"
)

// GetCDNBaseURL returns the appropriate CDN base URL based on the client's country
// For China (CF-IPCountry: CN), returns CNCDN_URL if configured
// For other countries, returns empty string (use relative URLs)
func GetCDNBaseURL(c *gin.Context) string {
	// Check if China CDN is configured
	if config.AppConfig.CNCDNURL == "" {
		return ""
	}

	// Check CF-IPCountry header (set by Cloudflare)
	country := c.GetHeader("CF-IPCountry")

	// If request is from China, use China CDN
	if country == "CN" {
		return config.AppConfig.CNCDNURL
	}

	// For other countries, use relative URLs (served by main domain)
	return ""
}

package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// GetRealIP extracts the real client IP from Cloudflare headers
// Priority: CF-Connecting-IP > X-Real-IP > X-Forwarded-For > RemoteAddr
func GetRealIP(c *gin.Context) string {
	// Cloudflare passes the real IP in CF-Connecting-IP
	if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
		return ip
	}

	// Fallback to X-Real-IP
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}

	// Fallback to X-Forwarded-For (take the first IP)
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For can be: "client, proxy1, proxy2"
		// We want the first IP (the client)
		for i := 0; i < len(ip); i++ {
			if ip[i] == ',' || ip[i] == ' ' {
				return ip[:i]
			}
		}
		return ip
	}

	// Fallback to RemoteAddr
	return c.ClientIP()
}

// Logger is a custom logger middleware that:
// 1. Shows real client IP from Cloudflare headers
// 2. Skips logging for /api/health endpoint
// 3. Adds Cloudflare debugging headers to response
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for health check endpoint
		if c.Request.URL.Path == "/api/health" {
			c.Next()
			return
		}

		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Get real IP
		realIP := GetRealIP(c)

		// Get Cloudflare headers for debugging
		cfRay := c.GetHeader("CF-Ray")
		cfCountry := c.GetHeader("CF-IPCountry")
		cfCacheStatus := c.GetHeader("CF-Cache-Status")

		// Add Cloudflare debugging headers to response (helpful for frontend debugging)
		if cfRay != "" {
			c.Header("X-CF-Ray", cfRay)
		}
		if cfCacheStatus != "" {
			c.Header("X-CF-Cache-Status", cfCacheStatus)
		}

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		method := c.Request.Method

		// Build log message
		logMsg := fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %s",
			start.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			realIP,
			method,
			path,
		)

		// Add query string if present
		if raw != "" {
			logMsg += "?" + raw
		}

		// Add Cloudflare info if available
		cfInfo := ""
		if cfCountry != "" {
			cfInfo += fmt.Sprintf(" | Country: %s", cfCountry)
		}
		if cfRay != "" {
			cfInfo += fmt.Sprintf(" | Ray: %s", cfRay)
		}
		if cfCacheStatus != "" {
			cfInfo += fmt.Sprintf(" | Cache: %s", cfCacheStatus)
		}
		if cfInfo != "" {
			logMsg += cfInfo
		}

		// Print log
		fmt.Println(logMsg)

		// Log errors if any
		if len(c.Errors) > 0 {
			fmt.Println(c.Errors.String())
		}
	}
}

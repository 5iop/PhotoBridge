package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"photobridge/config"
)

// TurnstileResponse represents the response from Cloudflare Turnstile verification API
type TurnstileResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
	Action      string   `json:"action"`
	CData       string   `json:"cdata"`
}

// VerifyTurnstileToken verifies a Turnstile token with Cloudflare's API
func VerifyTurnstileToken(token string, remoteIP string) (bool, error) {
	// If Turnstile is not configured, skip verification
	if config.AppConfig.TurnstileSecretKey == "" {
		return true, nil
	}

	if token == "" {
		return false, fmt.Errorf("turnstile token is required")
	}

	// Prepare request to Cloudflare
	formData := url.Values{
		"secret":   {config.AppConfig.TurnstileSecretKey},
		"response": {token},
	}

	// Add remote IP if provided (optional but recommended)
	if remoteIP != "" {
		formData.Set("remoteip", remoteIP)
	}

	// Make POST request to Cloudflare
	resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", formData)
	if err != nil {
		return false, fmt.Errorf("failed to verify turnstile token: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var result TurnstileResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check if verification succeeded
	if !result.Success {
		return false, fmt.Errorf("turnstile verification failed: %v", result.ErrorCodes)
	}

	return true, nil
}

// GenerateVerificationCookie generates a secure, signed cookie value for verified users
// Format: timestamp.randomToken.signature
// The signature is HMAC-SHA256(timestamp + randomToken, JWTSecret)
func GenerateVerificationCookie() string {
	// Generate timestamp
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Generate 16 bytes of random data
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp only if random fails (should never happen)
		return fmt.Sprintf("verified_%s", timestamp)
	}
	randomToken := base64.URLEncoding.EncodeToString(randomBytes)

	// Create payload to sign
	payload := timestamp + "." + randomToken

	// Sign with HMAC-SHA256 using JWT secret
	h := hmac.New(sha256.New, []byte(config.AppConfig.JWTSecret))
	h.Write([]byte(payload))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Return signed cookie: timestamp.randomToken.signature
	return payload + "." + signature
}

// VerifyVerificationCookie verifies the signature of a verification cookie
// Also checks TTL (1 day) to prevent long-term cookie reuse
func VerifyVerificationCookie(cookie string) bool {
	// Split cookie into parts
	parts := strings.Split(cookie, ".")
	if len(parts) != 3 {
		return false
	}

	timestampStr := parts[0]
	randomToken := parts[1]
	providedSignature := parts[2]

	// Parse and verify timestamp (TTL: 1 day)
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return false
	}

	// Check if cookie is expired (1 day TTL)
	const cookieTTL = 24 * 60 * 60 // 1 day in seconds
	if time.Now().Unix()-timestamp > cookieTTL {
		return false
	}

	// Recreate payload
	payload := timestampStr + "." + randomToken

	// Compute expected signature
	h := hmac.New(sha256.New, []byte(config.AppConfig.JWTSecret))
	h.Write([]byte(payload))
	expectedSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Compare signatures using constant-time comparison
	return hmac.Equal([]byte(providedSignature), []byte(expectedSignature))
}

// GeneratePasswordCookie generates a secure, signed cookie value for password-verified users
// Format: timestamp.randomToken.signature
// The signature includes the shareToken to prevent cookie reuse across different share links
func GeneratePasswordCookie(shareToken string) string {
	// Generate timestamp
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Generate 16 bytes of random data
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp only if random fails (should never happen)
		return fmt.Sprintf("verified_%s_%s", shareToken, timestamp)
	}
	randomToken := base64.URLEncoding.EncodeToString(randomBytes)

	// Create payload to sign (includes shareToken to bind cookie to specific link)
	payload := timestamp + "." + randomToken + "." + shareToken

	// Sign with HMAC-SHA256 using JWT secret
	h := hmac.New(sha256.New, []byte(config.AppConfig.JWTSecret))
	h.Write([]byte(payload))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Return signed cookie: timestamp.randomToken.signature
	return timestamp + "." + randomToken + "." + signature
}

// VerifyPasswordCookie verifies the signature of a password verification cookie
// The cookie is bound to a specific shareToken and cannot be used for other share links
// Also checks TTL (1 day) to prevent long-term cookie reuse
func VerifyPasswordCookie(cookie string, shareToken string) bool {
	// Split cookie into parts
	parts := strings.Split(cookie, ".")
	if len(parts) != 3 {
		return false
	}

	timestampStr := parts[0]
	randomToken := parts[1]
	providedSignature := parts[2]

	// Parse and verify timestamp (TTL: 1 day)
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return false
	}

	// Check if cookie is expired (1 day TTL)
	const cookieTTL = 24 * 60 * 60 // 1 day in seconds
	if time.Now().Unix()-timestamp > cookieTTL {
		return false
	}

	// Recreate payload (must include shareToken)
	payload := timestampStr + "." + randomToken + "." + shareToken

	// Compute expected signature
	h := hmac.New(sha256.New, []byte(config.AppConfig.JWTSecret))
	h.Write([]byte(payload))
	expectedSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Compare signatures using constant-time comparison
	return hmac.Equal([]byte(providedSignature), []byte(expectedSignature))
}

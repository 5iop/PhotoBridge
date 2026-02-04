package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"testing"
	"time"

	"photobridge/config"
)

// Note: These tests focus on unit testing the functions in isolation.
// Integration tests with actual Cloudflare API should be done separately.

func TestGenerateVerificationCookie_Format(t *testing.T) {
	// Ensure config is initialized
	if config.AppConfig == nil || config.AppConfig.JWTSecret == "" {
		config.AppConfig = &config.Config{
			JWTSecret: "test-secret-for-testing",
		}
	}

	cookie := GenerateVerificationCookie()

	// Should be non-empty
	if cookie == "" {
		t.Error("Cookie should not be empty")
	}

	// Should have format: timestamp.randomToken.signature
	parts := strings.Split(cookie, ".")
	if len(parts) != 3 {
		t.Errorf("Cookie should have 3 parts (timestamp.randomToken.signature), got %d parts: %q", len(parts), cookie)
	}

	// First part should be a unix timestamp (number)
	timestamp := parts[0]
	if _, err := strconv.ParseInt(timestamp, 10, 64); err != nil {
		t.Errorf("First part should be a valid timestamp, got %q", timestamp)
	}

	// Second part should be base64-encoded random token
	randomToken := parts[1]
	if len(randomToken) == 0 {
		t.Error("Random token part should not be empty")
	}
	if _, err := base64.URLEncoding.DecodeString(randomToken); err != nil {
		t.Errorf("Random token should be valid base64, got %q", randomToken)
	}

	// Third part should be base64-encoded signature
	signature := parts[2]
	if len(signature) == 0 {
		t.Error("Signature part should not be empty")
	}
	if _, err := base64.URLEncoding.DecodeString(signature); err != nil {
		t.Errorf("Signature should be valid base64, got %q", signature)
	}
}

func TestGenerateVerificationCookie_Uniqueness(t *testing.T) {
	// Ensure config is initialized
	if config.AppConfig == nil || config.AppConfig.JWTSecret == "" {
		config.AppConfig = &config.Config{
			JWTSecret: "test-secret-for-testing",
		}
	}

	// Generate multiple cookies
	cookie1 := GenerateVerificationCookie()
	time.Sleep(time.Millisecond) // Small delay
	cookie2 := GenerateVerificationCookie()

	// Should be different (due to different timestamps and/or random tokens)
	if cookie1 == cookie2 {
		t.Error("Cookies should be unique")
	}
}

func TestVerifyVerificationCookie_Valid(t *testing.T) {
	// Ensure config is initialized
	if config.AppConfig == nil || config.AppConfig.JWTSecret == "" {
		config.AppConfig = &config.Config{
			JWTSecret: "test-secret-for-testing",
		}
	}

	// Generate a cookie
	cookie := GenerateVerificationCookie()

	// Should verify successfully
	if !VerifyVerificationCookie(cookie) {
		t.Error("Valid cookie should verify successfully")
	}
}

func TestVerifyVerificationCookie_Invalid(t *testing.T) {
	// Ensure config is initialized
	if config.AppConfig == nil || config.AppConfig.JWTSecret == "" {
		config.AppConfig = &config.Config{
			JWTSecret: "test-secret-for-testing",
		}
	}

	tests := []struct {
		name   string
		cookie string
	}{
		{"empty", ""},
		{"invalid format (2 parts)", "1234567890.abcdef"},
		{"invalid format (4 parts)", "1234567890.abcdef.signature.extra"},
		{"tampered timestamp", "9999999999.abcdef.signature"},
		{"tampered random", "1234567890.TAMPERED.signature"},
		{"tampered signature", "1234567890.abcdef.TAMPERED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if VerifyVerificationCookie(tt.cookie) {
				t.Errorf("Invalid cookie %q should not verify", tt.cookie)
			}
		})
	}
}

func TestVerifyVerificationCookie_DifferentSecret(t *testing.T) {
	// Generate with one secret
	config.AppConfig = &config.Config{
		JWTSecret: "secret1",
	}
	cookie := GenerateVerificationCookie()

	// Verify with different secret
	config.AppConfig.JWTSecret = "secret2"
	if VerifyVerificationCookie(cookie) {
		t.Error("Cookie signed with different secret should not verify")
	}

	// Restore original secret and verify
	config.AppConfig.JWTSecret = "secret1"
	if !VerifyVerificationCookie(cookie) {
		t.Error("Cookie should verify with original secret")
	}
}

func TestVerifyVerificationCookie_ManuallyConstructed(t *testing.T) {
	// Set up test secret
	config.AppConfig = &config.Config{
		JWTSecret: "test-secret",
	}

	// Manually construct a valid cookie
	timestamp := "1234567890"
	randomToken := "test-random-token"
	payload := timestamp + "." + randomToken

	// Compute correct signature
	h := hmac.New(sha256.New, []byte("test-secret"))
	h.Write([]byte(payload))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	validCookie := payload + "." + signature

	// Should verify
	if !VerifyVerificationCookie(validCookie) {
		t.Error("Manually constructed valid cookie should verify")
	}

	// Tamper with it
	tamperedCookie := timestamp + ".TAMPERED." + signature
	if VerifyVerificationCookie(tamperedCookie) {
		t.Error("Tampered cookie should not verify")
	}
}

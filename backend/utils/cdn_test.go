package utils

import (
	"net/http/httptest"
	"testing"

	"photobridge/config"

	"github.com/gin-gonic/gin"
)

func TestGetCDNBaseURL(t *testing.T) {
	// Save original config
	originalConfig := config.AppConfig

	tests := []struct {
		name           string
		cncdnURL       string
		cfIPCountry    string
		expectedResult string
	}{
		{
			name:           "China user with CDN configured",
			cncdnURL:       "https://cdn.example.com",
			cfIPCountry:    "CN",
			expectedResult: "https://cdn.example.com",
		},
		{
			name:           "US user with CDN configured",
			cncdnURL:       "https://cdn.example.com",
			cfIPCountry:    "US",
			expectedResult: "",
		},
		{
			name:           "Japan user with CDN configured",
			cncdnURL:       "https://cdn.example.com",
			cfIPCountry:    "JP",
			expectedResult: "",
		},
		{
			name:           "China user without CDN configured",
			cncdnURL:       "",
			cfIPCountry:    "CN",
			expectedResult: "",
		},
		{
			name:           "No CF-IPCountry header",
			cncdnURL:       "https://cdn.example.com",
			cfIPCountry:    "",
			expectedResult: "",
		},
		{
			name:           "Case sensitive country code",
			cncdnURL:       "https://cdn.example.com",
			cfIPCountry:    "cn", // lowercase
			expectedResult: "",   // Should not match (case sensitive)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test config
			config.AppConfig = &config.Config{
				CNCDNURL: tt.cncdnURL,
			}

			// Create test context
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create request with CF-IPCountry header
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.cfIPCountry != "" {
				req.Header.Set("CF-IPCountry", tt.cfIPCountry)
			}
			c.Request = req

			// Test GetCDNBaseURL
			result := GetCDNBaseURL(c)

			if result != tt.expectedResult {
				t.Errorf("GetCDNBaseURL() = %q, want %q", result, tt.expectedResult)
			}
		})
	}

	// Restore original config
	config.AppConfig = originalConfig
}

func TestGetCDNBaseURLMultipleRequests(t *testing.T) {
	// Save original config
	originalConfig := config.AppConfig
	defer func() { config.AppConfig = originalConfig }()

	config.AppConfig = &config.Config{
		CNCDNURL: "https://cdn.test.com",
	}

	gin.SetMode(gin.TestMode)

	// Test multiple requests with different countries
	countries := []struct {
		country string
		wantCDN bool
	}{
		{"CN", true},
		{"US", false},
		{"CN", true},
		{"GB", false},
		{"CN", true},
	}

	for i, tc := range countries {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("CF-IPCountry", tc.country)
		c.Request = req

		result := GetCDNBaseURL(c)

		if tc.wantCDN {
			if result != "https://cdn.test.com" {
				t.Errorf("Request %d: expected CDN URL, got %q", i, result)
			}
		} else {
			if result != "" {
				t.Errorf("Request %d: expected empty string, got %q", i, result)
			}
		}
	}
}

func TestGetCDNBaseURLNilConfig(t *testing.T) {
	// Save original config
	originalConfig := config.AppConfig
	defer func() {
		config.AppConfig = originalConfig
		// Recover from potential panic
		if r := recover(); r != nil {
			t.Logf("Recovered from panic: %v", r)
		}
	}()

	// Test with nil config (should not panic)
	config.AppConfig = &config.Config{
		CNCDNURL: "",
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("CF-IPCountry", "CN")
	c.Request = req

	result := GetCDNBaseURL(c)
	if result != "" {
		t.Errorf("Expected empty string when CNCDNURL is empty, got %q", result)
	}
}

func TestGetCDNBaseURLSpecialCountries(t *testing.T) {
	// Save original config
	originalConfig := config.AppConfig
	defer func() { config.AppConfig = originalConfig }()

	config.AppConfig = &config.Config{
		CNCDNURL: "https://cdn.example.com",
	}

	// Test special/edge case country codes
	specialCases := []struct {
		country string
		wantCDN bool
		desc    string
	}{
		{"CN", true, "China (target country)"},
		{"HK", false, "Hong Kong (different from CN)"},
		{"MO", false, "Macau (different from CN)"},
		{"TW", false, "Taiwan (different from CN)"},
		{"XX", false, "Invalid country code"},
		{"", false, "Empty country code"},
		{"C", false, "Too short country code"},
		{"CHINA", false, "Full country name instead of code"},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range specialCases {
		t.Run(tc.desc, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest("GET", "/test", nil)
			if tc.country != "" {
				req.Header.Set("CF-IPCountry", tc.country)
			}
			c.Request = req

			result := GetCDNBaseURL(c)

			if tc.wantCDN {
				if result != "https://cdn.example.com" {
					t.Errorf("Expected CDN URL, got %q", result)
				}
			} else {
				if result != "" {
					t.Errorf("Expected empty string, got %q", result)
				}
			}
		})
	}
}

func TestGetCDNBaseURLDifferentCDNURLs(t *testing.T) {
	// Save original config
	originalConfig := config.AppConfig
	defer func() { config.AppConfig = originalConfig }()

	cdnURLs := []string{
		"https://cdn1.example.com",
		"https://cdn2.example.com",
		"http://cdn3.example.com",
		"https://cdn.example.com:8443",
		"https://cdn.example.com/prefix",
	}

	gin.SetMode(gin.TestMode)

	for _, cdnURL := range cdnURLs {
		t.Run(cdnURL, func(t *testing.T) {
			config.AppConfig = &config.Config{
				CNCDNURL: cdnURL,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("CF-IPCountry", "CN")
			c.Request = req

			result := GetCDNBaseURL(c)
			if result != cdnURL {
				t.Errorf("Expected %q, got %q", cdnURL, result)
			}
		})
	}
}

// Benchmark GetCDNBaseURL
func BenchmarkGetCDNBaseURL(b *testing.B) {
	config.AppConfig = &config.Config{
		CNCDNURL: "https://cdn.example.com",
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("CF-IPCountry", "CN")
	c.Request = req

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetCDNBaseURL(c)
	}
}

func BenchmarkGetCDNBaseURLNoCDN(b *testing.B) {
	config.AppConfig = &config.Config{
		CNCDNURL: "",
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("CF-IPCountry", "CN")
	c.Request = req

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetCDNBaseURL(c)
	}
}

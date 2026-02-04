package utils

import (
	"strconv"
	"testing"
)

func TestGenerateSharePassword(t *testing.T) {
	// Generate multiple passwords
	passwords := make(map[string]bool)
	for i := 0; i < 100; i++ {
		password := GenerateSharePassword()

		// Should be 4 characters
		if len(password) != 4 {
			t.Errorf("Password should be 4 characters, got %d: %q", len(password), password)
		}

		// Should be numeric
		num, err := strconv.Atoi(password)
		if err != nil {
			t.Errorf("Password should be numeric, got %q: %v", password, err)
		}

		// Should be in range 1000-9999
		if num < 1000 || num > 9999 {
			t.Errorf("Password should be in range 1000-9999, got %d", num)
		}

		// Track uniqueness (should have some variety)
		passwords[password] = true
	}

	// Should generate at least some different passwords in 100 tries
	// (Not a guarantee, but statistically very likely with 9000 possible values)
	if len(passwords) < 10 {
		t.Errorf("Expected at least 10 different passwords in 100 tries, got %d", len(passwords))
	}
}

func TestValidateSharePassword_Valid(t *testing.T) {
	tests := []string{
		"1000",
		"1234",
		"5678",
		"9999",
		"0000", // Edge case: technically valid format
	}

	for _, password := range tests {
		if !ValidateSharePassword(password) {
			t.Errorf("Password %q should be valid", password)
		}
	}
}

func TestValidateSharePassword_Invalid(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"empty", ""},
		{"too short", "123"},
		{"too long", "12345"},
		{"contains letters", "12a4"},
		{"contains special chars", "12@4"},
		{"contains spaces", "12 4"},
		{"non-numeric", "abcd"},
		{"negative", "-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ValidateSharePassword(tt.password) {
				t.Errorf("Password %q should be invalid", tt.password)
			}
		})
	}
}

func TestGenerateSharePassword_Format(t *testing.T) {
	// Test that generated passwords always pass validation
	for i := 0; i < 100; i++ {
		password := GenerateSharePassword()
		if !ValidateSharePassword(password) {
			t.Errorf("Generated password %q should pass validation", password)
		}
	}
}

func TestGenerateSharePassword_Randomness(t *testing.T) {
	// Generate many passwords and check distribution
	passwords := make(map[string]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		password := GenerateSharePassword()
		passwords[password]++
	}

	// Should have a reasonable number of unique values
	// With 9000 possible values and 1000 iterations, expect mostly unique
	uniqueCount := len(passwords)
	expectedMinUnique := iterations * 9 / 10 // At least 90% unique

	if uniqueCount < expectedMinUnique {
		t.Errorf("Expected at least %d unique passwords, got %d (may indicate poor randomness)", expectedMinUnique, uniqueCount)
	}

	// Check no password appears too frequently
	maxFrequency := 0
	for _, count := range passwords {
		if count > maxFrequency {
			maxFrequency = count
		}
	}

	// With good randomness, no password should appear more than ~5 times in 1000 iterations
	// (statistical outliers possible but very unlikely)
	if maxFrequency > 10 {
		t.Errorf("Maximum password frequency too high: %d (may indicate poor randomness)", maxFrequency)
	}
}

func TestValidateSharePassword_Boundaries(t *testing.T) {
	// Test boundary values
	tests := []struct {
		password string
		valid    bool
	}{
		{"0999", true},  // Just below minimum
		{"1000", true},  // Minimum
		{"1001", true},  // Just above minimum
		{"9998", true},  // Just below maximum
		{"9999", true},  // Maximum
		{"10000", false}, // Too long (5 digits)
		{"999", false},  // Too short (3 digits)
	}

	for _, tt := range tests {
		result := ValidateSharePassword(tt.password)
		if result != tt.valid {
			t.Errorf("ValidateSharePassword(%q) = %v, want %v", tt.password, result, tt.valid)
		}
	}
}

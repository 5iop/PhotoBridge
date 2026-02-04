package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// GenerateSharePassword generates a random 4-digit password (1000-9999)
func GenerateSharePassword() string {
	min := int64(1000)
	max := int64(9999)

	n, err := rand.Int(rand.Reader, big.NewInt(max-min+1))
	if err != nil {
		// Fallback: use timestamp last 4 digits
		return fmt.Sprintf("%04d", time.Now().Unix()%10000)
	}

	return fmt.Sprintf("%04d", n.Int64()+min)
}

// ValidateSharePassword validates that the password is exactly 4 digits
func ValidateSharePassword(password string) bool {
	if len(password) != 4 {
		return false
	}
	for _, c := range password {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

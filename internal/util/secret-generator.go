package util

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateRandomString returns a securely generated random string of the given length.
func GenerateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)[:length]
}

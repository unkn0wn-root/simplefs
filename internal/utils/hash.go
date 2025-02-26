package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// HashString creates a consistent hash for a string
func HashString(s string) string {
	hasher := sha256.New()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}

// ShortHash creates a shorter hash for a string (first 16 characters)
func ShortHash(s string) string {
	return HashString(s)[:16]
}

// IsValidHash checks if a string is a valid hex hash
func IsValidHash(s string) bool {
	// Check if string is a valid hex string
	_, err := hex.DecodeString(s)
	return err == nil
}

// HashPath creates a hash for a file path, normalizing the path first
func HashPath(path string) string {
	// Normalize path by removing leading/trailing slashes and converting backslashes
	normalized := strings.TrimPrefix(path, "/")
	normalized = strings.TrimSuffix(normalized, "/")
	normalized = strings.ReplaceAll(normalized, "\\", "/")

	return HashString(normalized)
}

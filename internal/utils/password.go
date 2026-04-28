package utils

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword generates a bcrypt hash from a plaintext password. We use
// cost 12, high enough to be resistant to brute force, low enough that
// login requests don't feel sluggish. bcrypt is chosen over argon2 because
// the Go standard extended library ships it, and for this application the
// security difference is negligible.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword compares a plaintext password against a bcrypt hash.
// Returns true if they match. The comparison is constant-time to prevent
// timing attacks, bcrypt handles this internally.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

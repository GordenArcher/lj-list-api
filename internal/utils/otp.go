package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const otpMaxValue = 1000000

// GenerateOTP returns a cryptographically random 6-digit numeric code.
func GenerateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(otpMaxValue))
	if err != nil {
		return "", fmt.Errorf("generate otp: %w", err)
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
}

package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenClaims struct {
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type TokenType string

const (
	AccessTokenType  TokenType = "access"
	RefreshTokenType TokenType = "refresh"
)

type RefreshTokenClaims struct {
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a short-lived JWT (15 minutes). The access
// token goes in the httpOnly cookie and is sent on every request. Short
// expiry limits the damage if the cookie leaks.
func GenerateAccessToken(userID, role, secret string) (string, error) {
	claims := TokenClaims{
		UserID:    userID,
		Role:      role,
		TokenType: AccessTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "lj-list-api",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken creates a long-lived JWT (7 days). It's stored in
// a separate httpOnly cookie with a restricted path so it's only sent to
// auth endpoints, not the entire API. If a refresh token is compromised,
// the attacker can get new access tokens but only for 7 days.
func GenerateRefreshToken(userID, secret string) (string, error) {
	claims := RefreshTokenClaims{
		TokenType: RefreshTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "lj-list-api",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateTokenPair returns both tokens. The caller sets two cookies:
// one for the access token (path=/), one for the refresh token
// (path=/api/v1/auth).
func GenerateTokenPair(userID, role, secret string) (*TokenPair, error) {
	access, err := GenerateAccessToken(userID, role, secret)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refresh, err := GenerateRefreshToken(userID, secret)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

// ParseToken validates a token and returns its claims. The secret is
// passed explicitly so tests can supply their own signing key. Works
// for both access and refresh tokens since both use the same secret.
func ParseToken(tokenString, secret string, expectedType TokenType) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	if claims.TokenType != expectedType {
		return nil, fmt.Errorf("invalid token type: expected %s", expectedType)
	}

	return claims, nil
}

// ParseRefreshToken is a convenience wrapper that only validates the
// standard claims (expiry, issuer, subject) without requiring custom
// claims like Role. Refresh tokens carry only sub (user ID), not role.
func ParseRefreshToken(tokenString, secret string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*RefreshTokenClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid refresh token")
	}
	if claims.TokenType != RefreshTokenType {
		return "", fmt.Errorf("invalid refresh token type")
	}

	if claims.Subject == "" {
		return "", fmt.Errorf("missing subject in refresh token")
	}

	return claims.Subject, nil
}

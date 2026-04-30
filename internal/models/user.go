package models

import "time"

// User is the canonical representation of an account in the system. Every
// user, customer or admin shares this struct. Role distinguishes them.
// Sensitive identity fields remain server-only by default; handlers that need
// to expose them, such as the profile endpoint, should build explicit payloads.
type User struct {
	ID              string     `json:"id"`
	PasswordHash    string     `json:"-"` // never serialized, never leaves the server
	DisplayName     string     `json:"display_name"`
	PhoneNumber     string     `json:"phone_number"`
	StaffNumber     string     `json:"-"`
	Institution     string     `json:"-"`
	GhanaCardNumber string     `json:"-"`
	IsActive        bool       `json:"-"`
	OTPHash         *string    `json:"-"`
	OTPExpiresAt    *time.Time `json:"-"`
	Role            string     `json:"role"` // "customer" or "admin"
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

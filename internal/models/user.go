package models

import "time"

// User is the canonical representation of an account in the system. Every
// user, customer or admin shares this struct. Role distinguishes them.
// The Phone field is optional at signup but required before an application
// can be submitted; the service layer enforces this, not the database.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // never serialized, never leaves the server
	DisplayName  string    `json:"display_name"`
	Phone        *string   `json:"phone,omitempty"`
	Role         string    `json:"role"` // "customer" or "admin"
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

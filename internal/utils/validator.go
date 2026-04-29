package utils

import (
	"regexp"
	"strings"
)

// ValidateEmail checks that an email string is non-empty and roughly
// matches an email pattern. This is not RFC 5322 compliant, it's a
// pragmatic check that catches typos without rejecting valid but unusual
// addresses. If you need full compliance, swap the regex.
func ValidateEmail(email string) bool {
	if strings.TrimSpace(email) == "" {
		return false
	}
	pattern := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(pattern, email)
	return match
}

// ValidatePassword ensures a password is at least 8 characters. We don't
// enforce complexity rules (uppercase, numbers, symbols), NIST guidelines
// recommend length over complexity, and the bcrypt cost factor provides
// the real security. If a client demands stricter rules, add them here.
func ValidatePassword(password string) bool {
	return len(password) >= 8
}

// ValidateDisplayName ensures the display name is not blank and has a
// reasonable maximum length to prevent abuse.
func ValidateDisplayName(name string) bool {
	trimmed := strings.TrimSpace(name)
	return len(trimmed) >= 2 && len(trimmed) <= 100
}

// NormalizePhone removes common formatting characters so we store a stable
// representation regardless of whether the client submits spaces, dashes, or
// parentheses. A leading + is preserved for international numbers.
func NormalizePhone(phone string) string {
	trimmed := strings.TrimSpace(phone)
	if trimmed == "" {
		return ""
	}

	var b strings.Builder
	for i, r := range trimmed {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' && i == 0 && b.Len() == 0:
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '(' || r == ')':
			continue
		default:
			return trimmed
		}
	}

	return b.String()
}

// ValidatePhone accepts pragmatic phone numbers rather than country-specific
// formats. After normalization, it requires 7-15 digits with an optional
// leading + for international notation.
func ValidatePhone(phone string) bool {
	normalized := NormalizePhone(phone)
	if normalized == "" {
		return false
	}

	digits := normalized
	if normalized[0] == '+' {
		digits = normalized[1:]
	}

	if len(digits) < 7 || len(digits) > 15 {
		return false
	}

	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

// ValidateRequired checks that a string field is not empty after trimming.
// Used for staff_number, mandate_number, and other required application
// fields.
func ValidateRequired(field string) bool {
	return strings.TrimSpace(field) != ""
}

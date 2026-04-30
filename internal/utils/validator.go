package utils

import "strings"

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

// NormalizePhone removes formatting noise and canonicalizes Ghana numbers to
// +233XXXXXXXXX so 054..., 23354..., and +23354... all compare the same.
// Non-Ghana numbers keep a pragmatic normalized shape with punctuation
// removed and an optional leading + preserved.
func NormalizePhone(phone string) string {
	trimmed := strings.TrimSpace(phone)
	if trimmed == "" {
		return ""
	}

	var digits strings.Builder
	sawLeadingPlus := false
	for i, r := range trimmed {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r == '+' && i == 0 && digits.Len() == 0:
			sawLeadingPlus = true
		case r == ' ' || r == '-' || r == '(' || r == ')':
			continue
		default:
			return trimmed
		}
	}

	normalizedDigits := digits.String()
	if normalizedDigits == "" {
		return ""
	}

	switch {
	case len(normalizedDigits) == 10 && normalizedDigits[0] == '0':
		return "+233" + normalizedDigits[1:]
	case len(normalizedDigits) == 12 && strings.HasPrefix(normalizedDigits, "233"):
		return "+" + normalizedDigits
	case sawLeadingPlus:
		return "+" + normalizedDigits
	default:
		return normalizedDigits
	}
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

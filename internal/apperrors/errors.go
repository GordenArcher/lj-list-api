package apperrors

// Kind is a stable error category used for HTTP mapping.
type Kind string

const (
	KindInvalidRequest Kind = "INVALID_REQUEST"
	KindValidation     Kind = "VALIDATION_ERROR"
	KindMinimumOrder   Kind = "MINIMUM_ORDER_NOT_MET"
	KindUnauthorized   Kind = "UNAUTHORIZED"
	KindForbidden      Kind = "FORBIDDEN"
	KindNotFound       Kind = "NOT_FOUND"
	KindConflict       Kind = "CONFLICT"
	KindRateLimited    Kind = "RATE_LIMIT_EXCEEDED"
	KindInternal       Kind = "INTERNAL_ERROR"
)

// Error is the shared domain error shape across handlers/services/middleware.
// Message is safe to expose to clients. Err is optional internal context.
type Error struct {
	Kind    Kind
	Message string
	Details map[string][]string
	Err     error
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Kind)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func New(kind Kind, message string, details map[string][]string) *Error {
	return &Error{
		Kind:    kind,
		Message: message,
		Details: details,
	}
}

func Wrap(kind Kind, message string, err error) *Error {
	return &Error{
		Kind:    kind,
		Message: message,
		Err:     err,
	}
}

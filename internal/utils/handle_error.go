package utils

import (
	"errors"
	"net/http"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/gin-gonic/gin"
)

// HandleError maps domain errors to the standard API envelope.
// Unknown errors are treated as internal errors.
func HandleError(c *gin.Context, err error, fallbackMessage string) {
	var appErr *apperrors.Error
	if errors.As(err, &appErr) {
		status, code := mapKindToHTTP(appErr.Kind)
		message := appErr.Message
		if message == "" {
			if fallbackMessage != "" {
				message = fallbackMessage
			} else {
				message = defaultMessage(appErr.Kind)
			}
		}
		Error(c, status, code, message, appErr.Details)
		return
	}

	message := fallbackMessage
	if message == "" {
		message = defaultMessage(apperrors.KindInternal)
	}
	Error(c, http.StatusInternalServerError, string(apperrors.KindInternal), message, nil)
}

func mapKindToHTTP(kind apperrors.Kind) (int, string) {
	switch kind {
	case apperrors.KindInvalidRequest:
		return http.StatusUnprocessableEntity, string(apperrors.KindInvalidRequest)
	case apperrors.KindValidation:
		return http.StatusUnprocessableEntity, string(apperrors.KindValidation)
	case apperrors.KindMinimumOrder:
		return http.StatusUnprocessableEntity, string(apperrors.KindMinimumOrder)
	case apperrors.KindUnauthorized:
		return http.StatusUnauthorized, string(apperrors.KindUnauthorized)
	case apperrors.KindForbidden:
		return http.StatusForbidden, string(apperrors.KindForbidden)
	case apperrors.KindNotFound:
		return http.StatusNotFound, string(apperrors.KindNotFound)
	case apperrors.KindConflict:
		return http.StatusConflict, string(apperrors.KindConflict)
	case apperrors.KindRateLimited:
		return http.StatusTooManyRequests, string(apperrors.KindRateLimited)
	default:
		return http.StatusInternalServerError, string(apperrors.KindInternal)
	}
}

func defaultMessage(kind apperrors.Kind) string {
	switch kind {
	case apperrors.KindInvalidRequest:
		return "Invalid request"
	case apperrors.KindValidation:
		return "Validation failed"
	case apperrors.KindMinimumOrder:
		return "Minimum order not met"
	case apperrors.KindUnauthorized:
		return "Authentication required"
	case apperrors.KindForbidden:
		return "Forbidden"
	case apperrors.KindNotFound:
		return "Resource not found"
	case apperrors.KindConflict:
		return "Conflict"
	case apperrors.KindRateLimited:
		return "Too many requests. Please try again later."
	default:
		return "Internal server error"
	}
}

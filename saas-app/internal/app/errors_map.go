package app

import (
	"errors"
	"net/http"

	"saas/internal/dao"
	"saas/internal/service"
)

// httpStatusOf maps a service error to an HTTP status code.
func httpStatusOf(err error) int {
	var badInput service.ValidationError
	switch {
	case errors.As(err, &badInput):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrForbidden), errors.Is(err, service.ErrNotPlatformAdmin):
		return http.StatusForbidden
	case errors.Is(err, service.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, service.ErrQuotaExceeded):
		return http.StatusPaymentRequired
	case errors.Is(err, dao.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, service.ErrInvalidCredentials):
		return http.StatusUnauthorized
	default:
		if _, ok := errStatusMessages(err); ok {
			return http.StatusBadRequest
		}
		return http.StatusInternalServerError
	}
}

// errStatus returns an app-level code mirroring the HTTP status.
func errStatus(err error) int {
	return httpStatusOf(err)
}

// errStatusMessages detects "business" errors that map to 400 (user-fixable).
func errStatusMessages(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	s := err.Error()
	subs := []string{
		"is required", "already", "invalid", "expired", "used up",
		"you cannot remove yourself", "password", "nickname",
	}
	for _, sub := range subs {
		if contains(s, sub) {
			return s, true
		}
	}
	return "", false
}

func contains(s, sub string) bool {
	if sub == "" {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
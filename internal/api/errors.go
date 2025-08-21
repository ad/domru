package api

import (
	"errors"
	"net/http"

	"github.com/ad/domru/internal/upstream"
)

// Sentinel errors (Step 5)
var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrBadRequest   = errors.New("bad request")
	ErrTokenExpired = errors.New("token expired")
	ErrUnknown      = errors.New("unknown upstream error")
)

// mapUpstreamErr converts upstream errors to sentinel ones
func mapUpstreamErr(err error) error {
	if err == nil {
		return nil
	}
	if ue, ok := err.(*upstream.UpstreamError); ok {
		switch ue.StatusCode {
		case http.StatusBadRequest:
			return ErrBadRequest
		case http.StatusUnauthorized:
			return ErrUnauthorized
		case http.StatusForbidden:
			return ErrForbidden
		case http.StatusNotFound:
			return ErrNotFound
		case 409:
			return ErrTokenExpired
		default:
			return ErrUnknown
		}
	}
	return err
}

// StatusFromError maps sentinel error to HTTP code
func StatusFromError(err error) int {
	switch err {
	case ErrBadRequest:
		return http.StatusBadRequest
	case ErrUnauthorized, ErrForbidden, ErrTokenExpired:
		return http.StatusUnauthorized
	case ErrNotFound:
		return http.StatusNotFound
	}
	return http.StatusBadGateway
}

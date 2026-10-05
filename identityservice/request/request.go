// Package request defines the request identity type and validation.
package request

import "strings"

// ErrInvalidRequest is returned when a request lacks a valid identity.
var ErrInvalidRequest = &invalidRequestError{}

type invalidRequestError struct{}

func (*invalidRequestError) Error() string { return "invalid request: empty request ID" }

// Request carries the identity and kind for an operation.
type Request struct {
	RequestID string
	Kind      string
}

// Validate checks that the request has a non-empty identity.
func (r Request) Validate() error {
	if r.RequestID == "" {
		return ErrInvalidRequest
	}
	return nil
}

// Normalize returns the normalized form of kind by lowercasing and trimming whitespace.
func Normalize(kind string) string {
	return normalizeKind(kind)
}

// normalizeKind lowercases and trims whitespace from kind.
func normalizeKind(kind string) string {
	return strings.TrimSpace(strings.ToLower(kind))
}

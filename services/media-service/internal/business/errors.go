package business

import "errors"

var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrNotFound       = errors.New("not found")
	ErrForbidden      = errors.New("forbidden")
	ErrConflict       = errors.New("conflict")
	ErrRateLimited    = errors.New("rate limited")
	ErrObjectMissing  = errors.New("storage object missing")
	ErrMimeMismatch   = errors.New("mime mismatch")
	ErrSizeExceeded   = errors.New("size exceeded")
	ErrInvalidStatus  = errors.New("invalid status")
)

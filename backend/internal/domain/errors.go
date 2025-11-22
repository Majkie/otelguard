package domain

import "errors"

// Common domain errors
var (
	ErrNotFound       = errors.New("resource not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrValidation     = errors.New("validation error")
	ErrConflict       = errors.New("resource conflict")
	ErrInternal       = errors.New("internal error")
	ErrInvalidInput   = errors.New("invalid input")
	ErrDuplicateEntry = errors.New("duplicate entry")
)

// ValidationError contains field-level validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

// Error implements the error interface
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "validation failed"
	}
	return ve[0].Message
}

// Add adds a new validation error
func (ve *ValidationErrors) Add(field, message string) {
	*ve = append(*ve, ValidationError{Field: field, Message: message})
}

// HasErrors returns true if there are validation errors
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

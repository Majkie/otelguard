package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/otelguard/otelguard/internal/domain"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string      `json:"error"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// ErrorHandler returns middleware for handling errors
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there were any errors during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			handleError(c, err)
		}
	}
}

// handleError maps domain errors to HTTP responses
func handleError(c *gin.Context, err error) {
	var validationErrors domain.ValidationErrors

	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Resource not found",
		})
	case errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Access denied",
		})
	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "conflict",
			Message: "Resource already exists",
		})
	case errors.Is(err, domain.ErrDuplicateEntry):
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "duplicate_entry",
			Message: "Resource already exists",
		})
	case errors.Is(err, domain.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_input",
			Message: err.Error(),
		})
	case errors.As(err, &validationErrors):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Validation failed",
			Details: validationErrors,
		})
	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "An internal error occurred",
		})
	}
}

// RespondWithError sends an error response
func RespondWithError(c *gin.Context, statusCode int, errType, message string) {
	c.JSON(statusCode, ErrorResponse{
		Error:   errType,
		Message: message,
	})
}

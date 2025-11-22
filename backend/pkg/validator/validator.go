// Package validator provides request validation utilities
package validator

import (
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var (
	once     sync.Once
	validate *validator.Validate
)

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// ValidationErrors is a slice of ValidationError
type ValidationErrors []ValidationError

// Error implements the error interface
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "validation failed"
	}
	var messages []string
	for _, e := range ve {
		messages = append(messages, e.Message)
	}
	return strings.Join(messages, "; ")
}

// Init initializes the validator with custom validators
func Init() {
	once.Do(func() {
		if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			validate = v

			// Register custom tag name function to use JSON tags
			v.RegisterTagNameFunc(func(fld reflect.StructField) string {
				name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
				if name == "-" {
					return ""
				}
				return name
			})

			// Register custom validators
			_ = v.RegisterValidation("uuid", validateUUID)
			_ = v.RegisterValidation("slug", validateSlug)
			_ = v.RegisterValidation("status", validateStatus)
			_ = v.RegisterValidation("spantype", validateSpanType)
			_ = v.RegisterValidation("datatype", validateDataType)
			_ = v.RegisterValidation("source", validateSource)
		}
	})
}

// Get returns the validator instance
func Get() *validator.Validate {
	Init()
	return validate
}

// ParseValidationErrors converts validator.ValidationErrors to ValidationErrors
func ParseValidationErrors(err error) ValidationErrors {
	var validationErrors ValidationErrors

	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, e := range ve {
			field := e.Field()
			tag := e.Tag()

			validationErrors = append(validationErrors, ValidationError{
				Field:   field,
				Tag:     tag,
				Message: formatErrorMessage(field, tag, e.Param()),
			})
		}
	}

	return validationErrors
}

// formatErrorMessage creates a human-readable error message
func formatErrorMessage(field, tag, param string) string {
	switch tag {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email address"
	case "min":
		return field + " must be at least " + param + " characters"
	case "max":
		return field + " must be at most " + param + " characters"
	case "uuid":
		return field + " must be a valid UUID"
	case "slug":
		return field + " must be a valid slug (lowercase letters, numbers, and hyphens)"
	case "oneof":
		return field + " must be one of: " + param
	case "url":
		return field + " must be a valid URL"
	case "status":
		return field + " must be a valid status (success, error, pending)"
	case "spantype":
		return field + " must be a valid span type (llm, retrieval, tool, agent, embedding, custom)"
	case "datatype":
		return field + " must be a valid data type (numeric, boolean, categorical)"
	case "source":
		return field + " must be a valid source (api, llm_judge, human, user_feedback)"
	case "gte":
		return field + " must be greater than or equal to " + param
	case "lte":
		return field + " must be less than or equal to " + param
	case "gt":
		return field + " must be greater than " + param
	case "lt":
		return field + " must be less than " + param
	default:
		return field + " is invalid"
	}
}

// Custom validators

// validateUUID checks if a string is a valid UUID
func validateUUID(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	if val == "" {
		return true // Let required handle empty
	}
	// UUID pattern: 8-4-4-4-12 hex chars
	if len(val) != 36 {
		return false
	}
	parts := strings.Split(val, "-")
	if len(parts) != 5 {
		return false
	}
	lengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != lengths[i] {
			return false
		}
		for _, c := range part {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

// validateSlug checks if a string is a valid slug
func validateSlug(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	if val == "" {
		return true
	}
	for _, c := range val {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	// Cannot start or end with hyphen
	if val[0] == '-' || val[len(val)-1] == '-' {
		return false
	}
	return true
}

// validateStatus checks if a string is a valid status
func validateStatus(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	if val == "" {
		return true // Default will be set
	}
	validStatuses := map[string]bool{
		"success": true,
		"error":   true,
		"pending": true,
	}
	return validStatuses[val]
}

// validateSpanType checks if a string is a valid span type
func validateSpanType(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	validTypes := map[string]bool{
		"llm":       true,
		"retrieval": true,
		"tool":      true,
		"agent":     true,
		"embedding": true,
		"custom":    true,
	}
	return validTypes[val]
}

// validateDataType checks if a string is a valid data type for scores
func validateDataType(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	validTypes := map[string]bool{
		"numeric":     true,
		"boolean":     true,
		"categorical": true,
	}
	return validTypes[val]
}

// validateSource checks if a string is a valid score source
func validateSource(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	if val == "" {
		return true // Default to "api"
	}
	validSources := map[string]bool{
		"api":           true,
		"llm_judge":     true,
		"human":         true,
		"user_feedback": true,
	}
	return validSources[val]
}

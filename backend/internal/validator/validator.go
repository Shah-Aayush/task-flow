package validator

import (
	"errors"
	"reflect"
	"strings"

	"github.com/Shah-Aayush/task-flow-zomato-takehome/backend/internal/domain"
	"github.com/go-playground/validator/v10"
)

// Validator wraps go-playground/validator to extract structured field errors.
// This is the only place that knows about the validator library — everything
// else in the codebase works with the domain.ValidationError type.
type Validator struct {
	v *validator.Validate
}

// New creates a configured Validator instance.
func New() *Validator {
	v := validator.New()
	// Use JSON tag names in error messages (e.g., "email" not "Email")
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return &Validator{v: v}
}

// Validate runs struct validation and returns a *domain.ValidationError if
// any fields fail, or nil if validation passes.
// This converts go-playground's specific error types into our domain error type
// so the rest of the codebase doesn't depend on the validator library.
func (val *Validator) Validate(s interface{}) error {
	err := val.v.Struct(s)
	if err == nil {
		return nil
	}

	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		// Unexpected error from the validator itself
		return err
	}

	fields := make(map[string]string)
	for _, fe := range validationErrors {
		field := fe.Field()
		fields[field] = translateTag(fe.Tag(), fe.Param())
	}

	return domain.NewValidationError(fields)
}

// translateTag converts a validator tag name to a human-readable error message.
func translateTag(tag, param string) string {
	switch tag {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "must be at least " + param + " characters"
	case "max":
		return "must be at most " + param + " characters"
	case "oneof":
		return "must be one of: " + strings.ReplaceAll(param, " ", ", ")
	default:
		return "is invalid"
	}
}

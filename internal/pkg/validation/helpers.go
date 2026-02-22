package validation

import (
	"github.com/go-playground/validator/v10"
)

// Errors extracts field-level validation errors from a validator.ValidationErrors
// and returns them as a map of field name to failed validation tag.
func Errors(err error) map[string]string {
	errs := make(map[string]string)
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, e := range ve {
			errs[e.Field()] = e.Tag()
		}
	}
	return errs
}

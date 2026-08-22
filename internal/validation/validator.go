// Package validation wraps github.com/go-playground/validator/v10 with a
// reusable check for dangerous control characters in string fields, so HTTP
// server handlers can detect them and return an error to the caller instead
// of trying to sanitize the input.
package validation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	bodyValidator *validator.Validate
)

func init() {
	bodyValidator = validator.New()
	err := bodyValidator.RegisterValidation(controlCharsTagName, func(fl validator.FieldLevel) bool {
		if fl.Field().Kind() != reflect.String {
			return true // only validate string fields
		}
		return !ContainsControlChars(fl.Field().String())
	})
	if err != nil {
		panic("validation: failed to register " + controlCharsTagName + " tag: " + err.Error())
	}
}

// ValidateBody valida a estrutura do corpo e retorna todos os erros detectados
// em um formato amigável. Retorna nil se não houver erros.
func ValidateBody(body any) error {
	if err := bodyValidator.Struct(body); err != nil {
		var valErrs validator.ValidationErrors
		if errors.As(err, &valErrs) {
			return formatValidationErrors(valErrs)
		}
		return fmt.Errorf("validation error: %w", err)
	}
	return nil
}

// formatValidationErrors converte os erros do validator para um único erro
// amigável no formato "field: message; field: message".
func formatValidationErrors(valErrs validator.ValidationErrors) error {
	messages := make([]string, 0, len(valErrs))
	for _, fe := range valErrs {
		switch fe.Tag() {
		case controlCharsTagName:
			messages = append(messages, fmt.Sprintf("%s: contains invalid characters", fe.Field()))
		default:
			messages = append(messages, fmt.Sprintf("%s: invalid value", fe.Field()))
		}
	}
	return fmt.Errorf("validation errors: %s", strings.Join(messages, "; "))
}

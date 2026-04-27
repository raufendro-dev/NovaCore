package validation

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func Struct(value any) map[string]string {
	if err := validate.Struct(value); err != nil {
		out := map[string]string{}
		for _, e := range err.(validator.ValidationErrors) {
			out[strings.ToLower(e.Field())] = e.Tag()
		}
		return out
	}
	return nil
}

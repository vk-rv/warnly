package server

import (
	"fmt"
	"net/http"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	validate "github.com/go-playground/validator/v10"
	entranslations "github.com/go-playground/validator/v10/translations/en"
	"github.com/gorilla/schema"
)

const (
	// htmxHeader is the HTTP header used by HTMX to indicate an HTMX request.
	htmxHeader = "Hx-Request"
	// htmxTarget is the HTTP header used by HTMX to indicate the target element for the response.
	htmxTarget = "Hx-Target"
)

var (
	decoder   = schema.NewDecoder()
	validator *validate.Validate
	trans     ut.Translator
)

func initValidator() {
	english := en.New()
	uni := ut.New(english, english)
	trans, _ = uni.GetTranslator("en")
	validator = validate.New(validate.WithRequiredStructEnabled())
	if err := entranslations.RegisterDefaultTranslations(validator, trans); err != nil {
		panic(err)
	}
}

//nolint:ireturn,errorlint // will be removed in the future.
func decodeValid[T any](r *http.Request) (obj T, problems map[string]string, err error) {
	if err := r.ParseForm(); err != nil {
		return obj, nil, fmt.Errorf("parse form http request: %w", err)
	}

	if err := decoder.Decode(&obj, r.PostForm); err != nil {
		return obj, nil, fmt.Errorf("decode form: %w", err)
	}

	if err := validator.Struct(obj); err != nil {
		errs, ok := err.(validate.ValidationErrors)
		if !ok {
			return obj, nil, err
		}
		problems := errs.Translate(trans)

		return obj, problems, fmt.Errorf("invalid %T: %d problems", obj, len(problems))
	}

	return obj, nil, nil
}

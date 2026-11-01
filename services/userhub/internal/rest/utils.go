package rest

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"mime"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/mold/v4"
	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/microcosm-cc/bluemonday"
)

var requestNormalizer = func() *mold.Transformer {
	normalizer := modifiers.New()
	policy := bluemonday.StrictPolicy()
	normalizer.Register("strip_html", func(_ context.Context, field mold.FieldLevel) error {
		value := field.Field()
		// mold unwraps populated pointers but leaves omitted optional fields as nil pointers.
		if value.Kind() == reflect.Pointer && value.Type().Elem().Kind() == reflect.String && value.IsNil() {
			return nil
		}
		if value.Kind() != reflect.String {
			return errors.New("strip_html requires a string field")
		}

		var sanitized strings.Builder
		if err := policy.SanitizeReaderToWriter(strings.NewReader(value.String()), &sanitized); err != nil {
			return fmt.Errorf("strip html: %w", err)
		}
		// Decode entities for plain-text storage, not for insertion as trusted HTML.
		value.SetString(html.UnescapeString(sanitized.String()))
		return nil
	})
	return normalizer
}()

var requestValidator = func() *validator.Validate {
	validate := validator.New(validator.WithRequiredStructEnabled())
	slugPattern := regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	if err := validate.RegisterValidation("slug", func(field validator.FieldLevel) bool {
		return field.Field().Kind() == reflect.String && slugPattern.MatchString(field.Field().String())
	}); err != nil {
		panic(fmt.Errorf("register slug validation: %w", err))
	}

	// Report JSON field names instead of Go names
	// in validation errors. This changes error field names, not validation rules.
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		tag := field.Tag.Get("json")
		if tag == "-" {
			return ""
		}
		name, _, _ := strings.Cut(tag, ",")
		return name
	})
	return validate
}()

type messageResponse struct {
	OK      bool                   `json:"ok"`
	Message string                 `json:"message"`
	Fields  []validationFieldError `json:"fields,omitempty"`
}

func successMessage(message string) messageResponse {
	return messageResponse{OK: true, Message: message}
}

func errorMessage(message string) messageResponse {
	return messageResponse{OK: false, Message: message}
}

type validationFieldError struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
	Param string `json:"param"`
}

// normalizeRequest applies mod tags in place before validation.
// On error it writes a JSON 500 response. The handler must return immediately.
func normalizeRequest(w http.ResponseWriter, r *http.Request, request any) error {
	if err := requestNormalizer.Struct(r.Context(), request); err != nil {
		slog.ErrorContext(r.Context(), "normalize request", "error", err, "tag", "rest")
		writeJSON(w, r, http.StatusInternalServerError,
			errorMessage(http.StatusText(http.StatusInternalServerError)),
		)
		return fmt.Errorf("normalize request: %w", err)
	}
	return nil
}

// validateRequestBody writes a JSON 400 response for invalid request body fields.
// Internal validation errors produce a 500 response. The handler must return on error.
func validateRequestBody(w http.ResponseWriter, r *http.Request, request any) error {
	err := requestValidator.StructCtx(r.Context(), request)
	if err == nil {
		return nil
	}

	fieldErrors, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		slog.ErrorContext(r.Context(), "validate request", "error", err, "tag", "rest")
		writeJSON(w, r, http.StatusInternalServerError,
			errorMessage(http.StatusText(http.StatusInternalServerError)),
		)
		return fmt.Errorf("validate request: %w", err)
	}

	fields := make([]validationFieldError, 0, len(fieldErrors))
	for _, fieldError := range fieldErrors {
		fields = append(fields, validationFieldError{
			Field: fieldError.Field(),
			Rule:  fieldError.Tag(),
			Param: fieldError.Param(),
		})
	}

	slog.WarnContext(r.Context(), "invalid request fields", "field_count", len(fields), "tag", "rest")
	response := errorMessage("Request validation failed")
	response.Fields = fields
	writeJSON(w, r, http.StatusBadRequest, response)
	return fmt.Errorf("validate request: %w", err)
}

// validateRequestParam writes an empty 404 response for an invalid URL parameter.
// Internal validation errors produce a 500 response. The handler must return on error.
func validateRequestParam(w http.ResponseWriter, r *http.Request, value string, rules string) error {
	err := requestValidator.VarCtx(r.Context(), value, rules)
	if err == nil {
		return nil
	}

	if _, ok := errors.AsType[validator.ValidationErrors](err); !ok {
		slog.ErrorContext(r.Context(), "validate request parameter", "error", err, "tag", "rest")
		writeJSON(w, r, http.StatusInternalServerError,
			errorMessage(http.StatusText(http.StatusInternalServerError)),
		)
		return fmt.Errorf("validate request parameter: %w", err)
	}

	slog.WarnContext(r.Context(), "invalid request parameter", "error", err, "tag", "rest")
	writeJSON(w, r, http.StatusNotFound, nil)
	return fmt.Errorf("validate request parameter: %w", err)
}

// readJSON decodes a single JSON value from the request body into dst.
// dst must be a non-nil pointer.
// It requires an application/json Content-Type, optionally with parameters.
// On error it logs the failure and writes a 415 or 400 response.
// The handler must return immediately after an error.
func readJSON[T any](w http.ResponseWriter, r *http.Request, dst *T) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		var contentTypeErr error
		if err != nil {
			contentTypeErr = fmt.Errorf("parse request content type: %w", err)
		} else {
			contentTypeErr = errors.New("request content type must be application/json")
		}

		slog.WarnContext(r.Context(), "invalid request content type",
			"error", contentTypeErr,
			"tag", "rest",
		)
		writeJSON(w, r, http.StatusUnsupportedMediaType,
			errorMessage("Content-Type must be application/json"),
		)
		return contentTypeErr
	}

	if err := json.UnmarshalRead(r.Body, dst); err != nil {
		slog.WarnContext(r.Context(), "read json request",
			"error", err,
			"tag", "rest",
		)

		writeJSON(w, r, http.StatusBadRequest, errorMessage("Invalid JSON request body"))
		return fmt.Errorf("read json request: %w", err)
	}

	return nil
}

// readAndValidateJSON decodes, normalizes, and validates dst in that order.
// On error a response has already been written; the handler must return immediately.
func readAndValidateJSON[T any](w http.ResponseWriter, r *http.Request, dst *T) error {
	if err := readJSON(w, r, dst); err != nil {
		return err
	}
	if err := normalizeRequest(w, r, dst); err != nil {
		return err
	}
	return validateRequestBody(w, r, dst)
}

// writeJSON writes a JSON response with the given status and body.
// Call it at the end of a handler, or return immediately after calling it.
func writeJSON(w http.ResponseWriter, r *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")

	if body == nil {
		w.WriteHeader(status)
		return
	}

	data, err := json.Marshal(body)
	if err != nil {
		slog.ErrorContext(r.Context(), "marshal json response",
			"error", err,
			"tag", "rest",
		)

		status = http.StatusInternalServerError
		data = []byte(`{"ok":false,"message":"Internal Server Error"}`)
	}

	w.WriteHeader(status)

	if _, err := w.Write(data); err != nil {
		slog.ErrorContext(r.Context(), "write json response",
			"error", err,
			"tag", "rest",
		)
	}
}

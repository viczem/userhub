package rest

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestReadAndValidateJSON(t *testing.T) {
	for _, tc := range []struct {
		name        string
		body        string
		contentType string
		status      int
		message     string
		fields      []validationFieldError
	}{
		{
			name:        "valid request",
			body:        `{"name":"admin","is_default":true}`,
			contentType: "application/json",
		},
		{
			name:        "invalid fields",
			body:        `{"name":""}`,
			contentType: "application/json",
			status:      http.StatusBadRequest,
			message:     "Request validation failed",
			fields:      []validationFieldError{{Field: "name", Rule: "required"}},
		},
		{
			name:        "decoding error stops validation",
			body:        `{"name":`,
			contentType: "application/json",
			status:      http.StatusBadRequest,
			message:     "Invalid JSON request body",
		},
		{
			name:        "content type error stops decoding and validation",
			body:        `{"name":`,
			contentType: "text/plain",
			status:      http.StatusUnsupportedMediaType,
			message:     "Content-Type must be application/json",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()

			var request struct {
				Name      string `json:"name" validate:"required"`
				IsDefault bool   `json:"is_default"`
			}

			err := readAndValidateJSON(w, r, &request)
			if tc.status == 0 {
				if err != nil {
					t.Fatalf("readAndValidateJSON: %v", err)
				}
				if request.Name != "admin" || !request.IsDefault {
					t.Errorf("decoded request = %+v, want admin with is_default true", request)
				}
				if w.Body.Len() != 0 || len(w.Header()) != 0 || w.Flushed {
					t.Error("successful reading and validation must not write a response")
				}
				return
			}
			if err == nil {
				t.Fatal("expected an error")
			}
			_, validationFailed := errors.AsType[validator.ValidationErrors](err)
			if validationFailed != (len(tc.fields) > 0) {
				t.Errorf("validation error = %t, want %t", validationFailed, len(tc.fields) > 0)
			}
			if len(tc.fields) == 0 {
				assertJSONErrorResponse(t, w, tc.status, tc.message)
				return
			}

			if w.Code != tc.status {
				t.Errorf("status = %d, want %d", w.Code, tc.status)
			}
			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("content type = %q, want application/json", got)
			}

			var response messageResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response, json.RejectUnknownMembers(true)); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.OK || response.Message != tc.message || !slices.Equal(response.Fields, tc.fields) {
				t.Errorf("response = %+v, want ok false, message %q and fields %+v", response, tc.message, tc.fields)
			}
		})
	}
}

func TestNormalizeRequest(t *testing.T) {
	type request struct {
		Name     string `mod:"trim" validate:"required"`
		Password string
		Enabled  bool
	}

	for _, tc := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "surrounding whitespace", input: " \tadmin\n ", want: "admin"},
		{name: "Unicode whitespace", input: "\u00a0admin\u2003", want: "admin"},
		{name: "internal whitespace preserved", input: " Content  Manager ", want: "Content  Manager"},
		{name: "whitespace only", input: " \t\n "},
		{name: "empty string"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			input := request{Name: tc.input, Password: "  secret  ", Enabled: true}

			if err := normalizeRequest(w, r, &input); err != nil {
				t.Fatalf("normalizeRequest: %v", err)
			}
			if input.Name != tc.want {
				t.Errorf("name = %q, want %q", input.Name, tc.want)
			}
			if input.Password != "  secret  " || !input.Enabled {
				t.Errorf("untagged fields changed: %+v", input)
			}
			if w.Body.Len() != 0 || len(w.Header()) != 0 || w.Flushed {
				t.Error("successful normalization must not write a response")
			}
		})
	}
}

func TestNormalizeRequestInternalError(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request any
	}{
		{name: "nil"},
		{name: "nil pointer", request: (*struct{ Name string })(nil)},
		{name: "struct value", request: struct{ Name string }{Name: "admin"}},
		{name: "non-struct pointer", request: new(string)},
		{
			name: "strip_html on non-string field",
			request: &struct {
				Value int `mod:"strip_html"`
			}{Value: 42},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", nil)

			err := normalizeRequest(w, r, tc.request)
			if err == nil {
				t.Fatal("expected a normalization error")
			}
			if errors.Unwrap(err) == nil || !strings.HasPrefix(err.Error(), "normalize request: ") {
				t.Errorf("error = %v, want wrapped normalization error", err)
			}
			assertJSONErrorResponse(t, w, http.StatusInternalServerError, "Internal Server Error")
		})
	}
}

func TestStripHTML(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string"},
		{name: "plain text", input: "Content Manager", want: "Content Manager"},
		{name: "nested tags", input: "<div><b>Content</b> <em>Manager</em></div>", want: "Content Manager"},
		{name: "attributes removed", input: `<a href="javascript:alert(1)" onclick="alert(2)">Admin</a>`, want: "Admin"},
		{name: "script content removed", input: `<script>alert("secret")</script>Admin`, want: "Admin"},
		{name: "style content removed", input: "<style>body { display: none; }</style>Admin", want: "Admin"},
		{name: "comments removed", input: "<!-- private note -->Admin", want: "Admin"},
		{name: "image removed", input: `<img src="invalid" onerror="alert(1)">Admin`, want: "Admin"},
		{name: "tags only", input: "<div><br></div>"},
		{name: "entities decoded", input: "R&amp;D &#65; &#x42;", want: "R&D A B"},
		{name: "encoded markup remains literal text", input: "&lt;b&gt;Admin&lt;/b&gt;", want: "<b>Admin</b>"},
		{name: "inline markup does not split words", input: "Ad<b>mi</b>n", want: "Admin"},
		{name: "no separators inserted", input: "<p>Hello</p><p>World</p>", want: "HelloWorld"},
		{name: "whitespace preserved", input: " <b>Admin</b> \n", want: " Admin \n"},
		{name: "unclosed tag", input: "<b>Admin", want: "Admin"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := struct {
				Value string `mod:"strip_html"`
			}{Value: tc.input}

			if err := requestNormalizer.Struct(context.Background(), &input); err != nil {
				t.Fatalf("strip_html: %v", err)
			}
			if input.Value != tc.want {
				t.Errorf("value = %q, want %q", input.Value, tc.want)
			}
		})
	}
}

func TestStripHTMLInvalidFieldType(t *testing.T) {
	input := struct {
		Value int `mod:"strip_html"`
	}{Value: 42}

	err := requestNormalizer.Struct(context.Background(), &input)
	if err == nil || !strings.Contains(err.Error(), "strip_html requires a string field") {
		t.Fatalf("error = %v, want invalid field type error", err)
	}
	if input.Value != 42 {
		t.Errorf("value = %d, want unchanged 42", input.Value)
	}
}

func TestReadAndValidateJSONNormalization(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		want    string
		wantErr bool
	}{
		{name: "HTML and whitespace removed before length validation", body: `{"name":" <b>admin</b> "}`, want: "admin"},
		{name: "whitespace becomes empty before required validation", body: `{"name":"   "}`, wantErr: true},
		{name: "script becomes empty before required validation", body: `{"name":"<script>alert(1)</script>"}`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			var input struct {
				Name string `json:"name" mod:"strip_html,trim" validate:"required,max=5"`
			}

			err := readAndValidateJSON(w, r, &input)
			if input.Name != tc.want {
				t.Errorf("name = %q, want %q", input.Name, tc.want)
			}
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("readAndValidateJSON: %v", err)
				}
				if w.Body.Len() != 0 || len(w.Header()) != 0 || w.Flushed {
					t.Error("successful normalization and validation must not write a response")
				}
				return
			}

			fieldErrors, ok := errors.AsType[validator.ValidationErrors](err)
			if !ok || len(fieldErrors) != 1 || fieldErrors[0].Tag() != "required" {
				t.Fatalf("error = %v, want a required validation error", err)
			}
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", w.Code)
			}
		})
	}
}

func TestValidateRequestBodyValid(t *testing.T) {
	type request struct {
		Name      string `json:"name" validate:"required,max=5"`
		IsDefault bool   `json:"is_default"`
	}

	for _, tc := range []struct {
		name    string
		request any
	}{
		{name: "value", request: request{Name: "admin"}},
		{name: "pointer", request: &request{Name: "user", IsDefault: true}},
		{name: "without validation tags", request: struct{ Name string }{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", nil)

			if err := validateRequestBody(w, r, tc.request); err != nil {
				t.Fatalf("validateRequestBody: %v", err)
			}
			if w.Body.Len() != 0 || len(w.Header()) != 0 || w.Flushed {
				t.Error("successful validation must not write a response")
			}
		})
	}
}

func TestValidateRequestBodyInvalidFields(t *testing.T) {
	type request struct {
		Name        string `json:"name,omitempty" validate:"required,max=5"`
		Description string `json:"description" validate:"max=10"`
	}

	for _, tc := range []struct {
		name    string
		request any
		fields  []validationFieldError
	}{
		{
			name:    "required with JSON tag options",
			request: &request{},
			fields:  []validationFieldError{{Field: "name", Rule: "required"}},
		},
		{
			name:    "maximum length parameter",
			request: &request{Name: "administrator"},
			fields:  []validationFieldError{{Field: "name", Rule: "max", Param: "5"}},
		},
		{
			name:    "all invalid fields",
			request: &request{Description: "private administrator note"},
			fields: []validationFieldError{
				{Field: "name", Rule: "required"},
				{Field: "description", Rule: "max", Param: "10"},
			},
		},
		{
			name: "Go name without JSON tag",
			request: &struct {
				Name string `validate:"required"`
			}{},
			fields: []validationFieldError{{Field: "Name", Rule: "required"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", nil)

			err := validateRequestBody(w, r, tc.request)
			fieldErrors, ok := errors.AsType[validator.ValidationErrors](err)
			if !ok {
				t.Fatalf("error = %v, want wrapped validator.ValidationErrors", err)
			}
			if errors.Unwrap(err) == nil || len(fieldErrors) != len(tc.fields) {
				t.Errorf("error must wrap all %d field errors", len(tc.fields))
			}
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", w.Code)
			}
			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("content type = %q, want application/json", got)
			}

			var body struct {
				OK      *bool                  `json:"ok"`
				Message string                 `json:"message"`
				Fields  []validationFieldError `json:"fields"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body, json.RejectUnknownMembers(true)); err != nil {
				t.Fatalf("decode validation response: %v", err)
			}
			if body.OK == nil || *body.OK {
				t.Error("response must contain ok: false")
			}
			if body.Message != "Request validation failed" {
				t.Errorf("message = %q, want Request validation failed", body.Message)
			}
			if !slices.Equal(body.Fields, tc.fields) {
				t.Errorf("fields = %+v, want %+v", body.Fields, tc.fields)
			}
			if strings.Contains(w.Body.String(), "private administrator note") {
				t.Error("response must not expose submitted field values")
			}
		})
	}
}

func TestValidateRequestSlug(t *testing.T) {
	for _, tc := range []struct {
		name  string
		valid bool
	}{
		{name: "a", valid: true},
		{name: "admin", valid: true},
		{name: "project-admin1", valid: true},
		{name: "a-1-admin", valid: true},
		{name: ""},
		{name: "Admin"},
		{name: "1admin"},
		{name: "_admin"},
		{name: "project_admin"},
		{name: "-admin"},
		{name: "admin-"},
		{name: "project--admin"},
		{name: "project admin"},
		{name: "admin\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := struct {
				Name string `json:"name" validate:"slug"`
			}{Name: tc.name}
			err := requestValidator.Struct(request)
			if tc.valid {
				if err != nil {
					t.Fatalf("validate name %q: %v", tc.name, err)
				}
				return
			}

			fieldErrors, ok := errors.AsType[validator.ValidationErrors](err)
			if !ok || len(fieldErrors) != 1 || fieldErrors[0].Tag() != "slug" {
				t.Fatalf("error = %v, want a slug validation error", err)
			}
		})
	}
}

func TestValidateRequestBodyInternalError(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request any
	}{
		{name: "nil"},
		{name: "nil pointer", request: (*struct{ Name string })(nil)},
		{name: "non-struct", request: "invalid input"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", nil)

			err := validateRequestBody(w, r, tc.request)
			if _, ok := errors.AsType[*validator.InvalidValidationError](err); !ok {
				t.Fatalf("error = %v, want wrapped validator.InvalidValidationError", err)
			}
			if errors.Unwrap(err) == nil {
				t.Error("error must preserve the validation error chain")
			}
			assertJSONErrorResponse(t, w, http.StatusInternalServerError, "Internal Server Error")
		})
	}
}

func TestReadJSON(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		wantErr bool
	}{
		{name: "valid object", body: `{"name":"admin"}`},
		{name: "surrounding whitespace", body: " \n{\"name\":\"admin\"}\t\n"},
		{name: "empty body", wantErr: true},
		{name: "whitespace only", body: " \n\t", wantErr: true},
		{name: "malformed JSON", body: `{"name":`, wantErr: true},
		{name: "wrong field type", body: `{"name":123}`, wantErr: true},
		{name: "multiple values", body: `{"name":"admin"} {}`, wantErr: true},
		{name: "trailing garbage", body: `{"name":"admin"} invalid`, wantErr: true},
		{name: "duplicate names", body: `{"name":"admin","name":"user"}`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			var dst struct {
				Name string `json:"name"`
			}

			err := readJSON(w, r, &dst)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				assertJSONErrorResponse(t, w, http.StatusBadRequest, "Invalid JSON request body")
				return
			}
			if err != nil {
				t.Fatalf("readJSON: %v", err)
			}
			if w.Body.Len() != 0 || len(w.Header()) != 0 || w.Flushed {
				t.Error("successful decoding must not write a response")
			}
			if dst.Name != "admin" {
				t.Errorf("name = %q, want admin", dst.Name)
			}
		})
	}
}

func TestReadJSONReadError(t *testing.T) {
	wantErr := errors.New("body read failed")
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Body = io.NopCloser(jsonErrorReader{err: wantErr})
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	var dst any

	if err := readJSON(w, r, &dst); !errors.Is(err, wantErr) {
		t.Errorf("readJSON error = %v, want wrapped %v", err, wantErr)
	}
	assertJSONErrorResponse(t, w, http.StatusBadRequest, "Invalid JSON request body")
}

func TestReadJSONContentType(t *testing.T) {
	for _, tc := range []struct {
		name        string
		contentType string
		wantErr     bool
	}{
		{name: "JSON", contentType: "application/json"},
		{name: "UTF-8 parameter", contentType: "application/json; charset=utf-8"},
		{name: "case insensitive", contentType: "Application/JSON"},
		{name: "missing", wantErr: true},
		{name: "plain text", contentType: "text/plain", wantErr: true},
		{name: "form", contentType: "application/x-www-form-urlencoded", wantErr: true},
		{name: "malformed media type", contentType: "application", wantErr: true},
		{name: "malformed parameter", contentType: "application/json; charset", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"admin"}`))
			if tc.contentType != "" {
				r.Header.Set("Content-Type", tc.contentType)
			}
			w := httptest.NewRecorder()

			var dst struct {
				Name string `json:"name"`
			}

			err := readJSON(w, r, &dst)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected a content type error")
				}
				assertJSONErrorResponse(t, w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
				if dst.Name != "" {
					t.Error("unsupported content type must not decode the body")
				}
				return
			}
			if err != nil {
				t.Fatalf("readJSON: %v", err)
			}
			if dst.Name != "admin" {
				t.Errorf("name = %q, want admin", dst.Name)
			}
			if w.Body.Len() != 0 || len(w.Header()) != 0 || w.Flushed {
				t.Error("successful decoding must not write a response")
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		body     any
		wantBody string
	}{
		{
			name:   "object",
			status: http.StatusCreated,
			body: struct {
				Name string `json:"name"`
			}{Name: "administrator"},
			wantBody: `{"name":"administrator"}`,
		},
		{
			name:     "success message",
			status:   http.StatusOK,
			body:     messageResponse{OK: true, Message: "Role created"},
			wantBody: `{"ok":true,"message":"Role created"}`,
		},
		{
			name:     "empty fields omitted",
			status:   http.StatusBadRequest,
			body:     messageResponse{Message: "Invalid request", Fields: []validationFieldError{}},
			wantBody: `{"ok":false,"message":"Invalid request"}`,
		},
		{
			name:   "field errors included",
			status: http.StatusBadRequest,
			body: messageResponse{
				Message: "Request validation failed",
				Fields:  []validationFieldError{{Field: "name", Rule: "required"}},
			},
			wantBody: `{"ok":false,"message":"Request validation failed","fields":[{"field":"name","rule":"required","param":""}]}`,
		},
		{name: "nil body", status: http.StatusOK, wantBody: ""},
		{name: "array", status: http.StatusOK, body: []string{"admin", "user"}, wantBody: `["admin","user"]`},
		{name: "string", status: http.StatusAccepted, body: "hello", wantBody: `"hello"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			w.Header().Set("Content-Type", "text/plain")

			writeJSON(w, r, tc.status, tc.body)

			response := w.Result()
			t.Cleanup(func() {
				if err := response.Body.Close(); err != nil {
					t.Errorf("close response body: %v", err)
				}
			})
			if response.StatusCode != tc.status {
				t.Errorf("status = %d, want %d", response.StatusCode, tc.status)
			}
			if got := response.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("content type = %q, want application/json", got)
			}
			if got := w.Body.String(); got != tc.wantBody {
				t.Errorf("body = %q, want %q", got, tc.wantBody)
			}
		})
	}
}

func TestWriteJSONMarshalError(t *testing.T) {
	for _, tc := range []struct {
		name string
		body any
	}{
		{name: "unsupported type", body: make(chan int)},
		{name: "non-finite number", body: math.NaN()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			writeJSON(w, r, http.StatusCreated, tc.body)

			assertJSONErrorResponse(t, w, http.StatusInternalServerError, "Internal Server Error")
		})
	}
}

func TestWriteJSONWriteError(t *testing.T) {
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := &jsonErrorWriter{
		ResponseRecorder: httptest.NewRecorder(),
		err:              errors.New("response write failed"),
	}

	writeJSON(w, r, http.StatusCreated, "hello")

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("content type = %q, want application/json", got)
	}
	if w.writeCalls != 1 {
		t.Errorf("write calls = %d, want 1", w.writeCalls)
	}
	if string(w.data) != `"hello"` {
		t.Errorf("attempted body = %q, want JSON string", w.data)
	}
	for _, want := range []string{"level=ERROR", "write json response", "response write failed", "tag=rest"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("log = %q, want substring %q", logs.String(), want)
		}
	}
}

func assertJSONErrorResponse(t *testing.T, w *httptest.ResponseRecorder, status int, message string) {
	t.Helper()

	response := w.Result()
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	})
	if response.StatusCode != status {
		t.Errorf("status = %d, want %d", response.StatusCode, status)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("content type = %q, want application/json", got)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body["ok"] != false {
		t.Errorf("ok = %v, want false", body["ok"])
	}
	if body["message"] != message {
		t.Errorf("message = %v, want %q", body["message"], message)
	}
	if _, exists := body["fields"]; exists {
		t.Error("error response without field errors must omit fields")
	}
	if len(body) != 2 {
		t.Errorf("response properties = %d, want only ok and message", len(body))
	}
}

type jsonErrorWriter struct {
	*httptest.ResponseRecorder
	err        error
	writeCalls int
	data       []byte
}

func (w *jsonErrorWriter) Write(data []byte) (int, error) {
	w.writeCalls++
	w.data = append(w.data, data...)
	return 0, w.err
}

type jsonErrorReader struct {
	err error
}

func (r jsonErrorReader) Read([]byte) (int, error) {
	return 0, r.err
}

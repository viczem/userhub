package domain

import (
	"errors"
	"testing"
)

func TestErrorKindNewErrorf(t *testing.T) {
	t.Parallel()

	const kind ErrorKind = "keyring"

	err := kind.NewErrorf("key %d is missing", 7)

	if got, want := err.Error(), "keyring: key 7 is missing"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, kind) {
		t.Error("formatted error does not match its kind")
	}
}

func TestErrorKindWrapErrorf(t *testing.T) {
	t.Parallel()

	const kind ErrorKind = "keyring"
	cause := errors.New("invalid encoding")

	err := kind.WrapErrorf(cause, "decode key %d", 7)

	if got, want := err.Error(), "keyring: decode key 7: invalid encoding"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, kind) {
		t.Error("formatted error does not match its kind")
	}
	if !errors.Is(err, cause) {
		t.Error("formatted error does not retain its cause")
	}
}

func TestNewErrorf(t *testing.T) {
	t.Parallel()

	err := NewErrorf("key %d is missing", 7)

	if got, want := err.Error(), "key 7 is missing"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestWrapErrorf(t *testing.T) {
	t.Parallel()

	cause := errors.New("invalid encoding")

	err := WrapErrorf(cause, "decode key %d", 7)

	if got, want := err.Error(), "decode key 7: invalid encoding"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Error("formatted error does not retain its cause")
	}
}

func TestErrorKindError(t *testing.T) {
	t.Parallel()

	const kind ErrorKind = "keyring"

	if got, want := kind.Error(), "keyring"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestErrorError(t *testing.T) {
	t.Parallel()

	cause := errors.New("invalid encoding")

	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "kind message and cause",
			err:  &Error{Kind: "keyring", Message: "decode key", Err: cause},
			want: "keyring: decode key: invalid encoding",
		},
		{
			name: "kind and message",
			err:  &Error{Kind: "keyring", Message: "active key is missing"},
			want: "keyring: active key is missing",
		},
		{
			name: "kind and cause",
			err:  &Error{Kind: "keyring", Err: cause},
			want: "keyring: invalid encoding",
		},
		{
			name: "kind only",
			err:  &Error{Kind: "keyring"},
			want: "keyring",
		},
		{
			name: "message and cause",
			err:  &Error{Message: "decode key", Err: cause},
			want: "decode key: invalid encoding",
		},
		{
			name: "message only",
			err:  &Error{Message: "active key is missing"},
			want: "active key is missing",
		},
		{
			name: "cause only",
			err:  &Error{Err: cause},
			want: "invalid encoding",
		},
		{
			name: "empty",
			err:  &Error{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorKindNew(t *testing.T) {
	t.Parallel()

	const kind ErrorKind = "keyring"

	err := kind.NewError("active key is missing")

	if err.Kind != kind {
		t.Errorf("Kind = %q, want %q", err.Kind, kind)
	}
	if got, want := err.Message, "active key is missing"; got != want {
		t.Errorf("Message = %q, want %q", got, want)
	}
	if err.Err != nil {
		t.Errorf("Err = %v, want nil", err.Err)
	}
	if !errors.Is(err, kind) {
		t.Error("error does not match its kind")
	}
}

func TestErrorKindWrap(t *testing.T) {
	t.Parallel()

	const kind ErrorKind = "keyring"
	cause := errors.New("invalid encoding")

	err := kind.WrapError(cause, "decode key")

	if err.Kind != kind {
		t.Errorf("Kind = %q, want %q", err.Kind, kind)
	}
	if got, want := err.Message, "decode key"; got != want {
		t.Errorf("Message = %q, want %q", got, want)
	}
	if err.Unwrap() != cause {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), cause)
	}
	if !errors.Is(err, kind) {
		t.Error("error does not match its kind")
	}
	if !errors.Is(err, cause) {
		t.Error("error does not retain its cause")
	}
}

func TestNewError(t *testing.T) {
	t.Parallel()

	err := NewError("active key is missing")

	if err.Kind != "" {
		t.Errorf("Kind = %q, want empty", err.Kind)
	}
	if got, want := err.Message, "active key is missing"; got != want {
		t.Errorf("Message = %q, want %q", got, want)
	}
	if err.Err != nil {
		t.Errorf("Err = %v, want nil", err.Err)
	}
}

func TestWrapError(t *testing.T) {
	t.Parallel()

	cause := errors.New("invalid encoding")
	err := WrapError(cause, "decode key")

	if err.Kind != "" {
		t.Errorf("Kind = %q, want empty", err.Kind)
	}
	if got, want := err.Message, "decode key"; got != want {
		t.Errorf("Message = %q, want %q", got, want)
	}
	if err.Unwrap() != cause {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), cause)
	}
	if !errors.Is(err, cause) {
		t.Error("error does not retain its cause")
	}
}

func TestErrorIs(t *testing.T) {
	t.Parallel()

	const (
		keyringKind ErrorKind = "keyring"
		runtimeKind ErrorKind = "runtime session"
	)

	tests := []struct {
		name   string
		err    *Error
		target error
		want   bool
	}{
		{
			name:   "matching kind",
			err:    keyringKind.NewError("failed"),
			target: keyringKind,
			want:   true,
		},
		{
			name:   "different kind",
			err:    keyringKind.NewError("failed"),
			target: runtimeKind,
			want:   false,
		},
		{
			name:   "error without kind",
			err:    NewError("failed"),
			target: keyringKind,
			want:   false,
		},
		{
			name:   "empty target kind",
			err:    keyringKind.NewError("failed"),
			target: ErrorKind(""),
			want:   false,
		},
		{
			name:   "non-kind target",
			err:    keyringKind.NewError("failed"),
			target: errors.New("keyring"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestErrorAs(t *testing.T) {
	t.Parallel()

	const kind ErrorKind = "keyring"
	want := kind.NewError("active key is missing")

	var got *Error
	if !errors.As(want, &got) {
		t.Fatal("errors.As() = false, want true")
	}
	if got != want {
		t.Errorf("errors.As() error = %p, want %p", got, want)
	}
}

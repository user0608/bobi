package errs

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestErr_Error(t *testing.T) {
	t.Run("Test Error without wrapped error", func(t *testing.T) {
		err := &Err{
			message: "Test error",
		}

		got := err.Error()
		want := "error: Test error"
		if got != want {
			t.Errorf("Error() = %v, want %v", got, want)
		}
	})

	t.Run("Test Error with wrapped error", func(t *testing.T) {
		wrappedErr := fmt.Errorf("wrapped error")
		err := &Err{
			message: "Test error",
			wrapped: wrappedErr,
		}

		got := err.Error()
		want := "error: Test error; wrapped: wrapped error"
		if got != want {
			t.Errorf("Error() = %v, want %v", got, want)
		}
	})
}

func TestErr_Message(t *testing.T) {
	err := &Err{
		message: "Test message",
	}
	got := err.Message()
	want := "Test message"
	if got != want {
		t.Errorf("Message() = %v, want %v", got, want)
	}
}

func TestErr_Code(t *testing.T) {
	err := &Err{
		httpCode: http.StatusNotFound,
	}
	got := err.Code()
	want := http.StatusNotFound
	if got != want {
		t.Errorf("Code() = %v, want %v", got, want)
	}
}

func TestContainsMessage(t *testing.T) {
	err := newError(nil, "Test error message", http.StatusBadRequest)

	t.Run("Contains exact message", func(t *testing.T) {
		got := ContainsMessage(err, "Test error")
		if !got {
			t.Errorf("ContainsMessage() = %v, want true", got)
		}
	})

	t.Run("Does not contain message", func(t *testing.T) {
		got := ContainsMessage(err, "Different message")
		if got {
			t.Errorf("ContainsMessage() = %v, want false", got)
		}
	})
}

func TestBadRequestf(t *testing.T) {
	err := BadRequestError(nil, "Bad request: %s", "missing field")

	t.Run("Check error message", func(t *testing.T) {
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if err.(*Err).Message() != "Bad request: missing field" {
			t.Errorf("Expected message 'Bad request: missing field', got %v", err.(*Err).Message())
		}
	})

	t.Run("Check HTTP code", func(t *testing.T) {
		if err.(*Err).Code() != http.StatusBadRequest {
			t.Errorf("Expected HTTP code %v, got %v", http.StatusBadRequest, err.(*Err).Code())
		}
	})
}

func TestIsBadRequest(t *testing.T) {
	t.Run("IsBadRequest returns true for BadRequest error", func(t *testing.T) {
		err := newError(nil, "Bad request", http.StatusBadRequest)
		if !IsBadRequest(err) {
			t.Errorf("Expected IsBadRequest to return true, got false")
		}
	})

	t.Run("IsBadRequest returns false for non-BadRequest error", func(t *testing.T) {
		err := newError(nil, "Internal server error", http.StatusInternalServerError)
		if IsBadRequest(err) {
			t.Errorf("Expected IsBadRequest to return false, got true")
		}
	})
}

func TestToSummary(t *testing.T) {
	err := newError(nil, "Test summary", http.StatusBadRequest)

	t.Run("ToSummary returns correct summary", func(t *testing.T) {
		got := ToSummary(err)
		want := "Error 400: Test summary"
		if got != want {
			t.Errorf("ToSummary() = %v, want %v", got, want)
		}
	})

	t.Run("ToSummary returns 'No error' for nil error", func(t *testing.T) {
		got := ToSummary(nil)
		want := "No error"
		if got != want {
			t.Errorf("ToSummary() = %v, want %v", got, want)
		}
	})
}

func TestWrapError(t *testing.T) {
	baseErr := fmt.Errorf("Base error")
	wrappedErr := WrapError(baseErr, "Wrapped error", http.StatusInternalServerError)

	t.Run("WrapError with wrapped error", func(t *testing.T) {
		if wrappedErr == nil {
			t.Fatal("Expected wrapped error, got nil")
		}
		if wrappedErr.(*Err).wrapped != baseErr {
			t.Errorf("Expected wrapped error %v, got %v", baseErr, wrappedErr.(*Err).wrapped)
		}
	})

	t.Run("WrapError with nil error", func(t *testing.T) {
		if WrapError(nil, "No error", http.StatusOK) != nil {
			t.Errorf("Expected nil, got non-nil error")
		}
	})
}

func TestErrUnwrapAndAsType(t *testing.T) {
	baseErr := errors.New("base error")
	wrappedErr := WrapError(baseErr, "public error", http.StatusInternalServerError)

	if !errors.Is(wrappedErr, baseErr) {
		t.Fatal("expected errors.Is to find the wrapped error")
	}
	if customErr, ok := errors.AsType[*Err](wrappedErr); !ok || customErr.Message() != "public error" {
		t.Fatal("expected errors.AsType to find the custom error")
	}
}

func TestHTTPErrorConstructors(t *testing.T) {
	tests := []struct {
		name string
		code int
		make func(string) error
	}{
		{"bad request", http.StatusBadRequest, BadRequestDirect},
		{"not found", http.StatusNotFound, NotFoundDirect},
		{"internal", http.StatusInternalServerError, InternalErrorDirect},
		{"unauthorized", http.StatusUnauthorized, UnauthorizedDirect},
		{"forbidden", http.StatusForbidden, ForbiddenDirect},
		{"unsupported media type", http.StatusUnsupportedMediaType, UnsupportedMediaTypeDirect},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.make("test message")
			customErr, ok := errors.AsType[*Err](err)
			if !ok {
				t.Fatal("expected custom error")
			}
			if customErr.Code() != tt.code {
				t.Fatalf("Code() = %d, want %d", customErr.Code(), tt.code)
			}
			if customErr.Message() != "test message" {
				t.Fatalf("Message() = %q, want %q", customErr.Message(), "test message")
			}
		})
	}
}

func TestFormattedErrorConstructors(t *testing.T) {
	tests := []struct {
		name string
		code int
		make func(string, ...any) error
	}{
		{"bad request", http.StatusBadRequest, BadRequestf},
		{"not found", http.StatusNotFound, NotFoundf},
		{"internal", http.StatusInternalServerError, InternalErrorf},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.make("test %s", "message")
			customErr, ok := errors.AsType[*Err](err)
			if !ok {
				t.Fatal("expected custom error")
			}
			if customErr.Code() != tt.code {
				t.Fatalf("Code() = %d, want %d", customErr.Code(), tt.code)
			}
			if customErr.Message() != "test message" {
				t.Fatalf("Message() = %q, want %q", customErr.Message(), "test message")
			}
		})
	}
}

func TestNewWithMessagePreservesCodeAndCause(t *testing.T) {
	baseErr := errors.New("base error")
	original := InternalError(baseErr, "internal message")
	updated := NewWithMessage(original, "safe message")

	customErr, ok := errors.AsType[*Err](updated)
	if !ok {
		t.Fatal("expected custom error")
	}
	if customErr.Code() != http.StatusInternalServerError {
		t.Fatalf("Code() = %d, want %d", customErr.Code(), http.StatusInternalServerError)
	}
	if customErr.Message() != "safe message" {
		t.Fatalf("Message() = %q, want %q", customErr.Message(), "safe message")
	}
	if !errors.Is(updated, baseErr) {
		t.Fatal("expected NewWithMessage to preserve the original cause")
	}
}

func TestNewWithMessageDefaultsToBadRequest(t *testing.T) {
	baseErr := errors.New("base error")
	err := NewWithMessage(baseErr, "safe message")

	customErr, ok := errors.AsType[*Err](err)
	if !ok {
		t.Fatal("expected custom error")
	}
	if customErr.Code() != http.StatusBadRequest {
		t.Fatalf("Code() = %d, want %d", customErr.Code(), http.StatusBadRequest)
	}
	if !errors.Is(err, baseErr) {
		t.Fatal("expected NewWithMessage to preserve the original cause")
	}
}

func TestIsInternalError(t *testing.T) {
	if !IsInternalError(InternalErrorDirect("internal")) {
		t.Fatal("expected internal error")
	}
	if IsInternalError(BadRequestDirect("bad request")) {
		t.Fatal("did not expect bad request to be internal")
	}
}

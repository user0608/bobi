package answer

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/user0608/bobi/errs"
)

func newTestContext() (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	return e.NewContext(req, rec), rec
}

func assertResponse(t *testing.T, rec *httptest.ResponseRecorder, code int, message string, data any) {
	t.Helper()

	if rec.Code != code {
		t.Fatalf("expected status code %d, got %d", code, rec.Code)
	}

	var response Response
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid JSON response, got error %v and body %q", err, rec.Body.String())
	}

	if response.Message != message {
		t.Fatalf("expected message %q, got %q", message, response.Message)
	}

	if data == nil {
		if response.Data != nil {
			t.Fatalf("expected nil data, got %#v", response.Data)
		}

		return
	}

	expectedData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal expected data: %v", err)
	}

	actualData, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("failed to marshal actual data: %v", err)
	}

	if string(actualData) != string(expectedData) {
		t.Fatalf("expected data %s, got %s", expectedData, actualData)
	}
}

func TestOk(t *testing.T) {
	c, rec := newTestContext()
	payload := map[string]string{"id": "123"}

	err := Ok(c, payload)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertResponse(t, rec, http.StatusOK, "", payload)
}

func TestCreated(t *testing.T) {
	c, rec := newTestContext()

	err := Created(c)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertResponse(t, rec, http.StatusCreated, "Recurso creado exitosamente", nil)
}

func TestMessage(t *testing.T) {
	c, rec := newTestContext()

	err := Message(c, "custom message")

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertResponse(t, rec, http.StatusOK, "custom message", nil)
}

func TestSuccess(t *testing.T) {
	c, rec := newTestContext()

	err := Success(c)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertResponse(t, rec, http.StatusOK, "Operación completada exitosamente", nil)
}

func TestNoContent(t *testing.T) {
	c, rec := newTestContext()

	err := NoContent(c)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status code %d, got %d", http.StatusNoContent, rec.Code)
	}

	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rec.Body.String())
	}
}

func TestErrWithDomainError(t *testing.T) {
	c, rec := newTestContext()
	err := errs.BadRequestError(nil, "Invalid value: %s", "email")

	got := Err(c, err)

	if got != nil {
		t.Fatalf("expected nil error, got %v", got)
	}

	assertResponse(t, rec, http.StatusBadRequest, "Invalid value: email", nil)
}

func TestErrWithWrappedDomainError(t *testing.T) {
	c, rec := newTestContext()
	baseErr := fmt.Errorf("database failed")
	err := errs.WrapError(baseErr, "Unexpected failure", http.StatusInternalServerError)

	got := Err(c, err)

	if got != nil {
		t.Fatalf("expected nil error, got %v", got)
	}

	assertResponse(t, rec, http.StatusInternalServerError, "Unexpected failure", nil)
}

func TestErrWithPrefixedBadRequest(t *testing.T) {
	c, rec := newTestContext()

	err := Err(c, errors.New(":invalid request"))

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertResponse(t, rec, http.StatusBadRequest, "invalid request", nil)
}

func TestErrWithInternalError(t *testing.T) {
	c, rec := newTestContext()

	err := Err(c, errors.New("database failed"))

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	assertResponse(t, rec, http.StatusInternalServerError, "Ocurrió un problema. Se produjo un error inesperado.", nil)
}

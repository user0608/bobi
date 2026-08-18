package binds

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestFormFieldJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	c := _newMultipartContext(t, map[string]string{
		"data": `{"name":"john","age":30}`,
	})

	var result payload

	err := FormFieldJSON(c, "data", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "john" {
		t.Fatalf("name mismatch: got %v, want %v", result.Name, "john")
	}

	if result.Age != 30 {
		t.Fatalf("age mismatch: got %v, want %v", result.Age, 30)
	}
}

func TestFormFieldJSON_MissingField(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	c := _newMultipartContext(t, map[string]string{})

	var result payload

	err := FormFieldJSON(c, "data", &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFormFieldJSON_EmptyField(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	c := _newMultipartContext(t, map[string]string{
		"data": "",
	})

	var result payload

	err := FormFieldJSON(c, "data", &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFormFieldJSON_BlankField(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	c := _newMultipartContext(t, map[string]string{
		"data": "   ",
	})

	var result payload

	err := FormFieldJSON(c, "data", &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFormFieldJSON_InvalidJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	c := _newMultipartContext(t, map[string]string{
		"data": `{"name":`,
	})

	var result payload

	err := FormFieldJSON(c, "data", &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFormFieldJSON_WithFileInMultipart(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	c := newMultipartContextWithFile(t, map[string]string{
		"data": `{"name":"john"}`,
	}, "file", "test.txt", []byte("hello"))

	var result payload

	err := FormFieldJSON(c, "data", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "john" {
		t.Fatalf("name mismatch: got %v, want %v", result.Name, "john")
	}
}

func _newMultipartContext(t *testing.T, fields map[string]string) *echo.Context {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	rec := httptest.NewRecorder()

	return e.NewContext(req, rec)
}

func newMultipartContextWithFile(
	t *testing.T,
	fields map[string]string,
	fileField string,
	fileName string,
	fileContent []byte,
) *echo.Context {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}

	fileWriter, err := writer.CreateFormFile(fileField, fileName)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if _, err := fileWriter.Write(fileContent); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	rec := httptest.NewRecorder()

	return e.NewContext(req, rec)
}

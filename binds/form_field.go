package binds

import (
	"encoding/json"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/user0608/bobi/errs"
)

func FormFieldJSON(c *echo.Context, fieldName string, payload any) error {
	value := strings.TrimSpace(c.FormValue(fieldName))
	if value == "" {
		return errs.BadRequestf("el campo requerido '%s' no fue enviado o está vacío", fieldName)
	}

	if err := json.Unmarshal([]byte(value), payload); err != nil {
		return errs.BadRequestError(err, "json del formulario inválido")
	}

	return nil
}

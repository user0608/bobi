package binds

import (
	"github.com/labstack/echo/v5"
	"github.com/user0608/bobi/errs"
)

func From(c *echo.Context, payload any) error {
	return JSON(c, payload)
}

func JSON(c *echo.Context, payload any) error {
	if err := echo.BindBody(c, payload); err != nil {
		return errs.BadRequestError(err, "json document invalido")
	}
	return nil
}

func Query(c *echo.Context, payload any) error {
	if err := echo.BindQueryParams(c, payload); err != nil {
		return errs.BadRequestError(err, "json document invalido")
	}
	return nil
}

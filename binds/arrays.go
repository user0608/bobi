package binds

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/user0608/bobi/errs"
	"github.com/user0608/bobi/types"
)

func RequestUUIDs(c *echo.Context) ([]uuid.UUID, error) {
	var payload map[string]json.RawMessage

	if err := echo.BindBody(c, &payload); err != nil {
		return nil, errs.BadRequestError(err, "invalid request body")
	}

	keys := []string{"code", "codes", "id", "ids", "uuid", "uuids", "values"}

	for _, k := range keys {
		if raw, ok := payload[k]; ok {
			var arr types.UUIDArray
			if err := json.Unmarshal(raw, &arr); err != nil {
				return nil, errs.BadRequestError(err, "invalid field '%s': must be a UUID or array of UUIDs", k)
			}
			return arr.Unique(), nil
		}
	}

	return nil, errs.BadRequestError(nil, "missing required field: expected one of %v", keys)
}

func RequestStrings(c *echo.Context) ([]string, error) {
	var payload map[string]json.RawMessage

	if err := echo.BindBody(c, &payload); err != nil {
		return nil, errs.BadRequestError(err, "invalid request body")
	}

	keys := []string{
		"code",
		"codes",
		"codigo",
		"codigos",

		"value",
		"values",
		"valor",
		"valores",

		"string",
		"strings",

		"text",
		"texts",

		"key",
		"keys",

		"id",
		"ids",

		"username",
		"usernames",
	}

	for _, k := range keys {
		if raw, ok := payload[k]; ok {
			var arr types.StrArray
			if err := json.Unmarshal(raw, &arr); err != nil {
				return nil, errs.BadRequestError(err, "invalid field '%s': must be a string or array of strings", k)
			}
			return arr.Unique().NonEmpty(), nil
		}
	}

	return nil, errs.BadRequestError(nil, "missing required field: expected one of %v", keys)
}

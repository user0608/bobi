---
name: errors-answer
description: Use Bobi errs and answer for typed HTTP errors, database error translation, and consistent Echo JSON responses. Trigger when returning API errors or success responses from handlers.
---

# Bobi Errors And Responses

Keep error handling separated by layer. Repositories return persistence errors,
services apply business rules and create typed Bobi errors, and handlers translate
those errors into HTTP responses.

Service example:

```go
func (service *Service) FindUser(ctx context.Context, id string) (*User, error) {
	user, err := service.repo.Find(ctx, id)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errs.NotFoundf(
			"usuario %q no encontrado",
			id,
		)
	}

	return user, nil
}
```

Handler example:

```go
func (handler *Handler) FindUser(c *echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	user, err := handler.service.FindUser(ctx, id)
	if err != nil {
		return answer.Err(c, err)
	}

	return answer.Ok(c, user)
}
```

Do not return `errs.NotFoundf`, `errs.BadRequestf`, or another Bobi error
directly from a handler. Return it from the service and use `answer.Err` in the
handler. Binding helpers may return Bobi errors because they are boundary helpers;
the handler should still translate them consistently.

Use `errs.BadRequestError`, `NotFoundError`, `InternalError`, `UnauthorizedError`, `ForbiddenError`, and `UnsupportedMediaTypeError` when wrapping an underlying error. Use the `*f` or `*Direct` variants when appropriate. `errs.WrapError(err, message, status)` supports a custom status code.

Use `errs.Dbf(err)` immediately after a GORM operation when database errors must become API-safe errors. It handles `gorm.ErrRecordNotFound`, PostgreSQL errors, and SQLite errors.

## Response helpers

- `answer.Ok(c, data)`: HTTP 200 with `{ "data": ... }`.
- `answer.Created(c)`: HTTP 201 with a success message.
- `answer.Message(c, message)`: HTTP 200 with a message.
- `answer.Success(c)`: HTTP 200 with the standard success message.
- `answer.NoContent(c)`: HTTP 204.
- `answer.Err(c, err)`: maps a Bobi error to its status and message.
- `answer.Auto(c, err)`: returns an error response when `err != nil`, otherwise success.

Do not send `err.Error()` directly to clients for internal errors. `answer.UnwrapErr`
uses the typed error message and logs unexpected/internal errors. Keep each layer
linear: obtain the value, check the error, apply the business rule, and return.

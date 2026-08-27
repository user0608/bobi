---
name: binds-kcheck
description: Use Bobi binds and kcheck for Echo request binding, multipart fields/files, UUID and string lists, and validation tags. Trigger when handling request payloads or validating Go structs with chk tags.
---

# Bobi Binding And Validation

Use `github.com/user0608/bobi/binds` at the Echo boundary:

```go
var input CreateUser
if err := binds.JSON(c, &input); err != nil {
	return err
}
if err := kcheck.Struct(input); err != nil {
	return errs.BadRequestError(err, "datos invalidos")
}
```

Available binders include `binds.From`, `JSON`, `Query`, `FormFieldJSON`, `FormFileBytesRequired`, and `FormFileBytesOptional`. The optional file helper returns `(nil, nil)` only when the multipart file is absent.

`RequestUUIDs` accepts a UUID or UUID array under common keys such as `id`, `ids`, `uuid`, or `uuids`; it deduplicates results. `RequestStrings` accepts common singular/plural keys, removes empty values, and deduplicates results.

## Validation tags

`kcheck.New()` registers the default validators. Tags use space-separated rules and `=` parameters:

```go
type CreateUser struct {
	Email string `chk:"required email"`
	Name  string `chk:"required min=3 max=80"`
	Role  string `chk:"oneof=admin,user"`
}
```

Default rules include `required`, `nonil`, `len`, `min`, `max`, `email`, `uuid`, `url`, `ip`, `ipv4`, `ipv6`, `alpha`, `alphanum`, `num`, `decimal`, `lower`, `upper`, `oneof`, `prefix`, `suffix`, `contains`, `date`, `time`, `datetime`, `utc`, `gt`, `gte`, `lt`, and `lte`.

Use `StructSkip` or `Valid(input, fields...)` to skip fields, and `StructSelect` or `ValidSelect(input, fields...)` to validate selected fields and matching nested paths. Register custom rules on a validator with `Register`.

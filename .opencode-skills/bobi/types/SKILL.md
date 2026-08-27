---
name: types
description: Use Bobi types for date-only, time-only, datetime-only, UUID-array, and string-array values with JSON and SQL support. Trigger when modeling API or database fields that need these representations.
---

# Bobi Common Types

Use the specialized types instead of carrying formatting logic through handlers:

```go
type Filter struct {
	Date  types.DateOnly    `json:"date"`
	Start types.JustTime    `json:"start"`
	Tags  types.StrArray    `json:"tags"`
	Users types.UUIDArray   `json:"users"`
}
```

- `DateOnly` uses `YYYY-MM-DD` and supports JSON, text, SQL scanning, and UTC/local day ranges.
- `DateTimeOnly` uses `YYYY-MM-DD HH:MM:SS` without fractional seconds or timezone data.
- `JustTime` represents a time of day as a duration and accepts `HH`, `HH:MM`, or `HH:MM:SS[.fraction]` within valid ranges.
- `StrArray` accepts a JSON string or array of strings; use `Trimmed`, `NonEmpty`, and `Unique` for normalization.
- `UUIDArray` accepts a UUID string or array of UUID strings and provides `Unique`.

Zero values for the date/time types serialize as JSON `null`. Check location semantics explicitly when converting date-only values to `time.Time`; `nil` locations default to UTC in Bobi conversion helpers.

package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cast"
)

type DateTimeOnly struct{ time.Time }

var _ json.Marshaler = DateTimeOnly{}
var _ json.Unmarshaler = (*DateTimeOnly)(nil)
var _ encoding.TextMarshaler = DateTimeOnly{}
var _ encoding.TextUnmarshaler = (*DateTimeOnly)(nil)
var _ gob.GobEncoder = DateTimeOnly{}
var _ gob.GobDecoder = (*DateTimeOnly)(nil)
var _ driver.Valuer = DateTimeOnly{}
var _ sql.Scanner = (*DateTimeOnly)(nil)
var _ fmt.Stringer = DateTimeOnly{}

func NewDateTimeOnly(t time.Time) DateTimeOnly {
	str := t.Format(time.DateTime)
	val, _ := time.Parse(time.DateTime, str)
	return DateTimeOnly{Time: val}
}

func (do DateTimeOnly) ToTimeInLocation(loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}

	return time.Date(
		do.Year(),
		do.Month(),
		do.Day(),
		do.Hour(),
		do.Minute(),
		do.Second(),
		0,
		loc,
	)
}

func (do DateTimeOnly) MarshalJSON() ([]byte, error) {
	if do.IsZero() {
		return []byte("null"), nil
	}

	return json.Marshal(do.Format(time.DateTime))
}

func (do *DateTimeOnly) UnmarshalJSON(data []byte) error {
	value := strings.TrimSpace(string(data))
	if value == "" || value == "null" {
		do.Time = time.Time{}
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("DateTimeOnly: JSON inválido: %w", err)
	}

	return do.UnmarshalParam(str)
}

func (do DateTimeOnly) MarshalText() ([]byte, error) {
	return []byte(do.Format(time.DateTime)), nil
}

func (do *DateTimeOnly) UnmarshalText(data []byte) error {
	return do.UnmarshalParam(string(data))
}

func (do *DateTimeOnly) UnmarshalParam(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || value == "null" {
		do.Time = time.Time{}
		return nil
	}

	value = strings.Trim(value, `"`)
	if value == "" || value == "null" {
		do.Time = time.Time{}
		return nil
	}

	t, err := time.Parse(time.DateTime, value)
	if err != nil {
		return fmt.Errorf("DateTimeOnly: error al parsear `%s`: %w", value, err)
	}

	do.Time = t
	return nil
}

func (do DateTimeOnly) GobEncode() ([]byte, error) {
	return json.Marshal(do)
}

func (do *DateTimeOnly) GobDecode(b []byte) error {
	return json.Unmarshal(b, do)
}

func (do DateTimeOnly) Value() (driver.Value, error) {
	return do.Time.Format(time.DateTime), nil
}

func (do *DateTimeOnly) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		do.Time = time.Time{}

	case time.Time:
		do.Time = v

	case *time.Time:
		if v != nil {
			do.Time = *v
		} else {
			do.Time = time.Time{}
		}

	case DateTimeOnly:
		*do = v

	case *DateTimeOnly:
		if v != nil {
			*do = *v
		} else {
			do.Time = time.Time{}
		}

	case string, []byte:
		str := cast.ToString(v)
		if str == "" {
			do.Time = time.Time{}
		} else {
			t, err := time.Parse(time.DateTime, str)
			if err != nil {
				t, err = cast.ToTimeE(v)
				if err != nil {
					return fmt.Errorf("DateTimeOnly: error al parsear `%s`: %w", str, err)
				}
			}
			do.Time = t
		}

	default:
		return fmt.Errorf("DateTimeOnly: tipo no soportado (%T): %v", v, v)
	}

	*do = NewDateTimeOnly(do.Time)
	return nil
}

func (do DateTimeOnly) String() string {
	return do.Format(time.DateTime)
}

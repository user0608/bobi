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

type DateOnly struct{ time.Time }

var _ json.Marshaler = DateOnly{}
var _ json.Unmarshaler = (*DateOnly)(nil)
var _ encoding.TextMarshaler = DateOnly{}
var _ encoding.TextUnmarshaler = (*DateOnly)(nil)
var _ gob.GobEncoder = DateOnly{}
var _ gob.GobDecoder = (*DateOnly)(nil)
var _ driver.Valuer = DateOnly{}
var _ sql.Scanner = (*DateOnly)(nil)

func NewDateOnlyFromString(value string) (DateOnly, error) {
	var do DateOnly
	if err := do.UnmarshalParam(value); err != nil {
		return do, err
	}
	return do, nil
}

func NewDateOnly(t time.Time) DateOnly {
	str := t.Format(time.DateOnly)
	val, _ := time.Parse(time.DateOnly, str)
	return DateOnly{Time: val}
}

func (do DateOnly) ToTimeInLocation(loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}

	return time.Date(
		do.Year(),
		do.Month(),
		do.Day(),
		0, 0, 0, 0,
		loc,
	)
}

func (do DateOnly) StartOfDayInLocation(loc *time.Location) time.Time {
	return do.ToTimeInLocation(loc)
}

func (do DateOnly) EndOfDayInLocation(loc *time.Location) time.Time {
	day := do.ToTimeInLocation(loc)

	return time.Date(
		day.Year(),
		day.Month(),
		day.Day(),
		23, 59, 59,
		int(time.Second-time.Nanosecond),
		day.Location(),
	)
}

func (do DateOnly) StartOfDayUTC(loc *time.Location) time.Time {
	return do.StartOfDayInLocation(loc).UTC()
}

func (do DateOnly) EndOfDayUTC(loc *time.Location) time.Time {
	return do.EndOfDayInLocation(loc).UTC()
}

func (do DateOnly) ToUTCDayRange(loc *time.Location) (time.Time, time.Time) {
	return do.StartOfDayUTC(loc), do.EndOfDayUTC(loc)
}

func (do DateOnly) BuildUTCDayRange(loc *time.Location, start, end JustTime) (time.Time, time.Time) {
	day := do.StartOfDayInLocation(loc)

	left := day.Add(time.Duration(start))
	right := day.Add(time.Duration(end))

	if !right.After(left) {
		right = right.Add(24 * time.Hour)
	}

	return left.UTC(), right.UTC()
}

func (do DateOnly) MarshalJSON() ([]byte, error) {
	if do.IsZero() {
		return []byte("null"), nil
	}

	return json.Marshal(do.Format(time.DateOnly))
}

func (do *DateOnly) UnmarshalJSON(data []byte) error {
	value := strings.TrimSpace(string(data))
	if value == "" || value == "null" {
		do.Time = time.Time{}
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("DateOnly: JSON inválido: %w", err)
	}

	return do.UnmarshalParam(str)
}

func (do DateOnly) MarshalText() ([]byte, error) {
	return []byte(do.Format(time.DateOnly)), nil
}

func (do *DateOnly) UnmarshalText(data []byte) error {
	return do.UnmarshalParam(string(data))
}

func (do *DateOnly) UnmarshalParam(value string) error {
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

	t, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return fmt.Errorf("DateOnly: error al parsear `%s`: %w", value, err)
	}

	do.Time = t
	return nil
}

func (do DateOnly) GobEncode() ([]byte, error) {
	return json.Marshal(do)
}

func (do *DateOnly) GobDecode(b []byte) error {
	return json.Unmarshal(b, do)
}

func (do DateOnly) Value() (driver.Value, error) {
	return do.Time.Format(time.DateOnly), nil
}

func (do *DateOnly) Scan(value any) error {
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

	case DateOnly:
		*do = v

	case *DateOnly:
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
			t, err := time.Parse(time.DateOnly, str)
			if err != nil {
				t, err = cast.ToTimeE(v)
				if err != nil {
					return fmt.Errorf("DateOnly: error al parsear `%s`: %w", str, err)
				}
			}
			do.Time = t
		}

	default:
		return fmt.Errorf("DateOnly: tipo no soportado (%T): %v", v, v)
	}

	*do = NewDateOnly(do.Time)
	return nil
}

func (do DateOnly) String() string {
	return do.Format(time.DateOnly)
}

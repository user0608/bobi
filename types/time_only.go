package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type JustTime time.Duration

var _ driver.Valuer = JustTime(0)
var _ sql.Scanner = (*JustTime)(nil)
var _ json.Marshaler = JustTime(0)
var _ json.Unmarshaler = (*JustTime)(nil)
var _ encoding.TextMarshaler = JustTime(0)
var _ encoding.TextUnmarshaler = (*JustTime)(nil)

func NewJustTimeFromString(value string) (JustTime, error) {
	var jt JustTime
	if err := jt.UnmarshalParam(value); err != nil {
		return 0, err
	}
	return jt, nil
}

func NewJustTime(t time.Time) JustTime {
	return JustTime(
		time.Duration(t.Hour())*time.Hour +
			time.Duration(t.Minute())*time.Minute +
			time.Duration(t.Second())*time.Second +
			time.Duration(t.Nanosecond())*time.Nanosecond,
	)
}

func (jt JustTime) String() string {
	return jt.Format()
}

func (jt JustTime) Duration() time.Duration {
	return time.Duration(jt)
}

func (jt JustTime) IsZero() bool {
	return jt == 0
}

func (jt JustTime) Add(d time.Duration) JustTime {
	const day = 24 * time.Hour

	v := (time.Duration(jt) + d) % day
	if v < 0 {
		v += day
	}

	return JustTime(v)
}

func (jt JustTime) Sub(other JustTime) time.Duration {
	return time.Duration(jt) - time.Duration(other)
}

func (jt JustTime) ToTime() time.Time {
	now := time.Now()
	return jt.ToTimeOnDateInLocation(now, now.Location())
}

func (jt JustTime) ToTimeInLocation(loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}

	return jt.ToTimeOnDateInLocation(time.Now().In(loc), loc)
}

func (jt JustTime) ToTimeOnDate(date time.Time) time.Time {
	return jt.ToTimeOnDateInLocation(date, date.Location())
}

func (jt JustTime) ToTimeOnDateInLocation(date time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}

	date = date.In(loc)

	return time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		jt.hours(),
		jt.minutes(),
		jt.seconds(),
		jt.nanoseconds(),
		loc,
	)
}

func (jt JustTime) Format() string {
	if jt.nanoseconds() > 0 {
		return fmt.Sprintf("%02d:%02d:%02d.%09d", jt.hours(), jt.minutes(), jt.seconds(), jt.nanoseconds())
	}
	return fmt.Sprintf("%02d:%02d:%02d", jt.hours(), jt.minutes(), jt.seconds())
}

func (jt JustTime) MarshalJSON() ([]byte, error) {
	if jt == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(jt.Format())
}

func (jt *JustTime) UnmarshalJSON(data []byte) error {
	value := strings.TrimSpace(string(data))
	if value == "" || value == "null" {
		*jt = 0
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("JustTime: JSON inválido: %w", err)
	}

	return jt.UnmarshalParam(str)
}

func (jt JustTime) MarshalText() ([]byte, error) {
	return []byte(jt.Format()), nil
}

func (jt *JustTime) UnmarshalText(data []byte) error {
	return jt.UnmarshalParam(string(data))
}

func (jt *JustTime) UnmarshalParam(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || value == "null" {
		*jt = 0
		return nil
	}

	parts := strings.Split(value, ":")
	if len(parts) < 1 || len(parts) > 3 {
		return fmt.Errorf("JustTime: formato inválido `%s`", value)
	}

	h, err := parseTimePart(parts[0], "h", value)
	if err != nil {
		return err
	}

	m := 0
	s := 0
	n := 0

	if len(parts) >= 2 {
		m, err = parseTimePart(parts[1], "m", value)
		if err != nil {
			return err
		}
	}

	if len(parts) == 3 {
		secParts := strings.Split(parts[2], ".")
		if len(secParts) > 2 {
			return fmt.Errorf("JustTime: formato inválido `%s`", value)
		}

		s, err = parseTimePart(secParts[0], "s", value)
		if err != nil {
			return err
		}

		if len(secParts) == 2 {
			n, err = parseNanoseconds(secParts[1], value)
			if err != nil {
				return err
			}
		}
	}

	if h < 0 || h > 23 || m < 0 || m > 59 || s < 0 || s > 59 || n < 0 || n > 999999999 {
		return fmt.Errorf("JustTime: fuera de rango `%s` (h=%d m=%d s=%d n=%d)", value, h, m, s, n)
	}

	*jt = JustTime(
		time.Duration(h)*time.Hour +
			time.Duration(m)*time.Minute +
			time.Duration(s)*time.Second +
			time.Duration(n)*time.Nanosecond,
	)

	return nil
}

func (jt JustTime) Value() (driver.Value, error) {
	return jt.Format(), nil
}

func (jt *JustTime) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*jt = 0
	case time.Time:
		*jt = NewJustTime(v)
	case *time.Time:
		if v == nil {
			*jt = 0
			return nil
		}
		*jt = NewJustTime(*v)
	case JustTime:
		*jt = v
	case *JustTime:
		if v == nil {
			*jt = 0
			return nil
		}
		*jt = *v
	case string:
		if err := jt.UnmarshalParam(v); err != nil {
			return fmt.Errorf("JustTime: error al parsear `%s`: %w", v, err)
		}
	case []byte:
		str := string(v)
		if err := jt.UnmarshalParam(str); err != nil {
			return fmt.Errorf("JustTime: error al parsear `%s`: %w", str, err)
		}
	default:
		return fmt.Errorf("JustTime: tipo no soportado (%T): %v", v, v)
	}

	return nil
}

func (jt JustTime) GobEncode() ([]byte, error) {
	return json.Marshal(jt)
}

func (jt *JustTime) GobDecode(b []byte) error {
	return json.Unmarshal(b, jt)
}

func (jt JustTime) hours() int {
	return int(time.Duration(jt).Truncate(time.Hour).Hours())
}

func (jt JustTime) minutes() int {
	return int((time.Duration(jt) % time.Hour).Truncate(time.Minute).Minutes())
}

func (jt JustTime) seconds() int {
	return int((time.Duration(jt) % time.Minute).Truncate(time.Second).Seconds())
}

func (jt JustTime) nanoseconds() int {
	return int((time.Duration(jt) % time.Second).Nanoseconds())
}

func parseTimePart(value string, name string, original string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("JustTime: formato inválido `%s`", original)
	}

	v, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("JustTime: componente %s inválido en `%s`", name, original)
	}

	return v, nil
}

func parseNanoseconds(value string, original string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 9 {
		return 0, fmt.Errorf("JustTime: fracción inválida en `%s`", original)
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("JustTime: fracción inválida en `%s`", original)
		}
	}

	value += strings.Repeat("0", 9-len(value))

	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("JustTime: fracción inválida en `%s`", original)
	}

	return n, nil
}

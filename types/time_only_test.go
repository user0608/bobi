package types_test

import (
	"github.com/user0608/bobi/types"
	"bytes"

	"encoding/gob"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewJustTimeFromString_Format(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{`12`, "12:00:00", false},
		{`12:30`, "12:30:00", false},
		{`12:30:45`, "12:30:45", false},
		{`00:00:00`, "00:00:00", false},
		{`12:30:45.123456789`, "12:30:45.123456789", false},
		{`23:59:59.999999999`, "23:59:59.999999999", false},

		// errores
		{`25:61:61`, "", true},
		{`abc`, "", true},
		{`12:30:60`, "", true},
		{`12:60`, "", true},
		{`-1:00:00`, "", true},
		{`12:30:45.1000000000`, "", true},
	}

	for _, tt := range tests {
		jt := types.JustTime(0)
		err := jt.UnmarshalParam(tt.input)

		if tt.wantErr {
			if err == nil {
				t.Errorf("input %s: expected error, got none (jt.String()=%s)", tt.input, jt.String())
			}
			continue
		}
		if err != nil {
			t.Errorf("input %s: unexpected error: %v (jt.String()=%s)", tt.input, err, jt.String())
			continue
		}
		if got := jt.Format(); got != tt.expected {
			t.Errorf("input %s: expected %s, got %s (jt.String()=%s)", tt.input, tt.expected, got, jt.String())
		}
	}
}

func TestJustTime_Scan(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
		wantErr  bool
	}{
		{"time.Time", time.Date(1, 1, 1, 12, 0, 0, 0, time.UTC), "12:00:00", false},
		{"*time.Time", ptrTime(time.Date(1, 1, 1, 23, 59, 59, 0, time.UTC)), "23:59:59", false},
		{"JustTime", types.NewJustTime(time.Date(1, 1, 1, 7, 15, 0, 0, time.UTC)), "07:15:00", false},
		{"*JustTime", ptrJT(types.NewJustTime(time.Date(1, 1, 1, 1, 2, 3, 0, time.UTC))), "01:02:03", false},
		{"string", "18:45:59", "18:45:59", false},
		{"[]byte", []byte("22:22:22"), "22:22:22", false},
		{"string con nanos", "12:00:00.000000123", "12:00:00.000000123", false},
		{"nil", nil, "00:00:00", false},
		{"invalid string", "bad-time", "", true},
		{"unsupported type", 123, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jt types.JustTime
			err := jt.Scan(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, jt.String())
			}
		})
	}
}

func TestJustTime_String(t *testing.T) {
	jt := types.NewJustTime(time.Date(1, 1, 1, 8, 8, 8, 0, time.UTC))
	assert.Equal(t, "08:08:08", jt.String())

	jtNano := types.NewJustTime(time.Date(1, 1, 1, 1, 1, 1, 999, time.UTC))
	assert.Contains(t, jtNano.String(), "01:01:01.")
}

func TestJustTime_MarshalJSON(t *testing.T) {
	jt := types.NewJustTime(time.Date(1, 1, 1, 14, 0, 0, 0, time.UTC))
	data, err := json.Marshal(jt)
	assert.NoError(t, err)
	assert.Equal(t, `"14:00:00"`, string(data))

	var zero types.JustTime
	zeroData, err := json.Marshal(zero)
	assert.NoError(t, err)
	assert.Equal(t, "null", string(zeroData))
}

func TestJustTime_UnmarshalJSON(t *testing.T) {
	var jt types.JustTime
	err := json.Unmarshal([]byte(`"05:45:30"`), &jt)
	assert.NoError(t, err)
	assert.Equal(t, "05:45:30", jt.String())

	err = json.Unmarshal([]byte(`"null"`), &jt)
	assert.NoError(t, err)
	assert.Equal(t, "00:00:00", jt.String())

	err = json.Unmarshal([]byte(`"bad"`), &jt)
	assert.Error(t, err)
}

func TestJustTime_UnmarshalParam(t *testing.T) {
	var jt types.JustTime
	err := jt.UnmarshalParam("06:00:00")
	assert.NoError(t, err)
	assert.Equal(t, "06:00:00", jt.String())

	err = jt.UnmarshalParam("invalid")
	assert.Error(t, err)
}

func TestJustTime_GobEncodeDecode(t *testing.T) {
	original := types.NewJustTime(time.Date(1, 1, 1, 20, 0, 0, 0, time.UTC))

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(original)
	assert.NoError(t, err)

	var decoded types.JustTime
	dec := gob.NewDecoder(&buf)
	err = dec.Decode(&decoded)
	assert.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func ptrJT(t types.JustTime) *types.JustTime {
	return &t
}

func TestJustTime_ToTime(t *testing.T) {
	jt := types.NewJustTime(time.Date(1, 1, 1, 14, 30, 0, 0, time.UTC))
	now := time.Now()
	expected := time.Date(now.Year(), now.Month(), now.Day(), 14, 30, 0, 0, time.UTC)

	result := jt.ToTime()

	assert.Equal(t, expected.Year(), result.Year())
	assert.Equal(t, expected.Month(), result.Month())
	assert.Equal(t, expected.Day(), result.Day())
	assert.Equal(t, expected.Hour(), result.Hour())
	assert.Equal(t, expected.Minute(), result.Minute())
	assert.Equal(t, expected.Second(), result.Second())
}

func TestJustTime_ToTimeInLocation(t *testing.T) {
	loc, err := time.LoadLocation("America/Lima")
	assert.NoError(t, err)

	jt := types.NewJustTime(time.Date(1, 1, 1, 9, 15, 0, 0, time.UTC))
	now := time.Now().In(loc)
	expected := time.Date(now.Year(), now.Month(), now.Day(), 9, 15, 0, 0, loc)

	result := jt.ToTimeInLocation(loc)

	assert.Equal(t, expected.Year(), result.Year())
	assert.Equal(t, expected.Month(), result.Month())
	assert.Equal(t, expected.Day(), result.Day())
	assert.Equal(t, expected.Hour(), result.Hour())
	assert.Equal(t, expected.Minute(), result.Minute())
	assert.Equal(t, expected.Second(), result.Second())
	assert.Equal(t, loc.String(), result.Location().String())

	assert.Equal(t, expected.Format(time.RFC3339), result.Format(time.RFC3339))
}

func TestNewJustTimeFromString(t *testing.T) {
	jt, err := types.NewJustTimeFromString("16:20:30.1")

	assert.NoError(t, err)
	assert.Equal(t, "16:20:30.100000000", jt.String())

	_, err = types.NewJustTimeFromString("bad-time")
	assert.Error(t, err)
}

func TestJustTime_Duration(t *testing.T) {
	var jt types.JustTime
	err := jt.UnmarshalParam("01:02:03.000000004")

	assert.NoError(t, err)
	assert.Equal(t, time.Hour+2*time.Minute+3*time.Second+4*time.Nanosecond, jt.Duration())
}

func TestJustTime_IsZero(t *testing.T) {
	var jt types.JustTime

	assert.True(t, jt.IsZero())

	err := jt.UnmarshalParam("00:00:01")
	assert.NoError(t, err)

	assert.False(t, jt.IsZero())
}

func TestJustTime_Add(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		add      time.Duration
		expected string
	}{
		{"same day", "10:00:00", 90 * time.Minute, "11:30:00"},
		{"wrap next day", "23:30:00", 90 * time.Minute, "01:00:00"},
		{"wrap previous day", "00:30:00", -1 * time.Hour, "23:30:00"},
		{"more than one day", "10:00:00", 25 * time.Hour, "11:00:00"},
		{"negative more than one day", "10:00:00", -25 * time.Hour, "09:00:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jt types.JustTime
			err := jt.UnmarshalParam(tt.input)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, jt.Add(tt.add).String())
		})
	}
}

func TestJustTime_Sub(t *testing.T) {
	var a, b types.JustTime

	assert.NoError(t, a.UnmarshalParam("12:30:00"))
	assert.NoError(t, b.UnmarshalParam("10:00:00"))

	assert.Equal(t, 150*time.Minute, a.Sub(b))
	assert.Equal(t, -150*time.Minute, b.Sub(a))
}

func TestJustTime_ToTimeOnDate(t *testing.T) {
	loc, err := time.LoadLocation("America/Lima")
	assert.NoError(t, err)

	var jt types.JustTime
	assert.NoError(t, jt.UnmarshalParam("14:30:15.123456789"))

	date := time.Date(2026, 5, 8, 23, 59, 59, 0, loc)
	result := jt.ToTimeOnDate(date)

	expected := time.Date(2026, 5, 8, 14, 30, 15, 123456789, loc)

	assert.Equal(t, expected, result)
}

func TestJustTime_ToTimeOnDateInLocation(t *testing.T) {
	loc, err := time.LoadLocation("America/Lima")
	assert.NoError(t, err)

	var jt types.JustTime
	assert.NoError(t, jt.UnmarshalParam("21:15:30"))

	date := time.Date(2026, 5, 9, 2, 0, 0, 0, time.UTC)
	result := jt.ToTimeOnDateInLocation(date, loc)

	expected := time.Date(2026, 5, 8, 21, 15, 30, 0, loc)

	assert.Equal(t, expected, result)
	assert.Equal(t, loc.String(), result.Location().String())
}

func TestJustTime_ToTimeInLocation_NilLocationDefaultsUTC(t *testing.T) {
	var jt types.JustTime
	assert.NoError(t, jt.UnmarshalParam("09:15:00"))

	result := jt.ToTimeInLocation(nil)

	assert.Equal(t, "UTC", result.Location().String())
	assert.Equal(t, 9, result.Hour())
	assert.Equal(t, 15, result.Minute())
	assert.Equal(t, 0, result.Second())
}

func TestJustTime_ToTimeOnDateInLocation_NilLocationDefaultsUTC(t *testing.T) {
	var jt types.JustTime
	assert.NoError(t, jt.UnmarshalParam("08:45:30"))

	date := time.Date(2026, 5, 8, 12, 0, 0, 0, time.FixedZone("TEST", -5*60*60))
	result := jt.ToTimeOnDateInLocation(date, nil)

	assert.Equal(t, "UTC", result.Location().String())
	assert.Equal(t, 2026, result.Year())
	assert.Equal(t, time.May, result.Month())
	assert.Equal(t, 8, result.Day())
	assert.Equal(t, 8, result.Hour())
	assert.Equal(t, 45, result.Minute())
	assert.Equal(t, 30, result.Second())
}

func TestJustTime_MarshalText(t *testing.T) {
	var jt types.JustTime
	assert.NoError(t, jt.UnmarshalParam("17:25:40"))

	data, err := jt.MarshalText()

	assert.NoError(t, err)
	assert.Equal(t, "17:25:40", string(data))
}

func TestJustTime_MarshalText_Zero(t *testing.T) {
	var jt types.JustTime

	data, err := jt.MarshalText()

	assert.NoError(t, err)
	assert.Equal(t, "00:00:00", string(data))
}

func TestJustTime_UnmarshalText(t *testing.T) {
	var jt types.JustTime

	err := jt.UnmarshalText([]byte("17:25:40.1"))

	assert.NoError(t, err)
	assert.Equal(t, "17:25:40.100000000", jt.String())
}

func TestJustTime_UnmarshalText_Empty(t *testing.T) {
	var jt types.JustTime

	err := jt.UnmarshalText([]byte(""))

	assert.NoError(t, err)
	assert.Equal(t, "00:00:00", jt.String())
}

func TestJustTime_UnmarshalJSON_RawNull(t *testing.T) {
	var jt types.JustTime

	err := json.Unmarshal([]byte(`null`), &jt)

	assert.NoError(t, err)
	assert.Equal(t, "00:00:00", jt.String())
}

func TestJustTime_UnmarshalJSON_InvalidType(t *testing.T) {
	var jt types.JustTime

	err := json.Unmarshal([]byte(`123`), &jt)

	assert.Error(t, err)
}

func TestJustTime_Value(t *testing.T) {
	var jt types.JustTime
	assert.NoError(t, jt.UnmarshalParam("18:45:59.123456789"))

	value, err := jt.Value()

	assert.NoError(t, err)
	assert.Equal(t, "18:45:59.123456789", value)
}

func TestJustTime_Value_Zero(t *testing.T) {
	var jt types.JustTime

	value, err := jt.Value()

	assert.NoError(t, err)
	assert.Equal(t, "00:00:00", value)
}

func TestJustTime_ScanNilPointersAndEmptyValues(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"nil *time.Time", (*time.Time)(nil)},
		{"nil *JustTime", (*types.JustTime)(nil)},
		{"empty string", ""},
		{"empty bytes", []byte("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jt types.JustTime
			assert.NoError(t, jt.UnmarshalParam("10:00:00"))

			err := jt.Scan(tt.input)

			assert.NoError(t, err)
			assert.Equal(t, "00:00:00", jt.String())
		})
	}
}

func TestJustTime_UnmarshalParamFractionNormalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"12:30:45.1", "12:30:45.100000000"},
		{"12:30:45.12", "12:30:45.120000000"},
		{"12:30:45.123", "12:30:45.123000000"},
		{"12:30:45.1234", "12:30:45.123400000"},
		{"12:30:45.12345", "12:30:45.123450000"},
		{"12:30:45.123456", "12:30:45.123456000"},
		{"12:30:45.1234567", "12:30:45.123456700"},
		{"12:30:45.12345678", "12:30:45.123456780"},
		{"12:30:45.123456789", "12:30:45.123456789"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var jt types.JustTime

			err := jt.UnmarshalParam(tt.input)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, jt.String())
		})
	}
}

func TestJustTime_UnmarshalParamInvalidFormats(t *testing.T) {
	tests := []string{
		"24:00:00",
		"23:60:00",
		"23:59:60",
		"12:30:45.",
		"12:30:45.1234567890",
		"12:30:45.abc",
		"12:30:45.1.2",
		"12::45",
		":30:45",
		"12:30:",
		"12:30:45:10",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			var jt types.JustTime

			err := jt.UnmarshalParam(input)

			assert.Error(t, err)
		})
	}
}

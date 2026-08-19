package pointers_test

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user0608/bobi/tools/pointers"
)

type testValue struct {
	Count int
	Name  string
}

type testInt int

func TestValue(t *testing.T) {
	t.Run("non-zero scalar", func(t *testing.T) {
		assertValue(t, new(math.MinInt), math.MinInt)
	})

	t.Run("pointer to zero scalar", func(t *testing.T) {
		assertValue(t, new(0), 0)
	})

	t.Run("struct", func(t *testing.T) {
		want := testValue{Count: 42, Name: "answer"}
		assertValue(t, new(want), want)
	})

	t.Run("slice", func(t *testing.T) {
		want := []int{1, 2, 3}
		assertValue(t, new(want), want)
	})

	t.Run("map", func(t *testing.T) {
		want := map[string]int{"answer": 42}
		assertValue(t, new(want), want)
	})

	t.Run("function", func(t *testing.T) {
		want := func() int { return 42 }
		got := pointers.Value(new(want))
		require.NotNil(t, got)
		assert.Equal(t, want(), got())
	})

	t.Run("interface", func(t *testing.T) {
		var want any = []int{1, 2, 3}
		assertValue(t, new(want), want)
	})

	t.Run("pointer", func(t *testing.T) {
		want := new(42)
		assertValue(t, new(want), want)
	})

	t.Run("decimal", func(t *testing.T) {
		want := decimal.RequireFromString("1234567890.123456789")
		assert.True(t, pointers.Value(new(want)).Equal(want))
		assert.True(t, pointers.Value[decimal.Decimal](nil).IsZero())
	})

	t.Run("UUID", func(t *testing.T) {
		want := uuid.New()
		assert.Equal(t, want, pointers.Value(new(want)))
		assert.Equal(t, uuid.Nil, pointers.Value[uuid.UUID](nil))
	})

	t.Run("time", func(t *testing.T) {
		want := time.Date(2026, time.August, 19, 12, 30, 45, 123, time.UTC)
		assert.Equal(t, want, pointers.Value(new(want)))
		assert.True(t, pointers.Value[time.Time](nil).IsZero())
	})

	t.Run("nil pointer to scalar", func(t *testing.T) {
		assertValue[int](t, nil, 0)
	})

	t.Run("nil pointer to slice", func(t *testing.T) {
		assertValue[[]int](t, nil, nil)
	})

	t.Run("nil pointer to map", func(t *testing.T) {
		assertValue[map[string]int](t, nil, nil)
	})
}

func TestNilIfZero(t *testing.T) {
	t.Run("zero values return nil", func(t *testing.T) {
		tests := []struct {
			name string
			test func(*testing.T)
		}{
			{name: "int", test: func(t *testing.T) { assertNil(t, pointers.NilIfZero(0)) }},
			{name: "negative zero", test: func(t *testing.T) {
				assertNil(t, pointers.NilIfZero(math.Copysign(0, -1)))
			}},
			{name: "string", test: func(t *testing.T) { assertNil(t, pointers.NilIfZero("")) }},
			{name: "bool", test: func(t *testing.T) { assertNil(t, pointers.NilIfZero(false)) }},
			{name: "complex", test: func(t *testing.T) { assertNil(t, pointers.NilIfZero(complex128(0))) }},
			{name: "named type", test: func(t *testing.T) { assertNil(t, pointers.NilIfZero(testInt(0))) }},
			{name: "struct", test: func(t *testing.T) {
				assertNil(t, pointers.NilIfZero(testValue{}))
			}},
			{name: "array", test: func(t *testing.T) { assertNil(t, pointers.NilIfZero([2]int{})) }},
			{name: "pointer", test: func(t *testing.T) {
				assertNil(t, pointers.NilIfZero[*int](nil))
			}},
			{name: "channel", test: func(t *testing.T) {
				assertNil(t, pointers.NilIfZero[chan int](nil))
			}},
			{name: "interface", test: func(t *testing.T) {
				assertNil(t, pointers.NilIfZero[any](nil))
			}},
		}

		for _, tt := range tests {
			t.Run(tt.name, tt.test)
		}
	})

	t.Run("non-zero values return pointer", func(t *testing.T) {
		assertPointer(t, pointers.NilIfZero(math.MinInt), math.MinInt)
		assertPointer(t, pointers.NilIfZero(math.MaxInt), math.MaxInt)
		assertPointer(t, pointers.NilIfZero("value"), "value")
		assertPointer(t, pointers.NilIfZero(true), true)
		assertPointer(t, pointers.NilIfZero(complex(-1.5, 2.5)), complex(-1.5, 2.5))
		assertPointer(t, pointers.NilIfZero(testInt(-1)), testInt(-1))
		assertPointer(t, pointers.NilIfZero(testValue{Name: "value"}), testValue{Name: "value"})
		assertPointer(t, pointers.NilIfZero([2]int{0, 1}), [2]int{0, 1})

		value := new(42)
		assertPointer(t, pointers.NilIfZero(value), value)

		channel := make(chan int)
		defer close(channel)
		assertPointer(t, pointers.NilIfZero(channel), channel)
	})

	t.Run("NaN is not zero", func(t *testing.T) {
		got := pointers.NilIfZero(math.NaN())
		require.NotNil(t, got)
		assert.True(t, math.IsNaN(*got))
	})

	t.Run("infinity is not zero", func(t *testing.T) {
		assertPointer(t, pointers.NilIfZero(math.Inf(1)), math.Inf(1))
		assertPointer(t, pointers.NilIfZero(math.Inf(-1)), math.Inf(-1))
	})

	t.Run("non-comparable dynamic interface value is not zero", func(t *testing.T) {
		var value any = []int{1, 2, 3}
		got := pointers.NilIfZero(value)
		require.NotNil(t, got)
		assert.Equal(t, value, *got)
	})

	t.Run("decimal", func(t *testing.T) {
		assert.Nil(t, pointers.NilIfZero(decimal.Decimal{}))

		want := decimal.RequireFromString("-999999999.000000001")
		got := pointers.NilIfZero(want)
		require.NotNil(t, got)
		assert.True(t, got.Equal(want))
	})

	t.Run("UUID", func(t *testing.T) {
		assert.Nil(t, pointers.NilIfZero(uuid.Nil))

		want := uuid.New()
		assertPointer(t, pointers.NilIfZero(want), want)
	})

	t.Run("time", func(t *testing.T) {
		assert.Nil(t, pointers.NilIfZero(time.Time{}))

		want := time.Date(2026, time.August, 19, 12, 30, 45, 123, time.UTC)
		assertPointer(t, pointers.NilIfZero(want), want)
	})

	t.Run("returned pointer owns independent storage", func(t *testing.T) {
		value := 42
		got := pointers.NilIfZero(value)
		require.NotNil(t, got)

		*got = 100
		assert.Equal(t, 42, value)
	})
}

func TestValueAndNilIfZeroRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value int
	}{
		{name: "zero", value: 0},
		{name: "positive", value: 42},
		{name: "negative", value: -42},
		{name: "minimum", value: math.MinInt},
		{name: "maximum", value: math.MaxInt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pointers.Value(pointers.NilIfZero(tt.value))
			assert.Equal(t, tt.value, got)
		})
	}
}

func FuzzValueAndNilIfZeroRoundTrip(f *testing.F) {
	f.Add(int64(0), "")
	f.Add(int64(1), "value")
	f.Add(int64(-1), "\x00")
	f.Add(int64(math.MinInt64), "minimum")
	f.Add(int64(math.MaxInt64), "maximum")

	f.Fuzz(func(t *testing.T, integer int64, text string) {
		assert.Equal(t, integer, pointers.Value(pointers.NilIfZero(integer)))
		assert.Equal(t, text, pointers.Value(pointers.NilIfZero(text)))
	})
}

func assertValue[T any](t *testing.T, pointer *T, want T) {
	t.Helper()

	assert.Equal(t, want, pointers.Value(pointer))
}

func assertNil[T any](t *testing.T, got *T) {
	t.Helper()

	assert.Nil(t, got)
}

func assertPointer[T any](t *testing.T, got *T, want T) {
	t.Helper()

	require.NotNil(t, got)
	assert.Equal(t, want, *got)
}

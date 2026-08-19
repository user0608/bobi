package kcheck

import (
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStructCyclicPointersDoNotRecurseForever(t *testing.T) {
	type node struct {
		Name string `chk:"required"`
		Next *node
	}

	root := &node{Name: "root"}
	root.Next = root

	if err := Valid(root); err != nil {
		t.Fatalf("expected cyclic structure to be valid, got %v", err)
	}
}

func TestCyclicMapsDoNotRecurseForever(t *testing.T) {
	values := make(map[string]any)
	values["self"] = values

	type dto struct {
		Values map[string]any
	}

	if err := Valid(dto{Values: values}); err != nil {
		t.Fatalf("expected cyclic map without rules to be valid, got %v", err)
	}
}

func TestRequiredRejectsZeroAndNilValues(t *testing.T) {
	type dto struct {
		Count   int            `chk:"required"`
		Enabled bool           `chk:"required"`
		When    time.Time      `chk:"required"`
		IDs     []int          `chk:"required"`
		Labels  map[string]int `chk:"required"`
		Value   any            `chk:"required"`
	}

	err := Valid(dto{})
	if err == nil {
		t.Fatal("expected zero and nil values to fail required")
	}

	for _, field := range []string{"Count", "Enabled", "When", "IDs", "Labels", "Value"} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("expected required error for %s, got %v", field, err)
		}
	}
}

func TestRequiredAcceptsNonZeroCollectionsAndValues(t *testing.T) {
	type dto struct {
		Count   int            `chk:"required"`
		Enabled bool           `chk:"required"`
		When    time.Time      `chk:"required"`
		IDs     []int          `chk:"required"`
		Labels  map[string]int `chk:"required"`
		Value   any            `chk:"required"`
	}

	value := dto{
		Count:   1,
		Enabled: true,
		When:    time.Unix(1, 0),
		IDs:     []int{1},
		Labels:  map[string]int{"one": 1},
		Value:   "set",
	}
	if err := Valid(value); err != nil {
		t.Fatalf("expected non-zero values to satisfy required, got %v", err)
	}
}

func TestRequiredUUID(t *testing.T) {
	type dto struct {
		ID       uuid.UUID `chk:"required"`
		IDString string    `chk:"required"`
	}

	if err := Valid(dto{ID: uuid.Nil, IDString: ""}); err == nil {
		t.Fatal("expected nil UUID and empty UUID string to fail required")
	}

	value := uuid.New()
	if err := Valid(dto{ID: value, IDString: value.String()}); err != nil {
		t.Fatalf("expected non-zero UUID values to satisfy required, got %v", err)
	}
}

func TestNilValidatorRegistrationIsSafe(t *testing.T) {
	v := New()
	v.Register("broken", nil)

	type dto struct {
		Name string `chk:"broken"`
	}

	if err := v.Struct(dto{Name: "Kevin"}); err == nil {
		t.Fatal("expected unregistered nil validator to report an error")
	}
}

func TestMultiplePointersAreSupported(t *testing.T) {
	value := "ABC123"
	pointer := &value

	type dto struct {
		Code **string `chk:"required upper len=6"`
	}

	if err := Valid(dto{Code: &pointer}); err != nil {
		t.Fatalf("expected multiple pointers to validate, got %v", err)
	}
}

func TestNestedCollectionsAndInterfacesAreValidated(t *testing.T) {
	type address struct {
		City string `chk:"required min=3"`
	}
	type dto struct {
		Addresses []address
		ByName    map[string]address
		Any       any
	}

	value := dto{
		Addresses: []address{{City: "Li"}},
		ByName:    map[string]address{"home": {City: "Ok"}},
		Any:       address{City: "No"},
	}
	err := Valid(value)
	if err == nil {
		t.Fatal("expected nested collection validation errors")
	}

	for _, path := range []string{"Addresses[0].City", "ByName[home].City", "Any.City"} {
		if !strings.Contains(err.Error(), path) {
			t.Errorf("expected error path %s, got %v", path, err)
		}
	}
}

func TestSelectNestedCollectionByIndexedPath(t *testing.T) {
	type address struct {
		City string `chk:"required min=3"`
	}
	type dto struct {
		Addresses []address
		Name      string `chk:"required"`
	}

	value := dto{Addresses: []address{{City: "Li"}}}
	err := ValidSelect(value, "Addresses[0].City")
	if err == nil || !strings.Contains(err.Error(), "Addresses[0].City") {
		t.Fatalf("expected indexed selection error, got %v", err)
	}
}

func TestExactNumericComparisonsRejectPrecisionLoss(t *testing.T) {
	const value = uint64(1<<53 + 1)
	type dto struct {
		Value uint64 `chk:"gt=9007199254740992"`
	}

	if err := Valid(dto{Value: value}); err != nil {
		t.Fatalf("expected exact uint64 comparison to pass, got %v", err)
	}
}

func TestNumericComparisonsRejectNonFiniteFloats(t *testing.T) {
	type dto struct {
		Value float64 `chk:"min=0 max=10"`
	}

	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if err := Valid(dto{Value: value}); err == nil {
			t.Errorf("expected non-finite value %v to fail", value)
		}
	}
}

func TestURLRejectsNonHTTPProtocols(t *testing.T) {
	if err := urlValue(Field{Path: "URL", Value: "ftp://example.com/file"}); err == nil {
		t.Fatal("expected ftp URL to be rejected")
	}
}

func TestValidatorsAreSafeForConcurrentUse(t *testing.T) {
	v := New()
	v.Register("custom", func(Field) error { return nil })

	type dto struct {
		Name string `chk:"required custom"`
	}

	const workers = 32
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			if err := v.Struct(dto{Name: "Kevin"}); err != nil {
				t.Errorf("concurrent validation failed: %v", err)
			}
		}()
	}
	wg.Wait()
}

package list

import (
	"reflect"
	"testing"
)

func TestUnique(t *testing.T) {
	in := []int{1, 2, 2, 3, 1, 4, 4, 5}
	out := Unique(in)

	expected := []int{1, 2, 3, 4, 5}

	if len(out) != len(expected) {
		t.Fatalf("len mismatch: got %v, want %v", out, expected)
	}

	for i := range expected {
		if out[i] != expected[i] {
			t.Fatalf("value mismatch: got %v, want %v", out, expected)
		}
	}
}

func TestUniqueString(t *testing.T) {
	in := []string{"a", "b", "a", "c", "b", "d"}
	out := Unique(in)

	expected := []string{"a", "b", "c", "d"}

	if len(out) != len(expected) {
		t.Fatalf("len mismatch: got %v, want %v", out, expected)
	}

	for i := range expected {
		if out[i] != expected[i] {
			t.Fatalf("value mismatch: got %v, want %v", out, expected)
		}
	}
}

func TestNonZeroUniques_Int(t *testing.T) {
	input := []int{0, 1, 2, 2, 0, 3, 1}
	expected := []int{1, 2, 3}

	result := NonZeroUniques(input)

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestNonZeroUniques_String(t *testing.T) {
	input := []string{"", "a", "b", "a", "", "c"}
	expected := []string{"a", "b", "c"}

	result := NonZeroUniques(input)

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestNonZeroUniques_AllZero(t *testing.T) {
	input := []int{0, 0, 0}
	expected := []int{}

	result := NonZeroUniques(input)

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestNonZeroUniques_AlreadyUnique(t *testing.T) {
	input := []int{1, 2, 3}
	expected := []int{1, 2, 3}

	result := NonZeroUniques(input)

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestNonZeroUniques_Struct(t *testing.T) {
	type S struct {
		A int
	}

	input := []S{{}, {1}, {2}, {1}, {}}
	expected := []S{{1}, {2}}

	result := NonZeroUniques(input)

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestNonZeroUniques_Empty(t *testing.T) {
	input := []int{}
	expected := []int{}

	result := NonZeroUniques(input)

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestFilter(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}
	expected := []int{2, 4}

	result := Filter(input, func(n int) bool {
		return n%2 == 0
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestFilter_NoMatches(t *testing.T) {
	input := []int{1, 3, 5}
	expected := []int{}

	result := Filter(input, func(n int) bool {
		return n%2 == 0
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestFilter_Empty(t *testing.T) {
	input := []int{}
	expected := []int{}

	result := Filter(input, func(n int) bool {
		return n > 0
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestMap(t *testing.T) {
	input := []int{1, 2, 3}
	expected := []int{2, 4, 6}

	result := Map(input, func(n int) int {
		return n * 2
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestMap_ChangeType(t *testing.T) {
	input := []string{"go", "rust", "js"}
	expected := []int{2, 4, 2}

	result := Map(input, func(s string) int {
		return len(s)
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestMap_Empty(t *testing.T) {
	input := []int{}
	expected := []int{}

	result := Map(input, func(n int) int {
		return n * 2
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestReduce(t *testing.T) {
	input := []int{1, 2, 3, 4}
	expected := 10

	result := Reduce(input, 0, func(sum, n int) int {
		return sum + n
	})

	if result != expected {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestReduce_WithInitialValue(t *testing.T) {
	input := []int{1, 2, 3}
	expected := 16

	result := Reduce(input, 10, func(sum, n int) int {
		return sum + n
	})

	if result != expected {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestReduce_Empty(t *testing.T) {
	input := []int{}
	expected := 10

	result := Reduce(input, 10, func(sum, n int) int {
		return sum + n
	})

	if result != expected {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestFind(t *testing.T) {
	input := []int{1, 2, 3, 4}

	result, ok := Find(input, func(n int) bool {
		return n > 2
	})

	if !ok {
		t.Fatal("expected value to be found")
	}

	if result != 3 {
		t.Fatalf("got %v, want %v", result, 3)
	}
}

func TestFind_NotFound(t *testing.T) {
	input := []int{1, 2, 3}

	result, ok := Find(input, func(n int) bool {
		return n > 5
	})

	if ok {
		t.Fatal("expected value not to be found")
	}

	if result != 0 {
		t.Fatalf("got %v, want %v", result, 0)
	}
}

func TestFind_Empty(t *testing.T) {
	input := []string{}

	result, ok := Find(input, func(s string) bool {
		return s == "go"
	})

	if ok {
		t.Fatal("expected value not to be found")
	}

	if result != "" {
		t.Fatalf("got %v, want empty string", result)
	}
}

func TestPartition(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}

	matched, unmatched := Partition(input, func(n int) bool {
		return n%2 == 0
	})

	expectedMatched := []int{2, 4}
	expectedUnmatched := []int{1, 3, 5}

	if !reflect.DeepEqual(matched, expectedMatched) {
		t.Fatalf("matched: got %v, want %v", matched, expectedMatched)
	}

	if !reflect.DeepEqual(unmatched, expectedUnmatched) {
		t.Fatalf("unmatched: got %v, want %v", unmatched, expectedUnmatched)
	}
}

func TestPartition_AllMatch(t *testing.T) {
	input := []int{2, 4, 6}

	matched, unmatched := Partition(input, func(n int) bool {
		return n%2 == 0
	})

	expectedMatched := []int{2, 4, 6}
	expectedUnmatched := []int{}

	if !reflect.DeepEqual(matched, expectedMatched) {
		t.Fatalf("matched: got %v, want %v", matched, expectedMatched)
	}

	if !reflect.DeepEqual(unmatched, expectedUnmatched) {
		t.Fatalf("unmatched: got %v, want %v", unmatched, expectedUnmatched)
	}
}

func TestPartition_NoneMatch(t *testing.T) {
	input := []int{1, 3, 5}

	matched, unmatched := Partition(input, func(n int) bool {
		return n%2 == 0
	})

	expectedMatched := []int{}
	expectedUnmatched := []int{1, 3, 5}

	if !reflect.DeepEqual(matched, expectedMatched) {
		t.Fatalf("matched: got %v, want %v", matched, expectedMatched)
	}

	if !reflect.DeepEqual(unmatched, expectedUnmatched) {
		t.Fatalf("unmatched: got %v, want %v", unmatched, expectedUnmatched)
	}
}

func TestFlatMap(t *testing.T) {
	input := []int{1, 2, 3}
	expected := []int{1, 1, 2, 2, 3, 3}

	result := FlatMap(input, func(n int) []int {
		return []int{n, n}
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestFlatMap_ChangeType(t *testing.T) {
	input := []string{"go", "js"}
	expected := []rune{'g', 'o', 'j', 's'}

	result := FlatMap(input, func(s string) []rune {
		return []rune(s)
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestFlatMap_Empty(t *testing.T) {
	input := []int{}
	expected := []int{}

	result := FlatMap(input, func(n int) []int {
		return []int{n, n}
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestGroupBy(t *testing.T) {
	input := []string{"go", "js", "rust", "java"}
	expected := map[int][]string{
		2: {"go", "js"},
		4: {"rust", "java"},
	}

	result := GroupBy(input, func(s string) int {
		return len(s)
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestGroupBy_Empty(t *testing.T) {
	input := []string{}
	expected := map[int][]string{}

	result := GroupBy(input, func(s string) int {
		return len(s)
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestKeyBy(t *testing.T) {
	input := []string{"go", "rust", "js"}
	expected := map[int]string{
		2: "js",
		4: "rust",
	}

	result := KeyBy(input, func(s string) int {
		return len(s)
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestKeyBy_Empty(t *testing.T) {
	input := []string{}
	expected := map[int]string{}

	result := KeyBy(input, func(s string) int {
		return len(s)
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestCountBy(t *testing.T) {
	input := []string{"go", "js", "rust", "java", "c"}
	expected := map[int]int{
		1: 1,
		2: 2,
		4: 2,
	}

	result := CountBy(input, func(s string) int {
		return len(s)
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

func TestCountBy_Empty(t *testing.T) {
	input := []string{}
	expected := map[int]int{}

	result := CountBy(input, func(s string) int {
		return len(s)
	})

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("got %v, want %v", result, expected)
	}
}

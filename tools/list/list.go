package list

// Unique returns the unique values from list, keeping the original order.
// Example: Unique([]int{1, 2, 2, 3}) returns []int{1, 2, 3}.
func Unique[T comparable](list []T) []T {
	seen := make(map[T]struct{}, len(list))
	out := make([]T, 0, len(list))

	for _, v := range list {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}

	return out
}

// NonZeroUniques returns the unique nonzero values from list.
// Example: NonZeroUniques([]int{0, 1, 1, 2}) returns []int{1, 2}.
func NonZeroUniques[T comparable](list []T) []T {
	result := make([]T, 0, len(list))
	existMap := make(map[T]struct{}, len(list))

	var zeroVal T

	for _, val := range list {
		if val == zeroVal {
			continue
		}
		if _, ok := existMap[val]; ok {
			continue
		}
		existMap[val] = struct{}{}
		result = append(result, val)
	}

	return result
}

// Filter returns the values that satisfy keep.
// Example: Filter([]int{1, 2, 3}, func(n int) bool { return n > 1 }) returns []int{2, 3}.
func Filter[T any](list []T, keep func(T) bool) []T {
	out := make([]T, 0, len(list))

	for _, v := range list {
		if keep(v) {
			out = append(out, v)
		}
	}

	return out
}

// Map transforms each value from list into a new value.
// Example: Map([]int{1, 2}, func(n int) string { return fmt.Sprint(n) }) returns []string{"1", "2"}.
func Map[T any, R any](list []T, transform func(T) R) []R {
	out := make([]R, 0, len(list))

	for _, v := range list {
		out = append(out, transform(v))
	}

	return out
}

// Reduce combines all values into one result.
// Example: Reduce([]int{1, 2, 3}, 0, func(sum, n int) int { return sum + n }) returns 6.
func Reduce[T any, R any](list []T, initial R, reducer func(R, T) R) R {
	result := initial

	for _, v := range list {
		result = reducer(result, v)
	}

	return result
}

// Find returns the first value that satisfies match.
// Example: Find([]int{1, 2, 3}, func(n int) bool { return n > 1 }) returns 2, true.
func Find[T any](list []T, match func(T) bool) (T, bool) {
	for _, v := range list {
		if match(v) {
			return v, true
		}
	}

	var zero T
	return zero, false
}

// Partition splits list into values that match and values that do not.
// Example: Partition([]int{1, 2, 3}, func(n int) bool { return n%2 == 0 }) returns []int{2}, []int{1, 3}.
func Partition[T any](list []T, match func(T) bool) ([]T, []T) {
	matched := make([]T, 0, len(list))
	unmatched := make([]T, 0, len(list))

	for _, v := range list {
		if match(v) {
			matched = append(matched, v)
			continue
		}

		unmatched = append(unmatched, v)
	}

	return matched, unmatched
}

// FlatMap transforms each value into a slice and flattens the result.
// Example: FlatMap([]int{1, 2}, func(n int) []int { return []int{n, n} }) returns []int{1, 1, 2, 2}.
func FlatMap[T any, R any](list []T, transform func(T) []R) []R {
	out := make([]R, 0, len(list))

	for _, v := range list {
		out = append(out, transform(v)...)
	}

	return out
}

// GroupBy groups values by the key returned from key.
// Example: GroupBy([]string{"go", "js"}, func(s string) int { return len(s) }) returns map[int][]string{2: {"go", "js"}}.
func GroupBy[T any, K comparable](list []T, key func(T) K) map[K][]T {
	out := make(map[K][]T)

	for _, v := range list {
		k := key(v)
		out[k] = append(out[k], v)
	}

	return out
}

// KeyBy indexes values by the key returned from key.
// Example: KeyBy([]string{"go", "rust"}, func(s string) int { return len(s) }) returns map[int]string{2: "go", 4: "rust"}.
func KeyBy[T any, K comparable](list []T, key func(T) K) map[K]T {
	out := make(map[K]T, len(list))

	for _, v := range list {
		out[key(v)] = v
	}

	return out
}

// CountBy counts values by the key returned from key.
// Example: CountBy([]string{"go", "js", "rust"}, func(s string) int { return len(s) }) returns map[int]int{2: 2, 4: 1}.
func CountBy[T any, K comparable](list []T, key func(T) K) map[K]int {
	out := make(map[K]int)

	for _, v := range list {
		out[key(v)]++
	}

	return out
}

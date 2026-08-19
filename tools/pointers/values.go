package pointers

func Value[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	}
	return *v
}

func NilIfZero[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return new(v)
}

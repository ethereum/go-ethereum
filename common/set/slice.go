package set

func New[T comparable](slice []T) map[T]struct{} {
	m := make(map[T]struct{}, len(slice))

	for _, el := range slice {
		m[el] = struct{}{}
	}

	return m
}

func ToSlice[T comparable](m map[T]struct{}) []T {
	slice := make([]T, len(m))

	var i int

	for k := range m {
		slice[i] = k
		i++
	}

	return slice
}

func Deduplicate[T comparable](slice []T) []T {
	return ToSlice(New(slice))
}

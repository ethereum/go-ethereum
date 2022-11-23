package set

func New[T comparable](slice []T) map[T]struct{} {
	m := make(map[T]struct{}, len(slice))

	for _, el := range slice {
		m[el] = struct{}{}
	}

	return m
}

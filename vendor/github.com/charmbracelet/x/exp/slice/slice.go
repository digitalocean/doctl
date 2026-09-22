package slice

// GroupBy groups a slice of items by a key function.
func GroupBy[T any, K comparable](list []T, key func(T) K) map[K][]T {
	groups := make(map[K][]T)

	for _, item := range list {
		k := key(item)
		groups[k] = append(groups[k], item)
	}

	return groups
}

// Take returns the first n elements of the given slice. If there are not
// enough elements in the slice, the whole slice is returned.
func Take[A any](slice []A, n int) []A {
	if n > len(slice) {
		return slice
	}
	return slice[:n]
}

// Uniq returns a new slice with all duplicates removed.
func Uniq[T comparable](list []T) []T {
	seen := make(map[T]struct{}, len(list))
	uniqList := make([]T, 0, len(list))

	for _, item := range list {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			uniqList = append(uniqList, item)
		}
	}

	return uniqList
}

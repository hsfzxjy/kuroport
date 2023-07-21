package ku

func Map[T1, T2 any](arr []T1, f func(T1) T2) []T2 {
	result := make([]T2, len(arr))
	for i, v := range arr {
		result[i] = f(v)
	}
	return result
}

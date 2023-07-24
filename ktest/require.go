//go:build test

package ktest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func RequireAllEqual[T any](t *testing.T, values []T) {
	for i := 1; i < len(values); i++ {
		require.Equalf(t, values[0], values[i], "values[0] != values[%d]", i)
	}
}

func RequireAllNotEqual[T any](t *testing.T, values []T) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			require.NotEqualf(t, values[i], values[j], "values[%d] == values[%d]", i, j)
		}
	}
}

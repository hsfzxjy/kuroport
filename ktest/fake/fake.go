//go:build test

package kfake

import "math/rand"

func Bytes(n int) []byte {
	var buf = make([]byte, n)
	rand.Read(buf)
	return buf
}

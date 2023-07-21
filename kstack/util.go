package kstack

import (
	"unsafe"

	"go4.org/intern"
)

type Hash struct{ v *intern.Value }

func (h Hash) Uint64() uint64 {
	return uint64(uintptr(unsafe.Pointer(h.v)))
}

func GetHashForString(str string) Hash {
	return Hash{intern.GetByString(str)}
}

func GetHash(obj any) Hash {
	return Hash{intern.Get(obj)}
}

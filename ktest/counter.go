package ktest

import (
	"sync"
	"unsafe"
)

type Counter uint64

func asWaitGroup(c *Counter) *sync.WaitGroup {
	return (*sync.WaitGroup)(unsafe.Pointer(c))
}

func (c *Counter) Add() {
	asWaitGroup(c).Done()
}

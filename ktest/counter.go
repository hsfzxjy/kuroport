package ktest

import (
	"reflect"
	"sync"
	"unsafe"
)

type Counter uint64

func asWaitGroup(c *Counter) *sync.WaitGroup {
	return (*sync.WaitGroup)(unsafe.Pointer(c))
}

func (c *Counter) expect() {
	v := int(*c)
	*c = 0
	asWaitGroup(c).Add(v)
}

func (c *Counter) Add() {
	asWaitGroup(c).Done()
}

func ResetTracker(tracker any) <-chan struct{} {
	trackerV := reflect.ValueOf(tracker)
	trackerT := reflect.TypeOf(tracker).Elem()

	trackerPtr := trackerV.UnsafePointer()

	var wgs []*sync.WaitGroup

	for i := 0; i < trackerT.NumField(); i++ {
		f := trackerT.Field(i)
		if f.Type.PkgPath() == "ktest" && f.Type.Name() == "Counter" {
			c := (*Counter)(unsafe.Add(trackerPtr, f.Offset))
			c.expect()
			wg := asWaitGroup(c)
			wgs = append(wgs, wg)
		}
	}

	waitCh := make(chan struct{})
	go func() {
		for _, wg := range wgs {
			wg.Wait()
		}
		close(waitCh)
	}()
	return waitCh
}

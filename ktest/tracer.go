//go:build test

package ktest

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unsafe"
)

func ResetTracer(tracer any) <-chan struct{} {
	tracerV := reflect.ValueOf(tracer)
	tracerT := reflect.TypeOf(tracer).Elem()

	tracerPtr := tracerV.UnsafePointer()

	var wgs []*sync.WaitGroup

	for i := 0; i < tracerT.NumField(); i++ {
		f := tracerT.Field(i)
		if f.Type.PkgPath() == "ktest" && f.Type.Name() == "Counter" {
			c := (*Counter)(unsafe.Add(tracerPtr, f.Offset))
			v := uint64(*c)
			*c = 0
			wg := (*sync.WaitGroup)(unsafe.Pointer(c))
			wg.Add(int(v))
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

func ReportTracer(tracer any, expected any) string {
	builder := &strings.Builder{}

	tracerV := reflect.ValueOf(tracer)
	tracerT := reflect.TypeOf(tracer).Elem()

	tracerPtr := tracerV.UnsafePointer()
	expectedPtr := reflect.ValueOf(expected).UnsafePointer()

	for i := 0; i < tracerT.NumField(); i++ {
		f := tracerT.Field(i)
		if f.Type.PkgPath() == "ktest" && f.Type.Name() == "Counter" {
			v := *(*uint64)(unsafe.Add(tracerPtr, f.Offset))
			e := *(*uint64)(unsafe.Add(expectedPtr, f.Offset))

			v = v >> 32
			if v != 0 {
				fmt.Fprintf(builder, "\nCounter %s:\texpected %d, got %d", f.Name, e, e-v)
			}

		}
	}

	builder.WriteRune('\n')
	return builder.String()
}

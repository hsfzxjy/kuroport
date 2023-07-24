//go:build test

package tracer

import (
	"ktest"
	"time"
)

const Enabled = true

var expected *_Tracer

func init() {
	T = new(_Tracer)
}

var waitCh <-chan struct{}

func doWait(t _HasDeadline) {
	if p := recover(); p != nil {
		panic(p)
	}

	var timeout time.Duration
	if ddl, ok := t.Deadline(); ok {
		timeout = time.Until(ddl)
		if timeout > 0 {
			timeout = timeout / 2
		}
	}

	if timeout <= 0 {
		<-waitCh
	} else {
		select {
		case <-waitCh:
		case <-time.After(timeout):
			panic(ktest.ReportTracer(T, expected))
		}
	}
}

func doExpect(t _Tracer) chainer {
	*T = t
	expected = &t
	waitCh = ktest.ResetTracer(T)
	return chainer{}
}

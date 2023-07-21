package tracer

import (
	"fmt"
	"ktest"
	"testing"
	"time"
)

type _Tracer struct {
	TrSlotDeleted ktest.Counter
	_             uint32
	ConnDeleted   ktest.Counter
	_             uint32
	waitCh        <-chan struct{}
}

var T *_Tracer

type Type = _Tracer

func Wait(t *testing.T) {
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
		<-T.waitCh
	} else {
		select {
		case <-T.waitCh:
		case <-time.After(timeout):
			panic(fmt.Sprintf("%#v", T))
		}
	}
}

func Expect(expected _Tracer) {
	*T = expected
	T.waitCh = ktest.ResetTracker(T)
}

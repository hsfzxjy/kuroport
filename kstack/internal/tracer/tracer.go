package tracer

import (
	"ktest"
	"time"
)

type _Tracer struct {
	TrSlotDeleted ktest.Counter
	_             uint32
	ConnDeleted   ktest.Counter
	_             uint32
}

type _HasDeadline interface {
	Deadline() (time.Time, bool)
}

var T *_Tracer

type Type = _Tracer

func Wait(t _HasDeadline) {
	doWait(t)
}

func Expect(t _Tracer) chainer {
	return doExpect(t)
}

type chainer struct{}

func (chainer) Wait(t _HasDeadline) { Wait(t) }

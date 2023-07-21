package kstack

import "ktest"

type _Tracker struct {
	TrDeleted   ktest.Counter
	_           uint32
	ConnDeleted ktest.Counter
	_           uint32
	waitCh      <-chan struct{}
}

func (t *_Tracker) Reset(expected _Tracker) {
	*t = expected
	t.waitCh = ktest.ResetTracker(t)
}

var tracker *_Tracker

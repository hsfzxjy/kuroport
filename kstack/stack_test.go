package kstack

import (
	"fmt"
	"testing"
	"time"
)

const TestFamily = testFamily

func init() {
	tracker = Tracker
}

var Tracker = &_Tracker{}

type TrackerValue = _Tracker

func (tracker *_Tracker) Wait(t *testing.T, timeout time.Duration) {
	if p := recover(); p != nil {
		panic(p)
	}
	if timeout <= 0 {
		<-tracker.waitCh
	} else {
		select {
		case <-tracker.waitCh:
		case <-time.After(timeout):
			panic(fmt.Sprintf("%#v", tracker))
		}
	}
}

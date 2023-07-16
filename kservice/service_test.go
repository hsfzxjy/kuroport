package kservice_test

import (
	"context"
	"errors"
	"kservice"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/ryboe/q"
	"github.com/stretchr/testify/require"
)

func collectEventsUntilStopped(service *kservice.Service, nStopped int) <-chan []kservice.Event {
	result := make(chan []kservice.Event)
	ch, dispose := service.Events().Listen()
	go func() {
		var eventList []kservice.Event
		defer dispose()

		i := 0
		for event := range ch {
			eventList = append(eventList, event)
			if event.State == kservice.Stopped {
				i += 1
			}
			if i == nStopped {
				break
			}
		}
		result <- eventList
	}()
	return result
}

type option struct {
	errToReturn       error
	shouldWaitContext bool
	waitTime          time.Duration

	// service closes this before the callback is going to block
	readyCh chan struct{}

	// service waits on this before the callback is going to return
	returnCh chan option
}

func (o *option) handleBeforeReturn() {
	if o.returnCh != nil {
		newOption, ok := <-o.returnCh
		if ok {
			*o = newOption
		}
	}
}

func (o *option) createTimer() (<-chan struct{}, context.CancelFunc) {
	if o.waitTime > 0 {
		ddlCtx, ddlCancel := context.WithTimeout(
			context.Background(),
			o.waitTime,
		)
		return ddlCtx.Done(), ddlCancel
	}
	return nil, func() {}
}

type testCallbacks struct {
	x int

	startOption option
	runOption   option

	timerCancel atomic.Value
}

func (s *testCallbacks) do(ctx context.Context, option *option) (err error) {
	defer option.handleBeforeReturn()

	err = option.errToReturn
	defer func() {
		if err == nil {
			s.x += 1
		}
	}()

	s.x += 1

	timerC, timerCancel := option.createTimer()
	defer timerCancel()
	s.timerCancel.Store(timerCancel)

	if option.readyCh != nil {
		close(option.readyCh)
	}

	if option.shouldWaitContext {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timerC:
			return
		}
	}

	if timerC != nil {
		<-timerC
		return
	}

	return
}

// OnServiceStart implements kservice._ICallbacks.
func (s *testCallbacks) OnServiceStart(ctx context.Context) (err error) {
	return s.do(ctx, &s.startOption)
}

// OnServiceRun implements kservice._ICallbacks.
func (s *testCallbacks) OnServiceRun(ctx context.Context) (err error) {
	return s.do(ctx, &s.runOption)
}

// OnServiceStop implements kservice._ICallbacks.
func (s *testCallbacks) OnServiceStop() {
	if ddlCancel, ok := s.timerCancel.Load().(context.CancelFunc); ok {
		ddlCancel()
	}
}

var Err1 = errors.New("Error1")

func TestService(t *testing.T) {
	cb := &testCallbacks{}
	service := kservice.New(cb)

	// first run
	events := collectEventsUntilStopped(service, 2)
	service.Start()

	require.Equal(t, []kservice.Event{
		{kservice.Stopped, nil},
		{kservice.Starting, nil},
		{kservice.Started, nil},
		{kservice.Stopped, nil},
	}, <-events)

	// second run
	events = collectEventsUntilStopped(service, 2)
	service.Start()

	require.Equal(t, []kservice.Event{
		{kservice.Stopped, nil},
		{kservice.Starting, nil},
		{kservice.Started, nil},
		{kservice.Stopped, nil},
	}, <-events)

	require.Equal(t, 8, cb.x)
}

func TestServiceReturnsError(t *testing.T) {
	cb := &testCallbacks{startOption: option{
		errToReturn: Err1,
	}}
	service := kservice.New(cb)

	// first run
	events := collectEventsUntilStopped(service, 2)
	service.Start()

	require.Equal(t, []kservice.Event{
		{kservice.Stopped, nil},
		{kservice.Starting, nil},
		{kservice.Stopped, Err1},
	}, <-events)

	require.Equal(t, 1, cb.x)

	// second run
	cb = &testCallbacks{runOption: option{
		errToReturn: Err1,
	}}
	service = kservice.New(cb)

	events = collectEventsUntilStopped(service, 2)
	service.Start()

	require.Equal(t, []kservice.Event{
		{kservice.Stopped, nil},
		{kservice.Starting, nil},
		{kservice.Started, nil},
		{kservice.Stopped, Err1},
	}, <-events)

	require.Equal(t, 3, cb.x)
}

func TestServiceCancelStart(t *testing.T) {
	cb := &testCallbacks{startOption: option{
		errToReturn:       Err1,
		shouldWaitContext: true,
		readyCh:           make(chan struct{}),
	}}
	service := kservice.New(cb)
	events := collectEventsUntilStopped(service, 2)
	service.Start()
	<-cb.startOption.readyCh
	service.Stop()

	require.Equal(t, []kservice.Event{
		{kservice.Stopped, nil},
		{kservice.Starting, nil},
		{kservice.Stopped, context.Canceled},
	}, <-events)

	require.Equal(t, 1, cb.x)
}

func TestServiceCancelRun(t *testing.T) {
	cb := &testCallbacks{runOption: option{
		errToReturn:       Err1,
		shouldWaitContext: true,
		readyCh:           make(chan struct{}),
	}}
	service := kservice.New(cb)
	events := collectEventsUntilStopped(service, 2)
	service.Start()
	<-cb.runOption.readyCh
	service.Stop()

	require.Equal(t, []kservice.Event{
		{kservice.Stopped, nil},
		{kservice.Starting, nil},
		{kservice.Started, nil},
		{kservice.Stopping, nil},
		{kservice.Stopped, context.Canceled},
	}, <-events)

	require.Equal(t, 3, cb.x)
}

func TestServiceStopStartWithoutWaitContext(t *testing.T) {
	cb := &testCallbacks{startOption: option{
		errToReturn: Err1,
		waitTime:    1 * time.Second,
		readyCh:     make(chan struct{}),
	}}
	service := kservice.New(cb)
	events := collectEventsUntilStopped(service, 2)
	service.Start()
	<-cb.startOption.readyCh
	service.Stop()

	require.Equal(t, []kservice.Event{
		{kservice.Stopped, nil},
		{kservice.Starting, nil},
		{kservice.Stopped, Err1},
	}, <-events)

	require.Equal(t, 1, cb.x)
}

func TestServiceStopRunWithoutWaitContext(t *testing.T) {
	cb := &testCallbacks{startOption: option{
		readyCh: make(chan struct{}),
	}, runOption: option{
		errToReturn: Err1,
		waitTime:    1 * time.Second,
		readyCh:     make(chan struct{}),
	}}
	service := kservice.New(cb)
	events := collectEventsUntilStopped(service, 2)
	service.Start()
	<-cb.startOption.readyCh
	<-cb.runOption.readyCh
	service.Stop()

	require.Equal(t, []kservice.Event{
		{kservice.Stopped, nil},
		{kservice.Starting, nil},
		{kservice.Started, nil},
		{kservice.Stopping, nil},
		{kservice.Stopped, Err1},
	}, <-events)

	require.Equal(t, 3, cb.x)
}

func TestServiceStopStartAndTogglingSeveralTimes(t *testing.T) {
	cb := &testCallbacks{startOption: option{
		errToReturn: Err1,
		waitTime:    1 * time.Second,
		readyCh:     make(chan struct{}),
		returnCh:    make(chan option),
	}}
	service := kservice.New(cb)
	events := collectEventsUntilStopped(service, 2)
	service.Start()
	<-cb.startOption.readyCh
	service.Stop()

	service.Toggle()
	service.Start()
	service.Stop()
	service.Toggle()
	service.Stop()
	close(cb.startOption.returnCh)

	require.Equal(t, []kservice.Event{
		{kservice.Stopped, nil},
		{kservice.Starting, nil},
		{kservice.Stopped, Err1},
	}, <-events)

	require.Equal(t, 1, cb.x)
}

func TestServiceStopStartAndTogglingSeveralTimesButEventuallyRestart(t *testing.T) {
	cb := &testCallbacks{startOption: option{
		errToReturn: Err1,
		waitTime:    1 * time.Second,
		readyCh:     make(chan struct{}),
		returnCh:    make(chan option),
	}}
	service := kservice.New(cb)
	events := collectEventsUntilStopped(service, 3)
	service.Start()
	<-cb.startOption.readyCh
	service.Stop()

	service.Toggle()
	service.Start()
	service.Stop()
	service.Toggle()
	service.Start()
	newOption := option{
		errToReturn: Err1,
		readyCh:     make(chan struct{}),
		returnCh:    make(chan option),
	}
	cb.startOption.returnCh <- newOption

	<-newOption.readyCh
	close(newOption.returnCh)

	require.Equal(t, []kservice.Event{
		{kservice.Stopped, nil},
		{kservice.Starting, nil},
		{kservice.Stopped, Err1},
		{kservice.Starting, nil},
		{kservice.Stopped, Err1},
	}, <-events)

	require.Equal(t, 2, cb.x)
}

func TestServiceStopRunAndTogglingSeveralTimes(t *testing.T) {
	cb := &testCallbacks{runOption: option{
		errToReturn: Err1,
		waitTime:    1 * time.Second,
		readyCh:     make(chan struct{}),
		returnCh:    make(chan option),
	}}
	service := kservice.New(cb)
	events := collectEventsUntilStopped(service, 2)
	service.Start()
	<-cb.runOption.readyCh
	service.Stop()

	service.Toggle()
	service.Start()
	service.Stop()
	service.Toggle()
	service.Stop()
	close(cb.runOption.returnCh)

	require.Equal(t, []kservice.Event{
		{kservice.Stopped, nil},
		{kservice.Starting, nil},
		{kservice.Started, nil},
		{kservice.Stopping, nil},
		{kservice.Stopped, Err1},
	}, <-events)

	require.Equal(t, 3, cb.x)
}

func TestServiceStopRunAndTogglingSeveralTimesButEventuallyRestart(t *testing.T) {
	cb := &testCallbacks{runOption: option{
		errToReturn: Err1,
		waitTime:    1 * time.Second,
		readyCh:     make(chan struct{}),
		returnCh:    make(chan option),
	}}
	service := kservice.New(cb)
	events := collectEventsUntilStopped(service, 3)
	service.Start()
	<-cb.runOption.readyCh
	service.Stop()

	service.Toggle()
	service.Start()
	service.Stop()
	service.Toggle()
	service.Start()
	newOption := option{
		errToReturn: Err1,
		readyCh:     make(chan struct{}),
		returnCh:    make(chan option),
	}
	cb.runOption.returnCh <- newOption

	<-newOption.readyCh
	close(newOption.returnCh)

	require.Equal(t, []kservice.Event{
		{kservice.Stopped, nil},
		{kservice.Starting, nil},
		{kservice.Started, nil},
		{kservice.Stopping, nil},
		{kservice.Stopped, Err1},
		{kservice.Starting, nil},
		{kservice.Started, nil},
		{kservice.Stopped, Err1},
	}, <-events)

	require.Equal(t, 6, cb.x)
}

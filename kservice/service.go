package kservice

import (
	"context"
	"sync"

	"github.com/hsfzxjy/pipe"
)

type IService interface {
	Start()
	Stop()
	Toggle()
	Events() pipe.ListenableCM[Event]
}

type Service struct {
	callbacks _ICallbacks
	events    *pipe.ControllerCM[Event]

	mu     sync.Mutex
	cancel context.CancelFunc

	state       int16
	towardStart bool
}

func New(callbacks _ICallbacks) *Service {
	return &Service{
		callbacks: callbacks,
		state:     int16(Stopped),
		events:    pipe.NewControllerCM(Event{Stopped, nil}, true),
	}
}

func (s *Service) Events() pipe.ListenableCM[Event] {
	return s.events
}

func (s *Service) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startLocked()
}

func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

func (s *Service) Toggle() {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch s.currentState() {
	case Started:
		s.stopLocked()
	case Stopped:
		s.startLocked()
	case Stopping, Starting:
		s.towardStart = !s.towardStart
	}
}

func (s *Service) currentState() State {
	return State(s.state)
}

func (s *Service) sendEvent(ev Event) {
	s.state = int16(ev.State)
	s.events.Send(ev)
}

func (s *Service) startLocked() {
	switch s.currentState() {
	case Started:
	case Starting, Stopping:
		s.towardStart = true
	case Stopped:
		s.sendEvent(Event{Starting, nil})
		s.towardStart = true

		s.cancel = runWithCancelTimeout(
			s.callbacks.OnServiceStart,
			s.afterOnStartFinished,
		)
	}
}

func (s *Service) afterOnStartFinished(wasCanceled bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancel = nil

	if s.towardStart && err == nil {
		s.sendEvent(Event{Started, nil})
		s.cancel = runWithCancelTimeout(
			s.callbacks.OnServiceRun,
			s.afterOnRunFinished,
		)
		return
	}

	s.sendEvent(Event{Stopped, err})

	if s.towardStart && wasCanceled {
		s.startLocked()
	}
}

func (s *Service) afterOnRunFinished(wasCanceled bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancel = nil

	state := s.currentState()
	switch state {
	case Started:
		s.sendEvent(Event{Stopped, err})
	case Stopping:
		s.sendEvent(Event{Stopped, err})
		if s.towardStart {
			s.startLocked()
		}
	default:
		panic("bad state: " + state.String())
	}
}

func (s *Service) stopLocked() {
	s.towardStart = false
	switch s.currentState() {
	case Stopped, Stopping:
	case Started:
		s.sendEvent(Event{Stopping, nil})
		fallthrough
	case Starting:
		s.towardStart = false
		s.cancel()
		s.callbacks.OnServiceStop()
	}
}

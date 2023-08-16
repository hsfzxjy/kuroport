//go:build test

package ktest

import (
	"sync"
)

type _Scope struct {
	nRunning int
	mu       sync.Mutex
	cond     sync.Cond
}

func Scope() *_Scope {
	s := new(_Scope)
	s.cond.L = &s.mu
	return s
}

func (s *_Scope) Go(cb func()) {
	s.mu.Lock()
	s.nRunning++
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			s.nRunning--
			s.cond.Broadcast()
			s.mu.Unlock()
		}()
		cb()
	}()
}

func (s *_Scope) Wait() {
	s.mu.Lock()
	for s.nRunning != 0 {
		s.cond.Wait()
	}
	s.mu.Unlock()
}

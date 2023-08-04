package slot

import (
	"kstack/internal"
	"kstack/internal/conn/slot/muxed"
	"kstack/internal/conn/slot/not_muxed"
	ku "kutil"
	"sync"
)

type Slot struct {
	impl        internal.Impl
	disposeSelf ku.F

	mu   sync.RWMutex
	cond sync.Cond

	muxedConns ku.List[muxed.Tracked]

	nWaitingOrDialing uint32
	nNotMuxedConns    uint32
}

func New(impl internal.Impl, disposeSelf ku.F) *Slot {
	s := &Slot{
		impl:        impl,
		disposeSelf: disposeSelf,
	}
	s.cond.L = &s.mu
	return s
}

func (s *Slot) nAliveConnsRLocked() uint32 {
	return uint32(s.muxedConns.ActiveCount()) + s.nNotMuxedConns
}

func (s *Slot) DialAndTrack(dialF ku.Awaiter[internal.IConn], failFast bool) ku.Awaiter[internal.ITrackedConn] {
	opt := s.impl.ImplOption()

	if opt.Mux {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if c, ok := s.muxedConns.Get(); ok {
			return ku.Resolve[internal.ITrackedConn](c)
		}
	}

	return func() (internal.ITrackedConn, error) {
		s.mu.Lock()

		var maxDialing = opt.ConnPerAddrMaxDialing
		if opt.Mux {
			if c, ok := s.muxedConns.Get(); ok {
				s.mu.Unlock()
				return c, nil
			}
			maxDialing = 1
		}

		if s.nWaitingOrDialing == internal.MAX_SIZE ||
			failFast && maxDialing <= s.nWaitingOrDialing && !opt.Mux {
			s.mu.Unlock()
			return nil, internal.ErrTryAgain
		}

		for maxDialing <= s.nWaitingOrDialing {
			s.cond.Wait()
		}

		if opt.Mux {
			if c, ok := s.muxedConns.Get(); ok {
				s.mu.Unlock()
				s.cond.Signal()
				return c, nil
			}
		}

		s.nWaitingOrDialing++

		s.mu.Unlock()

		ic, err := dialF()

		s.mu.Lock()
		defer s.mu.Unlock()
		s.nWaitingOrDialing--
		s.cond.Signal()

		if err != nil {
			if s.isEmptyRLocked() {
				s.disposeSelf()
			}
			return nil, err
		}

		return s.trackLocked(ic, false)
	}
}

func (s *Slot) trackMuxedLocked(ic internal.IConn, isInbound bool) (internal.ITrackedConn, error) {
	var index int
	c, err := muxed.New(s.impl, ic, isInbound, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.muxedConns.Delete(index)
		if s.isEmptyRLocked() {
			s.disposeSelf()
		}
	})

	if err != nil {
		return nil, err
	}

	index = s.muxedConns.Add(c)

	return c, nil
}

func (s *Slot) trackNotMuxedLocked(ic internal.IConn, isInbound bool) (internal.ITrackedConn, error) {
	s.nNotMuxedConns++
	return not_muxed.New(s.impl, ic, isInbound, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.nNotMuxedConns--
		if s.isEmptyRLocked() {
			s.disposeSelf()
		}
	})
}

func (s *Slot) trackLocked(ic internal.IConn, isInbound bool) (internal.ITrackedConn, error) {
	opt := s.impl.ImplOption()
	if s.nAliveConnsRLocked() >= opt.ConnPerAddrMaxAlive {
		return nil, internal.ErrTryAgain
	}

	if opt.Mux {
		return s.trackMuxedLocked(ic, isInbound)
	} else {
		return s.trackNotMuxedLocked(ic, isInbound)
	}
}

func (s *Slot) isEmptyRLocked() bool {
	return s.nAliveConnsRLocked() == 0 && s.nWaitingOrDialing == 0
}

func (s *Slot) Track(ic internal.IConn, isInbound bool) (internal.ITrackedConn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.trackLocked(ic, isInbound)
}

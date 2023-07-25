package slot

import (
	"kstack/internal"
	"kstack/internal/transport/slot/muxed"
	"kstack/internal/transport/slot/not_muxed"
	ku "kutil"
	"sync"
)

type Slot struct {
	impl        internal.Impl
	disposeSelf ku.F

	mu   sync.RWMutex
	cond sync.Cond

	muxedTrs ku.List[muxed.Tracked]

	nWaitingOrDialing uint32
	nNotMuxedTrs      uint32
}

func New(impl internal.Impl, disposeSelf ku.F) *Slot {
	s := &Slot{
		impl:        impl,
		disposeSelf: disposeSelf,
	}
	s.cond.L = &s.mu
	return s
}

func (s *Slot) nAliveTrsRLocked() uint32 {
	return uint32(s.muxedTrs.ActiveCount()) + s.nNotMuxedTrs
}

func (s *Slot) DialAndTrack(dialF ku.Awaiter[internal.ITransport], failFast bool) ku.Awaiter[internal.TrackedTransport] {
	opt := s.impl.ImplOption()

	if opt.Mux {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if tr, ok := s.muxedTrs.Get(); ok {
			return ku.Resolve[internal.TrackedTransport](tr)
		}
	}

	return func() (internal.TrackedTransport, error) {
		s.mu.Lock()

		var maxDialing = opt.TransportPerAddrMaxDialing
		if opt.Mux {
			if tr, ok := s.muxedTrs.Get(); ok {
				s.mu.Unlock()
				return tr, nil
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
			if tr, ok := s.muxedTrs.Get(); ok {
				s.mu.Unlock()
				s.cond.Signal()
				return tr, nil
			}
		}

		s.nWaitingOrDialing++

		s.mu.Unlock()

		itr, err := dialF()

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

		return s.trackLocked(itr, false)
	}
}

func (s *Slot) trackMuxedLocked(itr internal.ITransport, isInbound bool) (internal.TrackedTransport, error) {
	var index int
	tr, err := muxed.New(s.impl, itr, isInbound, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.muxedTrs.Delete(index)
		if s.isEmptyRLocked() {
			s.disposeSelf()
		}
	})

	if err != nil {
		return nil, err
	}

	index = s.muxedTrs.Add(tr)

	return tr, nil
}

func (s *Slot) trackNotMuxedLocked(itr internal.ITransport, isInbound bool) (internal.TrackedTransport, error) {
	s.nNotMuxedTrs++
	return not_muxed.New(s.impl, itr, isInbound, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.nNotMuxedTrs--
		if s.isEmptyRLocked() {
			s.disposeSelf()
		}
	})
}

func (s *Slot) trackLocked(itr internal.ITransport, isInbound bool) (internal.TrackedTransport, error) {
	opt := s.impl.ImplOption()
	if s.nAliveTrsRLocked() >= opt.TransportPerAddrMaxAlive {
		return nil, internal.ErrTryAgain
	}

	if opt.Mux {
		return s.trackMuxedLocked(itr, isInbound)
	} else {
		return s.trackNotMuxedLocked(itr, isInbound)
	}
}

func (s *Slot) isEmptyRLocked() bool {
	return s.nAliveTrsRLocked() == 0 && s.nWaitingOrDialing == 0
}

func (s *Slot) Track(itr internal.ITransport, isInbound bool) (internal.TrackedTransport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.trackLocked(itr, isInbound)
}

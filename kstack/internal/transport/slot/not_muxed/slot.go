package not_muxed

import (
	"kstack/internal"
	ku "kutil"
	"sync"
)

type _Slot struct {
	impl        internal.Impl
	disposeSelf ku.F

	mu   sync.Mutex
	cond sync.Cond

	nMaxDialing uint16
	nWaiting    uint16
	nDialing    uint16
	nAliveTrs   uint16
}

func New(impl internal.Impl, disposeSelf ku.F) *_Slot {
	s := &_Slot{
		impl:        impl,
		disposeSelf: disposeSelf,
		nMaxDialing: ^uint16(0),
	}
	s.cond.L = &s.mu
	return s
}

func (s *_Slot) DialAndTrack(dialF ku.Awaiter[internal.ITransport], failFast bool) ku.Awaiter[internal.TrackedTransport] {
	return func() (internal.TrackedTransport, error) {
		s.mu.Lock()
		if failFast && s.nMaxDialing == s.nDialing {
			s.mu.Unlock()
			return nil, internal.ErrTryAgain
		}
		for s.nMaxDialing == s.nDialing {
			s.nWaiting++
			s.cond.Wait()
			s.nWaiting--
		}
		s.nDialing++
		s.mu.Unlock()

		itr, err := dialF()

		s.mu.Lock()
		defer s.mu.Unlock()
		s.nDialing--

		if err != nil {
			return nil, err
		}

		return s.trackLocked(itr, false)
	}
}

func (s *_Slot) trackLocked(itr internal.ITransport, isRemote bool) (*_TrackedTransport, error) {
	s.nAliveTrs++
	return newTrackedTrNotMuxed(s.impl, itr, isRemote, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.nAliveTrs--
		if s.isEmptyLocked() {
			s.disposeSelf()
		}
	})
}

func (s *_Slot) isEmptyLocked() bool {
	return s.nAliveTrs == 0 && s.nDialing == 0 && s.nWaiting == 0
}

func (s *_Slot) Track(itr internal.ITransport, isRemote bool) (internal.TrackedTransport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.trackLocked(itr, isRemote)
}

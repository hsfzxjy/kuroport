package muxed

import (
	"kstack/internal"
	ku "kutil"
	"sync"
)

type _Slot struct {
	impl        internal.Impl
	disposeSelf ku.F

	mu   sync.Mutex
	trs  ku.List[_Tracked]
	call *ku.Call[internal.TrackedTransport]
}

func New(impl internal.Impl, disposeSelf ku.F) *_Slot {
	return &_Slot{
		impl:        impl,
		disposeSelf: disposeSelf,
	}
}

func (s *_Slot) DialAndTrack(dialF ku.Awaiter[internal.ITransport], failFast bool) ku.Awaiter[internal.TrackedTransport] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.call != nil {
		return s.call.WaitAndGet
	}
	if tr, ok := s.trs.Get(); ok {
		return ku.Resolve[internal.TrackedTransport](tr)
	}

	s.call = ku.NewCall[internal.TrackedTransport]()
	return s.call.DoOrGetAsync(func() (internal.TrackedTransport, error) {
		itr, err := dialF()

		s.mu.Lock()
		defer s.mu.Unlock()

		var tr internal.TrackedTransport
		if err == nil {
			tr, err = s.trackLocked(itr, false)
		}

		if s.call == nil {
			panic("_Slot: call is nil in ResolveCall()")
		}
		s.call = nil

		return tr, err
	})
}

func (s *_Slot) trackLocked(itr internal.ITransport, isRemote bool) (internal.TrackedTransport, error) {
	var err error
	var index int

	tr, err := newTracked(s.impl, itr, isRemote, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.trs.Delete(index)
		if s.isEmptyLocked() {
			s.disposeSelf()
		}
	})

	if err != nil {
		return nil, err
	}

	index = s.trs.Add(tr)

	return tr, nil
}

func (s *_Slot) Track(itr internal.ITransport, isRemote bool) (internal.TrackedTransport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.trackLocked(itr, isRemote)
}

func (s *_Slot) isEmptyLocked() bool {
	return s.trs.IsEmpty() && s.call == nil
}

package kstack

import (
	ku "kutil"
	"sync"
)

// A _TrSlot manages multiple transport items that correspond to a same destination address.
// _TrSlot is used as the value type of _TrManager.
type _TrSlot struct {
	impl        *Impl
	disposeSelf ku.F

	mu   sync.RWMutex
	trs  ku.List[_Tr]
	call *ku.Call[_Tr]
}

func newTrSlot(impl *Impl, disposeSelf ku.F) *_TrSlot {
	return &_TrSlot{
		impl:        impl,
		disposeSelf: disposeSelf,
	}
}

func (s *_TrSlot) GetOpenableTr(dialF ku.Awaiter[ITransport], failFast bool) ku.Awaiter[_Tr] {
	if !s.impl.Option.Mux {
		return func() (_Tr, error) {
			itr, err := dialF()
			var tr _Tr
			if err == nil {
				tr, err = s.Track(itr, false)
			}
			return tr, err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.call != nil {
		return s.call.WaitAndGet
	}
	if tr, ok := s.trs.Get(); ok {
		return ku.Resolve(tr)
	}

	s.call = ku.NewCall[_Tr]()
	return s.call.DoOrGetAsync(func() (_Tr, error) {
		defer s.resolveCall()
		itr, err := dialF()
		var tr _Tr
		if err == nil {
			tr, err = s.Track(itr, false)
		}
		return tr, err
	})
}

func (s *_TrSlot) resolveCall() {
	s.mu.Lock()
	defer s.mu.Unlock()
	call := s.call
	if call == nil {
		panic("_TrSlot: call is nil in ResolveCall()")
	}
	s.call = nil
}

func (s *_TrSlot) Track(itr ITransport, isRemote bool) (_Tr, error) {
	// if !s.impl.Option.TransportMuxEnabled {
	// 	return newTr(s.impl, itr, isRemote, nil), nil
	// }

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.trs.AddFunc(func(index int) _Tr {
		return newTr(s.impl, itr, isRemote, func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.trs.Delete(index)
			if s.isEmptyLocked() {
				s.disposeSelf()
			}
		})
	}), nil
}

func (s *_TrSlot) isEmptyLocked() bool {
	return s.trs.IsEmpty() && s.call == nil
}

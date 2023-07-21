package kstack

import (
	"context"
	"errors"
	"hash/maphash"
	ku "kutil"

	"github.com/puzpuzpuz/xsync/v2"
)

type _TrManager struct {
	impl *Impl
	m    *xsync.MapOf[Hash, *_TrSlot]
}

func newTrManager(impl *Impl) *_TrManager {
	return &_TrManager{
		impl,
		xsync.NewTypedMapOf[Hash, *_TrSlot](func(s maphash.Seed, k Hash) uint64 {
			return k.Uint64()
		}),
	}
}

func (m *_TrManager) disposeSlot(addrHash Hash) ku.F {
	return func() {
		if _Tracking {
			tracker.TrDeleted.Add()
		}
		m.m.Delete(addrHash)
	}
}

func (m *_TrManager) Track(itr ITransport) (tr _Tr, err error) {
	hash := itr.Addr().Hash()
	m.m.Compute(hash, func(slot *_TrSlot, loaded bool) (*_TrSlot, bool) {
		if !loaded {
			slot = newTrSlot(m.impl, m.disposeSlot(hash))
		}
		tr, err = slot.Track(itr, true)
		return slot, false
	})
	return tr, err
}

var ErrTryAgain = errors.New("kstack: transports reach max capacity")

func (m *_TrManager) Dial(ctx context.Context, addr IAddr, failFast bool) (tr _Tr, err error) {
	dialer := m.impl.Dialer

	hash := addr.Hash()

COMPUTE:
	select {
	case <-ctx.Done():
		return emptyTr, nil
	default:
	}
	var awaiter ku.Awaiter[_Tr]
	m.m.Compute(hash, func(slot *_TrSlot, loaded bool) (*_TrSlot, bool) {
		if !loaded {
			slot = newTrSlot(m.impl, m.disposeSlot(hash))
		}
		awaiter = slot.GetOpenableTr(func() (ITransport, error) {
			return dialer.DialAddr(ctx, addr)
		}, failFast)
		return slot, false
	})
	tr, err = awaiter()
	if err == nil {
		return tr, nil
	}
	goto COMPUTE
}

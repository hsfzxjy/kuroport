package transport

import (
	"context"
	"errors"
	"hash/maphash"
	"kstack/internal"
	"kstack/internal/tracer"
	"kstack/internal/transport/slot"
	ku "kutil"

	"github.com/puzpuzpuz/xsync/v2"
)

type _Manager struct {
	impl internal.Impl
	m    *xsync.MapOf[ku.Hash, slot.ISlot]
}

func NewManager(impl internal.Impl) *_Manager {
	return &_Manager{
		impl,
		xsync.NewTypedMapOf[ku.Hash, slot.ISlot](func(s maphash.Seed, k ku.Hash) uint64 {
			return k.Uint64()
		}),
	}
}

func (m *_Manager) slotDisposer(addrHash ku.Hash) ku.F {
	return func() {
		if tracer.Enabled {
			tracer.T.TrSlotDeleted.Add()
		}
		m.m.Delete(addrHash)
	}
}

func (m *_Manager) TrackRemote(itr internal.ITransport) (tr internal.TrackedTransport, err error) {
	hash := itr.Addr().Hash()
	m.m.Compute(hash, func(s slot.ISlot, loaded bool) (slot.ISlot, bool) {
		if !loaded {
			s = slot.New(m.impl, m.slotDisposer(hash))
		}
		tr, err = s.Track(itr, true)
		return s, false
	})
	return tr, err
}

var ErrTryAgain = errors.New("kstack: transports reach max capacity")

func (m *_Manager) Dial(ctx context.Context, addr internal.IAddr, failFast bool) (tr internal.TrackedTransport, err error) {
	dialer := m.impl.Dialer()

	hash := addr.Hash()

	var loadedSlot slot.ISlot
COMPUTE:
	select {
	case <-ctx.Done():
		if loadedSlot != nil {
			loadedSlot.TryDispose()
		}
		return nil, ctx.Err()
	default:
	}
	var awaiter ku.Awaiter[internal.TrackedTransport]
	var dialedByThisCall = false
	m.m.Compute(hash, func(s slot.ISlot, loaded bool) (slot.ISlot, bool) {
		if !loaded {
			s = slot.New(m.impl, m.slotDisposer(hash))
		}
		loadedSlot = s
		awaiter = s.DialAndTrack(func() (internal.ITransport, error) {
			dialedByThisCall = true
			return dialer.DialAddr(ctx, addr)
		}, failFast)
		return s, false
	})
	tr, err = awaiter()
	if err == nil {
		return tr, nil
	}
	if dialedByThisCall {
		return nil, err
	}
	goto COMPUTE
}

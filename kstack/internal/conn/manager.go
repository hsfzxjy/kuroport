package conn

import (
	"context"
	"errors"
	"hash/maphash"
	"kstack/internal"
	"kstack/internal/conn/slot"
	"kstack/internal/tracer"
	ku "kutil"
	"sync"

	"github.com/puzpuzpuz/xsync/v2"
)

type _Manager struct {
	impl internal.Impl
	m    *xsync.MapOf[ku.Hash, *slot.Slot]
}

func NewManager(impl internal.Impl) *_Manager {
	return &_Manager{
		impl,
		xsync.NewTypedMapOf[ku.Hash, *slot.Slot](func(s maphash.Seed, k ku.Hash) uint64 {
			return k.Uint64()
		}),
	}
}

func (m *_Manager) slotDisposer(addrHash ku.Hash) ku.F {
	var once sync.Once
	return func() {
		once.Do(func() {
			if tracer.Enabled {
				tracer.T.TrSlotDeleted.Add()
			}
			m.m.Delete(addrHash)
		})
	}
}

func (m *_Manager) TrackRemote(ic internal.IConn) (c internal.ITrackedConn, err error) {
	hash := ic.Addr().Hash()
	m.m.Compute(hash, func(s *slot.Slot, loaded bool) (*slot.Slot, bool) {
		if !loaded {
			s = slot.New(m.impl, m.slotDisposer(hash))
		}
		c, err = s.Track(ic, true)
		return s, false
	})
	return c, err
}

var ErrTryAgain = errors.New("kstack: conns reach max capacity")

func (m *_Manager) Dial(ctx context.Context, addr internal.IAddr, failFast bool) (c internal.ITrackedConn, err error) {
	dialer := m.impl.Dialer()

	hash := addr.Hash()

	var awaiter ku.Awaiter[internal.ITrackedConn]
	m.m.Compute(hash, func(s *slot.Slot, loaded bool) (*slot.Slot, bool) {
		if !loaded {
			s = slot.New(m.impl, m.slotDisposer(hash))
		}
		awaiter = s.DialAndTrack(func() (internal.IConn, error) {
			return dialer.DialAddr(ctx, addr)
		}, failFast)
		return s, false
	})
	return awaiter()
}

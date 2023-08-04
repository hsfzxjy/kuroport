package not_muxed

import (
	"errors"
	"kstack/internal"
	ku "kutil"
	"sync/atomic"
)

type Tracked struct {
	iface    internal.IConn
	isOpened atomic.Bool
}

func New(impl internal.Impl, ic internal.IConn, isInbound bool, disposeSelf ku.F) (*Tracked, error) {
	t := &Tracked{iface: ic}
	go t.runLoop(impl, disposeSelf)

	if isInbound {
		t.isOpened.Store(true)
		if _, err := impl.StreamMgr().Track(ic, nil, isInbound); err != nil {
			ic.Close()
			return nil, err
		}
	}

	return t, nil
}

func (t *Tracked) Open(impl internal.Impl) (internal.IStream, error) {
	if ok := t.isOpened.CompareAndSwap(false, true); !ok {
		return nil, errors.New("already opened")
	}
	return impl.StreamMgr().Track(t.iface, nil, false)
}

func (t *Tracked) runLoop(impl internal.Impl, disposeSelf ku.F) {
	<-t.iface.DiedCh()
	disposeSelf.Do()
}

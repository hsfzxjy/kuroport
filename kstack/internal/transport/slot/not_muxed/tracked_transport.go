package not_muxed

import (
	"errors"
	"kstack/internal"
	ku "kutil"
	"sync/atomic"
)

type _TrackedTransport struct {
	iface    internal.ITransport
	isOpened atomic.Bool
}

func newTrackedTrNotMuxed(impl internal.Impl, itr internal.ITransport, isRemote bool, disposeSelf ku.F) (*_TrackedTransport, error) {
	tr := &_TrackedTransport{iface: itr}
	go tr.runLoop(impl, disposeSelf)

	if isRemote {
		tr.isOpened.Store(true)
		if _, err := impl.ConnManager().Track(itr, nil, isRemote); err != nil {
			itr.Close()
			return nil, err
		}
	}

	return tr, nil
}

func (t *_TrackedTransport) Open(impl internal.Impl) (internal.IConn, error) {
	if ok := t.isOpened.CompareAndSwap(false, true); !ok {
		return nil, errors.New("already opened")
	}
	return impl.ConnManager().Track(t.iface, nil, false)
}

func (tr *_TrackedTransport) runLoop(impl internal.Impl, disposeSelf ku.F) {
	<-tr.iface.DiedCh()
	disposeSelf.Do()
}

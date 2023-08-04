package muxed

import (
	"kstack/internal"
	ku "kutil"

	"github.com/hsfzxjy/smux"
)

type Tracked struct {
	iface       internal.IConn
	smuxSession *smux.Session
}

func New(impl internal.Impl, ic internal.IConn, isInbound bool, disposeSelf ku.F) (t Tracked, err error) {
	var session *smux.Session
	if isInbound {
		session, err = smux.Server(ic, nil)
	} else {
		session, err = smux.Client(ic, nil)
	}

	if err != nil {
		return Tracked{}, err
	}

	t = Tracked{ic, session}
	go t.runLoop(impl, disposeSelf)

	return t, nil
}

func (t Tracked) IsZero() bool {
	return t.iface == nil
}

func (t Tracked) Open(impl internal.Impl) (internal.IStream, error) {
	stream, err := t.smuxSession.OpenStream()
	if err != nil {
		return nil, err
	}
	return impl.StreamMgr().Track(t.iface, stream, false)
}

func (t Tracked) runLoop(impl internal.Impl, disposeSelf ku.F) {
	defer t.smuxSession.Close()
	for {
		stream, err := t.smuxSession.AcceptStream()
		if err != nil {
			break
		}
		impl.StreamMgr().Track(t.iface, stream, true)
	}
	<-t.iface.DiedCh()
	disposeSelf.Do()
}

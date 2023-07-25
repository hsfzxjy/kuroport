package muxed

import (
	"kstack/internal"
	ku "kutil"

	"github.com/hsfzxjy/smux"
)

type Tracked struct {
	iface       internal.ITransport
	smuxSession *smux.Session
}

func New(impl internal.Impl, itr internal.ITransport, isInbound bool, disposeSelf ku.F) (tr Tracked, err error) {
	var session *smux.Session
	if isInbound {
		session, err = smux.Server(itr, nil)
	} else {
		session, err = smux.Client(itr, nil)
	}

	if err != nil {
		return Tracked{}, err
	}

	tr = Tracked{itr, session}
	go tr.runLoop(impl, disposeSelf)

	return tr, nil
}

func (tr Tracked) IsZero() bool {
	return tr.iface == nil
}

func (tr Tracked) Open(impl internal.Impl) (internal.IConn, error) {
	stream, err := tr.smuxSession.OpenStream()
	if err != nil {
		return nil, err
	}
	return impl.ConnManager().Track(tr.iface, stream, false)
}

func (tr Tracked) runLoop(impl internal.Impl, disposeSelf ku.F) {
	defer tr.smuxSession.Close()
	for {
		stream, err := tr.smuxSession.AcceptStream()
		if err != nil {
			break
		}
		impl.ConnManager().Track(tr.iface, stream, true)
	}
	<-tr.iface.DiedCh()
	disposeSelf.Do()
}

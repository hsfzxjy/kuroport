package muxed

import (
	"kstack/internal"
	ku "kutil"

	"github.com/hsfzxjy/smux"
)

type _Tracked struct {
	iface       internal.ITransport
	smuxSession *smux.Session
}

func newTracked(impl internal.Impl, itr internal.ITransport, isRemote bool, disposeSelf ku.F) (_Tracked, error) {
	var session *smux.Session
	var tr _Tracked
	var err error
	if isRemote {
		session, err = smux.Server(itr, nil)
	} else {
		session, err = smux.Client(itr, nil)
	}

	if err != nil {
		return _Tracked{}, err
	}

	tr = _Tracked{itr, session}
	go tr.runLoop(impl, disposeSelf)

	return tr, nil
}

func (tr _Tracked) IsZero() bool {
	return tr.iface == nil
}

func (tr _Tracked) Open(impl internal.Impl) (internal.IConn, error) {
	stream, err := tr.smuxSession.OpenStream()
	if err != nil {
		return nil, err
	}
	return impl.ConnManager().Track(tr.iface, stream, false)
}

func (tr _Tracked) runLoop(impl internal.Impl, disposeSelf ku.F) {
	if tr.smuxSession != nil {
		defer tr.smuxSession.Close()
		for {
			stream, err := tr.smuxSession.AcceptStream()
			if err != nil {
				break
			}
			impl.ConnManager().Track(tr.iface, stream, true)
		}
	}
	<-tr.iface.DiedCh()
	disposeSelf.Do()
}

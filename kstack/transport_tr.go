package kstack

import (
	ku "kutil"

	"github.com/hsfzxjy/smux"
)

// A _Tr bundles the underlying transport and its corresponding smux.Session instance.
// It is used as the element type _TrList
type _Tr struct {
	iface       ITransport
	smuxSession *smux.Session
}

var emptyTr _Tr

func newTr(impl *Impl, itr ITransport, isRemote bool, disposeSelf ku.F) _Tr {
	if !impl.Option.Mux {
		tr := _Tr{itr, nil}
		go tr.runLoop(impl, disposeSelf)
		return tr
	}

	var session *smux.Session
	var tr _Tr
	if isRemote {
		session, _ = smux.Server(itr, nil)
	} else {
		session, _ = smux.Client(itr, nil)
	}

	tr = _Tr{itr, session}
	go tr.runLoop(impl, disposeSelf)

	return tr
}

func (tr _Tr) IsZero() bool {
	return tr.iface == nil
}

func (tr *_Tr) Open(impl *Impl) (IConn, error) {
	if impl.Option.Mux {
		stream, err := tr.smuxSession.OpenStream()
		if err != nil {
			return nil, err
		}
		return impl.connManager.Track(tr.iface, stream, false)
	}
	return impl.connManager.Track(tr.iface, nil, false)
}

func (tr _Tr) runLoop(impl *Impl, disposeSelf ku.F) {
	if tr.smuxSession != nil {
		defer tr.smuxSession.Close()
		for {
			stream, err := tr.smuxSession.AcceptStream()
			if err != nil {
				break
			}
			impl.connManager.Track(tr.iface, stream, true)
		}
	}
	<-tr.iface.DiedCh()
	disposeSelf.Do()
}

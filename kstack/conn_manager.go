package kstack

import (
	"errors"
	"sync/atomic"

	"github.com/hsfzxjy/smux"
	"github.com/puzpuzpuz/xsync/v2"
)

type _ConnManager struct {
	impl *Impl
	m    *xsync.MapOf[ConnID, IConn]

	nextId atomic.Uint64
}

func newConnManager(impl *Impl) *_ConnManager {
	m := new(_ConnManager)
	m.impl = impl
	m.m = xsync.NewIntegerMapOf[ConnID, IConn]()
	return m
}

var ErrStackWontAcceptConn = errors.New("kstack: stack won't accept conn, specify non-nil RemoteConns to fix")

func (m *_ConnManager) Track(itr ITransport, stream *smux.Stream, isRemote bool) (IConn, error) {
	var c IConn

	if !m.impl.Option.Mux {
		if stream != nil {
			panic("kstack: stream must be nil")
		}
	}

	if isRemote && m.impl.stack.option.RemoteConns == nil {
		if stream != nil {
			stream.Close()
		}
		return nil, ErrStackWontAcceptConn
	}

	var idOk bool
	var id ConnID
	for !idOk {
		nextId := m.nextId.Add(1)
		id = ConnID(nextId)
		m.m.Compute(id, func(conn IConn, loaded bool) (_ IConn, delete bool) {
			if loaded {
				idOk = false
				return conn, false
			}
			idOk = true

			disposeConn := func() {
				if _Tracking {
					tracker.ConnDeleted.Add()
				}
				m.m.Delete(id)
			}

			if stream == nil {
				c = newTrConn(id, itr, disposeConn)
			} else {
				c = newSmuxStreamConn(id, itr, stream, disposeConn)
			}
			return c, false
		})
	}

	if isRemote {
		m.impl.stack.option.RemoteConns <- c
	}

	return c, nil
}

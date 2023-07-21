package conn

import (
	"errors"
	"kstack/internal"
	"kstack/internal/tracer"
	"sync/atomic"

	"github.com/hsfzxjy/smux"
	"github.com/puzpuzpuz/xsync/v2"
)

type _Manager struct {
	impl internal.Impl
	m    *xsync.MapOf[internal.ConnID, internal.IConn]

	nextId atomic.Uint64
}

func NewManager(impl internal.Impl) *_Manager {
	m := new(_Manager)
	m.impl = impl
	m.m = xsync.NewIntegerMapOf[internal.ConnID, internal.IConn]()
	return m
}

var ErrStackWontAcceptConn = errors.New("kstack: stack won't accept conn, specify non-nil RemoteConns to fix")

func (m *_Manager) Track(itr internal.ITransport, stream *smux.Stream, isRemote bool) (internal.IConn, error) {
	var c internal.IConn

	if !m.impl.ImplOption().Mux {
		if stream != nil {
			panic("kstack: stream must be nil")
		}
	}

	if isRemote && m.impl.StackOption().RemoteConns == nil {
		if stream != nil {
			stream.Close()
		}
		return nil, ErrStackWontAcceptConn
	}

	var idOk bool
	var id internal.ConnID
	for !idOk {
		nextId := m.nextId.Add(1)
		id = internal.ConnID(nextId)
		m.m.Compute(id, func(conn internal.IConn, loaded bool) (_ internal.IConn, delete bool) {
			if loaded {
				idOk = false
				return conn, false
			}
			idOk = true

			disposeConn := func() {
				if tracer.Enabled {
					tracer.T.ConnDeleted.Add()
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
		m.impl.StackOption().RemoteConns <- c
	}

	return c, nil
}

package stream

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
	m    *xsync.MapOf[internal.StreamID, internal.IStream]

	nextId atomic.Uint64
}

func NewManager(impl internal.Impl) *_Manager {
	m := new(_Manager)
	m.impl = impl
	m.m = xsync.NewIntegerMapOf[internal.StreamID, internal.IStream]()
	return m
}

var ErrStackWontAcceptConn = errors.New("kstack: stack won't accept conn, specify non-nil RemoteConns to fix")

func (m *_Manager) Track(ic internal.IConn, stream *smux.Stream, isInbound bool) (internal.IStream, error) {
	var c internal.IStream

	if !m.impl.ImplOption().Mux {
		if stream != nil {
			panic("kstack: stream must be nil")
		}
	}

	if isInbound && m.impl.StackOption().RemoteConns == nil {
		if stream != nil {
			stream.Close()
		}
		return nil, ErrStackWontAcceptConn
	}

	var idOk bool
	var id internal.StreamID
	for !idOk {
		nextId := m.nextId.Add(1)
		id = internal.StreamID(nextId)
		m.m.Compute(id, func(conn internal.IStream, loaded bool) (_ internal.IStream, delete bool) {
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
				c = newStream(id, ic, disposeConn)
			} else {
				c = newMuxedStream(id, ic, stream, disposeConn)
			}
			return c, false
		})
	}

	if isInbound {
		m.impl.StackOption().RemoteConns <- c
	}

	return c, nil
}

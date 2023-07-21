package conn

import (
	"kstack/internal"
	ku "kutil"

	"github.com/hsfzxjy/smux"
)

type _MuxedConn struct {
	*smux.Stream
	id  internal.ConnID
	itr internal.ITransport
}

func newMuxedConn(
	id internal.ConnID,
	itr internal.ITransport,
	stream *smux.Stream,
	disposeSelf ku.F) *_MuxedConn {
	c := new(_MuxedConn)
	c.Stream = stream
	c.id = id
	c.itr = itr
	go c.runLoop(disposeSelf)
	return c
}

func (c *_MuxedConn) Transport() internal.ITransport {
	return c.itr
}

func (c *_MuxedConn) ID() internal.ConnID {
	return c.id
}

func (c *_MuxedConn) runLoop(disposeSelf ku.F) {
	<-c.Stream.GetDieCh()
	disposeSelf.Do()
}

type _Conn struct {
	id internal.ConnID
	internal.ITransport
}

func newConn(id internal.ConnID, itr internal.ITransport, disposeSelf ku.F) *_Conn {
	c := &_Conn{id, itr}
	go c.runLoop(disposeSelf)
	return c
}

func (c *_Conn) runLoop(disposeSelf ku.F) {
	<-c.DiedCh()
	disposeSelf.Do()
}

func (c *_Conn) ID() internal.ConnID {
	return c.id
}

func (c *_Conn) Transport() internal.ITransport {
	return c.ITransport
}

package conn

import (
	"kstack/internal"
	ku "kutil"

	"github.com/hsfzxjy/smux"
)

type _SmuxStreamConn struct {
	*smux.Stream
	id  internal.ConnID
	itr internal.ITransport
}

func newSmuxStreamConn(
	id internal.ConnID,
	itr internal.ITransport,
	stream *smux.Stream,
	disposeSelf ku.F) *_SmuxStreamConn {
	c := new(_SmuxStreamConn)
	c.Stream = stream
	c.id = id
	c.itr = itr
	go c.runLoop(disposeSelf)
	return c
}

func (c *_SmuxStreamConn) Transport() internal.ITransport {
	return c.itr
}

func (c *_SmuxStreamConn) ID() internal.ConnID {
	return c.id
}

func (c *_SmuxStreamConn) runLoop(disposeSelf ku.F) {
	<-c.Stream.GetDieCh()
	disposeSelf.Do()
}

type _TrConn struct {
	id internal.ConnID
	internal.ITransport
}

func newTrConn(id internal.ConnID, itr internal.ITransport, disposeSelf ku.F) *_TrConn {
	c := &_TrConn{id, itr}
	go c.runLoop(disposeSelf)
	return c
}

func (c *_TrConn) runLoop(disposeSelf ku.F) {
	<-c.DiedCh()
	disposeSelf.Do()
}

func (c *_TrConn) ID() internal.ConnID {
	return c.id
}

func (c *_TrConn) Transport() internal.ITransport {
	return c.ITransport
}

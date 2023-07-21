package kstack

import (
	"io"
	ku "kutil"
	"time"

	"github.com/hsfzxjy/smux"
)

type IConn interface {
	io.ReadWriteCloser
	ID() ConnID
	Transport() ITransport
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

type ConnID uint64

type _SmuxStreamConn struct {
	*smux.Stream
	id  ConnID
	itr ITransport
}

func newSmuxStreamConn(
	id ConnID,
	itr ITransport,
	stream *smux.Stream,
	disposeSelf ku.F) *_SmuxStreamConn {
	c := new(_SmuxStreamConn)
	c.Stream = stream
	c.id = id
	c.itr = itr
	go c.runLoop(disposeSelf)
	return c
}

func (c *_SmuxStreamConn) Transport() ITransport {
	return c.itr
}

func (c *_SmuxStreamConn) ID() ConnID {
	return c.id
}

func (c *_SmuxStreamConn) runLoop(disposeSelf ku.F) {
	<-c.Stream.GetDieCh()
	disposeSelf.Do()
}

type _TrConn struct {
	id ConnID
	ITransport
}

func newTrConn(id ConnID, itr ITransport, disposeSelf ku.F) *_TrConn {
	c := &_TrConn{id, itr}
	go c.runLoop(disposeSelf)
	return c
}

func (c *_TrConn) runLoop(disposeSelf ku.F) {
	<-c.DiedCh()
	disposeSelf.Do()
}

func (c *_TrConn) ID() ConnID {
	return c.id
}

func (c *_TrConn) Transport() ITransport {
	return c.ITransport
}

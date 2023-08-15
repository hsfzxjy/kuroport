package kstack

import (
	"errors"
	"io"
	"kstack/internal"
	"kstack/peer"
	"sync"
	"time"
)

type IRawConn interface {
	io.ReadWriteCloser
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

var ErrBadAddress = errors.New("bad kstack address")

type AddrProvider interface {
	Addr() IAddr
	Family() Family
}

type WrapOption struct {
	CloseOnError bool
	AddrProvider
}

type _ConnWrapper[C IRawConn] struct {
	rawConn  C
	diedCh   chan struct{}
	diedOnce sync.Once
	WrapOption
}

func WrapTransport[C IRawConn](rawConn C, config WrapOption) internal.IConn {
	return &_ConnWrapper[C]{
		rawConn:    rawConn,
		diedCh:     make(chan struct{}),
		WrapOption: config,
	}
}

func (w *_ConnWrapper[C]) doDie() {
	w.diedOnce.Do(func() {
		close(w.diedCh)
	})
}

func (w *_ConnWrapper[C]) Close() error {
	w.doDie()
	return w.rawConn.Close()
}

func (w *_ConnWrapper[C]) DiedCh() <-chan struct{} {
	return w.diedCh
}

func (*_ConnWrapper[C]) RemoteID() peer.ID {
	return ""
}

func (w *_ConnWrapper[C]) Read(p []byte) (n int, err error) {
	n, err = w.rawConn.Read(p)
	if err != nil && w.CloseOnError {
		w.Close()
	}
	return n, err
}

func (w *_ConnWrapper[C]) SetDeadline(t time.Time) error {
	err := w.rawConn.SetDeadline(t)
	if err != nil && w.CloseOnError {
		w.Close()
	}
	return err
}

func (w *_ConnWrapper[C]) SetReadDeadline(t time.Time) error {
	err := w.rawConn.SetReadDeadline(t)
	if err != nil && w.CloseOnError {
		w.Close()
	}
	return err
}

func (w *_ConnWrapper[C]) SetWriteDeadline(t time.Time) error {
	err := w.rawConn.SetWriteDeadline(t)
	if err != nil && w.CloseOnError {
		w.Close()
	}
	return err
}

func (w *_ConnWrapper[C]) Write(p []byte) (n int, err error) {
	n, err = w.rawConn.Write(p)
	if err != nil && w.CloseOnError {
		w.Close()
	}
	return n, err
}

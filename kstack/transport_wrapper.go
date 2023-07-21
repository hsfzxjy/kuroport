package kstack

import (
	"io"
	"sync"
	"time"
)

type IRawTransport interface {
	io.ReadWriteCloser
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

type WrapOption struct {
	CloseOnError bool
	AddrProvider
}

type _TransportWrapper[RTR IRawTransport] struct {
	rtr      RTR
	diedCh   chan struct{}
	diedOnce sync.Once
	WrapOption
}

func WrapTransport[RTR IRawTransport](rtr RTR, config WrapOption) ITransport {
	return &_TransportWrapper[RTR]{
		rtr:        rtr,
		diedCh:     make(chan struct{}),
		WrapOption: config,
	}
}

func (w *_TransportWrapper[RTR]) doDie() {
	w.diedOnce.Do(func() {
		close(w.diedCh)
	})
}

// Close implements ITransport.
func (w *_TransportWrapper[RTR]) Close() error {
	w.doDie()
	return w.rtr.Close()
}

// DiedCh implements ITransport.
func (w *_TransportWrapper[RTR]) DiedCh() <-chan struct{} {
	return w.diedCh
}

// Read implements ITransport.
func (w *_TransportWrapper[RTR]) Read(p []byte) (n int, err error) {
	n, err = w.rtr.Read(p)
	if err != nil && w.CloseOnError {
		w.Close()
	}
	return n, err
}

// SetDeadline implements ITransport.
func (w *_TransportWrapper[RTR]) SetDeadline(t time.Time) error {
	err := w.rtr.SetDeadline(t)
	if err != nil && w.CloseOnError {
		w.Close()
	}
	return err
}

// SetReadDeadline implements ITransport.
func (w *_TransportWrapper[RTR]) SetReadDeadline(t time.Time) error {
	err := w.rtr.SetReadDeadline(t)
	if err != nil && w.CloseOnError {
		w.Close()
	}
	return err
}

// SetWriteDeadline implements ITransport.
func (w *_TransportWrapper[RTR]) SetWriteDeadline(t time.Time) error {
	err := w.rtr.SetWriteDeadline(t)
	if err != nil && w.CloseOnError {
		w.Close()
	}
	return err
}

// Write implements ITransport.
func (w *_TransportWrapper[RTR]) Write(p []byte) (n int, err error) {
	n, err = w.rtr.Write(p)
	if err != nil && w.CloseOnError {
		w.Close()
	}
	return n, err
}

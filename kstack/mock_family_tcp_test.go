package kstack_test

import (
	"context"
	"kservice"
	"kstack"
	"net"
	"sync/atomic"
)

type _TestTcpAddr struct {
	*net.TCPAddr
}

// Family implements kstack.IAddr.
func (a _TestTcpAddr) Family() kstack.Family {
	return kstack.TestFamily
}

// Hash implements kstack.IAddr.
func (a _TestTcpAddr) Hash() kstack.Hash {
	return kstack.GetHash(a.TCPAddr)
}

// ResolveDevice implements kstack.IAddr.
func (_TestTcpAddr) ResolveDevice() (kstack.IDevice, error) {
	panic("unimplemented")
}

// Addr implements kstack.IAddrProvider.
func (a _TestTcpAddr) Addr() kstack.IAddr {
	return a
}

var testTcpAddr _TestTcpAddr

func init() {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:18888")
	testTcpAddr = _TestTcpAddr{tcpAddr}
}

func newTestTcpTransport(c *net.TCPConn) kstack.ITransport {
	return kstack.WrapTransport(c, kstack.WrapOption{
		CloseOnError: true,
		AddrProvider: _TestTcpAddr{c.RemoteAddr().(*net.TCPAddr)},
	})
}

type _TestTcpListener struct {
	*kservice.Service
	ln atomic.Pointer[net.TCPListener]
	ch chan<- kstack.ITransport
}

// AcceptTransport implements kstack.IListener.
func (l *_TestTcpListener) AcceptTransport(ch chan<- kstack.ITransport) {
	l.ch = ch
}

// OnServiceRun implements kservice._ICallbacks.
func (l *_TestTcpListener) OnServiceRun(ctx context.Context) (err error) {
	ln, err := net.ListenTCP("tcp", testTcpAddr.TCPAddr)
	if err != nil {
		return err
	}
	l.ln.Store(ln)
	select {
	case <-ctx.Done():
		ln.Close()
		return nil
	default:
	}
	for {
		c, err := ln.AcceptTCP()
		if err != nil {
			return err
		}
		l.ch <- newTestTcpTransport(c)
	}
}

// OnServiceStart implements kservice._ICallbacks.
func (*_TestTcpListener) OnServiceStart(ctx context.Context) (err error) {
	return nil
}

// OnServiceStop implements kservice._ICallbacks.
func (l *_TestTcpListener) OnServiceStop() {
	ln := l.ln.Load()
	if ln != nil {
		ln.Close()
	}
}

func newTestTcpListener() kstack.IListener {
	l := &_TestTcpListener{}
	l.Service = kservice.New(l)
	return l
}

type _TestTcpDialer struct{}

func (_TestTcpDialer) DialAddr(ctx context.Context, addr kstack.IAddr) (kstack.ITransport, error) {
	c, err := net.DialTCP("tcp", nil, addr.(_TestTcpAddr).TCPAddr)
	if err != nil {
		return nil, err
	}
	return newTestTcpTransport(c), nil
}

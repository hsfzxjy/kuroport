package mock_tcp

import (
	"context"
	"kservice"
	"kstack"
	"kstack/internal"
	ku "kutil"
	"net"
	"sync/atomic"
)

type _Addr struct {
	*net.TCPAddr
}

func (a _Addr) Family() kstack.Family {
	return internal.TestFamily
}

func (a _Addr) Hash() ku.Hash {
	return ku.GetHash(a.TCPAddr)
}

func (_Addr) ResolveDevice() (kstack.IDevice, error) {
	panic("unimplemented")
}

func (a _Addr) Addr() kstack.IAddr {
	return a
}

var Addr _Addr

func init() {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:18888")
	Addr = _Addr{tcpAddr}
}

func newTransport(c *net.TCPConn) kstack.ITransport {
	return kstack.WrapTransport(c, kstack.WrapOption{
		CloseOnError: true,
		AddrProvider: _Addr{c.RemoteAddr().(*net.TCPAddr)},
	})
}

type _Listener struct {
	*kservice.Service
	ln atomic.Pointer[net.TCPListener]
	ch chan<- kstack.ITransport
}

func (l *_Listener) AcceptTransport(ch chan<- kstack.ITransport) {
	l.ch = ch
}

func (l *_Listener) OnServiceRun(ctx context.Context) (err error) {
	ln, err := net.ListenTCP("tcp", Addr.TCPAddr)
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
		l.ch <- newTransport(c)
	}
}

func (*_Listener) OnServiceStart(ctx context.Context) (err error) {
	return nil
}

func (l *_Listener) OnServiceStop() {
	ln := l.ln.Load()
	if ln != nil {
		ln.Close()
	}
}

func Listener() kstack.IListener {
	l := &_Listener{}
	l.Service = kservice.New(l)
	return l
}

type _Dialer struct{}

func Dialer() _Dialer {
	return _Dialer{}
}

func (_Dialer) DialAddr(ctx context.Context, addr kstack.IAddr) (kstack.ITransport, error) {
	c, err := net.DialTCP("tcp", nil, addr.(_Addr).TCPAddr)
	if err != nil {
		return nil, err
	}
	return newTransport(c), nil
}

package mocktcp

import (
	"context"
	"kservice"
	"kstack"
	"kstack/internal"
	ku "kutil"
	"net"
	"sync/atomic"
	"time"
)

type _Addr struct {
	*net.TCPAddr
	ch <-chan struct{}
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

func (a _Addr) WithCh(ch <-chan struct{}) _Addr {
	return _Addr{a.TCPAddr, ch}
}

var Addr _Addr
var AddrBad _Addr

func init() {
	tcpAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:18888")
	Addr = _Addr{TCPAddr: tcpAddr}
}

func newTransport(c *net.TCPConn) kstack.ITransport {
	return kstack.WrapTransport(c, kstack.WrapOption{
		CloseOnError: true,
		AddrProvider: _Addr{TCPAddr: c.RemoteAddr().(*net.TCPAddr)},
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
	ln := l.ln.Load()
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

func (l *_Listener) OnServiceStart(ctx context.Context) (err error) {
	ln, err := net.ListenTCP("tcp", Addr.TCPAddr)
	if err != nil {
		return err
	}
	l.ln.Store(ln)
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

type _DelayedDialer struct {
	duration time.Duration
}

func DelayedDialer(delay time.Duration) _DelayedDialer {
	return _DelayedDialer{delay}
}

func (d _DelayedDialer) DialAddr(ctx context.Context, addr kstack.IAddr) (kstack.ITransport, error) {
	timer := time.NewTimer(d.duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
	}
	c, err := net.DialTCP("tcp", nil, addr.(_Addr).TCPAddr)
	if err != nil {
		return nil, err
	}
	return newTransport(c), nil
}

type _NotifiableDialer struct {
	ch chan<- struct{}
}

func NotifiableDialer(ch chan<- struct{}) *_NotifiableDialer {
	return &_NotifiableDialer{ch: ch}
}

func (d *_NotifiableDialer) DialAddr(ctx context.Context, addr kstack.IAddr) (kstack.ITransport, error) {
	if d.ch != nil {
		close(d.ch)
	}

	if a, ok := addr.(_Addr); ok && a.ch != nil {
		<-a.ch
	}

	c, err := net.DialTCP("tcp", nil, addr.(_Addr).TCPAddr)
	if err != nil {
		return nil, err
	}
	return newTransport(c), nil
}

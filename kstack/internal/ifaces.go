package internal

import (
	"context"
	"io"
	"kservice"
	"kstack/peer"
	ku "kutil"
	"time"

	"github.com/hsfzxjy/smux"
)

type StreamID uint64

type IStream interface {
	io.ReadWriteCloser
	ID() StreamID
	Conn() IConn
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

type IAddr interface {
	Family() Family
	String() string
	Hash() ku.Hash
	ResolveDevice() (IUnidentifiedDevice, error)
}

type IConn interface {
	io.ReadWriteCloser
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	Family() Family
	Addr() IAddr
	DiedCh() <-chan struct{}
	RemoteID() peer.ID
}

type IUnidentifiedDevice interface {
	Family() Family
	Addrs() []IAddr
	Name() string
	DialFunc() func(context.Context) (IConn, error)
}

type IListener interface {
	kservice.IService
	AcceptTransport(chan<- IConn)
}

type IScanner interface {
	kservice.IService
	AcceptDevice(chan<- IUnidentifiedDevice)
}

type IDialer interface {
	DialAddr(ctx context.Context, addr IAddr) (IConn, error)
}

type IAdvertiser interface {
	kservice.IService
}

type IStreamMgr interface {
	Track(ic IConn, stream *smux.Stream, isInbound bool) (IStream, error)
}

type ITrackedConn interface {
	Open(impl Impl) (IStream, error)
}

type IConnMgr interface {
	TrackRemote(ic IConn) (c ITrackedConn, err error)
	Dial(ctx context.Context, addr IAddr, failFast bool) (c ITrackedConn, err error)
}

type Option struct {
	RemoteConns chan<- IStream
}

type Impl interface {
	Dialer() IDialer
	StackOption() Option
	ImplOption() ImplOption
	StreamMgr() IStreamMgr
	ConnMgr() IConnMgr
}

type ImplOption struct {
	Mux                   bool
	ConnMaxDialing        uint32
	ConnMaxAlive          uint32
	ConnPerAddrMaxDialing uint32
	ConnPerAddrMaxAlive   uint32
}

const MAX_SIZE = uint32(1 << 30)

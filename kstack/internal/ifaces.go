package internal

import (
	"context"
	"io"
	"kservice"
	ku "kutil"
	"time"

	"github.com/hsfzxjy/smux"
)

type ConnID uint64

type IConn interface {
	io.ReadWriteCloser
	ID() ConnID
	Transport() ITransport
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

type IAddr interface {
	Family() Family
	String() string
	Hash() ku.Hash
	ResolveDevice() (IDevice, error)
}

type ITransport interface {
	io.ReadWriteCloser
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	Family() Family
	Addr() IAddr
	DiedCh() <-chan struct{}
}

type IIdentity interface {
	Hash() ku.Hash
	Devices() []IDevice
}

type IDeviceID interface {
	Hash() ku.Hash
	String() string
	Family() Family
}

type IDevice interface {
	DeviceID() IDeviceID
	Family() Family
	Identity() IIdentity
	Addrs() []IAddr
	DialFunc() func(context.Context) (ITransport, error)
}

type IListener interface {
	kservice.IService
	AcceptTransport(chan<- ITransport)
}

type IScanner interface {
	kservice.IService
	AcceptDevice(chan<- IDevice)
}

type IDialer interface {
	DialAddr(ctx context.Context, addr IAddr) (ITransport, error)
}

type IAdvertiser interface {
	kservice.IService
}

type ConnManager interface {
	Track(itr ITransport, stream *smux.Stream, isRemote bool) (IConn, error)
}

type TrackedTransport interface {
	Open(impl Impl) (IConn, error)
}

type TrManager interface {
	TrackRemote(itr ITransport) (tr TrackedTransport, err error)
	Dial(ctx context.Context, addr IAddr, failFast bool) (tr TrackedTransport, err error)
}

type Option struct {
	RemoteConns chan<- IConn
}

type Impl interface {
	Dialer() IDialer
	StackOption() Option
	ImplOption() ImplOption
	ConnManager() ConnManager
	TrManager() TrManager
}

type ImplOption struct {
	Mux                        bool
	TransportMaxDialing        uint32
	TransportMaxAlive          uint32
	TransportPerAddrMaxDialing uint32
	TransportPerAddrMaxAlive   uint32
}

const MAX_SIZE = uint32(1 << 30)

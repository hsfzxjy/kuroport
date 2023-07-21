package kstack

import (
	"context"
	"kservice"
	"sync"
)

type Family int

const (
	BREDR Family = iota
	BLE
	INet
	testFamily
	maxFamily
)

type IIdentity interface {
	Hash() Hash
	Devices() []IDevice
}

type IDeviceID interface {
	Hash() Hash
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

type Stack struct {
	option  Option
	impls   [maxFamily]Impl
	runOnce sync.Once
}

func New(option Option) *Stack {
	stack := new(Stack)
	stack.option = option
	return stack
}

func (s *Stack) getImpl(family Family) *Impl {
	if family >= maxFamily {
		return nil
	}
	return &s.impls[family]
}

func (s *Stack) RegisterImpl(impl Impl) {
	family := impl.Family
	oldImpl := s.getImpl(family)
	if oldImpl.stack != nil {
		panic("kstack: family has been registered")
	}
	*oldImpl = impl
	oldImpl.stack = s
	oldImpl.trManager = newTrManager(oldImpl)
	oldImpl.connManager = newConnManager(oldImpl)
}

func (s *Stack) Run() {
	s.runOnce.Do(func() {
		var wg sync.WaitGroup
		for i := Family(0); i < maxFamily; i++ {
			impl := &s.impls[i]
			if impl.stack != nil {
				wg.Add(1)
				go impl.run(&wg)
			}
		}
		wg.Wait()
	})
}

type ImplOption struct {
	Mux                 bool
	TransportMaxTotal   uint
	TransportMaxPerAddr uint
}

type Impl struct {
	stack      *Stack
	Family     Family
	Option     ImplOption
	Listener   IListener
	Dialer     IDialer
	Scanner    IScanner
	Advertiser IAdvertiser

	trManager   *_TrManager
	connManager *_ConnManager
}

func (i *Impl) run(wg *sync.WaitGroup) {
	var itrCh chan ITransport
	if i.Listener != nil {
		itrCh = make(chan ITransport, 1)
		i.Listener.AcceptTransport(itrCh)
	}
	wg.Done()
	for {
		select {
		case itr := <-itrCh:
			tr, err := i.trManager.Track(itr)
			if err != nil {
				itr.Close()
			}
			if !i.Option.Mux {
				conn, err := i.connManager.Track(tr.iface, nil, true)
				if err != nil {
					conn.Close()
				}
			}
		}
	}
}

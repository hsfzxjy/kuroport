package kstack

import (
	"context"
	"kstack/internal"
	"kstack/internal/conn"
	"kstack/internal/transport"
	"sync"
)

type IDialer = internal.IDialer
type IListener = internal.IListener
type IAdvertiser = internal.IAdvertiser
type IScanner = internal.IScanner

type IIdentity = internal.IIdentity
type IConn = internal.IConn
type IAddr = internal.IAddr

type ITransport = internal.ITransport

type IDevice = internal.IDevice

type Family = internal.Family

type ConnID = internal.ConnID

type Option = internal.StackOption
type ImplOption = internal.ImplOption

type Stack struct {
	option  Option
	impls   [internal.MaxFamily]_Impl
	runOnce sync.Once
}

func New(option Option) *Stack {
	stack := new(Stack)
	stack.option = option
	return stack
}

func (s *Stack) getImpl(family Family) *_Impl {
	if family >= internal.MaxFamily {
		return nil
	}
	return &s.impls[family]
}

func (s *Stack) Run() {
	s.runOnce.Do(func() {
		var wg sync.WaitGroup
		for i := Family(0); i < internal.MaxFamily; i++ {
			impl := &s.impls[i]
			if impl.stack != nil {
				wg.Add(1)
				go impl.Run(&wg)
			}
		}
		wg.Wait()
	})
}

func (s *Stack) Register(impl Impl) {
	family := impl.Family
	oldImpl := s.getImpl(family)
	if oldImpl.stack != nil {
		panic("kstack: family has been registered")
	}
	oldImpl.i = impl
	oldImpl.stack = s
	oldImpl.trManager = transport.NewManager(oldImpl)
	oldImpl.connManager = conn.NewManager(oldImpl)
}

type Impl struct {
	Family     Family
	Option     ImplOption
	Listener   IListener
	Dialer     IDialer
	Scanner    IScanner
	Advertiser IAdvertiser
}

type _Impl struct {
	i           Impl
	stack       *Stack
	trManager   internal.TrManager
	connManager internal.ConnManager
}

func (i *_Impl) ConnManager() internal.ConnManager {
	return i.connManager
}

func (i *_Impl) Dialer() internal.IDialer {
	return i.i.Dialer
}

func (i *_Impl) ImplOption() internal.ImplOption {
	return i.i.Option
}

func (i *_Impl) StackOption() internal.StackOption {
	return i.stack.option
}

func (i *_Impl) TrManager() internal.TrManager {
	return i.trManager
}

func (i *_Impl) Run(wg *sync.WaitGroup) {
	var itrCh chan internal.ITransport
	if i.i.Listener != nil {
		itrCh = make(chan internal.ITransport, 1)
		i.i.Listener.AcceptTransport(itrCh)
	}
	wg.Done()
	for {
		select {
		case itr := <-itrCh:
			_, err := i.trManager.TrackRemote(itr)
			if err != nil {
				itr.Close()
			}
		}
	}
}

func (s *Stack) DialAddr(ctx context.Context, addr IAddr, failFast bool) (IConn, error) {
	impl := s.getImpl(addr.Family())
	if impl == nil || impl.i.Dialer == nil {
		return nil, ErrBadAddress
	}
	tr, err := impl.trManager.Dial(ctx, addr, failFast)
	if err != nil {
		return nil, err
	}
	return tr.Open(impl)
}

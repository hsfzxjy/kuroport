package kstack

import (
	"context"
	"kstack/internal"
	"kstack/internal/conn"
	"kstack/internal/stream"
	"sync"
)

type IDialer = internal.IDialer
type IListener = internal.IListener
type IAdvertiser = internal.IAdvertiser
type IScanner = internal.IScanner

type IStream = internal.IStream
type IAddr = internal.IAddr

type IConn = internal.IConn

type IDevice = internal.IUnidentifiedDevice

type Family = internal.Family

type ConnID = internal.StreamID

type Option = internal.Option
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
	s.validateImplOption(&oldImpl.i.Option)
	oldImpl.stack = s
	oldImpl.connMgr = conn.NewManager(oldImpl)
	oldImpl.streamMgr = stream.NewManager(oldImpl)
}

func (s *Stack) validateImplOption(o *ImplOption) {
	if o.ConnMaxAlive == 0 {
		o.ConnMaxAlive = internal.MAX_SIZE
	}
	if o.ConnMaxDialing == 0 {
		o.ConnMaxDialing = internal.MAX_SIZE
	}
	if o.ConnPerAddrMaxAlive == 0 {
		o.ConnPerAddrMaxAlive = internal.MAX_SIZE
	}
	if o.ConnPerAddrMaxDialing == 0 {
		o.ConnPerAddrMaxDialing = internal.MAX_SIZE
	}
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
	i         Impl
	stack     *Stack
	connMgr   internal.IConnMgr
	streamMgr internal.IStreamMgr
}

func (i *_Impl) ConnMgr() internal.IConnMgr {
	return i.connMgr
}

func (i *_Impl) Dialer() internal.IDialer {
	return i.i.Dialer
}

func (i *_Impl) ImplOption() internal.ImplOption {
	return i.i.Option
}

func (i *_Impl) StackOption() internal.Option {
	return i.stack.option
}

func (i *_Impl) StreamMgr() internal.IStreamMgr {
	return i.streamMgr
}

func (i *_Impl) Run(wg *sync.WaitGroup) {
	var connCh chan internal.IConn
	if i.i.Listener != nil {
		connCh = make(chan internal.IConn, 1)
		i.i.Listener.AcceptTransport(connCh)
	}
	wg.Done()
	for {
		select {
		case c := <-connCh:
			_, err := i.connMgr.TrackRemote(c)
			if err != nil {
				c.Close()
			}
		}
	}
}

func (s *Stack) DialAddr(ctx context.Context, addr IAddr, failFast bool) (IStream, error) {
	impl := s.getImpl(addr.Family())
	if impl == nil || impl.i.Dialer == nil {
		return nil, ErrBadAddress
	}
	c, err := impl.connMgr.Dial(ctx, addr, failFast)
	if err != nil {
		return nil, err
	}
	return c.Open(impl)
}

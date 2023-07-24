//go:build test

package mock

import (
	"context"
	"kservice"
	"kstack"
	"kstack/internal"
	ku "kutil"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type disposableStack struct {
	*kstack.Stack
	Disposer ku.F
}

type stackBuilder struct {
	s     *disposableStack
	start ku.F
}

func (b stackBuilder) Impl(impl kstack.Impl) stackBuilder {
	impl.Family = internal.TestFamily
	b.s.Register(impl)
	if impl.Listener != nil {
		b.start = b.start.With(func() {
			impl.Listener.Start()
			ch, cancel := impl.Listener.Events().Listen()
			defer cancel()
			for e := range ch {
				if e.State == kservice.Started {
					return
				}
			}
		})
		b.s.Disposer = b.s.Disposer.With(func() { impl.Listener.Stop() })
	}
	return b
}

func (b stackBuilder) Build() *disposableStack {
	b.s.Run()
	b.start.Do()
	return b.s
}

func Stack(option kstack.Option) stackBuilder {
	return stackBuilder{&disposableStack{
		kstack.New(option), nil,
	}, nil}
}

type CS struct {
	Ch      <-chan kstack.IConn
	C       *kstack.Stack
	S       *kstack.Stack
	dispose ku.F
}

func (cs *CS) Dispose() {
	cs.dispose.Do()
}

func (cs *CS) GetConns(t *testing.T, n int, addr kstack.IAddr) (clientConns, serverConns []kstack.IConn) {
	clientConns = make([]internal.IConn, n)
	serverConns = make([]internal.IConn, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			clientConn, err := cs.C.DialAddr(context.Background(), addr, false)
			require.ErrorIs(t, err, nil)
			clientConns[i] = clientConn

			serverConn, ok := <-cs.Ch
			require.True(t, ok)
			serverConns[i] = serverConn
		}(i)
	}
	wg.Wait()
	return clientConns, serverConns
}

func ClientServer(
	listener kstack.IListener,
	dialer kstack.IDialer,
	implOption kstack.ImplOption) *CS {
	cs := new(CS)
	ch := make(chan kstack.IConn, 10)
	cs.Ch = ch
	c := Stack(kstack.Option{}).
		Impl(kstack.Impl{
			Dialer: dialer,
			Option: implOption}).
		Build()
	s := Stack(kstack.Option{RemoteConns: ch}).
		Impl(kstack.Impl{
			Listener: listener,
			Option:   implOption}).
		Build()
	cs.C = c.Stack
	cs.S = s.Stack
	cs.dispose = c.Disposer.With(s.Disposer)
	return cs
}

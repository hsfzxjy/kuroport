package kstack_test

import (
	"context"
	"kstack"
	"kstack/internal/tests/mock"
	mock_tcp "kstack/internal/tests/mock/tcp"
	"kstack/internal/tracer"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTcpDial(t *testing.T) {
	tracer.Expect(tracer.Type{
		TrSlotDeleted: 2,
		ConnDeleted:   2,
	})
	defer tracer.Wait(t)
	connCh := make(chan kstack.IConn, 2)
	server := mock.Stack(kstack.Option{RemoteConns: connCh}).
		Impl(kstack.Impl{
			Listener: mock_tcp.Listener(),
			Option:   kstack.ImplOption{Mux: true}}).
		Build()
	defer server.Disposer.Do()
	client := mock.Stack(kstack.Option{}).
		Impl(kstack.Impl{
			Dialer: mock_tcp.Dialer(),
			Option: kstack.ImplOption{Mux: true}}).
		Build()
	defer client.Disposer.Do()
	clientConn, err := client.DialAddr(context.Background(), mock_tcp.Addr, false)
	require.ErrorIs(t, err, nil)
	defer clientConn.Transport().Close()
	serverConn, ok := <-connCh
	require.True(t, ok)
	defer serverConn.Transport().Close()
}

func TestTcpDialNoMux(t *testing.T) {
	tracer.Expect(tracer.Type{
		TrSlotDeleted: 2,
		ConnDeleted:   2,
	})
	defer tracer.Wait(t)
	connCh := make(chan kstack.IConn, 2)
	server := mock.Stack(kstack.Option{RemoteConns: connCh}).
		Impl(kstack.Impl{
			Listener: mock_tcp.Listener(),
			Option:   kstack.ImplOption{Mux: false}}).
		Build()
	defer server.Disposer.Do()
	client := mock.Stack(kstack.Option{}).
		Impl(kstack.Impl{
			Dialer: mock_tcp.Dialer(),
			Option: kstack.ImplOption{Mux: false}}).
		Build()
	defer client.Disposer.Do()
	clientConn, err := client.DialAddr(context.Background(), mock_tcp.Addr, false)
	require.ErrorIs(t, err, nil)
	defer clientConn.Close()
	serverConn, ok := <-connCh
	require.True(t, ok)
	defer serverConn.Close()
}

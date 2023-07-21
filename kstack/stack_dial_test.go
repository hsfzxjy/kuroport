package kstack_test

import (
	"context"
	"kstack"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTcpDial(t *testing.T) {
	kstack.Tracker.Reset(kstack.TrackerValue{
		TrDeleted:   2,
		ConnDeleted: 2,
	})
	defer kstack.Tracker.Wait(t, 0)
	connCh := make(chan kstack.IConn, 2)
	server := Stack(kstack.Option{connCh}).
		Impl(kstack.Impl{
			Listener: newTestTcpListener(),
			Option:   kstack.ImplOption{Mux: true}}).
		Build()
	defer server.Disposer.Do()
	client := Stack(kstack.Option{}).
		Impl(kstack.Impl{
			Dialer: _TestTcpDialer{},
			Option: kstack.ImplOption{Mux: true}}).
		Build()
	defer client.Disposer.Do()
	clientConn, err := client.DialAddr(context.Background(), testTcpAddr, false)
	require.ErrorIs(t, err, nil)
	defer clientConn.Transport().Close()
	serverConn, ok := <-connCh
	require.True(t, ok)
	defer serverConn.Transport().Close()
}

func TestTcpDialNoMux(t *testing.T) {
	kstack.Tracker.Reset(kstack.TrackerValue{
		TrDeleted:   2,
		ConnDeleted: 2,
	})
	defer kstack.Tracker.Wait(t, 500*time.Millisecond)
	connCh := make(chan kstack.IConn, 2)
	server := Stack(kstack.Option{connCh}).
		Impl(kstack.Impl{
			Listener: newTestTcpListener(),
			Option:   kstack.ImplOption{Mux: false}}).
		Build()
	defer server.Disposer.Do()
	client := Stack(kstack.Option{}).
		Impl(kstack.Impl{
			Dialer: _TestTcpDialer{},
			Option: kstack.ImplOption{Mux: false}}).
		Build()
	defer client.Disposer.Do()
	clientConn, err := client.DialAddr(context.Background(), testTcpAddr, false)
	require.ErrorIs(t, err, nil)
	defer clientConn.Close()
	serverConn, ok := <-connCh
	require.True(t, ok)
	defer serverConn.Close()
}

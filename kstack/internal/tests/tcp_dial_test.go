package kstack_test

import (
	"context"
	"kstack"
	"kstack/internal"
	"kstack/internal/mock"
	mocktcp "kstack/internal/mock/tcp"
	"kstack/internal/tracer"
	"ktest"
	ku "kutil"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const M = 10

func TestTcpDial(t *testing.T) {
	for N := 1; N <= M; N++ {
		t.Run(strconv.Itoa(N), func(t *testing.T) {
			defer tracer.Expect(tracer.Type{
				TrSlotDeleted: 2,
				ConnDeleted:   ktest.Counter(2 * N)}).Wait(t)

			cs := mock.ClientServer(
				mocktcp.Listener(),
				mocktcp.Dialer(),
				kstack.ImplOption{Mux: true})
			defer cs.Dispose()

			clientConns, serverConns := cs.GetConns(t, N, mocktcp.Addr)

			ktest.RequireAllEqual(t, ku.Map(clientConns, kstack.IStream.Conn))
			ktest.RequireAllEqual(t, ku.Map(serverConns, kstack.IStream.Conn))

			clientConns[0].Conn().Close()
			serverConns[0].Conn().Close()
		})
	}
}

func TestTcpDial_NoMux(t *testing.T) {
	for N := 1; N <= M; N++ {
		t.Run(strconv.Itoa(N), func(t *testing.T) {
			defer tracer.Expect(tracer.Type{
				TrSlotDeleted: ktest.Counter(N + 1),
				ConnDeleted:   ktest.Counter(2 * N)}).Wait(t)

			cs := mock.ClientServer(
				mocktcp.Listener(),
				mocktcp.Dialer(),
				kstack.ImplOption{Mux: false})
			defer cs.Dispose()

			clientConns, serverConns := cs.GetConns(t, N, mocktcp.Addr)

			ktest.RequireAllNotEqual(t, ku.Map(clientConns, kstack.IStream.Conn))
			ktest.RequireAllNotEqual(t, ku.Map(serverConns, kstack.IStream.Conn))

			ku.Map(clientConns, kstack.IStream.Close)
			ku.Map(serverConns, kstack.IStream.Close)
		})
	}
}

func TestTcpDial_Delayed(t *testing.T) {
	const N = M
	defer tracer.Expect(tracer.Type{
		TrSlotDeleted: 2,
		ConnDeleted:   ktest.Counter(2 * N)}).Wait(t)

	delay := 100 * time.Millisecond
	cs := mock.ClientServer(
		mocktcp.Listener(),
		mocktcp.DelayedDialer(delay),
		kstack.ImplOption{Mux: true})
	defer cs.Dispose()

	startTime := time.Now()
	clientConns, serverConns := cs.GetConns(t, N, mocktcp.Addr)
	require.WithinDuration(t, startTime.Add(delay), time.Now(), delay/2)

	ktest.RequireAllEqual(t, ku.Map(clientConns, kstack.IStream.Conn))
	ktest.RequireAllEqual(t, ku.Map(serverConns, kstack.IStream.Conn))

	clientConns[0].Conn().Close()
	serverConns[0].Conn().Close()
}

func TestTcpDial_NoMux_Delayed(t *testing.T) {
	for N := 1; N <= M; N++ {
		t.Run(strconv.Itoa(N), func(t *testing.T) {
			defer tracer.Expect(tracer.Type{
				TrSlotDeleted: ktest.Counter(N + 1),
				ConnDeleted:   ktest.Counter(2 * N)}).Wait(t)

			cs := mock.ClientServer(
				mocktcp.Listener(),
				mocktcp.DelayedDialer(10*time.Millisecond),
				kstack.ImplOption{Mux: false})
			defer cs.Dispose()

			clientConns, serverConns := cs.GetConns(t, N, mocktcp.Addr)

			ktest.RequireAllNotEqual(t, ku.Map(clientConns, kstack.IStream.Conn))
			ktest.RequireAllNotEqual(t, ku.Map(serverConns, kstack.IStream.Conn))

			ku.Map(clientConns, kstack.IStream.Close)
			ku.Map(serverConns, kstack.IStream.Close)
		})
	}
}

func TestTcpDial_NoMux_Delayed_FailFast(t *testing.T) {
	defer tracer.Expect(tracer.Type{
		TrSlotDeleted: 2,
		ConnDeleted:   2}).Wait(t)

	firstDialing := make(chan struct{})
	secondDialing := make(chan struct{})

	cs := mock.ClientServer(
		mocktcp.Listener(),
		mocktcp.NotifiableDialer(firstDialing),
		kstack.ImplOption{
			Mux:                   false,
			ConnPerAddrMaxDialing: 1})
	defer cs.Dispose()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		c, err := cs.C.DialAddr(context.Background(), mocktcp.Addr.WithCh(secondDialing), false)
		require.Nil(t, err)
		c.Close()
	}()

	<-firstDialing
	go func() {
		defer wg.Done()
		_, err := cs.C.DialAddr(context.Background(), mocktcp.Addr, true)
		require.ErrorIs(t, err, internal.ErrTryAgain)
		close(secondDialing)
	}()
	wg.Wait()
	(<-cs.Ch).Close()
}

func TestTcpDial_Delayed_Error(t *testing.T) {
	const N = M
	defer tracer.Expect(tracer.Type{
		TrSlotDeleted: 1,
		ConnDeleted:   0}).Wait(t)

	delay := 10 * time.Millisecond
	cs := mock.ClientServer(
		mocktcp.Listener(),
		mocktcp.DelayedDialer(delay),
		kstack.ImplOption{Mux: true})
	defer cs.Dispose()

	startTime := time.Now()
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := cs.C.DialAddr(context.Background(), mocktcp.AddrBad, false)
			require.NotNil(t, err)
		}(i)
	}
	wg.Wait()
	elapsedTime := time.Since(startTime)
	require.GreaterOrEqual(t, elapsedTime, delay*N)
}

func TestTcpDial_NoMux_Delayed_Error(t *testing.T) {
	const N = M
	defer tracer.Expect(tracer.Type{
		TrSlotDeleted: 1,
		ConnDeleted:   0}).Wait(t)

	delay := 10 * time.Millisecond
	cs := mock.ClientServer(
		mocktcp.Listener(),
		mocktcp.DelayedDialer(delay),
		kstack.ImplOption{Mux: false})
	defer cs.Dispose()

	startTime := time.Now()
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := cs.C.DialAddr(context.Background(), mocktcp.AddrBad, false)
			require.NotNil(t, err)
		}(i)
	}
	wg.Wait()
	elapsedTime := time.Since(startTime)
	require.GreaterOrEqual(t, elapsedTime, delay)
}

func TestTcpDial_Delayed_Error_CancelWaiter(t *testing.T) {
	for _, mux := range [2]bool{false, true} {
		name := map[bool]string{false: "NoMux", true: "Mux"}[mux]
		t.Run(name, func(t *testing.T) {
			defer tracer.Expect(tracer.Type{
				TrSlotDeleted: 1,
				ConnDeleted:   0}).Wait(t)

			delay := 10 * time.Millisecond
			cs := mock.ClientServer(
				mocktcp.Listener(),
				mocktcp.DelayedDialer(delay),
				kstack.ImplOption{Mux: mux})
			defer cs.Dispose()

			done := make(chan struct{})
			go func() {
				close(done)
				_, err := cs.C.DialAddr(context.Background(), mocktcp.AddrBad, false)
				require.NotNil(t, err)
			}()

			<-done
			time.Sleep(delay / 10)
			done = make(chan struct{})
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				close(done)
				_, err := cs.C.DialAddr(ctx, mocktcp.AddrBad, false)
				require.ErrorIs(t, err, ctx.Err())
			}()
			<-done
			time.Sleep(delay / 10)
			cancel()
		})
	}
}

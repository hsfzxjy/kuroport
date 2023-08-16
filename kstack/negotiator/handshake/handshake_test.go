//go:build test

package handshake_test

import (
	"context"
	"kstack"
	"kstack/internal/mock"
	nego "kstack/negotiator"
	"kstack/negotiator/core"
	"ktest"
	kfake "ktest/fake"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

type Party struct{ mock.Party }

func (p Party) asNegotiator(store core.IStore) nego.INegotiator {
	return nego.New(core.Config{
		LocalID:  p.ID,
		LocalKey: p.Priv,
	}, store)
}

func (p Party) Out(conn kstack.IConn, store core.IStore, oopt core.OutboundOption) (kstack.IConn, error) {
	n := p.asNegotiator(store)
	return n.HandleOutbound(context.Background(), conn, oopt)
}

func (p Party) In(conn kstack.IConn, store core.IStore) (kstack.IConn, error) {
	n := p.asNegotiator(store)
	return n.HandleInbound(context.Background(), conn)

}

var A, B = Party{mock.NewParty(42)}, Party{mock.NewParty(43)}

func ConnPair() (initiator kstack.IConn, responder kstack.IConn) {
	return ktest.TcpPair(func(c net.Conn, initiator bool) kstack.IConn { return kstack.WrapTransport(c, kstack.WrapOption{}) })
}

type Store struct{ PassCode core.PassCode }

func (s *Store) GetPassCode() core.PassCode { return s.PassCode }

func Test_Handshake_FirstTime(t *testing.T) {
	cA, cB := ConnPair()
	defer cA.Close()
	defer cB.Close()
	passCode := core.PassCode("abcd")

	scope := ktest.Scope()

	testDataA, testDataB := kfake.Bytes(16), kfake.Bytes(16)

	scope.Go(func() {
		conn, err := A.Out(cA, nil, core.OutboundOption{PassCode: passCode})
		require.ErrorIs(t, err, nil)
		ktest.RequireWriteSuccess(t, conn, testDataA[:])
		ktest.RequireReadEqual(t, conn, testDataB[:])

	})

	scope.Go(func() {
		conn, err := B.In(cB, &Store{passCode})
		require.ErrorIs(t, err, nil)
		ktest.RequireReadEqual(t, conn, testDataA[:])
		ktest.RequireWriteSuccess(t, conn, testDataB[:])
	})

	scope.Wait()
}

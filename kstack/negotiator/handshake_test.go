//go:build test

package nego_test

import (
	"kstack/internal/mock"
	mocktcp "kstack/internal/mock/tcp"
	nego "kstack/negotiator"
	"kstack/negotiator/core"
	"kstack/negotiator/handshake"
	"ktest"
	kfake "ktest/fake"
	"testing"

	"github.com/stretchr/testify/require"
)

var A, B = mock.NewParty(42), mock.NewParty(43)

type Store struct{ PassCode core.PassCode }

func (s *Store) GetPassCode() core.PassCode { return s.PassCode }

func Test_Handshake_FirstTime(t *testing.T) {
	c := mocktcp.Pair()
	defer c.Close()
	passCode := core.PassCode("abcd")

	scope := ktest.Scope()

	testDataA, testDataB := kfake.Bytes(16), kfake.Bytes(16)

	scope.Go(func() {
		conn, err := nego.M(
			c.A,
			handshake.Party(A),
			handshake.HSOpt(&core.HSOpt{PassCode: passCode}))
		require.ErrorIs(t, err, nil)
		ktest.RequireWriteSuccess(t, conn, testDataA[:])
		ktest.RequireReadEqual(t, conn, testDataB[:])

	})

	scope.Go(func() {
		conn, err := nego.M(
			c.B,
			handshake.Party(B),
			handshake.Store(&Store{passCode}))
		require.ErrorIs(t, err, nil)
		ktest.RequireReadEqual(t, conn, testDataA[:])
		ktest.RequireWriteSuccess(t, conn, testDataB[:])
	})

	scope.Wait()
}

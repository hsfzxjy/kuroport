//go:build test

package nego_test

import (
	"fmt"
	"io"
	"kstack/internal/mock"
	mocktcp "kstack/internal/mock/tcp"
	nego "kstack/negotiator"
	"kstack/negotiator/core"
	"kstack/negotiator/handshake"
	sec "kstack/security"
	"ktest"
	kfake "ktest/fake"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

var A, B = mock.NewParty(42), mock.NewParty(43)

type Store struct{ PassCode core.PassCode }

func (*Store) HandleInitiatorMsg(m core.InitiatorMsg) (r core.ReplyToInitiator, err error) {
	return core.ReplyToInitiator{
		CalibratedExpiration: m.Expiration}, nil
}

func (*Store) HandleNegotiatedPeer(p core.NegotiatedPeer) error {
	return nil
}

func (*Store) HandleResponderMsg(m core.ResponderMsg) (err error) {
	return nil
}

func (s *Store) GetPassCode() core.PassCode { return s.PassCode }

type er struct {
	i, r error
}

func (e *er) IsEmpty() bool { return e.i == nil && e.r == nil }

func (e *er) Merge(other er) {
	if e.i == nil {
		e.i = other.i
	}
	if e.r == nil {
		e.r = other.r
	}
}

type au struct {
	iCap   sec.AuthCap
	iLevel sec.AuthLevel
	rCap   sec.AuthCap
}

var auList []au

func init() {
	for _, iCap := range []sec.AuthCap{false, true} {
		for _, iLevel := range []sec.AuthLevel{sec.RequireNoAuth, sec.AuthWhatever, sec.RequireAuth} {
			for _, rCap := range []sec.AuthCap{false, true} {
				auList = append(auList, au{iCap, iLevel, rCap})
			}
		}
	}
}

func (a au) IValid() bool {
	_, ok := a.iCap.Normalize(a.iLevel)
	return ok
}

func (a au) RValid() bool {
	normed, _ := a.iCap.Normalize(a.iLevel)
	_, ok := a.rCap.Decide(normed)
	return ok
}

func (a au) Use() bool {
	normed, _ := a.iCap.Normalize(a.iLevel)
	use, _ := a.rCap.Decide(normed)
	return use
}

func (a au) String() string {
	return fmt.Sprintf("%v--[%v]-->%v", a.iCap, a.iLevel, a.rCap)
}

type se struct {
	iCap   sec.SecCap
	iLevel sec.SecLevel
	rCap   sec.SecCap
}

var seList []se

func init() {
	for _, iCap := range []sec.SecCap{false, true} {
		for _, iLevel := range []sec.SecLevel{sec.RequireCleartext, sec.SecWhatever, sec.RequireEncrypted} {
			for _, rCap := range []sec.SecCap{false, true} {
				seList = append(seList, se{iCap, iLevel, rCap})
			}
		}
	}
}

func (s se) String() string {
	return fmt.Sprintf("%v--[%v]-->%v", s.iCap, s.iLevel, s.rCap)
}

func (s se) IValid() bool {
	_, ok := s.iCap.Normalize(s.iLevel)
	return ok
}

func (s se) RValid() bool {
	normed, _ := s.iCap.Normalize(s.iLevel)
	_, ok := s.rCap.Decide(normed)
	return ok
}

func (s se) Use() bool {
	normed, _ := s.iCap.Normalize(s.iLevel)
	use, _ := s.rCap.Decide(normed)
	return use
}

type testCase struct {
	au
	se
	err er
}

func (c *testCase) String() string {
	return fmt.Sprintf("%v++%v", c.au, c.se)
}

var testCases []testCase

func init() {
	for _, a := range auList {
		for _, s := range seList {
			tc := testCase{au: a, se: s}
			if !a.IValid() || !s.IValid() {
				tc.err.Merge(er{handshake.ErrBadOption, io.EOF})
			}
			if !a.RValid() || !s.RValid() {
				tc.err.Merge(er{io.EOF, handshake.ErrBadOption})
			}
			if !tc.au.Use() && tc.se.Use() {
				tc.err.Merge(er{io.EOF, handshake.ErrBadOption})
			}
			testCases = append(testCases, tc)
		}
	}
}

func _Test_Handshake_FirstTime(t *testing.T, tc testCase) {
	c := mocktcp.Pair()
	defer c.Close()
	passCode := core.PassCode("abcd")

	scope := ktest.Scope()

	testDataA, testDataB := kfake.Bytes(16), kfake.Bytes(16)

	scope.Go(func() {
		conn, err := nego.M(
			c.A,
			handshake.I,
			handshake.Cfg(&core.Config{
				SecCap:  tc.se.iCap,
				AuthCap: tc.au.iCap,
			}),
			handshake.Party(A),
			handshake.HSOpt(&core.HSOpt{
				PassCode:  passCode,
				SecLevel:  tc.se.iLevel,
				AuthLevel: tc.au.iLevel,
			}),
			handshake.Store(&Store{}))
		require.ErrorIs(t, err, tc.err.i)
		if err == nil {
			require.Equal(t, B.ID, conn.RemoteID())
			require.Equal(t, tc.se.Use(), conn.IsSecure())
			ktest.RequireWriteSuccess(t, conn, testDataA[:])
			ktest.RequireReadEqual(t, conn, testDataB[:])
		}
	})

	scope.Go(func() {
		conn, err := nego.M(
			c.B,
			handshake.R,
			handshake.Cfg(&core.Config{
				SecCap:  tc.se.rCap,
				AuthCap: tc.au.rCap,
			}),
			handshake.Party(B),
			handshake.Store(&Store{passCode}))
		require.ErrorIs(t, err, tc.err.r)
		if err == nil {
			require.Equal(t, A.ID, conn.RemoteID())
			require.Equal(t, tc.se.Use(), conn.IsSecure())
			ktest.RequireReadEqual(t, conn, testDataA[:])
			ktest.RequireWriteSuccess(t, conn, testDataB[:])
		}
	})

	scope.Wait()
}

func Test_Handshake_FirstTime(t *testing.T) {
	for idx, tc := range testCases[:] {
		t.Run(strconv.Itoa(idx)+":"+tc.String(), func(t *testing.T) {
			_Test_Handshake_FirstTime(t, tc)
		})
	}
}

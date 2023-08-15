package handshake_test

import (
	"context"
	"kstack"
	kc "kstack/crypto"
	"kstack/internal"
	"kstack/negotiator/core"
	"kstack/negotiator/handshake"
	"kstack/peer"
	mrand "math/rand"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type Party struct {
	ID   peer.ID
	Pub  kc.PubKey
	Priv kc.PrivKey
}

func (p Party) NegotiateOn(conn kstack.IConn, initiator bool, passCode core.PassCode) error {
	var store = new(Store)
	var oopt core.OutboundOption
	if initiator {
		oopt.PassCode = passCode
	} else {
		store.PassCode = passCode
	}
	ctx := context.Background()
	cfg := &core.Config{
		LocalID:  p.ID,
		LocalKey: p.Priv,
	}
	var err error
	if initiator {
		_, err = handshake.Initiate(ctx, cfg, oopt, store, conn)
	} else {
		_, err = handshake.Respond(ctx, cfg, store, conn)
	}
	return err
}

func makeParty(seed int64) Party {
	rng := mrand.New(mrand.NewSource(seed))
	priv, pub, _ := kc.GenerateEd25519Key(rng)
	id, _ := peer.IDFromPublicKey(pub)
	return Party{id, pub, priv}
}

var A, B = makeParty(42), makeParty(43)

type Conn struct {
	net.Conn
	remotePeer peer.ID
}

// Addr implements internal.IConn.
func (*Conn) Addr() internal.IAddr { panic("unimplemented") }

// DiedCh implements internal.IConn.
func (*Conn) DiedCh() <-chan struct{} { panic("unimplemented") }

// Family implements internal.IConn.
func (*Conn) Family() internal.Family { panic("unimplemented") }

func (c *Conn) RemoteID() peer.ID { return c.remotePeer }

func ConnPair() (initiator kstack.IConn, responder kstack.IConn) {
	l, _ := net.Listen("tcp4", ":0")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer l.Close()
		c, _ := l.Accept()
		responder = &Conn{c, ""}
	}()
	c, _ := net.Dial("tcp4", l.Addr().String())
	initiator = &Conn{c, ""}
	wg.Wait()
	return
}

type Store struct {
	PassCode core.PassCode
}

func (s *Store) GetPassCode() core.PassCode { return s.PassCode }

func Test_Handshake_FirstTime(t *testing.T) {
	ci, cr := ConnPair()
	defer ci.Close()
	defer cr.Close()
	passCode := core.PassCode("abcd")
	var wg sync.WaitGroup
	wg.Add(2)
	var errA, errB error
	go func() {
		defer wg.Done()
		errA = A.NegotiateOn(ci, true, passCode)
		require.ErrorIs(t, errA, nil)
	}()
	go func() {
		defer wg.Done()
		errB = B.NegotiateOn(cr, false, passCode)
		require.ErrorIs(t, errB, nil)
	}()
	wg.Wait()
}

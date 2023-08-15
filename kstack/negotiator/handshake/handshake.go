package handshake

import (
	"context"
	"kstack"
	"kstack/negotiator/core"
	"kstack/negotiator/rw"
	"kstack/peer"
	"sync"
	"time"

	"slices"

	"github.com/flynn/noise"
)

var cipherSuite = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)

type Result struct {
	UseEncryption bool
	RemoteID      peer.ID
	Enc, Dec      *noise.CipherState
	RW            rw.RW
}

type _Session struct {
	conn      kstack.IConn
	initiator bool
	cfg       *core.Config
	oopt      core.OutboundOption
	store     core.IStore
	rw        _RW

	Error   error
	Version uint8
	Stage   [2]byte

	Result
}

var sessionPool = sync.Pool{
	New: func() any { return new(_Session) },
}

func Initiate(ctx context.Context, config *core.Config, oopt core.OutboundOption, store core.IStore, conn kstack.IConn) (Result, error) {
	s := sessionPool.Get().(*_Session)
	defer func() {
		*s = _Session{}
		sessionPool.Put(s)
	}()
	s.conn = conn
	s.initiator = true
	s.cfg = config
	s.oopt = oopt
	s.store = store
	err := s.run(ctx)
	if err != nil {
		return Result{}, err
	}
	return s.Result, nil
}

func Respond(ctx context.Context, config *core.Config, store core.IStore, conn kstack.IConn) (Result, error) {
	s := sessionPool.Get().(*_Session)
	defer func() {
		*s = _Session{}
		sessionPool.Put(s)
	}()
	s.conn = conn
	s.initiator = false
	s.cfg = config
	s.store = store
	err := s.run(ctx)
	if err != nil {
		return Result{}, err
	}
	return s.Result, nil
}

func (s *_Session) run(ctx context.Context) error {
	var (
		wg       sync.WaitGroup
		doneCh   = make(chan struct{}, 1)
		canceled = false
	)
	wg.Add(1)
	conn := s.conn
	go func() {
		defer wg.Done()
		select {
		case <-doneCh:
		case <-ctx.Done():
			canceled = true
		}
		conn.Close()
	}()
	err := s.doRun()
	doneCh <- struct{}{}
	wg.Wait()
	if canceled {
		return ctx.Err()
	}
	if err != nil {
		return _Error{
			Version:   s.Version,
			Initiator: s.initiator,
			Stage:     s.Stage,
			Wrapped:   err,
		}
	}
	return nil
}

func (s *_Session) doRun() (errToReturn error) {
	defer func() {
		s.Error = errToReturn
	}()
	s.conn.SetDeadline(time.Now().Add(core.Timeout))

	defer s.rw.Init(s, 2<<10)()

	if s.initiator {
		// Stage 0.0: Send Hello1 to Responder
		{
			s.Stage = [2]byte{0x0, 0x0}

			var hello1 = _Hello1_Payload{
				ICanEncrypt: true,
				Versions:    [4]uint8{0x01},
			}
			if err := s.rw.WriteMessage(&hello1); err != nil {
				return err
			}
		}

		// Stage 0.1: Recv Resp1 from Responder
		{
			s.Stage = [2]byte{0x0, 0x1}

			var resp1 _Resp1_Payload
			if err := s.rw.ReadMessage(&resp1); err != nil {
				return err
			}
			s.UseEncryption = resp1.UseEncryption
			if !s.UseEncryption { // TODO: Support cleartext handshake
				return ErrAuthFailed
			}
			if resp1.ChosenVersion != 0x01 {
				return ErrUnsupportedVersion
			}

			s.Version = resp1.ChosenVersion
		}

		proto := protocols[s.Version]
		if s.UseEncryption {
			return proto.HandleInitiator(s)
		} else {
			return proto.HandleInitiatorCleartext(s)
		}

	} else {
		// Stage 0.0: Recv Hello1 from Initiator
		{
			s.Stage = [2]byte{0x0, 0x0}

			var hello1 _Hello1_Payload
			if err := s.rw.ReadMessage(&hello1); err != nil {
				return err
			}
			if !hello1.ICanEncrypt {
				return ErrEncryptionRequired
			}
			s.UseEncryption = true
			if !s.UseEncryption {
				return ErrEncryptionRequired
			}
			if !slices.Contains(hello1.Versions[:], 0x01) {
				return ErrUnsupportedVersion
			}
		}

		// Stage 0.1: Send Resp1 to Initiator
		{
			s.Stage = [2]byte{0x0, 0x1}

			var resp1 = _Resp1_Payload{
				UseEncryption: true,
				ChosenVersion: 0x01,
			}
			if err := s.rw.WriteMessage(resp1); err != nil {
				return err
			}
			s.Version = resp1.ChosenVersion
		}

		proto := protocols[s.Version]
		if s.UseEncryption {
			return proto.HandleResponder(s)
		} else {
			return proto.HandleResponderCleartext(s)
		}

	}
}

func (s *_Session) setCipherStates(cs1, cs2 *noise.CipherState) {
	if s.initiator {
		s.Enc = cs1
		s.Dec = cs2
	} else {
		s.Enc = cs2
		s.Dec = cs1
	}
}
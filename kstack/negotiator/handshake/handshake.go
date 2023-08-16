package handshake

import (
	"context"
	"kstack"
	"kstack/negotiator/core"
	"kstack/negotiator/rw"
	"kstack/peer"
	"sync"
	"time"

	"github.com/flynn/noise"
)

var cipherSuite = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)

type Result struct {
	UseEncryption bool
	RemoteID      peer.ID
	Enc, Dec      *noise.CipherState
	RW            rw.RW
}

type _Stage [2]byte

func (s *_Stage) Set(a, b byte) {
	*s = _Stage{a, b}
}

type _Session struct {
	conn      kstack.IConn
	initiator bool
	cfg       *core.Config
	oopt      core.OutboundOption
	store     core.IStore
	rw        _RW

	Error   error
	Version _Version
	Stage   _Stage

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
	}()
	err := s.doRun()
	doneCh <- struct{}{}
	wg.Wait()
	if canceled {
		err = ctx.Err()
	} else if err != nil {
		err = _Error{
			Version:   s.Version,
			Initiator: s.initiator,
			Stage:     s.Stage,
			Wrapped:   err,
		}
	}
	if err != nil {
		conn.Close()
		return err
	}
	return nil
}

func (s *_Session) doRun() (errToReturn error) {
	defer func() {
		s.Error = errToReturn
	}()
	s.conn.SetDeadline(time.Now().Add(core.Timeout))

	defer s.rw.Init(s, 2<<10)()

	var hello1 _Hello1_Payload
	var resp1 _Resp1_Payload
	if s.initiator {
		// Stage 0.0: Send Hello1 to Responder
		{
			s.Stage.Set(0, 0)

			hello1.Pack(true)
			if err := s.rw.WriteMessage(&hello1); err != nil {
				return err
			}
		}

		// Stage 0.1: Recv Resp1 from Responder
		{
			s.Stage.Set(0, 1)

			if err := s.rw.ReadMessage(&resp1); err != nil {
				return err
			}
			if !resp1.VerifyVersion(&hello1) {
				return ErrUnsupportedVersion
			}
			s.UseEncryption = resp1.UseEncryption
			if !s.UseEncryption { // TODO: Support cleartext handshake
				return ErrAuthFailed
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
		var chosenVersion _Version
		{
			s.Stage.Set(0, 0)

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
			chosenVersion = hello1.ChooseVersion()
			if chosenVersion == 0 {
				return ErrUnsupportedVersion
			}
		}

		// Stage 0.1: Send Resp1 to Initiator
		{
			s.Stage.Set(0, 1)

			var resp1 = _Resp1_Payload{
				UseEncryption: true,
				ChosenVersion: chosenVersion,
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

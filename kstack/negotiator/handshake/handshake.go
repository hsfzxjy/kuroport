package handshake

import (
	"context"
	"kstack"
	"kstack/negotiator/core"
	"kstack/negotiator/rw"
	"kstack/peer"
	sec "kstack/security"
	"sync"
	"time"

	"github.com/flynn/noise"
)

var cipherSuite = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)

type Result struct {
	RemoteID peer.ID
	Enc, Dec *noise.CipherState
	RW       rw.RW
}

func (r *Result) IsEncrypted() bool {
	return r.Enc != nil && r.Dec != nil
}

type _Stage [2]byte

func (s *_Stage) Set(a, b byte) {
	*s = _Stage{a, b}
}

type _Session struct {
	Conn      kstack.IConn
	Initiator bool
	Cfg       *core.Config
	HSOpt     core.HSOpt
	Model     core.IModel
	Rw        _RW

	Error   error
	Version _Version
	Stage   _Stage

	core.HandshakeInfo

	Result
}

var sessionPool = sync.Pool{
	New: func() any { return new(_Session) },
}

func Initiate(ctx context.Context, config *core.Config, hsopt core.HSOpt, model core.IModel, conn kstack.IConn) (Result, error) {
	s := sessionPool.Get().(*_Session)
	defer func() {
		*s = _Session{}
		sessionPool.Put(s)
	}()
	s.Conn = conn
	s.Initiator = true
	s.Cfg = config
	s.HSOpt = hsopt
	s.Model = model
	err := s.run(ctx)
	if err != nil {
		return Result{}, err
	}
	return s.Result, nil
}

func Respond(ctx context.Context, config *core.Config, model core.IModel, conn kstack.IConn) (Result, error) {
	s := sessionPool.Get().(*_Session)
	defer func() {
		*s = _Session{}
		sessionPool.Put(s)
	}()
	s.Conn = conn
	s.Initiator = false
	s.Cfg = config
	s.Model = model
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
	conn := s.Conn
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
			Initiator: s.Initiator,
			Stage:     s.Stage,
			Wrapped:   err,
		}
	}
	if err != nil {
		conn.Close()
		return err
	}
	s.Result.RW = s.Rw.RW
	return nil
}

func (s *_Session) precheck() (doHandshake bool, err error) {
	hsopt := &s.HSOpt
	cfg := s.Cfg
	isSecure := s.Conn.IsSecure()

	if isSecure {
		hsopt.SecLevel = sec.RequireCleartext
	}

	remoteID := s.Conn.RemoteID()
	if !remoteID.IsEmpty() {
		return false, nil
	}

	if s.Initiator {
		if sl, ok := cfg.SecCap.Normalize(hsopt.SecLevel); !ok {
			return false, ErrBadOption
		} else {
			hsopt.SecLevel = sl
		}

		if al, ok := cfg.AuthCap.Normalize(hsopt.AuthLevel); !ok {
			return false, ErrBadOption
		} else {
			hsopt.AuthLevel = al
		}

		if hsopt.RemoteID.IsEmpty() {
			s.FirstTime = true
		}
	}

	return true, nil
}

func (s *_Session) doRun() (errToReturn error) {
	if doHandshake, err := s.precheck(); !doHandshake {
		return err
	}

	defer func() {
		s.Error = errToReturn
	}()
	s.Conn.SetDeadline(time.Now().Add(core.Timeout))

	defer s.Rw.Init(s, 2<<10)()

	var hello1 _Hello1_Payload
	var resp1 _Resp1_Payload
	if s.Initiator {
		// Stage 0.0: Send Hello1 to Responder
		{
			s.Stage.Set(0, 0)

			hello1.SecLevel = s.HSOpt.SecLevel
			hello1.AuthLevel = s.HSOpt.AuthLevel
			hello1.FirstTime = s.FirstTime
			hello1.PackVersions()
			if err := s.Rw.WriteMessage(&hello1); err != nil {
				return err
			}
		}

		// Stage 0.1: Recv Resp1 from Responder
		{
			s.Stage.Set(0, 1)

			if err := s.Rw.ReadMessage(&resp1); err != nil {
				return err
			}
			if err := hello1.VerifyResp(&resp1); err != nil {
				return err
			}
			s.Version = resp1.ChosenVersion
			s.UseAuth = resp1.UseAuth
			s.UseEncryption = resp1.UseEncryption
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
			s.Stage.Set(0, 0)

			if err := s.Rw.ReadMessage(&hello1); err != nil {
				return err
			}
			s.FirstTime = hello1.FirstTime

			if err := resp1.Handle(&hello1, s.Cfg); err != nil {
				return err
			}
		}

		// Stage 0.1: Send Resp1 to Initiator
		{
			s.Stage.Set(0, 1)

			if err := s.Rw.WriteMessage(resp1); err != nil {
				return err
			}
			s.Version = resp1.ChosenVersion
			s.UseAuth = resp1.UseAuth
			s.UseEncryption = resp1.UseEncryption
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
	if s.Initiator {
		s.Enc = cs1
		s.Dec = cs2
	} else {
		s.Enc = cs2
		s.Dec = cs1
	}
}

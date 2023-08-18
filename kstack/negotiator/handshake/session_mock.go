//go:build test

package handshake

import (
	"context"
	"kstack"
	"kstack/internal/mock"
	"kstack/negotiator/core"
	"time"
)

type Hello1_Payload = _Hello1_Payload
type Resp1_Payload = _Resp1_Payload
type Version = _Version

type MSession struct {
	_Session
}

type MOpt func(*MSession)

func Cfg(cfg *core.Config) MOpt {
	return func(s *MSession) {
		s.Cfg = cfg
	}
}

func Party(party mock.Party) MOpt {
	return func(s *MSession) {
		var c = s.Cfg
		c.LocalID = party.ID
		c.LocalKey = party.Priv
	}
}

func I(m *MSession) {
	m.Initiator = true
	m.Cfg = &core.Config{}
}

func R(m *MSession) {
	m.Initiator = false
	m.Cfg = &core.Config{}
}

func HSOpt(hsopt *core.HSOpt) MOpt {
	return func(s *MSession) {
		s.Initiator = true
		if hsopt != nil {
			s.HSOpt = *hsopt
		}
	}
}

func Store(store core.IStore) MOpt {
	return func(s *MSession) {
		s.Store = store
	}
}

func NormalRun(s *MSession) error {
	return s.run(context.Background())
}

func M(
	conn kstack.IConn,
	run func(s *MSession) error,
	opts ...MOpt,
) (Result, error) {
	s := new(MSession)
	for _, opt := range opts {
		opt(s)
	}
	s.Conn = conn
	s.Conn.SetDeadline(time.Now().Add(core.Timeout))
	defer s.Rw.Init(&s._Session, 2<<10)()

	err := run(s)
	if err != nil {
		return Result{}, err
	}
	return s.Result, nil
}

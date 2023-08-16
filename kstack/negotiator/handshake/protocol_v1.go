package handshake

import (
	"crypto/rand"

	"github.com/flynn/noise"
)

type _ProtocolV1 struct{}

func (_ProtocolV1) cryptoSuite(s *_Session) (dhkey noise.DHKey, hs *noise.HandshakeState, err error) {
	var prologue []byte
	if s.rw.RemoteID().IsEmpty() {
		var psk []byte
		if s.initiator {
			psk = s.oopt.PassCode
		} else {
			psk = s.store.GetPassCode()
		}
		prologue = make([]byte, 1+len(psk))
		copy(prologue[1:], psk)
	} else {
		prologue = make([]byte, 1)
	}
	prologue[0] = byte(s.Version)

	dhkey, err = noise.DH25519.GenerateKeypair(rand.Reader)
	if err != nil {
		return
	}
	hs, err = noise.NewHandshakeState(noise.Config{
		CipherSuite:   cipherSuite,
		Pattern:       noise.HandshakeXX,
		Initiator:     s.initiator,
		Prologue:      prologue,
		StaticKeypair: dhkey,
	})
	return
}

func (p _ProtocolV1) HandleInitiator(s *_Session) error {
	dhkey, hs, err := p.cryptoSuite(s)
	if err != nil {
		return err
	}
	// Stage 1.0: Send Ephemeral Key to Responder
	s.Stage.Set(1, 0)
	if err := s.rw.EncryptAndWriteMessage(nil, hs); err != nil {
		return err
	}

	// Stage 1.1: Recv Remote ID from Responder
	{
		s.Stage.Set(1, 1)

		var remotePayload _V1_Payload
		if err := s.rw.ReadAndDecryptMessage(&remotePayload, hs); err != nil {
			return err
		}
		remoteId, err := remotePayload.Verify(hs.PeerStatic())
		if err != nil {
			return err
		}
		s.RemoteID = remoteId
	}

	// Stage 2.0: Send Local ID to Responder
	{
		s.Stage.Set(2, 0)

		var localPayload _V1_Payload

		if err := localPayload.Seal(s.cfg.LocalID, s.cfg.LocalKey, dhkey.Public); err != nil {
			return err
		}

		if err := s.rw.EncryptAndWriteMessage(&localPayload, hs); err != nil {
			return err
		}
	}

	// Stage 2.1: Recv Confirmation from Responder
	s.Stage.Set(2, 1)
	if err := s.rw.ReadMessage(nil); err != nil {
		return err
	}

	return nil
}

func (_ProtocolV1) HandleInitiatorCleartext(s *_Session) error {
	panic("unimplemented")
}

func (p _ProtocolV1) HandleResponder(s *_Session) error {
	dhkey, hs, err := p.cryptoSuite(s)
	localID := s.cfg.LocalID
	if err != nil {
		return err
	}

	// Stage 1.0: Recv Ephemeral Key from Initiator
	s.Stage.Set(1, 0)
	if err := s.rw.ReadAndDecryptMessage(nil, hs); err != nil {
		return err
	}

	// Stage 1.1: Send Local ID to Initiator
	{
		s.Stage.Set(1, 1)

		var payload _V1_Payload
		err = payload.Seal(localID, s.cfg.LocalKey, dhkey.Public)
		if err != nil {
			return err
		}
		if err := s.rw.EncryptAndWriteMessage(&payload, hs); err != nil {
			return err
		}
	}

	// Stage 2.0 & 2.1: Recv Remote ID & Send Confirmation
	{
		s.Stage.Set(2, 0)

		var payload _V1_Payload
		if err := s.rw.ReadAndDecryptMessage(&payload, hs); err != nil {
			return err
		}
		remoteId, err := payload.Verify(hs.PeerStatic())
		if err != nil {
			return err
		}
		s.RemoteID = remoteId

		s.Stage.Set(2, 1)
		if err := s.rw.WriteMessage(nil); err != nil {
			return err
		}
	}
	return nil
}

func (_ProtocolV1) HandleResponderCleartext(s *_Session) error {
	panic("unimplemented")
}

var _ _Protocol = _ProtocolV1{}

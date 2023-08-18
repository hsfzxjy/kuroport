package handshake

import (
	"crypto/rand"
	"kstack/negotiator/core"
	"kstack/peer"

	"slices"

	"github.com/flynn/noise"
)

type _ProtocolV1 struct{}

type _CryptoSuite struct {
	dh noise.DHKey
	hs *noise.HandshakeState
}

func (cs *_CryptoSuite) RemoteStatic() []byte {
	if cs.hs == nil {
		return nil
	}
	return cs.hs.PeerStatic()
}

func (cs *_CryptoSuite) LocalStatic() []byte {
	return cs.dh.Public
}

func (_ProtocolV1) cryptoSuite(s *_Session) (cs _CryptoSuite, err error) {
	var prologue []byte
	if s.FirstTime {
		var psk []byte
		if s.Initiator {
			psk = s.HSOpt.PassCode
		} else {
			psk = s.Model.GetPassCode()
		}
		prologue = make([]byte, 1+len(psk))
		copy(prologue[1:], psk)
	} else {
		var responderID peer.ID
		if s.Initiator {
			responderID = s.HSOpt.RemoteID
		} else {
			responderID = s.Cfg.LocalID
		}
		prologue = make([]byte, 1+len(responderID))
		copy(prologue[1:], responderID)
	}
	prologue[0] = byte(s.Version)

	cs.dh, err = noise.DH25519.GenerateKeypair(rand.Reader)
	if err != nil {
		return _CryptoSuite{}, err
	}
	cs.hs, err = noise.NewHandshakeState(noise.Config{
		CipherSuite: cipherSuite,
		Pattern:     noise.HandshakeXX,
		// original responder should initiate the noise handshake
		Initiator:     !s.Initiator,
		Prologue:      prologue,
		StaticKeypair: cs.dh,
	})
	return
}

func (p _ProtocolV1) HandleResponder(s *_Session) error {
	cs, err := p.cryptoSuite(s)
	if err != nil {
		return err
	}
	// Stage 1.0: Send Ephemeral Key to Initiator
	s.Stage.Set(1, 0)
	if err := s.Rw.EncryptAndWriteMessage(nil, cs.hs); err != nil {
		return err
	}

	return p.responder_Stage_1_1_Onwards(s, cs)
}

func (p _ProtocolV1) HandleResponderCleartext(s *_Session) error {
	// Stage 1.0: Recv passcode from Initiator (if first time)
	s.Stage.Set(1, 0)
	if s.FirstTime {
		var passCode core.PassCode
		if err := s.Rw.ReadMessage(&passCode); err != nil {
			return err
		}
		if !slices.Equal(passCode, s.Model.GetPassCode()) {
			return ErrAuthFailed
		}
	}

	return p.responder_Stage_1_1_Onwards(s, _CryptoSuite{})
}

func (_ProtocolV1) responder_Stage_1_1_Onwards(s *_Session, cs _CryptoSuite) error {
	var initMsg core.InitiatorMsg
	var reply core.ReplyToInitiator
	// Stage 1.1: Recv Remote ID from Initiator
	{
		s.Stage.Set(1, 1)

		var initPayload _V1_InitiatorMsg_Payload
		if err := s.Rw.ReadAndDecryptMessage(&initPayload, cs.hs); err != nil {
			return err
		}
		remoteId, err := initPayload.Verify(cs.RemoteStatic(), s.UseAuth)
		if err != nil {
			return err
		}

		initMsg = core.InitiatorMsg{
			InitiatorID:   remoteId,
			HandshakeInfo: s.HandshakeInfo,
			Expiration:    initPayload.Expiration,
		}

		reply, err = s.Model.HandleInitiatorMsg(initMsg)
		if err != nil {
			return err
		}
	}

	// Stage 2.0: Send Local ID to Initiator
	{
		s.Stage.Set(2, 0)

		var respPayload _V1_ResponderMsg_Payload
		respPayload.ReplyToInitiator = reply
		if err := respPayload.Seal(s.Cfg.LocalID, s.Cfg.LocalKey, cs.LocalStatic(), s.UseAuth); err != nil {
			return err
		}

		if err := s.Rw.EncryptAndWriteMessage(&respPayload, cs.hs); err != nil {
			return err
		}
	}

	// Stage 2.1: Recv Confirmation from Initiator
	s.Stage.Set(2, 1)
	if err := s.Rw.ReadMessage(nil); err != nil {
		return err
	}

	if err := s.Model.HandleNegotiatedPeer(core.NegotiatedInitiator(initMsg, reply)); err != nil {
		return err
	}
	s.RemoteID = initMsg.InitiatorID

	return nil
}

func (p _ProtocolV1) HandleInitiator(s *_Session) error {
	cs, err := p.cryptoSuite(s)
	if err != nil {
		return err
	}

	// Stage 1.0: Recv Ephemeral Key from Responder
	s.Stage.Set(1, 0)
	if err := s.Rw.ReadAndDecryptMessage(nil, cs.hs); err != nil {
		return err
	}

	return p.initiator_Stage_1_1_Onwards(s, cs)
}

func (p _ProtocolV1) HandleInitiatorCleartext(s *_Session) error {
	// Stage 1.0: Send passcode to Responder (if first time)
	s.Stage.Set(1, 0)
	if s.FirstTime {
		if err := s.Rw.WriteMessage(s.HSOpt.PassCode); err != nil {
			return err
		}
	}
	return p.initiator_Stage_1_1_Onwards(s, _CryptoSuite{})
}

func (_ProtocolV1) initiator_Stage_1_1_Onwards(s *_Session, cs _CryptoSuite) error {
	var err error
	// Stage 1.1: Send Local ID to Responder
	{
		s.Stage.Set(1, 1)

		var initPayload _V1_InitiatorMsg_Payload
		err = initPayload.Seal(s.Cfg.LocalID, s.Cfg.LocalKey, cs.LocalStatic(), s.UseAuth)
		if err != nil {
			return err
		}
		if err := s.Rw.EncryptAndWriteMessage(&initPayload, cs.hs); err != nil {
			return err
		}
	}

	var respMsg core.ResponderMsg
	// Stage 2.0 & 2.1: Recv Remote ID & Send Confirmation
	{
		s.Stage.Set(2, 0)

		var respPayload _V1_ResponderMsg_Payload
		if err := s.Rw.ReadAndDecryptMessage(&respPayload, cs.hs); err != nil {
			return err
		}
		remoteId, err := respPayload.Verify(cs.RemoteStatic(), s.UseAuth)
		if err != nil {
			return err
		}

		if !s.FirstTime && s.HSOpt.RemoteID != remoteId {
			return ErrAuthFailed
		}

		respMsg = core.ResponderMsg{
			ResponderID:      remoteId,
			HandshakeInfo:    s.HandshakeInfo,
			ReplyToInitiator: respPayload.ReplyToInitiator,
		}

		if err := s.Model.HandleResponderMsg(respMsg); err != nil {
			return err
		}

		s.Stage.Set(2, 1)
		if err := s.Rw.WriteMessage(nil); err != nil {
			return err
		}
	}

	if err := s.Model.HandleNegotiatedPeer(core.NegotiatedResponder(respMsg)); err != nil {
		return err
	}
	s.RemoteID = respMsg.ResponderID
	return nil
}

var _ _Protocol = _ProtocolV1{}

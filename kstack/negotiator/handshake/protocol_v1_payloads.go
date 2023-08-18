package handshake

import (
	kc "kstack/crypto"
	"kstack/negotiator/core"
	"kstack/peer"
	"time"
)

//go:generate msgp -unexported -v -io=false

//msgp:tuple _V1_PeerInfo_Payload
type _V1_PeerInfo_Payload struct {
	PeerID []byte
	Sig    []byte
}

func (p *_V1_PeerInfo_Payload) Seal(id peer.ID, signer kc.PrivKey, static []byte, useAuth bool) error {
	p.PeerID = []byte(id)
	if !useAuth {
		return nil
	}
	toSign := make([]byte, len(id)+len(static))
	copy(toSign[:], id)
	copy(toSign[len(id):], static)
	sig, err := signer.Sign(toSign)
	if err != nil {
		return err
	}
	p.Sig = sig
	return nil
}

func (p *_V1_PeerInfo_Payload) Verify(remoteStatic []byte, useAuth bool) (peer.ID, error) {
	id, err := peer.IDFromBytes(p.PeerID)
	if err != nil {
		return "", err
	}
	if !useAuth {
		return id, nil
	}
	pubkey, err := id.ExtractPublicKey()
	if err != nil {
		return "", err
	}
	toSign := make([]byte, len(id)+len(remoteStatic))
	copy(toSign[:], id[:])
	copy(toSign[len(id):], remoteStatic)
	ok, err := pubkey.Verify(toSign, p.Sig)
	if !ok {
		return "", ErrAuthFailed
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

//msgp:tuple _V1_InitiatorMsg_Payload
type _V1_InitiatorMsg_Payload struct {
	_V1_PeerInfo_Payload
	Expiration time.Time
}

//msgp:tuple _V1_ResponderMsg_Payload
type _V1_ResponderMsg_Payload struct {
	_V1_PeerInfo_Payload
	core.ReplyToInitiator
}

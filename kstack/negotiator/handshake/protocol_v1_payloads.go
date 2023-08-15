package handshake

import (
	kc "kstack/crypto"
	"kstack/peer"
)

//go:generate msgp -unexported -v -io=false

//msgp:tuple _V1_Payload
type _V1_Payload struct {
	PeerID []byte
	Sig    []byte
}

func (p *_V1_Payload) Seal(id peer.ID, signer kc.PrivKey, static []byte) error {
	p.PeerID = []byte(id)
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

func (p *_V1_Payload) Verify(remoteStatic []byte) (peer.ID, error) {
	id, err := peer.IDFromBytes(p.PeerID)
	if err != nil {
		return "", err
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

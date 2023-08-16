//go:build test

package mock

import (
	kc "kstack/crypto"
	"kstack/peer"
	mrand "math/rand"
)

type Party struct {
	ID   peer.ID
	Priv kc.PrivKey
}

func NewParty(seed int64) Party {
	rng := mrand.New(mrand.NewSource(seed))
	priv, pub, _ := kc.GenerateEd25519Key(rng)
	id, _ := peer.IDFromPublicKey(pub)
	return Party{id, priv}
}

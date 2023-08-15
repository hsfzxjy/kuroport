package core

import (
	kc "kstack/crypto"
	"kstack/peer"
)

type Config struct {
	LocalID  peer.ID
	LocalKey kc.PrivKey
}

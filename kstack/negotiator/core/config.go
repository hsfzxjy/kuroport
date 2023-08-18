package core

import (
	kc "kstack/crypto"
	"kstack/peer"
	sec "kstack/security"
)

type Config struct {
	LocalID  peer.ID
	LocalKey kc.PrivKey
	sec.SecCap
	sec.AuthCap
}

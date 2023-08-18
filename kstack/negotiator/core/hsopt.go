package core

import (
	"kstack/peer"
	sec "kstack/security"
	"time"
)

type HSOpt struct {
	PassCode   PassCode
	RemoteID   peer.ID
	SecLevel   sec.SecLevel
	AuthLevel  sec.AuthLevel
	Expiration time.Time
}

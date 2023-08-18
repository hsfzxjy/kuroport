package core

import (
	"kstack/peer"
	"time"
)

/*
Initiator                  Responder
    |            <-            |
	|     -> InitiatorMsg      |
    |                   HandleInitiatorMsg
    |     <- ResponderMsg      |
HandleResponderMsg             |
    |     -> OK                |
HandleNegotiatedPeer HandleNegotiatedPeer
*/

//go:generate msgp

//msgp:ignore HandshakeInfo
type HandshakeInfo struct {
	UseEncryption bool
	UseAuth       bool
	FirstTime     bool
}

//msgp:ignore InitiatorMsg
type InitiatorMsg struct {
	InitiatorID peer.ID
	HandshakeInfo
	Expiration time.Time
}

//msgp:tuple ReplyToInitiator
type ReplyToInitiator struct {
	CalibratedExpiration time.Time
}

//msgp:ignore ResponderMsg
type ResponderMsg struct {
	ResponderID peer.ID
	HandshakeInfo
	ReplyToInitiator
}

//msgp:ignore NegotiatedPeer
type NegotiatedPeer struct {
	ID peer.ID
	HandshakeInfo
	Expiration time.Time
}

func NegotiatedInitiator(m InitiatorMsg, r ReplyToInitiator) NegotiatedPeer {
	return NegotiatedPeer{
		ID:            m.InitiatorID,
		HandshakeInfo: m.HandshakeInfo,
		Expiration:    r.CalibratedExpiration,
	}
}

func NegotiatedResponder(m ResponderMsg) NegotiatedPeer {
	return NegotiatedPeer{
		ID:            m.ResponderID,
		HandshakeInfo: m.HandshakeInfo,
		Expiration:    m.CalibratedExpiration,
	}
}

type IStore interface {
	GetPassCode() PassCode
	HandleInitiatorMsg(m InitiatorMsg) (r ReplyToInitiator, err error)
	HandleResponderMsg(m ResponderMsg) (err error)
	HandleNegotiatedPeer(p NegotiatedPeer) error
}

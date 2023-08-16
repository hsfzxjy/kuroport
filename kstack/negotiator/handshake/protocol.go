package handshake

type _Protocol interface {
	HandleInitiator(s *_Session) error
	HandleInitiatorCleartext(s *_Session) error
	HandleResponder(s *_Session) error
	HandleResponderCleartext(s *_Session) error
}

var protocols = map[byte]_Protocol{
	0x01: _ProtocolV1{},
}

// newer comes first
var protocolVersions = [...]byte{0x01}

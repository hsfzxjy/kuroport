package handshake

import (
	"errors"
	"fmt"
)

type _Error struct {
	Version   _Version
	Initiator bool
	Stage     _Stage
	Wrapped   error
}

func (e _Error) Error() string {
	var role = "responder"
	if e.Initiator {
		role = "initiator"
	}
	return fmt.Sprintf("handshake: %s, version=%02x, stage=%d.%d, role=%s", e.Wrapped, e.Version, e.Stage[0], e.Stage[1], role)
}

func (e _Error) Unwrap() error { return e.Wrapped }

var (
	ErrAuthFailed         = errors.New("authentication failed")
	ErrUnsupportedVersion = errors.New("unsupported version")
	ErrBadFormat          = errors.New("bad format")
	ErrBadOption          = errors.New("bad option")
)

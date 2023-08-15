package peer

import (
	"errors"
	"strconv"

	"github.com/google/uuid"
)

const anonFlag byte = 0xFF
const anonIDLen = len(uuid.UUID{}) + 1

var ErrBadAnon = errors.New("anonymous ID should have exactly " + strconv.Itoa(anonIDLen) + " bytes")

// IDFromUUID returns the Peer ID corresponding to the given UUIDv4.
func IDFromUUID(uid uuid.UUID) ID {
	b := make([]byte, anonIDLen)
	b[0] = anonFlag
	copy(b[1:], uid[:])
	return ID(b)
}

func (id ID) IsAnonymous() bool {
	return len(id) == anonIDLen && id[0] == anonFlag
}

func (id ID) validateAnonymous() (bool, error) {
	if id[0] == anonFlag {
		if len(id) != anonIDLen {
			return false, ErrBadAnon
		}
		return true, nil
	}
	return false, nil
}

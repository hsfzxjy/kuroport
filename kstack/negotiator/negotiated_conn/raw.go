package nc

import (
	"kstack"
	"kstack/negotiator/handshake"
	"kstack/negotiator/rw"
	"kstack/peer"
)

type _RawConn struct {
	rw.RW
	remoteID peer.ID
}

func _NewRawConn(r *handshake.Result) kstack.IConn {
	return &_RawConn{r.RW, r.RemoteID}
}

func (c *_RawConn) RemoteID() peer.ID {
	return c.remoteID
}

package nc

import (
	"kstack"
	"kstack/negotiator/handshake"
)

func New(r *handshake.Result) kstack.IConn {
	if r.UseEncryption {
		return _NewSecureConn(r)
	}
	panic("unimplemented")
}

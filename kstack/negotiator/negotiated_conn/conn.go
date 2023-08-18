package nc

import (
	"kstack"
	"kstack/negotiator/handshake"
)

func New(r *handshake.Result) kstack.IConn {
	if r.IsEncrypted() {
		return _NewSecureConn(r)
	}
	return _NewRawConn(r)
}

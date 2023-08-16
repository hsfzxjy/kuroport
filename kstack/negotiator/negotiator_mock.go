//go:build test

package nego

import (
	"kstack"
	"kstack/negotiator/handshake"
	nc "kstack/negotiator/negotiated_conn"
)

func M(conn kstack.IConn, opts ...handshake.MOpt) (kstack.IConn, error) {
	r, err := handshake.M(conn, handshake.NormalRun, opts...)
	if err != nil {
		return nil, err
	}
	return nc.New(&r), nil
}

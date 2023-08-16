//go:build test

package ktest

import (
	"net"
	"sync"
)

type ConnBuilder[T any] func(c net.Conn, initiator bool) T

func TcpPair[T any](builder ConnBuilder[T]) (initiator T, responder T) {
	l, _ := net.Listen("tcp4", ":0")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer l.Close()
		c, _ := l.Accept()
		responder = builder(c, false)
	}()
	c, _ := net.Dial("tcp4", l.Addr().String())
	initiator = builder(c, true)
	wg.Wait()
	return
}

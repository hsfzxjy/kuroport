package kstack

type Option struct {
	RemoteConns chan<- IConn
}

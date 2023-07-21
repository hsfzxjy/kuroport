package kstack

// An ITransport is an abstraction over a communication traWnsport in real world.
// e.g., a network socket, a classic bluetooth socket, etc.
type ITransport interface {
	IRawTransport
	Family() Family
	Addr() IAddr
	DiedCh() <-chan struct{}
}

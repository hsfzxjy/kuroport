package kstack

import "context"

func (s *Stack) DialAddr(ctx context.Context, addr IAddr, failFast bool) (IConn, error) {
	impl := s.getImpl(addr.Family())
	if impl == nil || impl.Dialer == nil {
		return nil, ErrBadAddress
	}
	tr, err := impl.trManager.Dial(ctx, addr, failFast)
	if err != nil {
		return nil, err
	}
	return tr.Open(impl)
}

func (s *Stack) DialDevice(ctx context.Context, d IDevice) (IConn, error) {
	panic("TODO")
}

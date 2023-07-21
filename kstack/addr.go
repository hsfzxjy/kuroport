package kstack

import "errors"

type IAddr interface {
	Family() Family
	String() string
	Hash() Hash
	ResolveDevice() (IDevice, error)
}

var ErrBadAddress = errors.New("bad kstack address")

type AddrProvider interface {
	Addr() IAddr
	Family() Family
}

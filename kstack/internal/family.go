package internal

type Family int

const (
	BREDR Family = iota
	BLE
	INet
	TestFamily
	MaxFamily
)

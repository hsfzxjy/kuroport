package kservice

//go:generate stringer -type=State
type State int

const (
	Stopped State = iota + 1
	Starting
	Started
	Stopping
)

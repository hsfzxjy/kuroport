package kservice

type Event struct {
	State
	Error error
}

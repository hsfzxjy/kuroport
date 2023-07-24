//go:build !test

package tracer

const Enabled = false

func doWait(t _HasDeadline) {}

func doExpect(t _Tracer) chainer { return chainer{} }

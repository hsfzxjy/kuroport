//go:build test

package tracer

const Enabled = true

func init() {
	T = new(_Tracer)
}

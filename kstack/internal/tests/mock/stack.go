package mock

import (
	"kstack"
	"kstack/internal"
	ku "kutil"
)

type disposableStack struct {
	*kstack.Stack
	Disposer ku.F
}

type stackBuilder struct {
	s     *disposableStack
	start ku.F
}

func (b stackBuilder) Impl(impl kstack.Impl) stackBuilder {
	impl.Family = internal.TestFamily
	b.s.Register(impl)
	if impl.Listener != nil {
		b.start = b.start.With(func() { impl.Listener.Start() })
		b.s.Disposer = b.s.Disposer.With(func() { impl.Listener.Stop() })
	}
	return b
}

func (b stackBuilder) Build() *disposableStack {
	b.s.Run()
	b.start.Do()
	return b.s
}

func Stack(option kstack.Option) stackBuilder {
	return stackBuilder{&disposableStack{
		kstack.New(option), nil,
	}, nil}
}

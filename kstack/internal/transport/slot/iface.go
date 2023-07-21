package slot

import (
	"kstack/internal"
	"kstack/internal/transport/slot/muxed"
	"kstack/internal/transport/slot/not_muxed"
	ku "kutil"
)

type ISlot interface {
	Track(itr internal.ITransport, isRemote bool) (internal.TrackedTransport, error)
	DialAndTrack(
		dialF ku.Awaiter[internal.ITransport],
		failFast bool,
	) ku.Awaiter[internal.TrackedTransport]
	TryDispose()
}

func New(impl internal.Impl, disposeSelf ku.F) ISlot {
	if impl.ImplOption().Mux {
		return muxed.New(impl, disposeSelf)
	} else {
		return not_muxed.New(impl, disposeSelf)
	}
}

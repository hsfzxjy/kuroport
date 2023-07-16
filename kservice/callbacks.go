package kservice

import "context"

type _ICallbacks interface {
	OnServiceStart(ctx context.Context) (err error)
	OnServiceRun(ctx context.Context) (err error)
	OnServiceStop()
}

package kservice

import (
	"context"
	"time"
)

func runWithCancelTimeout(
	f func(context.Context) error,
	afterF func(wasCanceled bool, err error),
) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := f(ctx)
		wasCanceled := false
		select {
		case <-ctx.Done():
			wasCanceled = true
		default:
		}
		afterF(wasCanceled, err)
	}()

	go func() {
		select {
		case <-done:
			return
		case <-ctx.Done():
		}
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		select {
		case <-timer.C:
			panic("service takes > 10s to stop")
		case <-done:
		}
	}()

	return cancel
}

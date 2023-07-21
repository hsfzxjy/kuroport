package ku

import "sync"

type Awaiter[T any] func() (T, error)

func Resolve[T any](value T) Awaiter[T] {
	return func() (T, error) { return value, nil }
}

func Reject[T any](err error) Awaiter[T] {
	return func() (T, error) {
		var emptyT T
		return emptyT, err
	}
}

type Call[R any] struct {
	result R
	err    error

	wg   sync.WaitGroup
	once sync.Once
}

func NewCall[R any]() *Call[R] {
	c := &Call[R]{}
	c.wg.Add(1)
	return c
}

func (c *Call[R]) DoOrGetAsync(f Awaiter[R]) Awaiter[R] {
	return func() (R, error) { return c.DoOrGet(f) }
}

func (c *Call[R]) DoOrGet(f Awaiter[R]) (R, error) {
	c.once.Do(func() {
		defer c.wg.Done()
		c.result, c.err = f()
	})
	return c.WaitAndGet()
}

func (c *Call[R]) WaitAndGet() (R, error) {
	c.wg.Wait()
	return c.result, c.err
}

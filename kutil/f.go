package ku

type F func()

func (f F) Do() {
	if f != nil {
		f()
	}
}

func (f F) With(f2 F) F {
	if f2 == nil {
		return f
	}
	if f == nil {
		return f2
	}
	return func() {
		f()
		f2()
	}
}

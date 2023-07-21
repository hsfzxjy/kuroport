package ku

type IsZeroer interface {
	IsZero() bool
}

type List[E IsZeroer] struct {
	nActive   int
	container []E
}

func (l *List[E]) Get() (item E, ok bool) {
	if l.nActive == 0 {
		return
	}
	for _, item := range l.container {
		if !item.IsZero() {
			return item, true
		}
	}
	return
}

func (l *List[E]) findUsableIndex() int {
	if l.nActive == len(l.container) {
		var emptyE E
		l.container = append(l.container, emptyE)
		return l.nActive
	} else {
		for i, item := range l.container {
			if item.IsZero() {
				return i
			}
		}
		panic("unreachable")
	}
}

func (l *List[E]) Add(value E) (index int) {
	index = l.findUsableIndex()
	l.container[index] = value
	l.nActive++
	return index
}

func (l *List[E]) AddFunc(valueF func(i int) E) E {
	index := l.findUsableIndex()
	value := valueF(index)
	l.container[index] = value
	l.nActive++
	return value
}

func (l *List[E]) Delete(index int) {
	var emptyE E
	l.container[index] = emptyE
	l.nActive--
	if l.nActive == 0 {
		l.container = nil
	}
}

func (l *List[E]) IsEmpty() bool {
	return l.nActive == 0
}

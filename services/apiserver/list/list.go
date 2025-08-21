package list

import container "container/list"

type List[T any] struct {
	list *container.List
}

func New[T any]() List[T] {
	return List[T]{
		list: container.New(),
	}
}

func (l List[T]) Len() int {
	return l.list.Len()
}

func (l List[T]) Push(v T) {
	l.list.PushBack(v)
}

func (l List[T]) Pop() (T, bool) {
	v := l.list.Back()
	if v != nil {
		val := l.list.Remove(v)
		return val.(T), true
	}
	var zero T
	return zero, false
}

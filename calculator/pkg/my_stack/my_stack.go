package my_stack

type MyStack[T any] struct {
	container []T
	size      int
}

func (s *MyStack[T]) Push(val T) {
	if s.size < len(s.container) {
		s.container[s.size] = val
		s.size++
	} else {
		s.container = append(s.container, val)
		s.size++
	}
}

func (s *MyStack[T]) Pop() T {
	s.size--
	return s.container[s.size]
}

func (s *MyStack[T]) Size() int {
	return s.size
}

func NewMyStack[T any]() *MyStack[T] {
	return &MyStack[T]{container: make([]T, 10), size: 0}
}

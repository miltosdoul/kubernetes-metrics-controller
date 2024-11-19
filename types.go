package main

type empty struct{}

// type t interface{}
type set[T comparable] map[T]empty

func (s set[T]) has(item T) bool {
	_, exists := s[item]
	return exists
}

func (s set[T]) insert(item T) {
	s[item] = empty{}
}

func (s set[T]) delete(item T) {
	delete(s, item)
}

func (s set[T]) len() int {
	return len(s)
}

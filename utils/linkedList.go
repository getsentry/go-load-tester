package utils

import (
	"errors"
)

type DoubleLinkedList[T any] struct {
	first *node[T]
	last  *node[T]
}

type node[T any] struct {
	val  T
	prev *node[T]
	next *node[T]
}

func (dll *DoubleLinkedList[T]) PushFront(val T) {
	if dll == nil {
		panic("pushing into a nil double linked list")
	}
	n := new(node[T])

	n.val = val
	n.prev = nil
	n.next = dll.first
	if dll.first != nil {
		dll.first.prev = n
	}
	dll.first = n

	if dll.last == nil {
		dll.last = n // first element set it as last as well
	}
}

func (dll *DoubleLinkedList[T]) PushBack(val T) {
	if dll == nil {
		panic("pushing into a nil double linked list")
	}

	n := new(node[T])

	n.val = val
	n.prev = dll.last
	n.next = nil

	if dll.last != nil {
		dll.last.next = n
	}
	dll.last = n

	if dll.first == nil {
		dll.first = n //first element set it as first as well
	}

}

func (dll *DoubleLinkedList[T]) PeekFront() *T {
	if dll == nil {
		panic("accessing a nil double linked list")
	}

	if dll.first == nil {
		return nil
	}
	return &dll.first.val
}

func (dll *DoubleLinkedList[T]) PopFront() (*T, error) {
	if dll == nil {
		panic("popping from a nil double linked list")
	}

	if dll.first == nil {
		return nil, errors.New("trying to pop an empty list")
	}

	temp := dll.first

	dll.first = temp.next

	if dll.last == temp {
		dll.last = nil // the last element also clean last
	}

	return &temp.val, nil
}

func (dll *DoubleLinkedList[T]) PopBack() (*T, error) {
	if dll == nil {
		panic("popping from a nil double linked list")
	}

	if dll.last == nil {
		return nil, errors.New("trying to pop an empty list")
	}

	temp := dll.last

	dll.last = temp.prev

	if dll.first == temp {
		dll.first = nil // the last element also clean first
	}

	return &temp.val, nil
}

func (dll *DoubleLinkedList[T]) PeekBack() *T {
	if dll == nil {
		panic("accessing a nil double linked list")
	}

	if dll.last == nil {
		return nil
	}
	return &dll.last.val
}

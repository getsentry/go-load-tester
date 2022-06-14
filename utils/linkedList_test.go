package utils

import (
	"fmt"
	"testing"
)

func TestOneElementIntLinkedList(t *testing.T) {

	var l = new(DoubleLinkedList[int])

	assertEmptyList(t, l, "new list")

	// pushFront & popFront
	l.PushFront(1)

	assertListWithOneElement(t, l, 1, "pushing at front")

	val, err := l.PopFront()

	if err != nil {
		t.Error(fmt.Errorf("Failed to popFront non empty list, after pushFront:\n%v", err))
	}

	if val == nil || *val != 1 {
		t.Errorf("Failed to popFront non empty list after pushFront, bad value:\n%v", val)
	}

	assertEmptyList(t, l, "popped from front")

	// pushFront & popBack
	l.PushFront(2)

	assertListWithOneElement(t, l, 2, "pushing at front")

	val, err = l.PopBack()

	if err != nil {
		t.Error(fmt.Errorf("Failed to popBack non empty list, after pushFront:\n%v", err))
	}

	if val == nil || *val != 2 {
		t.Errorf("Failed to popBack non empty list after pushFront, bad value:\n%v", val)
	}
	assertEmptyList(t, l, "popped from back")

	// pushBack & popFront
	l.PushBack(3)

	assertListWithOneElement(t, l, 3, "pushing at back")

	val, err = l.PopFront()

	if err != nil {
		t.Error(fmt.Errorf("Failed to popFront non empty list, after pushBack:\n%v", err))
	}

	if val == nil || *val != 3 {
		t.Errorf("Failed to popFront non empty list after pushBack, bad value:\n%v", val)
	}

	assertEmptyList(t, l, "popped from front after pushBack")

	// pushBack & popBack
	l.PushBack(4)

	assertListWithOneElement(t, l, 4, "pushing at Back")

	val, err = l.PopBack()

	if err != nil {
		t.Error(fmt.Errorf("Failed to popBack non empty list, after pushBack:\n%v", err))
	}

	if val == nil || *val != 4 {
		t.Errorf("Failed to popBack non empty list after pushBack, bad value:\n%v", val)
	}
	assertEmptyList(t, l, "popped from back after pushBack")
}

func TestPushingFromBackAndPullingAtFront(t *testing.T) {
	var l = DoubleLinkedList[int]{}

	l.PushBack(1)
	l.PushBack(2)
	l.PushBack(3)
	assertContents(t, []int{1, 2, 3}, l.PopFront)
}

func TestPushingFromBackAndPullingAtBack(t *testing.T) {
	var l = DoubleLinkedList[int]{}

	l.PushBack(1)
	l.PushBack(2)
	l.PushBack(3)
	assertContents(t, []int{3, 2, 1}, l.PopBack)
}

func TestPushingFromFrontAndPullingAtFront(t *testing.T) {
	var l = DoubleLinkedList[int]{}

	l.PushFront(1)
	l.PushFront(2)
	l.PushFront(3)
	assertContents(t, []int{3, 2, 1}, l.PopFront)
}

func TestPushingFromFrontAndPullingAtBack(t *testing.T) {
	var l = DoubleLinkedList[int]{}

	l.PushFront(1)
	l.PushFront(2)
	l.PushFront(3)
	assertContents(t, []int{1, 2, 3}, l.PopBack)
}

func TestPushingFromBothSidesAndPullingFront(t *testing.T) {
	var l = DoubleLinkedList[int]{}

	l.PushFront(1)
	l.PushBack(2)
	l.PushFront(3)
	l.PushBack(4)
	l.PushFront(5)
	assertContents(t, []int{5, 3, 1, 2, 4}, l.PopFront)
}

func TestPushingFromBothSidesAndPullingBack(t *testing.T) {
	var l = DoubleLinkedList[int]{}

	l.PushFront(1)
	l.PushBack(2)
	l.PushFront(3)
	l.PushBack(4)
	l.PushFront(5)
	assertContents(t, []int{4, 2, 1, 3, 5}, l.PopBack)
}

func assertEmptyList[T any](t *testing.T, l *DoubleLinkedList[T], auxMessage string) {
	if l.PeekFront() != nil {
		t.Errorf("Empty list returns data at Front (%s)", auxMessage)
	}
	if l.PeekBack() != nil {
		t.Errorf("Empty list returns data at Back (%s)", auxMessage)
	}
}

func assertListWithOneElement[T comparable](t *testing.T, l *DoubleLinkedList[T], expected T, auxMessage string) {

	val := l.PeekFront()
	if val == nil {
		t.Errorf("Non empty list returns nil Front (%s)", auxMessage)
	}
	if *val != expected {
		t.Errorf("Garbage returned by PeekFront (%s)", auxMessage)
	}

	val = l.PeekBack()
	if val == nil {
		t.Errorf("Non empty list returns nil Back (%s)", auxMessage)
	}
	if *val != expected {
		t.Errorf("Garbage returned by PeekBack (%s)", auxMessage)
	}

}

func assertContents[T comparable](t *testing.T, expected []T, extractor func() (*T, error)) {
	for _, val := range expected {
		actual, err := extractor()
		if err != nil {
			t.Errorf("assertContents failed, expected: %v got error:\n%v", val, err)
			return
		}
		if *actual != val {
			t.Errorf("assertContents failed, expected: %v got: %v", val, *actual)
			return
		}
	}
	//make sure it is empty
	actual, err := extractor()
	if err == nil {
		t.Errorf("Expected empty contenst got some stuff: %v", *actual)
	}
}

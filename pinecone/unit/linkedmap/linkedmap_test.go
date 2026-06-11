package linkedmap

import (
	"reflect"
	"testing"
)

func TestPutKeepsInsertionOrder(t *testing.T) {
	m := New[string, int]()

	if !m.Put("a", 1) {
		t.Fatal("first put should insert")
	}
	m.Put("b", 2)
	m.Put("c", 3)

	if got := m.Keys(); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("keys = %#v", got)
	}
	if got := m.Values(); !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Fatalf("values = %#v", got)
	}
}

func TestPutExistingUpdatesWithoutMoving(t *testing.T) {
	m := New[string, int]()
	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("c", 3)

	if m.Put("b", 20) {
		t.Fatal("existing put should not insert")
	}

	if got := m.Keys(); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("keys = %#v", got)
	}
	if got, ok := m.Get("b"); !ok || got != 20 {
		t.Fatalf("b = %d, %v", got, ok)
	}
}

func TestHasOwnProperty(t *testing.T) {
	m := New[string, int]()
	m.Put("a", 1)

	if !m.HasOwnProperty("a") {
		t.Fatal("a should exist")
	}
	if m.HasOwnProperty("b") {
		t.Fatal("b should not exist")
	}
	if !m.Has("a") {
		t.Fatal("Has should alias HasOwnProperty")
	}
}

func TestDeleteHeadMiddleTail(t *testing.T) {
	m := New[string, int]()
	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("c", 3)
	m.Put("d", 4)

	if !m.Delete("a") {
		t.Fatal("delete head")
	}
	if !m.Delete("c") {
		t.Fatal("delete middle")
	}
	if !m.Delete("d") {
		t.Fatal("delete tail")
	}
	if m.Delete("missing") {
		t.Fatal("delete missing should be false")
	}

	if got := m.Keys(); !reflect.DeepEqual(got, []string{"b"}) {
		t.Fatalf("keys = %#v", got)
	}
	if key, value, ok := m.Front(); !ok || key != "b" || value != 2 {
		t.Fatalf("front = %q, %d, %v", key, value, ok)
	}
	if key, value, ok := m.Back(); !ok || key != "b" || value != 2 {
		t.Fatalf("back = %q, %d, %v", key, value, ok)
	}
}

func TestRangeCanStop(t *testing.T) {
	m := New[string, int]()
	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("c", 3)

	var keys []string
	m.Range(func(key string, value int) bool {
		keys = append(keys, key)
		return key != "b"
	})

	if !reflect.DeepEqual(keys, []string{"a", "b"}) {
		t.Fatalf("range keys = %#v", keys)
	}
}

func TestHashRangeCanStop(t *testing.T) {
	m := New[string, int]()
	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("c", 3)

	count := 0
	m.HashRange(func(key string, value int) bool {
		count++
		return false
	})

	if count != 1 {
		t.Fatalf("hash range count = %d", count)
	}
}

func TestToHMapCopiesValues(t *testing.T) {
	m := New[string, int]()
	m.Put("a", 1)
	m.Put("b", 2)

	copied := m.ToHMap()
	if !reflect.DeepEqual(copied, map[string]int{"a": 1, "b": 2}) {
		t.Fatalf("copied = %#v", copied)
	}

	copied["a"] = 10
	if got, ok := m.Get("a"); !ok || got != 1 {
		t.Fatalf("source value = %d, %v", got, ok)
	}
}

func TestMoveToFrontAndBack(t *testing.T) {
	m := New[string, int]()
	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("c", 3)

	if !m.MoveToFront("c") {
		t.Fatal("move c to front")
	}
	if got := m.Keys(); !reflect.DeepEqual(got, []string{"c", "a", "b"}) {
		t.Fatalf("keys after front = %#v", got)
	}

	if !m.MoveToBack("c") {
		t.Fatal("move c to back")
	}
	if got := m.Keys(); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("keys after back = %#v", got)
	}

	if m.MoveToFront("missing") {
		t.Fatal("move missing should be false")
	}
}

func TestClearAndReuse(t *testing.T) {
	m := NewWithCapacity[string, int](4)
	m.Put("a", 1)
	m.Put("b", 2)

	m.Clear()
	if !m.Empty() {
		t.Fatal("map should be empty after clear")
	}
	if _, _, ok := m.Front(); ok {
		t.Fatal("front should be empty after clear")
	}

	m.Put("c", 3)
	if got := m.Keys(); !reflect.DeepEqual(got, []string{"c"}) {
		t.Fatalf("keys after reuse = %#v", got)
	}
}

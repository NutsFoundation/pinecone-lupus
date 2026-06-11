package linkedmap

type node[K comparable, V any] struct {
	key   K
	value V
	prev  *node[K, V]
	next  *node[K, V]
}

type Map[K comparable, V any] struct {
	index map[K]*node[K, V]
	head  *node[K, V]
	tail  *node[K, V]
}

func New[K comparable, V any]() *Map[K, V] {
	return NewWithCapacity[K, V](0)
}

func NewWithCapacity[K comparable, V any](capacity int) *Map[K, V] {
	if capacity < 0 {
		capacity = 0
	}
	return &Map[K, V]{
		index: make(map[K]*node[K, V], capacity),
	}
}

func (m *Map[K, V]) Len() int {
	if m == nil {
		return 0
	}
	return len(m.index)
}

func (m *Map[K, V]) Empty() bool {
	return m.Len() == 0
}

func (m *Map[K, V]) Has(key K) bool {
	return m.HasOwnProperty(key)
}

func (m *Map[K, V]) HasOwnProperty(key K) bool {
	if m == nil {
		return false
	}
	_, ok := m.index[key]
	return ok
}

func (m *Map[K, V]) Get(key K) (V, bool) {
	if m == nil {
		var zero V
		return zero, false
	}
	n, ok := m.index[key]
	if !ok {
		var zero V
		return zero, false
	}
	return n.value, true
}

func (m *Map[K, V]) Put(key K, value V) bool {
	if m.index == nil {
		m.index = make(map[K]*node[K, V])
	}
	if n, ok := m.index[key]; ok {
		n.value = value
		return false
	}

	n := &node[K, V]{
		key:   key,
		value: value,
	}
	m.index[key] = n
	m.pushBack(n)
	return true
}

func (m *Map[K, V]) Delete(key K) bool {
	if m == nil {
		return false
	}
	n, ok := m.index[key]
	if !ok {
		return false
	}
	delete(m.index, key)
	m.unlink(n)
	return true
}

func (m *Map[K, V]) Clear() {
	if m == nil {
		return
	}
	m.index = make(map[K]*node[K, V])
	m.head = nil
	m.tail = nil
}

func (m *Map[K, V]) Front() (K, V, bool) {
	if m == nil || m.head == nil {
		var key K
		var value V
		return key, value, false
	}
	return m.head.key, m.head.value, true
}

func (m *Map[K, V]) Back() (K, V, bool) {
	if m == nil || m.tail == nil {
		var key K
		var value V
		return key, value, false
	}
	return m.tail.key, m.tail.value, true
}

func (m *Map[K, V]) Keys() []K {
	if m == nil {
		return nil
	}
	keys := make([]K, 0, len(m.index))
	for n := m.head; n != nil; n = n.next {
		keys = append(keys, n.key)
	}
	return keys
}

func (m *Map[K, V]) Values() []V {
	if m == nil {
		return nil
	}
	values := make([]V, 0, len(m.index))
	for n := m.head; n != nil; n = n.next {
		values = append(values, n.value)
	}
	return values
}

func (m *Map[K, V]) ToHMap() map[K]V {
	if m == nil {
		return nil
	}
	result := make(map[K]V, len(m.index))
	for key, n := range m.index {
		result[key] = n.value
	}
	return result
}

func (m *Map[K, V]) Range(consumer func(key K, value V) bool) {
	if m == nil || consumer == nil {
		return
	}
	for n := m.head; n != nil; n = n.next {
		if !consumer(n.key, n.value) {
			return
		}
	}
}

func (m *Map[K, V]) HashRange(consumer func(key K, value V) bool) {
	if m == nil || consumer == nil {
		return
	}
	for key, n := range m.index {
		if !consumer(key, n.value) {
			return
		}
	}
}

func (m *Map[K, V]) MoveToFront(key K) bool {
	if m == nil {
		return false
	}
	n, ok := m.index[key]
	if !ok {
		return false
	}
	if n == m.head {
		return true
	}
	m.unlink(n)
	m.pushFront(n)
	return true
}

func (m *Map[K, V]) MoveToBack(key K) bool {
	if m == nil {
		return false
	}
	n, ok := m.index[key]
	if !ok {
		return false
	}
	if n == m.tail {
		return true
	}
	m.unlink(n)
	m.pushBack(n)
	return true
}

func (m *Map[K, V]) pushFront(n *node[K, V]) {
	n.prev = nil
	n.next = m.head
	if m.head == nil {
		m.tail = n
	} else {
		m.head.prev = n
	}
	m.head = n
}

func (m *Map[K, V]) pushBack(n *node[K, V]) {
	n.prev = m.tail
	n.next = nil
	if m.tail == nil {
		m.head = n
	} else {
		m.tail.next = n
	}
	m.tail = n
}

func (m *Map[K, V]) unlink(n *node[K, V]) {
	if n.prev == nil {
		m.head = n.next
	} else {
		n.prev.next = n.next
	}

	if n.next == nil {
		m.tail = n.prev
	} else {
		n.next.prev = n.prev
	}

	n.prev = nil
	n.next = nil
}

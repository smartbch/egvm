package extension

import (
	"strings"

	"github.com/dop251/goja"
	"github.com/tinylib/msgp/msgp"
	"modernc.org/b/v2"
)

type OrderedStrMapIter struct {
	e *b.Enumerator[string, string]
}

func (iter OrderedStrMapIter) Close() {
	iter.e.Close()
}

func (iter OrderedStrMapIter) Next() (string, string) {
	k, v, err := iter.e.Next()
	if err != nil {
		return "", ""
	}
	return k, v
}

func (iter OrderedStrMapIter) Prev() (string, string) {
	k, v, err := iter.e.Prev()
	if err != nil {
		return "", ""
	}
	return k, v
}

type OrderedStrMap struct {
	estimatedSize int
	tree          *b.Tree[string, string]
}

func NewOrderedStrMap() OrderedStrMap {
	return OrderedStrMap{tree: b.TreeNew[string, string](func(a, b string) int {
		return strings.Compare(a, b)
	})}
}

func (m *OrderedStrMap) loadFrom(b []byte) ([]byte, error) {
	m.tree.Clear()
	initSize := len(b)
	count, b, err := msgp.ReadIntBytes(b)
	if err != nil {
		return nil, err
	}
	for i := 0; i < count; i++ {
		k, v := "", ""
		k, b, err = msgp.ReadStringBytes(b)
		if err != nil {
			return nil, err
		}
		v, b, err = msgp.ReadStringBytes(b)
		if err != nil {
			return nil, err
		}
		m.tree.Set(k, v)
	}
	m.estimatedSize = initSize - len(b)
	return b, nil
}

func (m *OrderedStrMap) dumpTo(b []byte) []byte {
	b = msgp.AppendInt(b, m.tree.Len())
	if m.tree.Len() == 0 {
		return b
	}
	e, _ := m.tree.SeekFirst()
	defer e.Close()
	for k, v, err := e.Next(); err == nil; k, v, err = e.Next() {
		b = msgp.AppendString(b, k)
		b = msgp.AppendString(b, v)
	}
	return b
}

func (m *OrderedStrMap) Clear() {
	m.tree.Clear()
	m.estimatedSize = 0
}

func (m *OrderedStrMap) Delete(k string) {
	existed := m.tree.Delete(k)
	if existed {
		m.estimatedSize -= len(k)
	}
}

func (m *OrderedStrMap) Get(k string) (string, bool) {
	return m.tree.Get(k)
}

func (m *OrderedStrMap) Len() int {
	return m.tree.Len()
}

func (m *OrderedStrMap) Set(k string, v string) {
	if len(k) == 0 {
		panic(goja.NewSymbol("Empty key string"))
	}
	m.tree.Put(k, func(oldV string, exists bool) (newV string, write bool) {
		if exists {
			m.estimatedSize += len(v) - len(oldV)
		} else {
			m.estimatedSize += 10 + len(k) + len(v)
		}
		return v, true
	})
}

func (m *OrderedStrMap) Seek(k string) (OrderedStrMapIter, bool) {
	e, ok := m.tree.Seek(k)
	return OrderedStrMapIter{e: e}, ok
}

func (m *OrderedStrMap) SeekFirst() (OrderedStrMapIter, error) {
	e, err := m.tree.SeekFirst()
	return OrderedStrMapIter{e: e}, err
}

func (m *OrderedStrMap) SeekLast() (OrderedStrMapIter, error) {
	e, err := m.tree.SeekLast()
	return OrderedStrMapIter{e: e}, err
}

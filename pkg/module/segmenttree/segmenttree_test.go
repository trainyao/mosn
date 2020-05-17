package segmenttree

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func Test_segmentTree(t *testing.T) {
	ns := []Node{
		{
			value:      1,
			rangeStart: 0,
			rangeEnd:   1,
		},
		{
			value:      2,
			rangeStart: 1,
			rangeEnd:   2,
		},
		{
			value:      3,
			rangeStart: 2,
			rangeEnd:   3,
		},
	}

	f := func(l, r interface{}) interface{} {
		return l.(int) + r.(int)
	}

	tree := NewTree(ns, f)
	if !reflect.DeepEqual(tree.data, []interface{}{
		nil, 6, 5, 1, 2, 3,
	}) {
		t.FailNow()
	}
	if !reflect.DeepEqual(tree.rangeStart, map[int]uint64{
		1: 0,
		2: 1,
		3: 0,
		4: 1,
		5: 2,
	}) {
		t.FailNow()
	}
	if !reflect.DeepEqual(tree.rangeEnd, map[int]uint64{
		1: 3,
		2: 3,
		3: 1,
		4: 2,
		5: 3,
	}) {
		t.FailNow()
	}

	ns = []Node{
		{
			value:      1,
			rangeStart: 0,
			rangeEnd:   1,
		},
		{
			value:      2,
			rangeStart: 1,
			rangeEnd:   2,
		},
		{
			value:      3,
			rangeStart: 2,
			rangeEnd:   3,
		},
		{
			value:      4,
			rangeStart: 3,
			rangeEnd:   4,
		},
	}

	tree = NewTree(ns, f)
	if !reflect.DeepEqual(tree.data, []interface{}{
		nil, 10, 3, 7, 1, 2, 3, 4,
	}) {
		t.FailNow()
	}
	if !reflect.DeepEqual(tree.rangeStart, map[int]uint64{
		1: 0,
		2: 0,
		3: 2,
		4: 0,
		5: 1,
		6: 2,
		7: 3,
	}) {
		t.FailNow()
	}
	if !reflect.DeepEqual(tree.rangeEnd, map[int]uint64{
		1: 4,
		2: 2,
		3: 4,
		4: 1,
		5: 2,
		6: 3,
		7: 4,
	}) {
		t.FailNow()
	}

	fmt.Sprintf("%+v", tree)
}

func Test_updateTree(t *testing.T) {
	ns := []Node{
		{
			value:      1,
			rangeStart: 0,
			rangeEnd:   1,
		},
		{
			value:      2,
			rangeStart: 1,
			rangeEnd:   2,
		},
		{
			value:      3,
			rangeStart: 2,
			rangeEnd:   3,
		},
		{
			value:      4,
			rangeStart: 3,
			rangeEnd:   4,
		},
	}

	f := func(l, r interface{}) interface{} {
		return l.(int) + r.(int)
	}

	tree := NewTree(ns, f)
	leaf := tree.Leaf(3)
	if !assert.Equalf(t, 7, leaf.index, "leaf index should be 7")  {
		t.FailNow()
	}

	leaf.value = 10
	tree.Update(leaf)

	if !reflect.DeepEqual(tree.data, []interface{}{
		nil, 16, 3, 13, 1, 2, 3, 10,
	}) {
		t.FailNow()
	}
}

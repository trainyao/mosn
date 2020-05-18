package segmenttree

type SegmentTreeUpdateFunc func(leftChildData, rightChildData interface{}) (currentNodeData interface{})

type Tree struct {
	data       []interface{}
	rangeStart map[int]uint64
	rangeEnd   map[int]uint64
	leafCount  int
	updateFunc SegmentTreeUpdateFunc
}

func (t *Tree) Leaf(index int) *Node {
	leafIndex := t.leafCount + index
	data := t.data[leafIndex]
	rangeStart := t.rangeStart[leafIndex]
	rangeEnd := t.rangeEnd[leafIndex]

	return &Node{
		Value:      data,
		index:      leafIndex,
		RangeStart: rangeStart,
		RangeEnd:   rangeEnd,
	}
}

func (t *Tree) Update(n *Node) {
	index := n.index
	// update current node
	t.data[index] = n.Value

	// find root index
	leftIndex := index
	rightIndex := index + 1
	if index%2 == 1 {
		leftIndex = index - 1
		rightIndex = index
	}
	rootIndex := leftIndex / 2

	for rootIndex > 0 {
		t.data[rootIndex] = t.updateFunc(t.data[leftIndex], t.data[rightIndex])

		leftIndex = rootIndex
		rightIndex = leftIndex + 1
		if rootIndex%2 == 1 {
			leftIndex = rootIndex - 1
			rightIndex = rootIndex
		}
		rootIndex /= 2
	}
}

type Node struct {
	Value      interface{}
	index      int
	RangeStart uint64
	RangeEnd   uint64
}

func NewTree(nodes []Node, updateFunc SegmentTreeUpdateFunc) *Tree {
	t := &Tree{
		updateFunc: updateFunc,
		leafCount:  len(nodes),
	}
	t.data, t.rangeStart, t.rangeEnd = build(nodes, updateFunc)

	return t
}

func build(nodes []Node, updateFunc SegmentTreeUpdateFunc) ([]interface{}, map[int]uint64, map[int]uint64) {
	if len(nodes) == 0 {
		return nil, nil, nil
	}
	count := len(nodes)

	data := make([]interface{}, 2*count)
	rangeStart := make(map[int]uint64)
	rangeEnd := make(map[int]uint64)

	for i := 0; i < count; i++ {
		data[count+i] = nodes[i].Value
		rangeStart[count+i] = nodes[i].RangeStart
		rangeEnd[count+i] = nodes[i].RangeEnd
	}

	n := 2*count - 1
	for {
		// [0- 23 45 67 89][1011-1213 1415 1617 1819]
		//
		leftIndex := n - 1
		rightIndex := n
		rootIndex := leftIndex / 2

		data[rootIndex] = updateFunc(data[leftIndex], data[rightIndex])
		rangeStart[rootIndex] = rangeStart[leftIndex]
		if rangeStart[rightIndex] < rangeStart[leftIndex] {
			rangeStart[rootIndex] = rangeStart[rightIndex]
		}

		rangeEnd[rootIndex] = rangeEnd[leftIndex]
		if rangeEnd[rightIndex] > rangeEnd[leftIndex] {
			rangeEnd[rootIndex] = rangeEnd[rightIndex]
		}

		//left := &Node{
		//	Value:      data[leftIndex],
		//	index:      leftIndex,
		//	RangeStart: RangeStart[leftIndex],
		//	RangeEnd:   RangeEnd[leftIndex],
		//}
		//right := &Node{
		//	Value:      data[rightIndex],
		//	index:      rightIndex,
		//	RangeStart: RangeStart[rightIndex],
		//	RangeEnd:   RangeEnd[rightIndex],
		//}
		//root := &Node{
		//	Value:      data[rootIndex],
		//	index:      rootIndex,
		//	RangeStart: RangeStart[rootIndex],
		//	RangeEnd:   RangeEnd[rootIndex],
		//}
		//
		//updateFunc(root, left, right)
		//
		//data[rootIndex] = root.Value
		//RangeStart[rootIndex] = root.RangeStart
		//RangeEnd[rootIndex] = root.RangeEnd

		n -= 2

		if n/2 == 0 {
			break
		}
	}

	return data, rangeStart, rangeEnd
}

func (t *Tree) FindParent(currentNode *Node) *Node {
	rootIndex := currentNode.index / 2
	root := &Node{
		Value:      t.data[rootIndex],
		index:      rootIndex,
		RangeStart: t.rangeStart[rootIndex],
		RangeEnd:   t.rangeEnd[rootIndex],
	}
	return root
}

func (n *Node) IsRoot() bool {
	return n.index/2 == 0
}

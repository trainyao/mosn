package segmenttree

type SegmentTreeUpdateFunc func(currentNode *Node, leftChild *Node, rightChild *Node)

type Tree struct {
	data       []uint64
	rangeStart map[int]uint64
	rangeEnd   map[int]uint64
	leafCount  int
	updateFunc SegmentTreeUpdateFunc
}

func (t *Tree) Leaf(index int) *Node {
	data := t.data[t.leafCount+index]
	rangeStart := t.rangeStart[t.leafCount+index]
	rangeEnd := t.rangeEnd[t.leafCount+index]

	return &Node{
		value:      data,
		index:      index,
		rangeStart: rangeStart,
		rangeEnd:   rangeEnd,
	}
}

type Node struct {
	value      interface{}
	index      int
	rangeStart uint64
	rangeEnd   uint64
}

func NewTree(nodes []Node, updateFunc SegmentTreeUpdateFunc) *Tree {
	return &Tree{
		data:       build(nodes),
		updateFunc: updateFunc,
	}
}

func build(nodes []Node) []uint64 {
	if len(nodes) == 0 {
		return nil
	}
	count := len(nodes)

	data := make([]uint64, 2*count)
	rangeStart := make(map[int]uint64)
	rangeEnd := make(map[int]uint64)

	for i:= 

}

func (n *Node) Parent() *Node {

}
func (n *Node) IsRoot() bool {
	return n.index == 0
}

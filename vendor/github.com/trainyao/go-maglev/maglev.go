// Package maglev implements maglev consistent hashing
/*
   http://research.google.com/pubs/pub44824.html
*/
package maglev

import (
	"github.com/dchest/siphash"
)

const (
	SmallM = 65537
	BigM   = 655373
)

type Table struct {
	n             int
	lookup        []int
	permutations  [][]uint64
	offsets       []uint64
	skips         []uint64
	originOffsets []uint64
	m             uint64
}

func New(names []string, m uint64) *Table {
	offsets, skips := generatePermutations(names, m)
	t := &Table{
		n: len(names),
		//lookup:       lookup,
		//permutations: permutations,
		skips:   skips,
		offsets: offsets,
		originOffsets: make([]uint64, len(names)),
		m:       m,
	}
	copy(t.originOffsets, t.offsets)
	t.lookup = t.populate(m, nil)

	return t
}

func (t *Table) Lookup(key uint64) int {
	return t.lookup[key%uint64(len(t.lookup))]
}

func (t *Table) Rebuild(dead []int) {
	t.lookup = t.populate(t.m, dead)
}

func generatePermutations(names []string, M uint64) ([]uint64, []uint64) {
	//permutations := make([][]uint64, len(names))
	offsets := make([]uint64, len(names))
	skips := make([]uint64, len(names))

	for i, name := range names {
		b := []byte(name)
		h := siphash.Hash(0xdeadbeefcafebabe, 0, b)
		offsets[i], skips[i] = (h>>32)%M, ((h&0xffffffff)%(M-1) + 1)

		//p := make([]uint64, M)
		//idx := offset
		//for j := uint64(0); j < M; j++ {
		//	p[j] = idx
		//	idx += skip
		//	if idx >= M {
		//		idx -= M
		//	}
		//}
		//permutations[i] = p
	}

	return offsets, skips
	//
	//return permutations
}

func (t *Table) populate(M uint64, dead []int) []int {
	t.resetOffsets()
	//M := len(permutation[0]) // smallM
	N := len(t.offsets) // len(names)

	//next := make([]uint64, N)
	entry := make([]int, M)
	for j := range entry {
		entry[j] = -1
	}

	var n uint64
	for {
		d := dead
		for i := 0; i < N; i++ {
			if len(d) > 0 && d[0] == i {
				d = d[1:]
				continue
			}

			var c uint64
			//c = t.offsets[i]
			t.next(i, &c)

			//c := permutation[i][next[i]]
			for entry[c] >= 0 {
				t.next(i, &c)
				//next[i]++
				//c = permutation[i][next[i]]
			}
			entry[c] = i
			//next[i]++
			n++
			if n == M {
				return entry
			}
		}
	}
}

func (t *Table) next(i int, c *uint64) {
	*c = t.offsets[i]

	t.offsets[i] += t.skips[i]
	if t.offsets[i] >= t.m {
		t.offsets[i] -= t.m
	}
}

func (t *Table) resetOffsets() {
	copy(t.offsets, t.originOffsets)
}

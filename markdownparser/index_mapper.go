package markdownparser

import "sort"

type indexToLine struct {
	offsets []int
}

func newIndexToLine(content []byte) *indexToLine {
	offsets := []int{0}

	for i, b := range content {
		if b == '\n' {
			offsets = append(offsets, i+1)
		}
	}

	return &indexToLine{offsets: offsets}
}

func (m *indexToLine) lineFor(index int) int {
	if m == nil || index < 0 {
		return -1
	}

	pos := sort.Search(len(m.offsets), func(i int) bool {
		return m.offsets[i] > index
	})
	if pos == 0 {
		return 1
	}

	return pos
}

package goja

import (
	"fmt"
	"github.com/go-sourcemap/sourcemap"
	"sort"
	"strings"
	"sync"
)

type Position struct {
	Line, Col int
}

type SrcFile struct {
	name string
	src  string

	lineOffsets       []int
	lineOffsetsLock   sync.Mutex
	lastScannedOffset int
	sourceMap         *sourcemap.Consumer
}

func NewSrcFile(name, src string, sourceMap *sourcemap.Consumer) *SrcFile {
	return &SrcFile{
		name:      name,
		src:       src,
		sourceMap: sourceMap,
	}
}

func (f *SrcFile) Position(offset int) Position {
	var line int
	var lineOffsets []int
	f.lineOffsetsLock.Lock()
	if offset > f.lastScannedOffset {
		line = f.scanTo(offset)
		lineOffsets = f.lineOffsets
		f.lineOffsetsLock.Unlock()
	} else {
		lineOffsets = f.lineOffsets
		f.lineOffsetsLock.Unlock()
		line = sort.Search(len(lineOffsets), func(x int) bool { return lineOffsets[x] > offset }) - 1
	}

	var lineStart int
	if line >= 0 {
		lineStart = lineOffsets[line]
	}

	row := line + 2
	col := offset - lineStart + 1

	if f.sourceMap != nil {
		if _, _, row, col, ok := f.sourceMap.Source(row, col); ok {
			return Position{
				Line: row,
				Col:  col,
			}
		}
	}

	return Position{
		Line: row,
		Col:  col,
	}
}

func (f *SrcFile) scanTo(offset int) int {
	o := f.lastScannedOffset
	for o < offset {
		p := strings.Index(f.src[o:], "\n")
		if p == -1 {
			f.lastScannedOffset = len(f.src)
			return len(f.lineOffsets) - 1
		}
		o = o + p + 1
		f.lineOffsets = append(f.lineOffsets, o)
	}
	f.lastScannedOffset = o

	if o == offset {
		return len(f.lineOffsets) - 1
	}

	return len(f.lineOffsets) - 2
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Col)
}

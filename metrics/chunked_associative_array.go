package metrics

// Ported from
// https://github.com/dropwizard/metrics/blob/release/4.2.x/metrics-core/src/main/java/com/codahale/metrics/ChunkedAssociativeLongArray.java

import (
	"github.com/gammazero/deque"
	"sort"
	"strconv"
	"strings"
)

const (
	ChunkedAssociativeArrayDefaultChunkSize = 512
	ChunkedAssociativeArrayMaxCacheSize     = 128
)

type ChunkedAssociativeArray struct {
	defaultChunkSize int

	/*
	 * We use this ArrayDeque as cache to store chunks that are expired and removed from main data structure.
	 * Then instead of allocating new AssociativeArrayChunk immediately we are trying to poll one from this deque.
	 * So if you have constant or slowly changing load ChunkedAssociativeLongArray will never
	 * throw away old chunks or allocate new ones which makes this data structure almost garbage free.
	 */
	chunksCache *deque.Deque[*AssociativeArrayChunk]
	chunks      *deque.Deque[*AssociativeArrayChunk]
}

func NewChunkedAssociativeArray(chunkSize int) *ChunkedAssociativeArray {
	return &ChunkedAssociativeArray{
		defaultChunkSize: chunkSize,
		chunksCache:      deque.New[*AssociativeArrayChunk](ChunkedAssociativeArrayMaxCacheSize, ChunkedAssociativeArrayMaxCacheSize),
		chunks:           deque.New[*AssociativeArrayChunk](),
	}
}

func (caa *ChunkedAssociativeArray) Clear() {
	for i := 0; i < caa.chunks.Len(); i++ {
		chunk := caa.chunks.PopBack()
		caa.freeChunk(chunk)
	}
}

func (caa *ChunkedAssociativeArray) AllocateChunk() *AssociativeArrayChunk {
	if caa.chunksCache.Len() == 0 {
		return NewAssociativeArrayChunk(caa.defaultChunkSize)
	}

	chunk := caa.chunksCache.PopBack()
	chunk.cursor = 0
	chunk.startIndex = 0
	chunk.chunkSize = len(chunk.keys)

	return chunk
}

func (caa *ChunkedAssociativeArray) freeChunk(chunk *AssociativeArrayChunk) {
	if caa.chunksCache.Len() < ChunkedAssociativeArrayMaxCacheSize {
		caa.chunksCache.PushBack(chunk)
	}
}

func (caa *ChunkedAssociativeArray) Put(key int64, value int64) {
	var activeChunk *AssociativeArrayChunk
	if caa.chunks.Len() > 0 {
		activeChunk = caa.chunks.Back()
	}

	if activeChunk != nil && activeChunk.cursor != 0 && activeChunk.keys[activeChunk.cursor-1] > key {
		// Key must be the same as last inserted or bigger
		key = activeChunk.keys[activeChunk.cursor-1] + 1
	}
	if activeChunk == nil || activeChunk.cursor-activeChunk.startIndex == activeChunk.chunkSize {
		// The last chunk doesn't exist or full
		activeChunk = caa.AllocateChunk()
		caa.chunks.PushBack(activeChunk)
	}
	activeChunk.Append(key, value)
}

func (caa *ChunkedAssociativeArray) Values() []int64 {
	valuesSize := caa.Size()
	if valuesSize == 0 {
		// Empty
		return []int64{0}
	}

	values := make([]int64, 0, valuesSize)
	caa.chunks.Index(func(chunk *AssociativeArrayChunk) bool {
		values = append(values, chunk.values[chunk.startIndex:chunk.cursor]...)
		return false
	})

	return values
}

func (caa *ChunkedAssociativeArray) Size() int {
	var result int
	caa.chunks.Index(func(chunk *AssociativeArrayChunk) bool {
		result += chunk.cursor - chunk.startIndex
		return false
	})
	return result
}

func (caa *ChunkedAssociativeArray) String() string {
	var builder strings.Builder
	first := true
	caa.chunks.Index(func(chunk *AssociativeArrayChunk) bool {
		if first {
			first = false
		} else {
			builder.WriteString("->")
		}
		builder.WriteString("[")
		for i := chunk.startIndex; i < chunk.cursor; i++ {
			builder.WriteString("(")
			builder.WriteString(strconv.FormatInt(chunk.keys[i], 10))
			builder.WriteString(": ")
			builder.WriteString(strconv.FormatInt(chunk.values[i], 10))
			builder.WriteString(") ")
		}
		builder.WriteString("]")
		return false
	})

	return builder.String()
}

// Trim tries to trim all beyond specified boundaries
// startKey: the start value for which all elements less than it should be removed.
// endKey:   the end value for which all elements greater/equals than it should be removed
func (caa *ChunkedAssociativeArray) Trim(startKey int64, endKey int64) {
	/*
	 * [3, 4, 5, 9] -> [10, 13, 14, 15] -> [21, 24, 29, 30] -> [31] :: start layout
	 *       |5______________________________23|                    :: trim(5, 23)
	 *       [5, 9] -> [10, 13, 14, 15] -> [21]                     :: result layout
	 */
	// Remove elements that are too large
	indexBeforeEndKey := caa.chunks.RIndex(func(chunk *AssociativeArrayChunk) bool {
		if chunk.IsFirstElementEmptyOrGreaterEqualThanKey(endKey) {
			return false
		}

		chunk.cursor = chunk.FindFirstIndexOfGreaterEqualElements(endKey)
		return true
	})

	// Remove chunks that only contain elements that are too large
	if indexBeforeEndKey >= 0 {
		for i := caa.chunks.Len() - 1; i > indexBeforeEndKey; i-- {
			chunk := caa.chunks.PopBack()
			caa.freeChunk(chunk)
		}
	}

	// Remove elements that are too small
	indexAfterStartKey := caa.chunks.Index(func(chunk *AssociativeArrayChunk) bool {
		if chunk.IsLastElementEmptyOrLessThanKey(startKey) {
			return false
		}

		newStartIndex := chunk.FindFirstIndexOfGreaterEqualElements(startKey)
		if chunk.startIndex != newStartIndex {
			chunk.startIndex = newStartIndex
			chunk.chunkSize = chunk.cursor - chunk.startIndex
		}
		return true
	})

	// Remove chunks that only contain elements that are too small
	for i := 0; i < indexAfterStartKey; i++ {
		chunk := caa.chunks.PopFront()
		caa.freeChunk(chunk)
	}
}

type AssociativeArrayChunk struct {
	keys   []int64
	values []int64

	chunkSize  int
	startIndex int
	cursor     int
}

func NewAssociativeArrayChunk(chunkSize int) *AssociativeArrayChunk {
	return &AssociativeArrayChunk{
		keys:       make([]int64, chunkSize),
		values:     make([]int64, chunkSize),
		chunkSize:  chunkSize,
		startIndex: 0,
		cursor:     0,
	}
}

func (c *AssociativeArrayChunk) Append(key int64, value int64) {
	c.keys[c.cursor] = key
	c.values[c.cursor] = value
	c.cursor++
}

func (c *AssociativeArrayChunk) IsFirstElementEmptyOrGreaterEqualThanKey(key int64) bool {
	return c.cursor == c.startIndex || c.keys[c.startIndex] >= key
}

func (c *AssociativeArrayChunk) IsLastElementEmptyOrLessThanKey(key int64) bool {
	return c.cursor == c.startIndex || c.keys[c.cursor-1] < key
}

func (c *AssociativeArrayChunk) FindFirstIndexOfGreaterEqualElements(minKey int64) int {
	if c.cursor == c.startIndex || c.keys[c.startIndex] >= minKey {
		return c.startIndex
	}
	elements := c.keys[c.startIndex:c.cursor]
	keyIndex := sort.Search(len(elements), func(i int) bool { return elements[i] >= minKey })

	return c.startIndex + keyIndex
}

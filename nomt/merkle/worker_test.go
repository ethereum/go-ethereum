package merkle

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/nomt/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Unit tests for helpers ---

func TestPartitionByChildIndex(t *testing.T) {
	// Stem 0x00... → child 0, stem 0x04... → child 1, stem 0xFC... → child 63.
	skvs := []core.StemKeyValue{
		makeSKV(0x00),
		makeSKV(0x04), // 0x04 >> 2 = 1
		makeSKV(0xFC), // 0xFC >> 2 = 63
	}
	buckets := partitionByChildIndex(skvs)

	assert.Len(t, buckets[0], 1)
	assert.Len(t, buckets[1], 1)
	assert.Len(t, buckets[63], 1)

	// All other buckets should be empty.
	nonEmpty := 0
	for _, b := range buckets {
		if len(b) > 0 {
			nonEmpty++
		}
	}
	assert.Equal(t, 3, nonEmpty)
}

func TestChildPosition(t *testing.T) {
	// Child 0, left: 7 bits all false → depth 7.
	pos := childPosition(0, false)
	assert.Equal(t, uint16(7), pos.Depth())
	for i := range 7 {
		assert.False(t, pos.Bit(i), "bit %d should be 0", i)
	}

	// Child 0, right: 6 false + 1 true → depth 7.
	pos = childPosition(0, true)
	assert.Equal(t, uint16(7), pos.Depth())
	for i := range 6 {
		assert.False(t, pos.Bit(i), "bit %d should be 0", i)
	}
	assert.True(t, pos.Bit(6))

	// Child 63 (0b111111), left: 6 true + 1 false → depth 7.
	pos = childPosition(63, false)
	assert.Equal(t, uint16(7), pos.Depth())
	for i := range 6 {
		assert.True(t, pos.Bit(i), "bit %d should be 1", i)
	}
	assert.False(t, pos.Bit(6))

	// Child 63 (0b111111), right: 7 true → depth 7.
	pos = childPosition(63, true)
	assert.Equal(t, uint16(7), pos.Depth())
	for i := range 7 {
		assert.True(t, pos.Bit(i), "bit %d should be 1", i)
	}
}

func TestAssignToWorkers(t *testing.T) {
	// 3 non-empty buckets, 2 workers.
	var buckets [64][]core.StemKeyValue
	buckets[0] = []core.StemKeyValue{makeSKV(0x00)}
	buckets[10] = []core.StemKeyValue{makeSKV(0x28)} // 0x28>>2=10
	buckets[63] = []core.StemKeyValue{makeSKV(0xFC)}

	tasks := assignToWorkers(buckets, 2)
	require.Len(t, tasks, 2)
	// 3 items / 2 workers: first gets 2, second gets 1.
	assert.Len(t, tasks[0].children, 2)
	assert.Len(t, tasks[1].children, 1)
	assert.Equal(t, uint8(0), tasks[0].children[0].childIndex)
	assert.Equal(t, uint8(10), tasks[0].children[1].childIndex)
	assert.Equal(t, uint8(63), tasks[1].children[0].childIndex)
}

func TestAssignToWorkersMoreWorkersThanChildren(t *testing.T) {
	var buckets [64][]core.StemKeyValue
	buckets[5] = []core.StemKeyValue{makeSKV(0x14)} // 0x14>>2=5
	buckets[6] = []core.StemKeyValue{makeSKV(0x18)} // 0x18>>2=6

	tasks := assignToWorkers(buckets, 8)
	// Only 2 non-empty, so cap to 2 workers.
	require.Len(t, tasks, 2)
	assert.Len(t, tasks[0].children, 1)
	assert.Len(t, tasks[1].children, 1)
}

// --- Integration tests ---

// permissivePageSet wraps MemoryPageSet to return fresh pages for missing
// entries (matching pebblePageSet behavior). This is needed because the
// parallel workers descend into child pages that may not exist yet.
type permissivePageSet struct {
	*MemoryPageSet
}

func (ps *permissivePageSet) Get(pageID core.PageID) (*core.RawPage, PageOrigin, bool) {
	page, origin, ok := ps.MemoryPageSet.Get(pageID)
	if !ok {
		fresh := new(core.RawPage)
		return fresh, PageOrigin{Kind: PageOriginFresh}, true
	}
	return page, origin, true
}

func memoryPageSetFactory() PageSet {
	return &permissivePageSet{NewMemoryPageSet(true)}
}

// expectedWorkerRoot computes the expected root hash matching the depth-7
// child-index partitioning used by both singleThreadedUpdate and ParallelUpdate.
// This differs from expectedRoot (which splits at depth 1) because without
// leaf compaction, the splitting depth affects intermediate hashes.
func expectedWorkerRoot(skvs []core.StemKeyValue) core.Node {
	if len(skvs) == 0 {
		return core.Terminator
	}

	// Partition into 128 subtree roots (64 child indices × 2 sides).
	buckets := partitionByChildIndex(skvs)
	var roots [128]core.Node
	for ci := range 64 {
		if len(buckets[ci]) == 0 {
			continue
		}
		var leftKVs, rightKVs []core.StemKeyValue
		for i := range buckets[ci] {
			if (buckets[ci][i].Stem[0]>>1)&1 == 0 {
				leftKVs = append(leftKVs, buckets[ci][i])
			} else {
				rightKVs = append(rightKVs, buckets[ci][i])
			}
		}
		if len(leftKVs) > 0 {
			roots[ci*2] = core.BuildInternalTree(7, leftKVs, func(_ core.WriteNode) {})
		}
		if len(rightKVs) > 0 {
			roots[ci*2+1] = core.BuildInternalTree(7, rightKVs, func(_ core.WriteNode) {})
		}
	}

	// Hash up 7 levels: 128 → 64 → 32 → 16 → 8 → 4 → 2 → 1.
	nodes := make([]core.Node, 128)
	copy(nodes, roots[:])
	for len(nodes) > 1 {
		half := len(nodes) / 2
		next := make([]core.Node, half)
		for i := range half {
			left := nodes[i*2]
			right := nodes[i*2+1]
			if core.IsTerminator(&left) && core.IsTerminator(&right) {
				next[i] = core.Terminator
			} else {
				next[i] = core.HashInternal(&core.InternalData{Left: left, Right: right})
			}
		}
		nodes = next
	}

	return nodes[0]
}

func TestParallelUpdateEmpty(t *testing.T) {
	out := ParallelUpdate(core.Terminator, nil, 4, memoryPageSetFactory)
	assert.Equal(t, core.Terminator, out.Root)
}

func TestParallelUpdateSingleKey(t *testing.T) {
	skv := makeSKV(0x50)
	skvs := []core.StemKeyValue{skv}

	out := ParallelUpdate(core.Terminator, skvs, 4, memoryPageSetFactory)
	expected := expectedWorkerRoot(skvs)
	assert.Equal(t, expected, out.Root)
}

func TestParallelUpdateTwoKeysDifferentChildren(t *testing.T) {
	// 0x00 → child 0, 0x80 → child 32.
	skvs := []core.StemKeyValue{
		makeSKV(0x00),
		makeSKV(0x80),
	}

	out := ParallelUpdate(core.Terminator, skvs, 4, memoryPageSetFactory)
	expected := expectedWorkerRoot(skvs)
	assert.Equal(t, expected, out.Root)
}

func TestParallelUpdateSparseChildren(t *testing.T) {
	// Only children 0 and 63 have ops.
	skvs := []core.StemKeyValue{
		makeSKV(0x00),
		makeSKV(0xFC),
	}

	out := ParallelUpdate(core.Terminator, skvs, 4, memoryPageSetFactory)
	expected := expectedWorkerRoot(skvs)
	assert.Equal(t, expected, out.Root)
}

func TestParallelUpdateSingleChild(t *testing.T) {
	// All stems land in child 0 (first 6 bits = 000000).
	skvs := []core.StemKeyValue{
		makeSKV(0x00),
		makeSKV(0x01),
		makeSKV(0x02),
		makeSKV(0x03),
	}
	sort.Slice(skvs, func(i, j int) bool { return skvLess(&skvs[i], &skvs[j]) })

	out := ParallelUpdate(core.Terminator, skvs, 4, memoryPageSetFactory)
	expected := expectedWorkerRoot(skvs)
	assert.Equal(t, expected, out.Root)
}

func TestParallelUpdateFallbackSmallBatch(t *testing.T) {
	// Less than 64 ops → single-threaded fallback.
	skvs := randomSKVs(10, 42)
	out := ParallelUpdate(core.Terminator, skvs, 8, memoryPageSetFactory)
	expected := expectedWorkerRoot(skvs)
	assert.Equal(t, expected, out.Root)
}

func TestParallelUpdateDeterministic(t *testing.T) {
	skvs := randomSKVs(200, 99)

	r1 := ParallelUpdate(core.Terminator, skvs, 4, memoryPageSetFactory).Root
	r2 := ParallelUpdate(core.Terminator, skvs, 4, memoryPageSetFactory).Root
	assert.Equal(t, r1, r2, "same inputs should produce same root")
}

func TestParallelUpdateMatchesSingleThreaded(t *testing.T) {
	tests := []struct {
		name    string
		numSKVs int
		workers int
	}{
		{"1skv_2w", 1, 2},
		{"10skv_2w", 10, 2},
		{"100skv_2w", 100, 2},
		{"100skv_4w", 100, 4},
		{"100skv_8w", 100, 8},
		{"500skv_4w", 500, 4},
		{"1000skv_8w", 1000, 8},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			skvs := randomSKVs(tc.numSKVs, 12345)

			single := singleThreadedUpdate(
				core.Terminator, skvs, memoryPageSetFactory(),
			)
			parallel := ParallelUpdate(
				core.Terminator, skvs, tc.workers, memoryPageSetFactory,
			)

			assert.Equal(t, single.Root, parallel.Root,
				"parallel root should match single-threaded root")
		})
	}
}

// --- helpers ---

func randomSKVs(n int, seed int64) []core.StemKeyValue {
	rng := rand.New(rand.NewSource(seed))
	skvs := make([]core.StemKeyValue, n)
	seen := make(map[core.StemPath]bool, n)

	for i := range n {
		for {
			var stem core.StemPath
			rng.Read(stem[:])
			if seen[stem] {
				continue
			}
			seen[stem] = true
			var hash core.Node
			rng.Read(hash[:])
			// Ensure non-zero hash (avoid terminator).
			hash[0] |= 0x01
			skvs[i] = core.StemKeyValue{Stem: stem, Hash: hash}
			break
		}
	}

	sort.Slice(skvs, func(i, j int) bool { return skvLess(&skvs[i], &skvs[j]) })
	return skvs
}

func skvLess(a, b *core.StemKeyValue) bool {
	for i := range a.Stem {
		if a.Stem[i] < b.Stem[i] {
			return true
		}
		if a.Stem[i] > b.Stem[i] {
			return false
		}
	}
	return false
}

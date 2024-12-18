package common

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShrinkingMap_Shrink(t *testing.T) {
	m := NewShrinkingMap[int, int](10)

	for i := 0; i < 100; i++ {
		m.Set(i, i)
	}

	for i := 0; i < 100; i++ {
		val, exists := m.Get(i)
		require.Equal(t, true, exists)
		require.Equal(t, i, val)

		has := m.Has(i)
		require.Equal(t, true, has)
	}

	for i := 0; i < 9; i++ {
		m.Delete(i)
	}
	require.Equal(t, 9, m.deletedKeys)

	// Delete the 10th key -> shrinks the map
	m.Delete(9)
	require.Equal(t, 0, m.deletedKeys)

	for i := 0; i < 100; i++ {
		if i < 10 {
			val, exists := m.Get(i)
			require.Equal(t, false, exists)
			require.Equal(t, 0, val)

			has := m.Has(i)
			require.Equal(t, false, has)
		} else {
			val, exists := m.Get(i)
			require.Equal(t, true, exists)
			require.Equal(t, i, val)

			has := m.Has(i)
			require.Equal(t, true, has)
		}
	}

	require.Equal(t, 90, m.Size())
}

func TestNewShrinkingMap_NoShrinking(t *testing.T) {
	m := NewShrinkingMap[int, int](0)
	for i := 0; i < 10000; i++ {
		m.Set(i, i)
	}

	for i := 0; i < 10000; i++ {
		val, exists := m.Get(i)
		require.Equal(t, true, exists)
		require.Equal(t, i, val)

		m.Delete(i)
	}

	require.Equal(t, 0, m.Size())
	require.Equal(t, 10000, m.deletedKeys)
}

func TestShrinkingMap_MemoryShrinking(t *testing.T) {
	t.Skip("Only for manual testing and memory profiling")

	gcAndPrintAlloc("start")
	m := NewShrinkingMap[int, int](10000)

	const mapSize = 1_000_000

	for i := 0; i < mapSize; i++ {
		m.Set(i, i)
	}

	gcAndPrintAlloc("after map creation")

	for i := 0; i < mapSize/2; i++ {
		m.Delete(i)
	}

	gcAndPrintAlloc("after removing half of the elements")

	val, exist := m.Get(mapSize - 1)
	require.Equal(t, true, exist)
	require.Equal(t, mapSize-1, val)

	gcAndPrintAlloc("end")
}

func TestShrinkingMap_MemoryNoShrinking(t *testing.T) {
	t.Skip("Only for manual testing and memory profiling")

	gcAndPrintAlloc("start")
	m := NewShrinkingMap[int, int](0)

	const mapSize = 1_000_000

	for i := 0; i < mapSize; i++ {
		m.Set(i, i)
	}

	gcAndPrintAlloc("after map creation")

	for i := 0; i < mapSize/2; i++ {
		m.Delete(i)
	}

	gcAndPrintAlloc("after removing half of the elements")

	val, exist := m.Get(mapSize - 1)
	require.Equal(t, true, exist)
	require.Equal(t, mapSize-1, val)

	gcAndPrintAlloc("end")
}

func gcAndPrintAlloc(prefix string) {
	runtime.GC()

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	fmt.Printf(prefix+", Allocated memory %d KiB\n", stats.Alloc/1024)
}

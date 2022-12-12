package metrics

import (
	"testing"
)

// Ported from
// https://github.com/dropwizard/metrics/blob/release/4.2.x/metrics-core/src/main/java/com/codahale/metrics/ChunkedAssociativeLongArray.java

func TestChunkedAssociativeArray_Put(t *testing.T) {
	array := NewChunkedAssociativeArray(3)
	// Test that time cannot go backwards
	array.Put(7, 7)
	expectedStringBefore := "[(7: 7) ]"
	if array.String() != expectedStringBefore {
		t.Errorf("initial array string incorrect: %s", array.String())
	}
}

func TestChunkedAssociativeArray_ValuesEmpty(t *testing.T) {
	array := NewChunkedAssociativeArray(3)

	values := array.Values()
	if len(values) != 1 {
		t.Fatalf("unexpected length of values: %d", len(values))
	}
	if values[0] != 0 {
		t.Errorf("unexpected value in empty values: %d", values[0])
	}
}

func TestChunkedAssociativeArray_Trim(t *testing.T) {
	array := NewChunkedAssociativeArray(3)
	array.Put(-7, 7)
	array.Put(-5, 7)
	array.Put(-4, 7)
	array.Put(-3, 3)
	array.Put(-2, 1)
	array.Put(0, 5)
	array.Put(3, 0)
	array.Put(9, 8)
	array.Put(15, 0)
	array.Put(19, 5)
	array.Put(21, 5)
	array.Put(34, -9)
	array.Put(109, 5)

	expectedStringBefore := "[(-7: 7) (-5: 7) (-4: 7) ]->[(-3: 3) (-2: 1) (0: 5) ]->[(3: 0) (9: 8) (15: 0) ]->[(19: 5) (21: 5) (34: -9) ]->[(109: 5) ]"
	if array.String() != expectedStringBefore {
		t.Errorf("initial array string incorrect: %s", array.String())
	}
	valuesBefore := array.Values()
	expectedValuesBefore := []int64{7, 7, 7, 3, 1, 5, 0, 8, 0, 5, 5, -9, 5}
	if len(valuesBefore) != len(expectedValuesBefore) {
		t.Errorf("initial values returned incorrect length: %d", len(valuesBefore))
	} else {
		for i, value := range valuesBefore {
			if value != expectedValuesBefore[i] {
				t.Errorf("unexpected value %d at index %d", value, i)
			}
		}
	}
	if array.Size() != 13 {
		t.Errorf("initial array size incorrect: %d", array.Size())
	}

	array.Trim(-2, 20)

	expectedStringAfter := "[(-2: 1) (0: 5) ]->[(3: 0) (9: 8) (15: 0) ]->[(19: 5) ]"
	if array.String() != expectedStringAfter {
		t.Errorf("array string incorrect: %s", array.String())
	}
	valuesAfter := array.Values()
	expectedValuesAfter := []int64{1, 5, 0, 8, 0, 5}
	if len(valuesAfter) != len(expectedValuesAfter) {
		t.Errorf("values returned incorrect length: %d", len(valuesAfter))
	} else {
		for i, value := range valuesAfter {
			if value != expectedValuesAfter[i] {
				t.Errorf("unexpected value %d at index %d", value, i)
			}
		}
	}
	if array.Size() != 6 {
		t.Errorf("array size incorrect: %d", array.Size())
	}

	array.Trim(-2, 16)
	expectedStringAfter2 := "[(-2: 1) (0: 5) ]->[(3: 0) (9: 8) (15: 0) ]"
	if array.String() != expectedStringAfter2 {
		t.Errorf("array string incorrect: %s", array.String())
	}

	initialCacheCount := array.chunksCache.Len()

	// Have AllocateChunk take from cache
	array.Put(200, 555)
	expectedStringAfter3 := "[(-2: 1) (0: 5) ]->[(3: 0) (9: 8) (15: 0) ]->[(200: 555) ]"
	if array.String() != expectedStringAfter3 {
		t.Errorf("array string incorrect: %s", array.String())
	}

	if array.chunksCache.Len() >= initialCacheCount {
		t.Error("cache not used when allocating chunk")
	}
}

func TestAssociativeArrayChunk_IsFirstElementEmptyOrGreaterEqualThanKey(t *testing.T) {
	chunk := NewAssociativeArrayChunk(3)

	if !chunk.IsFirstElementEmptyOrGreaterEqualThanKey(5) {
		t.Error("empty test failed")
	}

	chunk.keys = []int64{41, 42, 43}
	chunk.startIndex = 1

	if chunk.IsFirstElementEmptyOrGreaterEqualThanKey(43) {
		t.Error("element less than key test failed")
	}
	if !chunk.IsFirstElementEmptyOrGreaterEqualThanKey(42) {
		t.Error("element greater than or equal to key test failed")
	}
}

func TestAssociativeArrayChunk_IsLastElementIsLessThanKey(t *testing.T) {
	chunk := NewAssociativeArrayChunk(3)

	if !chunk.IsLastElementEmptyOrLessThanKey(5) {
		t.Error("empty test failed")
	}

	chunk.keys = []int64{41, 42, 43}
	chunk.cursor = 2

	if !chunk.IsLastElementEmptyOrLessThanKey(43) {
		t.Error("element less than key test failed")
	}
	if chunk.IsLastElementEmptyOrLessThanKey(42) {
		t.Error("element greater than or equal to key test failed")
	}
}

func TestAssociativeArrayChunk_FindFirstIndexOfGreaterEqualElements(t *testing.T) {
	chunk := NewAssociativeArrayChunk(7)

	if chunk.FindFirstIndexOfGreaterEqualElements(5) != 0 {
		t.Error("empty test failed")
	}

	chunk.keys = []int64{41, 42, 43, 44, 45, 46, 48}
	chunk.startIndex = 1
	chunk.cursor = 6

	if chunk.FindFirstIndexOfGreaterEqualElements(41) != chunk.startIndex {
		t.Error("minKey less than first element test failed")
	}
	if chunk.FindFirstIndexOfGreaterEqualElements(42) != chunk.startIndex {
		t.Error("minKey greater than or equal to first element test failed")
	}
	if chunk.FindFirstIndexOfGreaterEqualElements(43) != 2 {
		t.Error("2nd element test failed")
	}
	if chunk.FindFirstIndexOfGreaterEqualElements(46) != 5 {
		t.Error("last element test failed")
	}
	if chunk.FindFirstIndexOfGreaterEqualElements(49) != 6 {
		t.Error("past last element test failed")
	}
}

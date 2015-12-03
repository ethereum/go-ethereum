package storage

import (
	"testing"
)

func testMemStore(l int64, branches int64, t *testing.T) {
	m := NewMemStore(nil, defaultCacheCapacity)
	testStore(m, l, branches, t)
}

func TestMemStore128_10000(t *testing.T) {
	testMemStore(10000, 128, t)
}

func TestMemStore128_1000(t *testing.T) {
	testMemStore(1000, 128, t)
}

func TestMemStore128_100(t *testing.T) {
	testMemStore(100, 128, t)
}

func TestMemStore2_100(t *testing.T) {
	testMemStore(100, 2, t)
}

func TestMemStoreNotFound(t *testing.T) {
	m := NewMemStore(nil, defaultCacheCapacity)
	_, err := m.Get(ZeroKey)
	if err != notFound {
		t.Errorf("Expected notFound, got %v", err)
	}
}

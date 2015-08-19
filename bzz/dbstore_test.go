package bzz

import (
	"os"
	"testing"
)

func initDbStore() (m *dbStore) {
	os.RemoveAll("/tmp/bzz")
	m, err := newDbStore("/tmp/bzz")
	if err != nil {
		panic("no dbStore")
	}
	return
}

func testDbStore(l int64, branches int64, t *testing.T) {
	m := initDbStore()
	defer m.close()
	testStore(m, l, branches, t)
}

func TestDbStore128_0x1000000(t *testing.T) {
	testDbStore(0x1000000, 128, t)
}

func TestDbStore128_10000(t *testing.T) {
	testDbStore(10000, 128, t)
}

func TestDbStore128_1000(t *testing.T) {
	testDbStore(1000, 128, t)
}

func TestDbStore128_100(t *testing.T) {
	testDbStore(100, 128, t)
}

func TestDbStore2_100(t *testing.T) {
	testDbStore(100, 2, t)
}

func TestDbStoreNotFound(t *testing.T) {
	m := initDbStore()
	defer m.close()
	zeroKey := make([]byte, 32)
	_, err := m.Get(zeroKey)
	if err != notFound {
		t.Errorf("Expected notFound, got %v", err)
	}
}

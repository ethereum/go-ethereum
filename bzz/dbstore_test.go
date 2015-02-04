package bzz

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/bzz/test"
)

func testDbStore(l int64, branches int64, t *testing.T) {

	os.RemoveAll("/tmp/bzz")
	m, err := newDbStore("/tmp/bzz")
	if err != nil {
		panic("no dbStore")
	}
	defer m.close()
	testStore(m, l, branches, t)
}

func TestDbStore128_10000(t *testing.T) {
	// defer test.Testlog(t).Detach()
	test.LogInit()
	testDbStore(10000, 128, t)
}

func TestDbStore128_1000(t *testing.T) {
	// defer test.Testlog(t).Detach()
	test.LogInit()
	testDbStore(1000, 128, t)
}

func TestDbStore128_100(t *testing.T) {
	// defer test.Testlog(t).Detach()
	test.LogInit()
	testDbStore(100, 128, t)
}

func TestDbStore2_100(t *testing.T) {
	// defer test.Testlog(t).Detach()
	test.LogInit()
	testDbStore(100, 2, t)
}

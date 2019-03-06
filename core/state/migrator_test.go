package state

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

func TestMigrator(t *testing.T) {
	tests := []struct {
		numWorkers, batchSize int
	}{
		{1, 1},
		{2, 1},
		{1, 2},
		{2, 2},
	}

	for _, tc := range tests {
		name := fmt.Sprintf("%d_Workers_%d_BatchSize", tc.numWorkers, tc.batchSize)
		t.Run(name, func(t *testing.T) {
			srcDB, srcRoot, srcAccounts := makeTestState()
			// Ensure nodes are persisted to the underlying database.
			srcDB.TrieDB().Commit(srcRoot, false)
			dstDB := ethdb.NewMemDatabase()

			m := NewMigrator(dstDB, srcDB.TrieDB().DiskDB(), srcRoot, tc.numWorkers, tc.batchSize)
			m.Start()
			if err := m.Wait(); err != nil {
				t.Fatalf("m.Wait() = %v, want <nil>", err)
			}

			checkStateAccounts(t, dstDB, srcRoot, srcAccounts)
		})
	}
}

func TestMigrator_SrcDBReturnsError_ShouldReturnError(t *testing.T) {
	srcDB := &failingDB{}
	dstDB := ethdb.NewMemDatabase()

	m := NewMigrator(dstDB, srcDB, common.Hash{} /* numWorkers */, 1 /* batchSize */, 1)
	m.Start()

	if err := m.Wait(); err == nil {
		t.Fatal("m.Wait() = <nil>, want <error>")
	}
}

func TestMigrator_DstDBReturnsError_ShouldReturnError(t *testing.T) {
	srcDB, srcRoot, _ := makeTestState()
	// Ensure nodes are persisted to the underlying database.
	srcDB.TrieDB().Commit(srcRoot, false)
	dstDB := &failingDB{}

	m := NewMigrator(dstDB, srcDB.TrieDB().DiskDB(), srcRoot /* numWorkers */, 1 /* batchSize */, 1)
	m.Start()

	if err := m.Wait(); err == nil {
		t.Fatal("m.Wait() = <nil>, want <error>")
	}
}

// failingDB implements trie.DatabaseReader and ethdb.Database, but
// returns a dummy error any time a method that returns an error is invoked.
type failingDB struct{}

func (*failingDB) Put(key, value []byte) error {
	return errors.New("failed")
}

func (*failingDB) Delete(key []byte) error {
	return errors.New("failed")
}

func (*failingDB) Get(key []byte) (value []byte, err error) {
	return nil, errors.New("failed")
}

func (*failingDB) Has(key []byte) (bool, error) {
	return false, errors.New("failed")
}

func (*failingDB) Close() {}

func (*failingDB) NewBatch() ethdb.Batch {
	return &failingBatch{}
}

type failingBatch struct {
	failingDB
}

func (*failingBatch) ValueSize() int {
	return 0
}

func (*failingBatch) Write() error {
	return errors.New("failed")
}

func (*failingBatch) Reset() {}

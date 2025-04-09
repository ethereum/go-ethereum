package state

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// BlockToBaseStateRoot maps block hashes to their corresponding base Merkle state roots
// during the Verkle transition period to uniquely identify states
type BlockToBaseStateRoot struct {
	blockHashToStateRoot map[common.Hash]common.Hash
	mutex                sync.RWMutex
}

// NewBlockToBaseStateRootMapping creates a new mapping instance
func NewBlockToBaseStateRootMapping() *BlockToBaseStateRoot {
	return &BlockToBaseStateRoot{
		blockHashToStateRoot: make(map[common.Hash]common.Hash),
	}
}

// Add associates a block hash with its base state root
func (m *BlockToBaseStateRoot) Add(blockHash common.Hash, stateRoot common.Hash) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.blockHashToStateRoot[blockHash] = stateRoot
	log.Debug("Added verkle transition mapping", "blockHash", blockHash, "baseStateRoot", stateRoot)
}

// Get retrieves the base state root for a given block hash
func (m *BlockToBaseStateRoot) Get(blockHash common.Hash) (common.Hash, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	root, exists := m.blockHashToStateRoot[blockHash]
	return root, exists
}

// Has checks if mapping exists for the given block hash
func (m *BlockToBaseStateRoot) Has(blockHash common.Hash) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	_, exists := m.blockHashToStateRoot[blockHash]
	return exists
}

// Store persists the mapping to the database
func (m *BlockToBaseStateRoot) Store(db ethdb.Database) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	batch := db.NewBatch()
	count := 0

	for blockHash, stateRoot := range m.blockHashToStateRoot {
		key := append([]byte("verkle-base-state:"), blockHash.Bytes()...)
		if err := batch.Put(key, stateRoot.Bytes()); err != nil {
			return err
		}
		count++
	}

	if err := batch.Write(); err != nil {
		return err
	}

	log.Debug("Stored verkle transition mappings", "count", count)
	return nil
}

// Load loads the mapping from the database
func LoadBlockToBaseStateRoot(db ethdb.Database) (*BlockToBaseStateRoot, error) {
	mapping := NewBlockToBaseStateRootMapping()

	it := db.NewIterator([]byte("verkle-base-state:"), nil)
	defer it.Release()

	count := 0

	for it.Next() {
		key := it.Key()
		blockHash := common.BytesToHash(key[len("verkle-base-state:"):])
		stateRoot := common.BytesToHash(it.Value())
		mapping.Add(blockHash, stateRoot)
		count++
	}

	if err := it.Error(); err != nil {
		return nil, err
	}

	log.Debug("Loaded verkle transition mappings", "count", count)
	return mapping, nil
}

package trie

import (
	"sync"

	"github.com/VictoriaMetrics/fastcache"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/trie/triedb/hashdb"
)

func (db *Database) GetLock() *sync.RWMutex {
	return db.backend.GetLock()
}

func (db *Database) GetCleans() *fastcache.Cache {
	hdb, ok := db.backend.(*hashdb.Database)
	if !ok {
		panic("only hashdb supported")
	}
	return hdb.GetCleans()
}

// EmptyRoot indicate what root is for an empty trie, it depends on its underlying implement (zktrie or common trie)
func (db *Database) EmptyRoot() common.Hash {
	if db.IsUsingZktrie() {
		return types.EmptyZkTrieRootHash
	} else {
		return types.EmptyLegacyTrieRootHash
	}
}

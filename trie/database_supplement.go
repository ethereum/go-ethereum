package trie

import (
	"sync"

	"github.com/VictoriaMetrics/fastcache"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie/triedb/hashdb"
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
		return common.Hash{}
	} else {
		return types.EmptyRootHash
	}
}

// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package hashtree

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func Print(db ethdb.Database, prefix []byte) {
	it := db.(*ethdb.LDBDatabase).NewIterator()
	defer it.Release()
	cnt := 0
	for it.Seek(prefix); it.Valid(); it.Next() {
		key := it.Key()
		if len(key) < len(prefix) || !bytes.Equal(key[:len(prefix)], prefix) {
			return
		}
		value := it.Value()
		cnt++
		fmt.Printf("CNT  %d    KEY  %x    HASH  %x    VALUE  %x\n", cnt, key[len(prefix):], crypto.Keccak256(value), value)
	}
}

type hasDataFn func(version uint64) func(position, hash []byte) bool

type GarbageCollector struct {
	db                       *ethdb.LDBDatabase
	prefix                   []byte
	hasData                  hasDataFn
	gcBlock                  uint64
	gcBlockHasData           func(position, hash []byte) bool
	delkeys                  [][]byte
	keysChecked, keysRemoved uint64
	refsChecked, refsRemoved uint64
	writeCounter             uint64
	writeLock                sync.Mutex
	valid                    bool
}

func NewGarbageCollector(db ethdb.Database, prefix []byte, hasData hasDataFn) *GarbageCollector {
	return &GarbageCollector{
		db:      db.(*ethdb.LDBDatabase),
		prefix:  prefix,
		hasData: hasData,
	}
}

func (g *GarbageCollector) run(startKey []byte, maxEntries uint64) (nextKey []byte) {
	g.writeLock.Lock()
	g.valid = true
	g.writeLock.Unlock()

	it := g.db.NewIterator()
	g.delkeys = nil

	defer func() {
		it.Release()
		g.writeLock.Lock()
		if g.valid {
			for _, key := range g.delkeys {
				g.db.Delete(key)
			}
			g.keysRemoved += uint64(len(g.delkeys))
		}
		g.writeLock.Unlock()
		var r util.Range
		if nextKey == nil {
			r = *util.BytesPrefix(g.prefix)
		} else {
			r.Limit = nextKey
		}
		r.Start = startKey
		g.db.LDB().CompactRange(r)
	}()

	g.gcBlockHasData = g.hasData(g.gcBlock)
	it.Seek(startKey)
	for it.Valid() {
		key := common.CopyBytes(it.Key())
		//log.Info("key", "key", key)
		if len(key) < len(g.prefix) || !bytes.Equal(key[:len(g.prefix)], g.prefix) {
			return nil
		}
		if maxEntries == 0 {
			nextKey = key
			return nextKey
		}

		if len(key) >= len(g.prefix)+33 && key[len(key)-1] == 0 {
			it.Next()
			var refkeys [][]byte
			for it.Valid() {
				refkey := common.CopyBytes(it.Key())
				//log.Info("ref", "key", refkey)
				if len(refkey) >= len(key) && bytes.Equal(refkey[:len(key)-1], key[:len(key)-1]) {
					if len(refkey) == len(key)+8 && refkey[len(key)+7] == 1 {
						refkeys = append(refkeys, refkey)
					} else {
						log.Error("Invalid hashtree ref", "key", refkey)
					}
					it.Next()
				} else {
					break
				}
			}
			g.gcEntry(key, refkeys)
			maxEntries--
		} else {
			log.Error("Invalid hashtree entry", "key", key)
			it.Next()
		}
	}
	return nil
}

func (g *GarbageCollector) gcEntry(key []byte, refkeys [][]byte) {
	refcount := len(refkeys)
	keylen := len(key)
	oldrefs := 0
	for oldrefs < refcount {
		version := binary.BigEndian.Uint64(refkeys[oldrefs][keylen-1 : keylen+7])
		if version >= g.gcBlock {
			break
		}
		oldrefs++
	}

	removerefs := 0
	if oldrefs > 0 {
		removerefs = oldrefs - 1
		if oldrefs == refcount && !g.gcBlockHasData(key[len(g.prefix):keylen-33], key[keylen-33:keylen-1]) {
			removerefs = refcount
		}
	}

	g.keysChecked++
	if removerefs == refcount {
		g.delkeys = append(g.delkeys, key)
	}
	g.refsChecked += uint64(refcount)
	g.refsRemoved += uint64(removerefs)
	for i := 0; i < removerefs; i++ {
		g.db.Delete(refkeys[i])
	}
}

func (g *GarbageCollector) FullGC(block uint64) {
	log.Info("Starting full GC", "block", block)
	g.gcBlock = block
	key := g.prefix
	for key != nil {
		key = g.run(key, 10000)
		k := key
		if len(k) > 8 {
			k = k[:8]
		}
		log.Info("Running...", "key", k, "keys checked", g.keysChecked, "keys removed", g.keysRemoved, "refs checked", g.refsChecked, "refs removed", g.refsRemoved)
	}
	log.Info("Finished full GC", "keys checked", g.keysChecked, "keys removed", g.keysRemoved, "refs checked", g.refsChecked, "refs removed", g.refsRemoved)
}

func (g *GarbageCollector) BackgroundGC(currentBlock func() *types.Block, processing, stop *int32, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		var gcCounter uint64
		key := g.prefix

		for atomic.LoadInt32(stop) == 0 {
			wc := atomic.LoadUint64(&g.writeCounter)
			diff := wc - gcCounter
			if diff > 10000 {
				gcCounter = wc - 10000
				diff = 10000
			}
			if diff >= 100 && atomic.LoadInt32(processing) == 0 {
				gcCounter += 100
				if key == nil {
					key = g.prefix
				}
				headBlock := currentBlock().NumberU64()
				if headBlock > 1000 {
					g.gcBlock = headBlock - 1000
					key = g.run(key, 1000)
					k := key
					if len(k) > 8 {
						k = k[:8]
					}
					log.Info("Running GC...", "key", k, "keys checked", g.keysChecked, "keys removed", g.keysRemoved, "refs checked", g.refsChecked, "refs removed", g.refsRemoved)
				}
			} else {
				time.Sleep(time.Second)
			}
		}
	}()
}

func (g *GarbageCollector) LockWrite() {
	g.writeLock.Lock()
	g.valid = false
}

func (g *GarbageCollector) UnlockWrite() {
	g.writeLock.Unlock()
}

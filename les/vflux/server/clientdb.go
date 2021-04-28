// Copyright 2020 The go-ethereum Authors
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

package server

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
	lru "github.com/hashicorp/golang-lru"
)

const (
	balanceCacheLimit = 8192 // the maximum number of cached items in service token balance queue

	// nodeDBVersion is the version identifier of the node data in db
	//
	// Changelog:
	// Version 0 => 1
	// * Replace `lastTotal` with `meta` in positive balance: version 0=>1
	//
	// Version 1 => 2
	// * Positive Balance and negative balance is changed:
	// * Cumulative time is replaced with expiration
	//
	// Version 2 => 3
	// * Positive balance is stored together with a serial number
	// * Currency balance is added
	nodeDBVersion = 3

	// dbCleanupCycle is the cycle of db for useless data cleanup
	dbCleanupCycle = time.Hour
)

var (
	positiveBalancePrefix = []byte("pb:")         // dbVersion(uint16 big endian) + positiveBalancePrefix + id -> balance + serialNumber
	negativeBalancePrefix = []byte("nb:")         // dbVersion(uint16 big endian) + negativeBalancePrefix + ip -> balance
	currencyBalancePrefix = []byte("cb:")         // dbVersion(uint16 big endian) + curBalancePrefix + currencyId + ":" + address -> currency balance
	expirationKey         = []byte("expiration:") // dbVersion(uint16 big endian) + expirationKey -> posExp, negExp
)

type nodeDB struct {
	db            ethdb.KeyValueStore
	cache         *lru.Cache
	auxbuf        []byte                                              // 37-byte auxiliary buffer for key encoding
	verbuf        [2]byte                                             // 2-byte auxiliary buffer for db version
	evictCallBack func(mclock.AbsTime, bool, utils.ExpiredValue) bool // Callback to determine whether the balance can be evicted.
	clock         mclock.Clock
	closeCh       chan struct{}
	cleanupHook   func() // Test hook used for testing
}

func newNodeDB(db ethdb.KeyValueStore, clock mclock.Clock) *nodeDB {
	cache, _ := lru.New(balanceCacheLimit)
	ndb := &nodeDB{
		db:      db,
		cache:   cache,
		auxbuf:  make([]byte, 37),
		clock:   clock,
		closeCh: make(chan struct{}),
	}
	binary.BigEndian.PutUint16(ndb.verbuf[:], uint16(nodeDBVersion))
	go ndb.expirer()
	return ndb
}

func (db *nodeDB) close() {
	close(db.closeCh)
}

func (db *nodeDB) getPrefix(neg bool) []byte {
	prefix := positiveBalancePrefix
	if neg {
		prefix = negativeBalancePrefix
	}
	return append(db.verbuf[:], prefix...)
}

func (db *nodeDB) key(id []byte, neg bool) []byte {
	prefix := positiveBalancePrefix
	if neg {
		prefix = negativeBalancePrefix
	}
	if len(prefix)+len(db.verbuf)+len(id) > len(db.auxbuf) {
		db.auxbuf = append(db.auxbuf, make([]byte, len(prefix)+len(db.verbuf)+len(id)-len(db.auxbuf))...)
	}
	copy(db.auxbuf[:len(db.verbuf)], db.verbuf[:])
	copy(db.auxbuf[len(db.verbuf):len(db.verbuf)+len(prefix)], prefix)
	copy(db.auxbuf[len(prefix)+len(db.verbuf):len(prefix)+len(db.verbuf)+len(id)], id)
	return db.auxbuf[:len(prefix)+len(db.verbuf)+len(id)]
}

func (db *nodeDB) getExpiration() (utils.Fixed64, utils.Fixed64) {
	blob, err := db.db.Get(append(db.verbuf[:], expirationKey...))
	if err != nil || len(blob) != 16 {
		return 0, 0
	}
	return utils.Fixed64(binary.BigEndian.Uint64(blob[:8])), utils.Fixed64(binary.BigEndian.Uint64(blob[8:16]))
}

func (db *nodeDB) setExpiration(pos, neg utils.Fixed64) {
	var buff [16]byte
	binary.BigEndian.PutUint64(buff[:8], uint64(pos))
	binary.BigEndian.PutUint64(buff[8:16], uint64(neg))
	db.db.Put(append(db.verbuf[:], expirationKey...), buff[:16])
}

type storedPosBalance struct {
	Balance      utils.ExpiredValue
	SerialNumber uint64
}

func (db *nodeDB) getOrNewPosBalance(id []byte) (utils.ExpiredValue, uint64) {
	key := db.key(id, false)
	item, exist := db.cache.Get(string(key))
	if exist {
		b := item.(storedPosBalance)
		return b.Balance, b.SerialNumber
	}
	enc, err := db.db.Get(key)
	if err != nil || len(enc) == 0 {
		return utils.ExpiredValue{}, 0
	}
	var b storedPosBalance
	if err := rlp.DecodeBytes(enc, &b); err != nil {
		log.Crit("Failed to decode positive balance", "err", err)
	}
	db.cache.Add(string(key), b)
	return b.Balance, b.SerialNumber
}

func (db *nodeDB) getOrNewNegBalance(id []byte) utils.ExpiredValue {
	key := db.key(id, true)
	item, exist := db.cache.Get(string(key))
	if exist {
		return item.(utils.ExpiredValue)
	}
	var b utils.ExpiredValue
	enc, err := db.db.Get(key)
	if err != nil || len(enc) == 0 {
		return b
	}
	if err := rlp.DecodeBytes(enc, &b); err != nil {
		log.Crit("Failed to decode negative balance", "err", err)
	}
	db.cache.Add(string(key), b)
	return b
}

func (db *nodeDB) setPosBalance(batch ethdb.KeyValueWriter, id []byte, balance utils.ExpiredValue, serialNumber uint64) {
	b := storedPosBalance{balance, serialNumber}
	key := db.key(id, false)
	enc, err := rlp.EncodeToBytes(&(b))
	if err != nil {
		log.Crit("Failed to encode positive balance", "err", err)
	}
	if batch != nil {
		batch.Put(key, enc)
	} else {
		db.db.Put(key, enc)
	}
	db.cache.Add(string(key), b)
}

func (db *nodeDB) setNegBalance(batch ethdb.KeyValueWriter, id []byte, b utils.ExpiredValue) {
	key := db.key(id, true)
	enc, err := rlp.EncodeToBytes(&(b))
	if err != nil {
		log.Crit("Failed to encode negative balance", "err", err)
	}
	if batch != nil {
		batch.Put(key, enc)
	} else {
		db.db.Put(key, enc)
	}
	db.cache.Add(string(key), b)
}

func (db *nodeDB) delPosBalance(batch ethdb.KeyValueWriter, id []byte) {
	key := db.key(id, false)
	if batch != nil {
		batch.Delete(key)
	} else {
		db.db.Delete(key)
	}
	db.cache.Add(string(key), storedPosBalance{})
}

func (db *nodeDB) delNegBalance(batch ethdb.KeyValueWriter, id []byte) {
	key := db.key(id, true)
	if batch != nil {
		batch.Delete(key)
	} else {
		db.db.Delete(key)
	}
	db.cache.Add(string(key), utils.ExpiredValue{})
}

// getPosBalanceIDs returns a lexicographically ordered list of IDs of accounts
// with a positive balance
func (db *nodeDB) getPosBalanceIDs(start, stop enode.ID, maxCount int) (result []enode.ID) {
	if maxCount <= 0 {
		return
	}
	prefix := db.getPrefix(false)
	keylen := len(prefix) + len(enode.ID{})

	it := db.db.NewIterator(prefix, start.Bytes())
	defer it.Release()

	for it.Next() {
		var id enode.ID
		if len(it.Key()) != keylen {
			return
		}
		copy(id[:], it.Key()[keylen-len(id):])
		if bytes.Compare(id.Bytes(), stop.Bytes()) >= 0 {
			return
		}
		result = append(result, id)
		if len(result) == maxCount {
			return
		}
	}
	return
}

// forEachBalance iterates all balances and passes values to callback.
func (db *nodeDB) forEachBalance(neg bool, callback func(id enode.ID, balance utils.ExpiredValue) bool) {
	prefix := db.getPrefix(neg)
	keylen := len(prefix) + len(enode.ID{})

	it := db.db.NewIterator(prefix, nil)
	defer it.Release()

	for it.Next() {
		var id enode.ID
		if len(it.Key()) != keylen {
			return
		}
		copy(id[:], it.Key()[keylen-len(id):])

		var b storedPosBalance
		if err := rlp.DecodeBytes(it.Value(), &b); err != nil {
			continue
		}
		if !callback(id, b.Balance) {
			return
		}
	}
}

func (db *nodeDB) expirer() {
	for {
		select {
		case <-db.clock.After(dbCleanupCycle):
			db.expireNodes()
		case <-db.closeCh:
			return
		}
	}
}

// expireNodes iterates the whole node db and checks whether the
// token balances can be deleted.
func (db *nodeDB) expireNodes() {
	var (
		visited int
		deleted int
		start   = time.Now()
	)
	for _, neg := range []bool{false, true} {
		iter := db.db.NewIterator(db.getPrefix(neg), nil)
		for iter.Next() {
			visited++
			var balance utils.ExpiredValue
			if neg {
				if err := rlp.DecodeBytes(iter.Value(), &balance); err != nil {
					log.Crit("Failed to decode negative balance", "err", err)
				}
			} else {
				var b storedPosBalance
				if err := rlp.DecodeBytes(iter.Value(), &b); err != nil {
					log.Crit("Failed to decode negative balance", "err", err)
				}
				balance = b.Balance
			}
			if db.evictCallBack != nil && db.evictCallBack(db.clock.Now(), neg, balance) {
				deleted++
				db.db.Delete(iter.Key())
			}
		}
	}
	// Invoke testing hook if it's not nil.
	if db.cleanupHook != nil {
		db.cleanupHook()
	}
	log.Debug("Expire nodes", "visited", visited, "deleted", deleted, "elapsed", common.PrettyDuration(time.Since(start)))
}

func (db *nodeDB) currencyKey(currencyID string, address []byte) []byte {
	currencyID = currencyID + ":"
	if len(db.verbuf)+len(currencyBalancePrefix)+len(currencyID)+len(address) > len(db.auxbuf) {
		db.auxbuf = append(db.auxbuf, make([]byte, len(db.verbuf)+len(currencyBalancePrefix)+len(currencyID)+len(address)-len(db.auxbuf))...)
	}
	copy(db.auxbuf[:len(db.verbuf)], db.verbuf[:])
	copy(db.auxbuf[len(db.verbuf):len(db.verbuf)+len(currencyBalancePrefix)], currencyBalancePrefix)
	copy(db.auxbuf[len(db.verbuf)+len(currencyBalancePrefix):len(db.verbuf)+len(currencyBalancePrefix)+len(currencyID)], currencyID)
	copy(db.auxbuf[len(db.verbuf)+len(currencyBalancePrefix)+len(currencyID):len(db.verbuf)+len(currencyBalancePrefix)+len(currencyID)+len(address)], address)
	return db.auxbuf[:len(db.verbuf)+len(currencyBalancePrefix)+len(currencyID)+len(address)]
}

func (db *nodeDB) getCurrencyBalance(currencyID string, address []byte) *big.Int {
	key := db.currencyKey(currencyID, address)
	item, exist := db.cache.Get(string(key))
	if exist {
		return item.(*big.Int)
	}
	v := big.NewInt(0)
	b, _ := db.db.Get(key)
	if len(b) > 0 {
		v.SetBytes(b)
	}
	db.cache.Add(string(key), v)
	return v
}

func (db *nodeDB) setOrDelCurrencyBalance(batch ethdb.KeyValueWriter, currencyID string, address []byte, balance *big.Int) {
	key := db.currencyKey(currencyID, address)
	var b []byte
	switch balance.Sign() {
	case -1:
		log.Error("Negative currency balance", "currency", currencyID, "address", address, "balance", balance)
		balance = big.NewInt(0)
	case 1:
		b = balance.Bytes()
	}
	if batch != nil {
		if b != nil {
			batch.Put(key, b)
		} else {
			batch.Delete(key)
		}
	} else {
		if b != nil {
			db.db.Put(key, b)
		} else {
			db.db.Delete(key)
		}
	}
	db.cache.Add(string(key), balance)
}

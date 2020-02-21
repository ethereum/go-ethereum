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

package les

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
	lru "github.com/hashicorp/golang-lru"
)

const balanceCacheLimit = 8192 // the maximum number of cached items in service token balance queue

// tokenBalance is a wrapper of expiredValue which represents the service token
// balance of clients. The balance value will decay exponentially over time and
// can be deleted when the amount is small enough.
type tokenBalance struct {
	value expiredValue
}

// EncodeRLP implements rlp.Encoder
func (b *tokenBalance) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{b.value.base, b.value.exp})
}

// DecodeRLP implements rlp.Decoder
func (b *tokenBalance) DecodeRLP(s *rlp.Stream) error {
	var entry struct {
		Base, Exp uint64
	}
	if err := s.Decode(&entry); err != nil {
		return err
	}
	b.value = expiredValue{base: entry.Base, exp: entry.Exp}
	return nil
}

// currencyBalance represents the client's currency balance.
type currencyBalance struct {
	amount uint64
	typ    string
}

// EncodeRLP implements rlp.Encoder
func (b *currencyBalance) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{b.amount, b.typ})
}

// DecodeRLP implements rlp.Decoder
func (b *currencyBalance) DecodeRLP(s *rlp.Stream) error {
	var entry struct {
		Amount uint64
		Type   string
	}
	if err := s.Decode(&entry); err != nil {
		return err
	}
	b.amount, b.typ = entry.Amount, entry.Type
	return nil
}

const (
	// nodeDBVersion is the version identifier of the node data in db
	//
	// Changelog:
	// * Replace `lastTotal` with `meta` in positive balance: version 0=>1
	// * Rework balance, add currency balance: version 1=>2
	nodeDBVersion = 2

	// dbCleanupCycle is the cycle of db for useless data cleanup
	dbCleanupCycle = time.Hour
)

var (
	posBalancePrefix = []byte("pb:")         // dbVersion(uint16 big endian) + posBalancePrefix + id -> positive balance
	negBalancePrefix = []byte("nb:")         // dbVersion(uint16 big endian) + negBalancePrefix + ip -> negative balance
	curBalancePrefix = []byte("cb:")         // dbVersion(uint16 big endian) + curBalancePrefix + id -> currency balance
	expirationKey    = []byte("expiration:") // dbVersion(uint16 big endian) + expirationKey -> posExp, negExp
)

type nodeDB struct {
	db            ethdb.Database
	cache         *lru.Cache
	clock         mclock.Clock
	closeCh       chan struct{}
	evictCallBack func(mclock.AbsTime, bool, tokenBalance) bool // Callback to determine whether the balance can be evicted.
	cleanupHook   func()                                        // Test hook used for testing
}

func newNodeDB(db ethdb.Database, clock mclock.Clock) *nodeDB {
	var buff [2]byte
	binary.BigEndian.PutUint16(buff[:], uint16(nodeDBVersion))

	cache, _ := lru.New(balanceCacheLimit)
	ndb := &nodeDB{
		db:      rawdb.NewTable(db, string(buff[:])),
		cache:   cache,
		clock:   clock,
		closeCh: make(chan struct{}),
	}
	go ndb.expirer()
	return ndb
}

func (db *nodeDB) close() {
	close(db.closeCh)
}

func (db *nodeDB) key(id []byte, neg bool) []byte {
	prefix := posBalancePrefix
	if neg {
		prefix = negBalancePrefix
	}
	return append(prefix, id...)
}

func (db *nodeDB) getExpiration() (float64, float64) {
	blob, err := db.db.Get(expirationKey)
	if err != nil || len(blob) != 16 {
		return 0, 0
	}
	return math.Float64frombits(binary.BigEndian.Uint64(blob[:8])), math.Float64frombits(binary.BigEndian.Uint64(blob[8:16]))
}

func (db *nodeDB) setExpiration(pos, neg float64) {
	var buff [16]byte
	binary.BigEndian.PutUint64(buff[:8], math.Float64bits(pos))
	binary.BigEndian.PutUint64(buff[8:16], math.Float64bits(neg))
	db.db.Put(expirationKey, buff[:16])
}

func (db *nodeDB) getCurrencyBalance(id enode.ID) currencyBalance {
	var b currencyBalance
	enc, err := db.db.Get(append(curBalancePrefix, id.Bytes()...))
	if err != nil || len(enc) == 0 {
		return b
	}
	if err := rlp.DecodeBytes(enc, &b); err != nil {
		log.Crit("Failed to decode positive balance", "err", err)
	}
	return b
}

func (db *nodeDB) setCurrencyBalance(id enode.ID, b currencyBalance) {
	enc, err := rlp.EncodeToBytes(&(b))
	if err != nil {
		log.Crit("Failed to encode currency balance", "err", err)
	}
	db.db.Put(append(curBalancePrefix, id.Bytes()...), enc)
}

func (db *nodeDB) getOrNewBalance(id []byte, neg bool) tokenBalance {
	key := db.key(id, neg)
	item, exist := db.cache.Get(string(key))
	if exist {
		return item.(tokenBalance)
	}
	var b tokenBalance
	enc, err := db.db.Get(key)
	if err != nil || len(enc) == 0 {
		return b
	}
	if err := rlp.DecodeBytes(enc, &b); err != nil {
		log.Crit("Failed to decode positive balance", "err", err)
	}
	db.cache.Add(string(key), b)
	return b
}

func (db *nodeDB) setBalance(id []byte, neg bool, b tokenBalance) {
	key := db.key(id, neg)
	enc, err := rlp.EncodeToBytes(&(b))
	if err != nil {
		log.Crit("Failed to encode positive balance", "err", err)
	}
	db.db.Put(key, enc)
	db.cache.Add(string(key), b)
}

func (db *nodeDB) delBalance(id []byte, neg bool) {
	key := db.key(id, neg)
	db.db.Delete(key)
	db.cache.Remove(string(key))
}

// getPosBalanceIDs returns a lexicographically ordered list of IDs of accounts
// with a positive balance
func (db *nodeDB) getPosBalanceIDs(start, stop enode.ID, maxCount int) (result []enode.ID) {
	if maxCount <= 0 {
		return
	}
	it := db.db.NewIteratorWithStart(db.key(start.Bytes(), false))
	defer it.Release()
	for i := len(stop[:]) - 1; i >= 0; i-- {
		stop[i]--
		if stop[i] != 255 {
			break
		}
	}
	stopKey := db.key(stop.Bytes(), false)
	keyLen := len(stopKey)

	for it.Next() {
		var id enode.ID
		if len(it.Key()) != keyLen || bytes.Compare(it.Key(), stopKey) == 1 {
			return
		}
		copy(id[:], it.Key()[keyLen-len(id):])
		result = append(result, id)
		if len(result) == maxCount {
			return
		}
	}
	return
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
// token balances can deleted.
func (db *nodeDB) expireNodes() {
	var (
		visited int
		deleted int
		start   = time.Now()
	)
	for index, prefix := range [][]byte{posBalancePrefix, negBalancePrefix} {
		iter := db.db.NewIteratorWithPrefix(prefix)
		for iter.Next() {
			visited += 1
			var balance tokenBalance
			if err := rlp.DecodeBytes(iter.Value(), &balance); err != nil {
				log.Crit("Failed to decode negative balance", "err", err)
			}
			if db.evictCallBack != nil && db.evictCallBack(db.clock.Now(), index != 0, balance) {
				deleted += 1
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

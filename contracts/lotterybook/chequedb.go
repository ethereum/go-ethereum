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

package lotterybook

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	lru "github.com/hashicorp/golang-lru"
)

const (
	lotteryCacheSize = 128
	chequeCacheSize  = 4096
)

var (
	lotteryPrefix    = []byte("-l") // lotteryPrefix + drawer_id(20bytes) + lottery_id(32byte) -> lottery
	tmpLotteryPrefix = []byte("-t") // tmpLotteryPrefix + drawer_id(20bytes) + lottery_id(32byte) -> lottery

	// For sender side, the schema is:
	//      chequePrefix + drawer_id(20bytes) + lottery_id(32byte) + drawee_id(20bytes) -> cheque
	// While for receiver side, the schema is:
	//      chequePrefix + drawee_id(20bytes) + lottery_id(32byte) + drawer_id(20bytes) -> cheque
	chequePrefix = []byte("-c")
)

// chequeDB keeps all signed cheques issued by payer. It's very important
// to save the cheques properly, otherwise the payee of cheques may can't claim
// the money back.
//
// Cheques are cumulatively confirmed, so only the latest version needs to be stored.
type chequeDB struct {
	lCache *lru.Cache // LRU cache for storing latest lotteries
	cCache *lru.Cache // LRU cache for storing latest cheques
	db     ethdb.Database
}

// newChequeDB intiailises the chequedb with given db handler.
func newChequeDB(db ethdb.Database) *chequeDB {
	lCache, _ := lru.New(lotteryCacheSize)
	cCache, _ := lru.New(chequeCacheSize)
	return &chequeDB{
		lCache: lCache,
		cCache: cCache,
		db:     db,
	}
}

// chequeKey returns the db key for cheque storing or retrieval.
func chequeKey(drawee, drawer common.Address, id common.Hash, sender bool) []byte {
	var key []byte
	key = append(key, chequePrefix...)
	if sender {
		// The scheme of cheque in sender side is:
		// prefix + drawer + lotteryid + drawee => cheque
		key = append(key, drawer.Bytes()...)
		key = append(key, id.Bytes()...)
		key = append(key, drawee.Bytes()...)
	} else {
		// The scheme of cheque in receiver side is:
		// prefix + drawee + lotteryid + drawer => cheque
		key = append(key, drawee.Bytes()...)
		key = append(key, id.Bytes()...)
		key = append(key, drawer.Bytes()...)
	}
	return key
}

// readCheque returns the cheque from disk with the specified identity.
func (db *chequeDB) readCheque(drawee, drawer common.Address, id common.Hash, sender bool) *Cheque {
	key := chequeKey(drawee, drawer, id, sender)
	if item, exist := db.cCache.Get(string(key)); exist {
		c := item.(Cheque)
		chequeDBCacheHitMeter.Mark(1)
		return &c
	}
	chequeDBCacheMissMeter.Mark(1)
	blob, err := db.db.Get(key)
	if err != nil {
		return nil
	}
	var cheque Cheque
	if err := rlp.DecodeBytes(blob, &cheque); err != nil {
		return nil
	}
	db.cCache.Add(string(key), cheque)
	chequeDBReadMeter.Mark(int64(len(blob)))
	return &cheque
}

// writeCheque writes the cheque into the disk with the specific identity.
func (db *chequeDB) writeCheque(drawee, drawer common.Address, cheque *Cheque, sender bool) {
	blob, err := rlp.EncodeToBytes(cheque)
	if err != nil {
		log.Crit("Failed to encode cheque", "error", err)
	}
	key := chequeKey(drawee, drawer, cheque.LotteryId, sender)
	err = db.db.Put(key, blob)
	if err != nil {
		log.Crit("Failed to store cheque", "error", err)
	}
	db.cCache.Add(string(key), *cheque)
	chequeDBWriteMeter.Mark(int64(len(blob)))
}

// deleteCheque deletes the specified cheque from disk.
func (db *chequeDB) deleteCheque(drawee, drawer common.Address, id common.Hash, sender bool) {
	key := chequeKey(drawee, drawer, id, sender)
	if err := db.db.Delete(key); err != nil {
		log.Crit("Failed to delete cheque", "error", err)
	}
	db.cCache.Remove(string(key))
}

// readLottery reads the specified lottery from disk.
func (db *chequeDB) readLottery(drawer common.Address, id common.Hash) *Lottery {
	key := append(lotteryPrefix, append(drawer.Bytes(), id.Bytes()...)...)
	if item, exist := db.lCache.Get(string(key)); exist {
		l := item.(Lottery)
		chequeDBCacheHitMeter.Mark(1)
		return &l
	}
	chequeDBCacheMissMeter.Mark(1)
	blob, err := db.db.Get(key)
	if err != nil {
		return nil
	}
	var lottery Lottery
	if err := rlp.DecodeBytes(blob, &lottery); err != nil {
		return nil
	}
	db.lCache.Add(string(key), lottery)
	chequeDBReadMeter.Mark(int64(len(blob)))
	return &lottery
}

// writeLottery writes the given lottery into disk.
func (db *chequeDB) writeLottery(drawer common.Address, id common.Hash, tmp bool, lottery *Lottery) {
	blob, err := rlp.EncodeToBytes(lottery)
	if err != nil {
		log.Crit("Failed to encode lottery", "error", err)
	}
	prefix := lotteryPrefix
	if tmp {
		prefix = tmpLotteryPrefix
	}
	key := append(prefix, append(drawer.Bytes(), id.Bytes()...)...)
	err = db.db.Put(key, blob)
	if err != nil {
		log.Crit("Failed to store lottery", "error", err)
	}
	if !tmp {
		db.lCache.Add(string(key), *lottery)
	}
	chequeDBWriteMeter.Mark(int64(len(blob)))
}

// deleteLottery deletes the specified lottery from disk
func (db *chequeDB) deleteLottery(drawer common.Address, id common.Hash, tmp bool) {
	prefix := lotteryPrefix
	if tmp {
		prefix = tmpLotteryPrefix
	}
	key := append(prefix, append(drawer.Bytes(), id.Bytes()...)...)
	if err := db.db.Delete(key); err != nil {
		log.Crit("Failed to delete lottery", "error", err)
	}
	db.lCache.Remove(string(key))
}

// listLotteries returns a list of lotteries issued by specified address.
func (db *chequeDB) listLotteries(addr common.Address, tmp bool) []*Lottery {
	prefix := lotteryPrefix
	if tmp {
		prefix = tmpLotteryPrefix
	}
	iter := db.db.NewIterator(append(prefix, addr.Bytes()...), nil)
	defer iter.Release()

	var lotteries []*Lottery
	var size int64
	for iter.Next() {
		if len(iter.Key()) != len(lotteryPrefix)+common.AddressLength+common.HashLength {
			continue
		}
		var lottery Lottery
		if err := rlp.DecodeBytes(iter.Value(), &lottery); err != nil {
			continue
		}
		lotteries = append(lotteries, &lottery)
		size += int64(len(iter.Value()) + len(iter.Key()))
	}
	chequeDBReadMeter.Mark(size)
	return lotteries
}

// listCheques returns a list of cheques associated with specified drawer/drawee
// address. Besides caller can specify a filter function to filter useless data.
//
// If the caller is cheque sender, all returned addresses are a batch of receiver
// addresses, otherwise the addresses are senders.
func (db *chequeDB) listCheques(addr common.Address, filterFn func(common.Address, common.Hash, *Cheque) bool) (cheques []*Cheque, addresses []common.Address) {
	iter := db.db.NewIterator(append(chequePrefix, addr.Bytes()...), nil)
	defer iter.Release()

	var size int64
	for iter.Next() {
		if len(iter.Key()) != len(chequePrefix)+2*common.AddressLength+common.HashLength {
			continue
		}
		id := iter.Key()[len(iter.Key())-common.HashLength-common.AddressLength : len(iter.Key())-common.AddressLength]
		addr := iter.Key()[len(iter.Key())-common.AddressLength:]
		var cheque Cheque
		if err := rlp.DecodeBytes(iter.Value(), &cheque); err != nil {
			continue
		}
		if filterFn != nil && !filterFn(common.BytesToAddress(addr), common.BytesToHash(id), &cheque) {
			continue
		}
		cheques = append(cheques, &cheque)
		addresses = append(addresses, common.BytesToAddress(addr))
		size += int64(len(iter.Value()) + len(iter.Key()))
	}
	chequeDBReadMeter.Mark(size)
	return
}

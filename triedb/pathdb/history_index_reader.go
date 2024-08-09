// Copyright 2023 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/

package pathdb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

type blockReader struct {
	restarts []uint32
	buf      []byte
}

func parseIndexBlock(blob []byte) ([]uint32, []byte, error) {
	if len(blob) < 4 {
		return nil, nil, fmt.Errorf("corrupted index block, len: %d", len(blob))
	}
	restartLen := binary.BigEndian.Uint32(blob[len(blob)-4:])
	if restartLen == 0 {
		return nil, nil, errors.New("corrupted index block, no restart")
	}
	tailLen := int(restartLen+1) * 4
	if len(blob) < tailLen {
		return nil, nil, fmt.Errorf("truncated restarts, size: %d, restarts: %d", len(blob), restartLen)
	}
	restarts := make([]uint32, 0, restartLen)
	for i := restartLen; i > 0; i-- {
		restart := binary.BigEndian.Uint32(blob[len(blob)-int(i+1)*4:])
		restarts = append(restarts, restart)
	}
	prev := restarts[0]
	for i := 1; i < len(restarts); i++ {
		if restarts[i] <= prev {
			return nil, nil, fmt.Errorf("restart out of order, prev: %d, next: %d", prev, restarts[i])
		}
		if int(restarts[i]) >= len(blob)-tailLen {
			return nil, nil, fmt.Errorf("invalid restart position, restart: %d, size: %d", restarts[i], len(blob)-tailLen)
		}
		prev = restarts[i]
	}
	return restarts, blob[:len(blob)-tailLen], nil
}

func newBlockReader(disk ethdb.KeyValueReader, addr common.Address, state common.Hash, id uint32) (*blockReader, error) {
	blob := rawdb.ReadStateIndexBlock(disk, addr, state, id)
	if len(blob) == 0 {
		return nil, errors.New("index block is not present")
	}
	restarts, data, err := parseIndexBlock(blob)
	if err != nil {
		return nil, err
	}
	return &blockReader{
		restarts: restarts,
		buf:      data, // safe to own the slice
	}, nil
}

func (br *blockReader) readGreaterThan(id uint64) (uint64, error) {
	var err error
	index := sort.Search(len(br.restarts), func(i int) bool {
		item, n := binary.Uvarint(br.buf[br.restarts[i]:])
		if n <= 0 {
			err = errors.New("failed to decode item at restart point")
		}
		return item > id
	})
	if err != nil {
		return 0, err
	}
	if index == 0 {
		item, _ := binary.Uvarint(br.buf[br.restarts[0]:])
		return item, nil
	}
	var (
		start  int
		limit  int
		result uint64
	)
	if index == len(br.restarts) {
		start = int(br.restarts[len(br.restarts)-1])
		limit = len(br.buf)
	} else {
		start = int(br.restarts[index-1])
		limit = int(br.restarts[index])
	}
	pos := start
	for pos < limit {
		x, n := binary.Uvarint(br.buf[pos:])
		if pos == start {
			result = x
		} else {
			result += x
		}
		if result > id {
			return result, nil
		}
		pos += n
	}
	return 0, errors.New("not found")
}

type indexReader struct {
	disk     ethdb.KeyValueReader
	descList []*indexBlockDesc
	readers  map[uint32]*blockReader
	owner    common.Address
	state    common.Hash
}

func parseIndex(blob []byte) ([]*indexBlockDesc, error) {
	if len(blob) == 0 {
		return nil, errors.New("state index not found")
	}
	if len(blob)%indexBlockDescSize != 0 {
		return nil, fmt.Errorf("corrupted state index, len: %d", len(blob))
	}
	var descList []*indexBlockDesc
	for i := 0; i < len(blob)/indexBlockDescSize; i++ {
		var desc indexBlockDesc
		desc.decode(blob[i*indexBlockDescSize : (i+1)*indexBlockDescSize])
		if desc.empty() {
			return nil, errors.New("empty state index block")
		}
		descList = append(descList, &desc)
	}
	return descList, nil
}

func newIndexReader(disk ethdb.KeyValueReader, owner common.Address, state common.Hash) (*indexReader, error) {
	descList, err := parseIndex(rawdb.ReadStateIndex(disk, owner, state))
	if err != nil {
		return nil, err
	}
	return &indexReader{
		descList: descList,
		readers:  make(map[uint32]*blockReader),
		disk:     disk,
		owner:    owner,
		state:    state,
	}, nil
}

func (r *indexReader) readGreaterThan(id uint64) (uint64, error) {
	index := sort.Search(len(r.descList), func(i int) bool {
		return id < r.descList[i].max
	})
	if index == len(r.descList) {
		return math.MaxUint64, nil
	}
	desc := r.descList[index]

	br, ok := r.readers[desc.id]
	if !ok {
		var err error
		br, err = newBlockReader(r.disk, r.owner, r.state, desc.id)
		if err != nil {
			return 0, err
		}
		r.readers[desc.id] = br
	}
	return br.readGreaterThan(id)
}

type historyReader struct {
	disk    ethdb.KeyValueReader
	freezer ethdb.AncientReader
	readers map[string]*indexReader
}

func newHistoryReader(disk ethdb.KeyValueReader, freezer ethdb.AncientReader) *historyReader {
	return &historyReader{
		disk:    disk,
		freezer: freezer,
		readers: make(map[string]*indexReader),
	}
}

func (r *historyReader) findAccount(account common.Address, id uint64, resolve func([]byte)) error {
	blob := rawdb.ReadStateAccountIndex(r.freezer, id)
	if len(blob)%accountIndexSize != 0 {
		return errors.New("corrupted account index")
	}
	n := len(blob) / accountIndexSize

	index := sort.Search(n, func(i int) bool {
		h := blob[accountIndexSize*i : accountIndexSize*i+common.HashLength]
		return bytes.Compare(h, account.Bytes()) >= 0
	})
	if index == n {
		return errors.New("account is not found")
	}
	if account != common.BytesToAddress(blob[accountIndexSize*index:accountIndexSize*index+common.AddressLength]) {
		return errors.New("account is not found")
	}
	resolve(blob[accountIndexSize*index : accountIndexSize*(index+1)])
	return nil
}

func (r *historyReader) findStorage(storageHash common.Hash, id uint64, slotOffset, slotLength int, resolve func([]byte)) error {
	blob := rawdb.ReadStateStorageIndex(r.freezer, id)
	if len(blob)%slotIndexSize != 0 {
		return errors.New("storage indices are not corrupted")
	}
	if slotIndexSize*(slotOffset+slotLength) > len(blob) {
		return errors.New("out of slice")
	}
	subSlice := blob[slotIndexSize*slotOffset : slotIndexSize*(slotOffset+slotLength)]

	index := sort.Search(slotLength, func(i int) bool {
		slotHash := subSlice[slotIndexSize*i : slotIndexSize*i+common.HashLength]
		return bytes.Compare(slotHash, storageHash.Bytes()) >= 0
	})
	if index == slotLength {
		return errors.New("storage is not found")
	}
	if storageHash != common.BytesToHash(subSlice[slotIndexSize*index:slotIndexSize*index+common.HashLength]) {
		return errors.New("storage is not found")
	}
	resolve(subSlice[slotIndexSize*index : slotIndexSize*(index+1)])
	return nil
}

func (r *historyReader) resolveAccount(accountHash common.Address, id uint64) ([]byte, error) {
	var (
		offset int
		length int
	)
	err := r.findAccount(accountHash, id, func(blob []byte) {
		length = int(blob[common.AddressLength])
		offset = int(binary.BigEndian.Uint32(blob[common.AddressLength+1 : common.AddressLength+5]))
	})
	if err != nil {
		return nil, err
	}
	// TODO(rj493456442) optimize it with partial read
	data := rawdb.ReadStateAccountHistory(r.freezer, id)
	if len(data) < offset+length {
		return nil, errors.New("corrupted account data")
	}
	return data[offset : offset+length], nil
}

func (r *historyReader) resolveStorage(account common.Address, storageHash common.Hash, id uint64) ([]byte, error) {
	var (
		slotOffset int
		slotLength int
		offset     int
		length     int
	)
	err := r.findAccount(account, id, func(blob []byte) {
		slotOffset = int(binary.BigEndian.Uint32(blob[common.AddressLength+5 : common.AddressLength+9]))
		slotLength = int(binary.BigEndian.Uint32(blob[common.AddressLength+9 : common.AddressLength+13]))
	})
	if err != nil {
		return nil, err
	}
	err = r.findStorage(storageHash, id, slotOffset, slotLength, func(blob []byte) {
		length = int(blob[common.HashLength])
		offset = int(binary.BigEndian.Uint32(blob[common.HashLength+1 : common.HashLength+5]))
	})
	if err != nil {
		return nil, err
	}
	// TODO(rj493456442) optimize it with partial read
	data := rawdb.ReadStateStorageHistory(r.freezer, id)
	if len(data) < offset+length {
		return nil, errors.New("corrupted storage data")
	}
	return data[offset : offset+length], nil
}

func (r *historyReader) resolve(owner common.Address, state common.Hash, id uint64) ([]byte, error) {
	if state == (common.Hash{}) {
		return r.resolveAccount(owner, id)
	}
	return r.resolveStorage(owner, state, id)
}

func (r *historyReader) read(owner common.Address, state common.Hash, targetID uint64, latestID uint64, latestValue []byte) ([]byte, error) {
	tail, err := r.freezer.Tail()
	if err != nil {
		return nil, err
	}
	// targetID == tail is allowed, as the first history object
	// available is tail+1
	if targetID < tail {
		return nil, errors.New("historic state is pruned")
	}
	head := rawdb.ReadStateHistoryIndexHead(r.disk)

	/*
		the available range of histories is [tail+1, head]
	*/
	if head == nil || *head <= targetID {
		return nil, errors.New("state history is not fully indexed")
	}
	ir, ok := r.readers[owner.Hex()+state.Hex()]
	if !ok {
		ir, err = newIndexReader(r.disk, owner, state)
		if err != nil {
			return nil, err
		}
		r.readers[owner.Hex()+state.Hex()] = ir
	}
	targetID, err = ir.readGreaterThan(targetID)
	if err != nil {
		return nil, err
	}
	if targetID == math.MaxUint64 {
		if *head < latestID {
			return nil, errors.New("state history is not fully indexed")
		}
		return latestValue, nil
	}
	return r.resolve(owner, state, targetID)
}

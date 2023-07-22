// Copyright 2022 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package pathdb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie/triestate"
	"golang.org/x/exp/slices"
)

// State history records the state changes involved in executing a block. The
// state can be reverted to the previous version by applying the associated
// history object (state reverse diff). State history objects are kept to
// guarantee that the system can perform state rollbacks in case of deep reorg.
//
// Each state transition will generate a state history object. Note that not
// every block has a corresponding state history object. If a block performs
// no state changes whatsoever, no state is created for it. Each state history
// will have a sequentially increasing number acting as its unique identifier.
//
// The state history is written to disk (ancient store) when the corresponding
// diff layer is merged into the disk layer. At the same time, system can prune
// the oldest histories according to config.
//
//                                                        Disk State
//                                                            ^
//                                                            |
//   +------------+     +---------+     +---------+     +---------+
//   | Init State |---->| State 1 |---->|   ...   |---->| State n |
//   +------------+     +---------+     +---------+     +---------+
//
//                     +-----------+      +------+     +-----------+
//                     | History 1 |----> | ...  |---->| History n |
//                     +-----------+      +------+     +-----------+
//
// # Rollback
//
// If the system wants to roll back to a previous state n, it needs to ensure
// all history objects from n+1 up to the current disk layer are existent. The
// history objects are applied to the state in reverse order, starting from the
// current disk layer.

const (
	accountIndexSize = common.AddressLength + 13 // The length of encoded account index
	storageIndexSize = common.HashLength + 5     // The length of encoded slot index
	historyMetaSize  = 9 + 2*common.HashLength   // The length of fixed size part of meta object

	stateHistoryVersion = uint8(0) // initial version of state history structure.
)

// accountIndex describes the metadata belongs to an account.
type accountIndex struct {
	Address       common.Address // The address of account
	Length        uint8          // The length of account data, size limited by 255
	Offset        uint32         // The offset of item in account data table
	StorageOffset uint32         // The offset of storage index in storage index table
	StorageNumber uint32         // The number of storage slots
}

// encode packs account index into byte stream.
func (i *accountIndex) encode() []byte {
	var buf [accountIndexSize]byte
	copy(buf[:], i.Address.Bytes())
	buf[common.AddressLength] = i.Length
	binary.BigEndian.PutUint32(buf[common.AddressLength+1:], i.Offset)
	binary.BigEndian.PutUint32(buf[common.AddressLength+5:], i.StorageOffset)
	binary.BigEndian.PutUint32(buf[common.AddressLength+9:], i.StorageNumber)
	return buf[:]
}

// decode unpacks account index from byte stream.
func (i *accountIndex) decode(blob []byte) {
	i.Address = common.BytesToAddress(blob[:common.AddressLength])
	i.Length = blob[common.AddressLength]
	i.Offset = binary.BigEndian.Uint32(blob[common.AddressLength+1:])
	i.StorageOffset = binary.BigEndian.Uint32(blob[common.AddressLength+5:])
	i.StorageNumber = binary.BigEndian.Uint32(blob[common.AddressLength+9:])
}

// slotIndex describes the metadata belongs to a storage slot.
type slotIndex struct {
	Hash   common.Hash // The hash of slot key
	Length uint8       // The length of storage slot, up to 32 bytes defined in protocol
	Offset uint32      // The offset of item in storage slot data table
}

// encode packs slot index into byte stream.
func (i *slotIndex) encode() []byte {
	var buf [storageIndexSize]byte
	copy(buf[:common.HashLength], i.Hash.Bytes())
	buf[common.HashLength] = i.Length
	binary.BigEndian.PutUint32(buf[common.HashLength+1:], i.Offset)
	return buf[:]
}

// decode unpack slot index from the byte stream.
func (i *slotIndex) decode(blob []byte) {
	i.Hash = common.BytesToHash(blob[:common.HashLength])
	i.Length = blob[common.HashLength]
	i.Offset = binary.BigEndian.Uint32(blob[common.HashLength+1:])
}

// meta describes the meta data of state history object.
type meta struct {
	version     uint8            // version tag of history object
	parent      common.Hash      // prev-state root before the state transition
	root        common.Hash      // post-state root after the state transition
	blockNumber uint64           // associated block number
	incomplete  []common.Address // list of address whose storage set is incomplete
}

// encode packs the meta object into byte stream.
func (m *meta) encode() []byte {
	buf := make([]byte, historyMetaSize+len(m.incomplete)*common.AddressLength)
	buf[0] = m.version
	copy(buf[1:1+common.HashLength], m.parent.Bytes())
	copy(buf[1+common.HashLength:1+2*common.HashLength], m.root.Bytes())
	binary.BigEndian.PutUint64(buf[1+2*common.HashLength:historyMetaSize], m.blockNumber)
	for i, h := range m.incomplete {
		copy(buf[i*common.AddressLength+historyMetaSize:], h.Bytes())
	}
	return buf[:]
}

// decode unpacks the meta object from byte stream.
func (m *meta) decode(blob []byte) error {
	if len(blob) < 1 {
		return fmt.Errorf("no version tag")
	}
	switch blob[0] {
	case stateHistoryVersion:
		if len(blob) < historyMetaSize {
			return fmt.Errorf("invalid state history meta, len: %d", len(blob))
		}
		if (len(blob)-historyMetaSize)%common.AddressLength != 0 {
			return fmt.Errorf("corrupted state history meta, len: %d", len(blob))
		}
		m.version = blob[0]
		m.parent = common.BytesToHash(blob[1 : 1+common.HashLength])
		m.root = common.BytesToHash(blob[1+common.HashLength : 1+2*common.HashLength])
		m.blockNumber = binary.BigEndian.Uint64(blob[1+2*common.HashLength : historyMetaSize])
		for pos := historyMetaSize; pos < len(blob); {
			m.incomplete = append(m.incomplete, common.BytesToAddress(blob[pos:pos+common.AddressLength]))
			pos += common.AddressLength
		}
		return nil
	default:
		return fmt.Errorf("unknown version %d", blob[0])
	}
}

// history represents a set of state changes belong to a block along with
// the metadata including the state roots involved in the state transition.
// State history objects in disk are linked with each other by a unique id
// (8-bytes integer), the oldest state history object can be pruned on demand
// in order to control the storage size.
type history struct {
	meta        *meta                                     // Meta data of history
	accounts    map[common.Address][]byte                 // Account data keyed by its address hash
	accountList []common.Address                          // Sorted account hash list
	storages    map[common.Address]map[common.Hash][]byte // Storage data keyed by its address hash and slot hash
	storageList map[common.Address][]common.Hash          // Sorted slot hash list
}

// newHistory constructs the state history object with provided state change set.
func newHistory(root common.Hash, parent common.Hash, blockNumber uint64, states *triestate.Set) *history {
	var (
		accountList []common.Address
		storageList = make(map[common.Address][]common.Hash)
		incomplete  []common.Address
	)
	for hash := range states.Accounts {
		accountList = append(accountList, hash)
	}
	slices.SortFunc(accountList, func(a, b common.Address) bool { return a.Less(b) })

	for addr, slots := range states.Storages {
		for slotHash := range slots {
			storageList[addr] = append(storageList[addr], slotHash)
		}
		slices.SortFunc(storageList[addr], func(a, b common.Hash) bool { return a.Less(b) })
	}
	for addr := range states.Incomplete {
		incomplete = append(incomplete, addr)
	}
	slices.SortFunc(incomplete, func(a, b common.Address) bool { return a.Less(b) })

	return &history{
		meta: &meta{
			version:     stateHistoryVersion,
			parent:      parent,
			root:        root,
			blockNumber: blockNumber,
			incomplete:  incomplete,
		},
		accounts:    states.Accounts,
		accountList: accountList,
		storages:    states.Storages,
		storageList: storageList,
	}
}

// encode serializes the state history and returns four byte streams represent
// concatenated account/storage data, account/storage indexes respectively.
func (h *history) encode() ([]byte, []byte, []byte, []byte) {
	var (
		slotNumber     uint32 // the number of processed slots
		accountData    []byte // the buffer for concatenated account data
		storageData    []byte // the buffer for concatenated storage data
		accountIndexes []byte // the buffer for concatenated account index
		storageIndexes []byte // the buffer for concatenated storage index
	)
	for _, addr := range h.accountList {
		aIndex := accountIndex{
			Address: addr,
			Length:  uint8(len(h.accounts[addr])),
			Offset:  uint32(len(accountData)),
		}
		slots, exist := h.storages[addr]
		if exist {
			// Encode storage slots in order
			for _, slotHash := range h.storageList[addr] {
				sIndex := slotIndex{
					Hash:   slotHash,
					Length: uint8(len(h.storages[addr][slotHash])),
					Offset: uint32(len(storageData)),
				}
				storageData = append(storageData, h.storages[addr][slotHash]...)
				storageIndexes = append(storageIndexes, sIndex.encode()...)
			}
			// Fill up the storage meta in account index
			aIndex.StorageOffset = slotNumber
			aIndex.StorageNumber = uint32(len(slots))
			slotNumber += uint32(len(slots))
		}
		accountData = append(accountData, h.accounts[addr]...)
		accountIndexes = append(accountIndexes, aIndex.encode()...)
	}
	return accountData, storageData, accountIndexes, storageIndexes
}

// decoder wraps the byte streams for decoding with extra meta fields.
type decoder struct {
	accountData    []byte // the buffer for concatenated account data
	storageData    []byte // the buffer for concatenated storage data
	accountIndexes []byte // the buffer for concatenated account index
	storageIndexes []byte // the buffer for concatenated storage index

	lastAccount     common.Address // the address of last resolved account
	lastAccountRead uint32         // the read-cursor position of account data
	lastSIndexRead  uint32         // the read-cursor position of storage index
	lastStorageRead uint32         // the read-cursor position of storage data
}

// verify validates the provided byte streams for decoding state history. A few
// checks will be performed to quickly detect data corruption. The byte stream
// is regarded as corrupted if:
//
// - account indexes buffer is empty(empty state set is invalid)
// - account indexes/storage indexer buffer is not aligned
//
// note, these situations are allowed:
//
// - empty account data: all accounts were not present
// - empty storage set: no slots are modified
func (r *decoder) verify() error {
	if len(r.accountIndexes)%accountIndexSize != 0 || len(r.accountIndexes) == 0 {
		return fmt.Errorf("invalid account index, len: %d", len(r.accountIndexes))
	}
	if len(r.storageIndexes)%storageIndexSize != 0 {
		return fmt.Errorf("invalid storage index, len: %d", len(r.storageIndexes))
	}
	return nil
}

// readAccount parses the account from the byte stream with specified position.
func (r *decoder) readAccount(pos int) (accountIndex, []byte, error) {
	// Decode account index from the index byte stream.
	var index accountIndex
	if (pos+1)*accountIndexSize > len(r.accountIndexes) {
		return accountIndex{}, nil, errors.New("account data buffer is corrupted")
	}
	index.decode(r.accountIndexes[pos*accountIndexSize : (pos+1)*accountIndexSize])

	// Perform validation before parsing account data, ensure
	// - account is sorted in order in byte stream
	// - account data is strictly encoded with no gap inside
	// - account data is not out-of-slice
	if bytes.Compare(r.lastAccount.Bytes(), index.Address.Bytes()) >= 0 {
		return accountIndex{}, nil, errors.New("account is not in order")
	}
	if index.Offset != r.lastAccountRead {
		return accountIndex{}, nil, errors.New("account data buffer is gaped")
	}
	last := index.Offset + uint32(index.Length)
	if uint32(len(r.accountData)) < last {
		return accountIndex{}, nil, errors.New("account data buffer is corrupted")
	}
	data := r.accountData[index.Offset:last]

	r.lastAccount = index.Address
	r.lastAccountRead = last

	return index, data, nil
}

// readStorage parses the storages from the byte stream with specified account.
func (r *decoder) readStorage(aIndex accountIndex) ([]common.Hash, map[common.Hash][]byte, error) {
	var (
		last    common.Hash
		list    []common.Hash
		storage = make(map[common.Hash][]byte)
	)
	for j := 0; j < int(aIndex.StorageNumber); j++ {
		var (
			index slotIndex
			start = (aIndex.StorageOffset + uint32(j)) * uint32(storageIndexSize)
			end   = (aIndex.StorageOffset + uint32(j+1)) * uint32(storageIndexSize)
		)
		// Perform validation before parsing storage slot data, ensure
		// - slot index is not out-of-slice
		// - slot data is not out-of-slice
		// - slot is sorted in order in byte stream
		// - slot indexes is strictly encoded with no gap inside
		// - slot data is strictly encoded with no gap inside
		if start != r.lastSIndexRead {
			return nil, nil, errors.New("storage index buffer is gapped")
		}
		if uint32(len(r.storageIndexes)) < end {
			return nil, nil, errors.New("storage index buffer is corrupted")
		}
		index.decode(r.storageIndexes[start:end])

		if last != (common.Hash{}) {
			if bytes.Compare(last.Bytes(), index.Hash.Bytes()) >= 0 {
				return nil, nil, errors.New("storage slot is not in order")
			}
		}
		if index.Offset != r.lastStorageRead {
			return nil, nil, errors.New("storage data buffer is gapped")
		}
		sEnd := index.Offset + uint32(index.Length)
		if uint32(len(r.storageData)) < sEnd {
			return nil, nil, errors.New("storage data buffer is corrupted")
		}
		storage[index.Hash] = r.storageData[r.lastStorageRead:sEnd]
		list = append(list, index.Hash)

		last = index.Hash
		r.lastSIndexRead = end
		r.lastStorageRead = sEnd
	}
	return list, storage, nil
}

// decode deserializes the account and storage data from the provided byte stream.
func (h *history) decode(accountData, storageData, accountIndexes, storageIndexes []byte) error {
	var (
		accounts    = make(map[common.Address][]byte)
		storages    = make(map[common.Address]map[common.Hash][]byte)
		accountList []common.Address
		storageList = make(map[common.Address][]common.Hash)

		r = &decoder{
			accountData:    accountData,
			storageData:    storageData,
			accountIndexes: accountIndexes,
			storageIndexes: storageIndexes,
		}
	)
	if err := r.verify(); err != nil {
		return err
	}
	for i := 0; i < len(accountIndexes)/accountIndexSize; i++ {
		// Resolve account first
		aIndex, aData, err := r.readAccount(i)
		if err != nil {
			return err
		}
		accounts[aIndex.Address] = aData
		accountList = append(accountList, aIndex.Address)

		// Resolve storage slots
		slotList, slotData, err := r.readStorage(aIndex)
		if err != nil {
			return err
		}
		if len(slotList) > 0 {
			storageList[aIndex.Address] = slotList
			storages[aIndex.Address] = slotData
		}
	}
	h.accounts = accounts
	h.accountList = accountList
	h.storages = storages
	h.storageList = storageList
	return nil
}

// readHistory reads and decodes the state history object by the given id.
func readHistory(freezer *rawdb.ResettableFreezer, id uint64) (*history, error) {
	blob := rawdb.ReadStateHistoryMeta(freezer, id)
	if len(blob) == 0 {
		return nil, fmt.Errorf("state history not found %d", id)
	}
	var m meta
	if err := m.decode(blob); err != nil {
		return nil, err
	}
	var (
		dec            = history{meta: &m}
		accountData    = rawdb.ReadStateAccountHistory(freezer, id)
		storageData    = rawdb.ReadStateStorageHistory(freezer, id)
		accountIndexes = rawdb.ReadStateAccountIndex(freezer, id)
		storageIndexes = rawdb.ReadStateStorageIndex(freezer, id)
	)
	if err := dec.decode(accountData, storageData, accountIndexes, storageIndexes); err != nil {
		return nil, err
	}
	return &dec, nil
}

// writeStateHistory writes the state history with provided state set. After
// storing the corresponding state history, it will also prune the stale
// histories from the disk with the given threshold.
func writeStateHistory(freezer *rawdb.ResettableFreezer, dl *diffLayer, limit uint64) error {
	// Short circuit if state set is not available.
	if dl.states == nil {
		return errors.New("state change set is not available")
	}
	var (
		err   error
		n     int
		start = time.Now()
		h     = newHistory(dl.rootHash, dl.parent().root(), dl.blockNumber, dl.states)
	)
	accountData, storageData, accountIndex, storageIndex := h.encode()
	dataSize := common.StorageSize(len(accountData) + len(storageData))
	indexSize := common.StorageSize(len(accountIndex) + len(storageIndex))

	// Write history data into five freezer table respectively.
	rawdb.WriteStateHistory(freezer, dl.stateID(), h.meta.encode(), accountIndex, storageIndex, accountData, storageData)

	// Prune stale state histories based on the config.
	if limit != 0 && dl.stateID() > limit {
		n, err = truncateFromTail(freezer, dl.stateID()-limit)
		if err != nil {
			return err
		}
	}
	historyDataSizeMeter.Mark(int64(dataSize))
	historyIndexSizeMeter.Mark(int64(indexSize))
	historyBuildTimeMeter.UpdateSince(start)
	log.Debug("Stored state history", "id", dl.stateID(), "blockNumber", dl.blockNumber, "data", dataSize, "index", indexSize, "pruned", n, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// checkHistories retrieves a batch of meta objects with the specified range
// and performs the callback on each item.
func checkHistories(freezer *rawdb.ResettableFreezer, start, count uint64, check func(*meta) error) error {
	for count > 0 {
		number := count
		if number > 10000 {
			number = 10000 // split the big read into small chunks
		}
		blobs, err := rawdb.ReadStateHistoryMetaList(freezer, start, number)
		if err != nil {
			return err
		}
		for _, blob := range blobs {
			var dec meta
			if err := dec.decode(blob); err != nil {
				return err
			}
			if err := check(&dec); err != nil {
				return err
			}
		}
		count -= uint64(len(blobs))
		start += uint64(len(blobs))
	}
	return nil
}

// truncateFromHead removes the extra state histories from the head with the given
// parameters. It returns the number of items removed from the head.
func truncateFromHead(freezer *rawdb.ResettableFreezer, nhead uint64) (int, error) {
	ohead, err := freezer.TruncateHead(nhead)
	if err != nil {
		return 0, err
	}
	return int(ohead - nhead), nil
}

// truncateFromTail removes the extra state histories from the tail with the given
// parameters. It returns the number of items removed from the tail.
func truncateFromTail(freezer *rawdb.ResettableFreezer, ntail uint64) (int, error) {
	otail, err := freezer.TruncateTail(ntail)
	if err != nil {
		return 0, err
	}
	return int(ntail - otail), nil
}

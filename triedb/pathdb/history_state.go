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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package pathdb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
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
	slotIndexSize    = common.HashLength + 5     // The length of encoded slot index
	historyMetaSize  = 9 + 2*common.HashLength   // The length of encoded history meta

	stateHistoryV0 = uint8(0)       // initial version of state history structure
	stateHistoryV1 = uint8(1)       // use the storage slot raw key as the identifier instead of the key hash
	historyVersion = stateHistoryV1 // the default state history version
)

// Each state history entry is consisted of five elements:
//
// # metadata
//  This object contains a few meta fields, such as the associated state root,
//  block number, version tag and so on. This object may contain an extra
//  accountHash list which means the storage changes belong to these accounts
//  are not complete due to large contract destruction. The incomplete history
//  can not be used for rollback and serving archive state request.
//
// # account index
//  This object contains some index information of account. For example, offset
//  and length indicate the location of the data belonging to the account. Besides,
//  storageOffset and storageSlots indicate the storage modification location
//  belonging to the account.
//
//  The size of each account index is *fixed*, and all indexes are sorted
//  lexicographically. Thus binary search can be performed to quickly locate a
//  specific account.
//
// # account data
//  Account data is a concatenated byte stream composed of all account data.
//  The account data can be solved by the offset and length info indicated
//  by corresponding account index.
//
//            fixed size
//         ^             ^
//        /               \
//        +-----------------+-----------------+----------------+-----------------+
//        | Account index 1 | Account index 2 |       ...      | Account index N |
//        +-----------------+-----------------+----------------+-----------------+
//        |
//        |     length
// offset |----------------+
//        v                v
//        +----------------+----------------+----------------+----------------+
//        | Account data 1 | Account data 2 |       ...      | Account data N |
//        +----------------+----------------+----------------+----------------+
//
// # storage index
//  This object is similar with account index. It's also fixed size and contains
//  the location info of storage slot data.
//
// # storage data
//  Storage data is a concatenated byte stream composed of all storage slot data.
//  The storage slot data can be solved by the location info indicated by
//  corresponding account index and storage slot index.
//
//                    fixed size
//                 ^             ^
//                /               \
//                +-----------------+-----------------+----------------+-----------------+
//                | Account index 1 | Account index 2 |       ...      | Account index N |
//                +-----------------+-----------------+----------------+-----------------+
//                |
//                |                    storage slots
// storage offset |-----------------------------------------------------+
//                v                                                     v
//                +-----------------+-----------------+-----------------+
//                | storage index 1 | storage index 2 | storage index 3 |
//                +-----------------+-----------------+-----------------+
//                |     length
//         offset |-------------+
//                v             v
//                +-------------+
//                | slot data 1 |
//                +-------------+

// accountIndex describes the metadata belonging to an account.
type accountIndex struct {
	address       common.Address // The address of account
	length        uint8          // The length of account data, size limited by 255
	offset        uint32         // The offset of item in account data table
	storageOffset uint32         // The offset of storage index in storage index table
	storageSlots  uint32         // The number of mutated storage slots belonging to the account
}

// encode packs account index into byte stream.
func (i *accountIndex) encode() []byte {
	var buf [accountIndexSize]byte
	copy(buf[:], i.address.Bytes())
	buf[common.AddressLength] = i.length
	binary.BigEndian.PutUint32(buf[common.AddressLength+1:], i.offset)
	binary.BigEndian.PutUint32(buf[common.AddressLength+5:], i.storageOffset)
	binary.BigEndian.PutUint32(buf[common.AddressLength+9:], i.storageSlots)
	return buf[:]
}

// decode unpacks account index from byte stream.
func (i *accountIndex) decode(blob []byte) {
	i.address = common.BytesToAddress(blob[:common.AddressLength])
	i.length = blob[common.AddressLength]
	i.offset = binary.BigEndian.Uint32(blob[common.AddressLength+1:])
	i.storageOffset = binary.BigEndian.Uint32(blob[common.AddressLength+5:])
	i.storageSlots = binary.BigEndian.Uint32(blob[common.AddressLength+9:])
}

// slotIndex describes the metadata belonging to a storage slot.
type slotIndex struct {
	// the identifier of the storage slot. Specifically
	// in v0, it's the hash of the raw storage slot key (32 bytes);
	// in v1, it's the raw storage slot key (32 bytes);
	id     common.Hash
	length uint8  // The length of storage slot, up to 32 bytes defined in protocol
	offset uint32 // The offset of item in storage slot data table
}

// encode packs slot index into byte stream.
func (i *slotIndex) encode() []byte {
	var buf [slotIndexSize]byte
	copy(buf[:common.HashLength], i.id.Bytes())
	buf[common.HashLength] = i.length
	binary.BigEndian.PutUint32(buf[common.HashLength+1:], i.offset)
	return buf[:]
}

// decode unpack slot index from the byte stream.
func (i *slotIndex) decode(blob []byte) {
	i.id = common.BytesToHash(blob[:common.HashLength])
	i.length = blob[common.HashLength]
	i.offset = binary.BigEndian.Uint32(blob[common.HashLength+1:])
}

// meta describes the meta data of state history object.
type meta struct {
	version uint8       // version tag of history object
	parent  common.Hash // prev-state root before the state transition
	root    common.Hash // post-state root after the state transition
	block   uint64      // associated block number
}

// encode packs the meta object into byte stream.
func (m *meta) encode() []byte {
	buf := make([]byte, historyMetaSize)
	buf[0] = m.version
	copy(buf[1:1+common.HashLength], m.parent.Bytes())
	copy(buf[1+common.HashLength:1+2*common.HashLength], m.root.Bytes())
	binary.BigEndian.PutUint64(buf[1+2*common.HashLength:historyMetaSize], m.block)
	return buf[:]
}

// decode unpacks the meta object from byte stream.
func (m *meta) decode(blob []byte) error {
	if len(blob) < 1 {
		return errors.New("no version tag")
	}
	switch blob[0] {
	case stateHistoryV0, stateHistoryV1:
		if len(blob) != historyMetaSize {
			return fmt.Errorf("invalid state history meta, len: %d", len(blob))
		}
		m.version = blob[0]
		m.parent = common.BytesToHash(blob[1 : 1+common.HashLength])
		m.root = common.BytesToHash(blob[1+common.HashLength : 1+2*common.HashLength])
		m.block = binary.BigEndian.Uint64(blob[1+2*common.HashLength : historyMetaSize])
		return nil
	default:
		return fmt.Errorf("unknown version %d", blob[0])
	}
}

// stateHistory represents a set of state changes belong to a block along with
// the metadata including the state roots involved in the state transition.
//
// State history objects in disk are linked with each other by a unique id
// (8-bytes integer), the oldest state history object can be pruned on demand
// in order to control the storage size.
type stateHistory struct {
	meta        *meta                                     // Meta data of history
	accounts    map[common.Address][]byte                 // Account data keyed by its address hash
	accountList []common.Address                          // Sorted account hash list
	storages    map[common.Address]map[common.Hash][]byte // Storage data keyed by its address hash and slot hash
	storageList map[common.Address][]common.Hash          // Sorted slot hash list
}

// newStateHistory constructs the state history object with provided states.
func newStateHistory(root common.Hash, parent common.Hash, block uint64, accounts map[common.Address][]byte, storages map[common.Address]map[common.Hash][]byte, rawStorageKey bool) *stateHistory {
	var (
		accountList = slices.SortedFunc(maps.Keys(accounts), common.Address.Cmp)
		storageList = make(map[common.Address][]common.Hash)
	)
	for addr, slots := range storages {
		storageList[addr] = slices.SortedFunc(maps.Keys(slots), common.Hash.Cmp)
	}
	version := historyVersion
	if !rawStorageKey {
		version = stateHistoryV0
	}
	return &stateHistory{
		meta: &meta{
			version: version,
			parent:  parent,
			root:    root,
			block:   block,
		},
		accounts:    accounts,
		accountList: accountList,
		storages:    storages,
		storageList: storageList,
	}
}

// typ implements the history interface, returning the historical data type held.
func (h *stateHistory) typ() historyType {
	return typeStateHistory
}

// forEach implements the history interface, returning an iterator to traverse the
// state entries in the history.
func (h *stateHistory) forEach() iter.Seq[stateIdent] {
	return func(yield func(stateIdent) bool) {
		for _, addr := range h.accountList {
			addrHash := crypto.Keccak256Hash(addr.Bytes())
			if !yield(newAccountIdent(addrHash)) {
				return
			}
			for _, slotKey := range h.storageList[addr] {
				// The hash of the storage slot key is used as the identifier because the
				// legacy history does not include the raw storage key, therefore, the
				// conversion from storage key to hash is necessary for non-v0 histories.
				slotHash := slotKey
				if h.meta.version != stateHistoryV0 {
					slotHash = crypto.Keccak256Hash(slotKey.Bytes())
				}
				if !yield(newStorageIdent(addrHash, slotHash)) {
					return
				}
			}
		}
	}
}

// stateSet returns the state set, keyed by the hash of the account address
// and the hash of the storage slot key.
func (h *stateHistory) stateSet() (map[common.Hash][]byte, map[common.Hash]map[common.Hash][]byte) {
	var (
		accounts = make(map[common.Hash][]byte)
		storages = make(map[common.Hash]map[common.Hash][]byte)
	)
	for addr, blob := range h.accounts {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		accounts[addrHash] = blob

		storage, exist := h.storages[addr]
		if !exist {
			continue
		}
		if h.meta.version == stateHistoryV0 {
			storages[addrHash] = storage
		} else {
			subset := make(map[common.Hash][]byte)
			for key, slot := range storage {
				subset[crypto.Keccak256Hash(key.Bytes())] = slot
			}
			storages[addrHash] = subset
		}
	}
	return accounts, storages
}

// encode serializes the state history and returns four byte streams represent
// concatenated account/storage data, account/storage indexes respectively.
func (h *stateHistory) encode() ([]byte, []byte, []byte, []byte) {
	var (
		slotNumber     uint32 // the number of processed slots
		accountData    []byte // the buffer for concatenated account data
		storageData    []byte // the buffer for concatenated storage data
		accountIndexes []byte // the buffer for concatenated account index
		storageIndexes []byte // the buffer for concatenated storage index
	)
	for _, addr := range h.accountList {
		accIndex := accountIndex{
			address: addr,
			length:  uint8(len(h.accounts[addr])),
			offset:  uint32(len(accountData)),
		}
		slots, exist := h.storages[addr]
		if exist {
			// Encode storage slots in order
			for _, slotHash := range h.storageList[addr] {
				sIndex := slotIndex{
					id:     slotHash,
					length: uint8(len(slots[slotHash])),
					offset: uint32(len(storageData)),
				}
				storageData = append(storageData, slots[slotHash]...)
				storageIndexes = append(storageIndexes, sIndex.encode()...)
			}
			// Fill up the storage meta in account index
			accIndex.storageOffset = slotNumber
			accIndex.storageSlots = uint32(len(slots))
			slotNumber += uint32(len(slots))
		}
		accountData = append(accountData, h.accounts[addr]...)
		accountIndexes = append(accountIndexes, accIndex.encode()...)
	}
	return accountData, storageData, accountIndexes, storageIndexes
}

// decoder wraps the byte streams for decoding with extra meta fields.
type decoder struct {
	accountData    []byte // the buffer for concatenated account data
	storageData    []byte // the buffer for concatenated storage data
	accountIndexes []byte // the buffer for concatenated account index
	storageIndexes []byte // the buffer for concatenated storage index

	lastAccount       *common.Address // the address of last resolved account
	lastAccountRead   uint32          // the read-cursor position of account data
	lastSlotIndexRead uint32          // the read-cursor position of storage slot index
	lastSlotDataRead  uint32          // the read-cursor position of storage slot data
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
	if len(r.storageIndexes)%slotIndexSize != 0 {
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
	if r.lastAccount != nil { // zero address is possible
		if bytes.Compare(r.lastAccount.Bytes(), index.address.Bytes()) >= 0 {
			return accountIndex{}, nil, errors.New("account is not in order")
		}
	}
	if index.offset != r.lastAccountRead {
		return accountIndex{}, nil, errors.New("account data buffer is gaped")
	}
	last := index.offset + uint32(index.length)
	if uint32(len(r.accountData)) < last {
		return accountIndex{}, nil, errors.New("account data buffer is corrupted")
	}
	data := r.accountData[index.offset:last]

	r.lastAccount = &index.address
	r.lastAccountRead = last

	return index, data, nil
}

// readStorage parses the storage slots from the byte stream with specified account.
func (r *decoder) readStorage(accIndex accountIndex) ([]common.Hash, map[common.Hash][]byte, error) {
	var (
		last    *common.Hash
		count   = int(accIndex.storageSlots)
		list    = make([]common.Hash, 0, count)
		storage = make(map[common.Hash][]byte, count)
	)
	for j := 0; j < count; j++ {
		var (
			index slotIndex
			start = (accIndex.storageOffset + uint32(j)) * uint32(slotIndexSize)
			end   = (accIndex.storageOffset + uint32(j+1)) * uint32(slotIndexSize)
		)
		// Perform validation before parsing storage slot data, ensure
		// - slot index is not out-of-slice
		// - slot data is not out-of-slice
		// - slot is sorted in order in byte stream
		// - slot indexes is strictly encoded with no gap inside
		// - slot data is strictly encoded with no gap inside
		if start != r.lastSlotIndexRead {
			return nil, nil, errors.New("storage index buffer is gapped")
		}
		if uint32(len(r.storageIndexes)) < end {
			return nil, nil, errors.New("storage index buffer is corrupted")
		}
		index.decode(r.storageIndexes[start:end])

		if last != nil {
			if bytes.Compare(last.Bytes(), index.id.Bytes()) >= 0 {
				return nil, nil, fmt.Errorf("storage slot is not in order, last: %x, current: %x", *last, index.id)
			}
		}
		if index.offset != r.lastSlotDataRead {
			return nil, nil, errors.New("storage data buffer is gapped")
		}
		sEnd := index.offset + uint32(index.length)
		if uint32(len(r.storageData)) < sEnd {
			return nil, nil, errors.New("storage data buffer is corrupted")
		}
		storage[index.id] = r.storageData[r.lastSlotDataRead:sEnd]
		list = append(list, index.id)

		last = &index.id
		r.lastSlotIndexRead = end
		r.lastSlotDataRead = sEnd
	}
	return list, storage, nil
}

// decode deserializes the account and storage data from the provided byte stream.
func (h *stateHistory) decode(accountData, storageData, accountIndexes, storageIndexes []byte) error {
	var (
		count       = len(accountIndexes) / accountIndexSize
		accounts    = make(map[common.Address][]byte, count)
		storages    = make(map[common.Address]map[common.Hash][]byte)
		accountList = make([]common.Address, 0, count)
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
	for i := 0; i < count; i++ {
		// Resolve account first
		accIndex, accData, err := r.readAccount(i)
		if err != nil {
			return err
		}
		accounts[accIndex.address] = accData
		accountList = append(accountList, accIndex.address)

		// Resolve storage slots
		slotList, slotData, err := r.readStorage(accIndex)
		if err != nil {
			return err
		}
		if len(slotList) > 0 {
			storageList[accIndex.address] = slotList
			storages[accIndex.address] = slotData
		}
	}
	h.accounts = accounts
	h.accountList = accountList
	h.storages = storages
	h.storageList = storageList
	return nil
}

// readStateHistoryMeta reads the metadata of state history with the specified id.
func readStateHistoryMeta(reader ethdb.AncientReader, id uint64) (*meta, error) {
	data := rawdb.ReadStateHistoryMeta(reader, id)
	if len(data) == 0 {
		return nil, fmt.Errorf("metadata is not found, %d", id)
	}
	var m meta
	err := m.decode(data)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// readStateHistory reads a single state history records with the specified id.
func readStateHistory(reader ethdb.AncientReader, id uint64) (*stateHistory, error) {
	mData, accountIndexes, storageIndexes, accountData, storageData, err := rawdb.ReadStateHistory(reader, id)
	if err != nil {
		return nil, err
	}
	var m meta
	if err := m.decode(mData); err != nil {
		return nil, err
	}
	h := stateHistory{meta: &m}
	if err := h.decode(accountData, storageData, accountIndexes, storageIndexes); err != nil {
		return nil, err
	}
	return &h, nil
}

// readStateHistories reads a list of state history records within the specified range.
func readStateHistories(freezer ethdb.AncientReader, start uint64, count uint64) ([]history, error) {
	var histories []history
	metaList, aIndexList, sIndexList, aDataList, sDataList, err := rawdb.ReadStateHistoryList(freezer, start, count)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(metaList); i++ {
		var m meta
		if err := m.decode(metaList[i]); err != nil {
			return nil, err
		}
		h := stateHistory{meta: &m}
		if err := h.decode(aDataList[i], sDataList[i], aIndexList[i], sIndexList[i]); err != nil {
			return nil, err
		}
		histories = append(histories, &h)
	}
	return histories, nil
}

// writeStateHistory persists the state history associated with the given diff layer.
func writeStateHistory(writer ethdb.AncientWriter, dl *diffLayer) error {
	// Short circuit if state set is not available.
	if dl.states == nil {
		return errors.New("state change set is not available")
	}
	var (
		start   = time.Now()
		history = newStateHistory(dl.rootHash(), dl.parentLayer().rootHash(), dl.block, dl.states.accountOrigin, dl.states.storageOrigin, dl.states.rawStorageKey)
	)
	accountData, storageData, accountIndex, storageIndex := history.encode()
	dataSize := common.StorageSize(len(accountData) + len(storageData))
	indexSize := common.StorageSize(len(accountIndex) + len(storageIndex))

	// Write history data into five freezer table respectively.
	if err := rawdb.WriteStateHistory(writer, dl.stateID(), history.meta.encode(), accountIndex, storageIndex, accountData, storageData); err != nil {
		return err
	}
	historyDataBytesMeter.Mark(int64(dataSize))
	historyIndexBytesMeter.Mark(int64(indexSize))
	historyBuildTimeMeter.UpdateSince(start)
	log.Debug("Stored state history", "id", dl.stateID(), "block", dl.block, "data", dataSize, "index", indexSize, "elapsed", common.PrettyDuration(time.Since(start)))

	return nil
}

// checkStateHistories retrieves a batch of metadata objects with the specified
// range and performs the callback on each item.
func checkStateHistories(reader ethdb.AncientReader, start, count uint64, check func(*meta) error) error {
	for count > 0 {
		number := count
		if number > 10000 {
			number = 10000 // split the big read into small chunks
		}
		blobs, err := rawdb.ReadStateHistoryMetaList(reader, start, number)
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

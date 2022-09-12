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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package beacon

import (
	"context"
	"encoding/binary"
	"math"

	//"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	lru "github.com/hashicorp/golang-lru"
)

var (
	blockRootsKey           = []byte("br-")
	stateRootsKey           = []byte("sr-")
	historicRootsKey        = []byte("hr-")
	beaconHeadTailKey       = []byte("ht")  // -> head slot and block root, tail slot values (short term, long term, init data)
	blockDataKey            = []byte("b-")  // bigEndian64(slot) + stateRoot -> RLP(BlockData)  (available starting from tailLongTerm)
	blockDataByBlockRootKey = []byte("bb-") // bigEndian64(slot) + blockRoot -> RLP(BlockData)  (not stored in db, only for caching)
	execNumberKey           = []byte("e-")  // bigEndian64(execNumber) + stateRoot -> RLP(slot)  (available starting from tailLongTerm after successful init)
	slotByBlockRootKey      = []byte("sb-") // blockRoot -> RLP(slot)  (available for init data blocks and all blocks starting from tailShortTerm)

	StateProofFormats   [HspFormatCount]indexMapFormat
	stateProofIndexMaps [HspFormatCount]map[uint64]int
	beaconHeaderFormat  indexMapFormat
)

const (
	HspLongTerm    = 1 << iota // state proof long term fields (latest_block_header.root, exec_head)
	HspShortTerm               // state proof short term fields (block_roots, state_roots, historical_roots, finalized.root)
	HspInitData                // state proof init data fields (genesis_time, genesis_validators_root, sync_committee_root, next_sync_committee_root)
	HspFormatCount             // number of possible format configurations

	HspAll = HspFormatCount - 1

	// beacon header fields
	BhiSlot          = 8
	BhiProposerIndex = 9
	BhiParentRoot    = 10
	BhiStateRoot     = 11
	BhiBodyRoot      = 12

	// beacon state fields		//TODO ??? fork-ot nem kene state-bol ellenorizni?
	BsiGenesisTime       = 32
	BsiGenesisValidators = 33
	BsiForkVersion       = 141 //TODO ??? osszes fork field? long vagy short term?
	BsiLatestHeader      = 36
	BsiBlockRoots        = 37
	BsiStateRoots        = 38
	BsiHistoricRoots     = 39
	BsiFinalBlock        = 105
	BsiSyncCommittee     = 54
	BsiNextSyncCommittee = 55
	BsiExecHead          = 908 // ??? 56

)

var BsiFinalExecHash = ChildIndex(ChildIndex(BsiFinalBlock, BhiStateRoot), BsiExecHead)

func init() {
	// initialize header proof format
	beaconHeaderFormat = NewIndexMapFormat()
	for i := uint64(8); i < 13; i++ {
		beaconHeaderFormat.AddLeaf(i, nil)
	}

	// initialize beacon state proof formats and index maps
	for i := range StateProofFormats {
		StateProofFormats[i] = NewIndexMapFormat()
		if i&HspLongTerm != 0 {
			StateProofFormats[i].AddLeaf(BsiLatestHeader, nil)
			StateProofFormats[i].AddLeaf(BsiExecHead, nil)
		}
		if i&HspShortTerm != 0 {
			StateProofFormats[i].AddLeaf(BsiStateRoots, nil)
			StateProofFormats[i].AddLeaf(BsiHistoricRoots, nil)
			StateProofFormats[i].AddLeaf(BsiFinalBlock, nil)
		}
		if i&HspInitData != 0 {
			StateProofFormats[i].AddLeaf(BsiGenesisTime, nil)
			StateProofFormats[i].AddLeaf(BsiGenesisValidators, nil)
			StateProofFormats[i].AddLeaf(BsiForkVersion, nil)
			StateProofFormats[i].AddLeaf(BsiSyncCommittee, nil)
			StateProofFormats[i].AddLeaf(BsiNextSyncCommittee, nil)
		}
		stateProofIndexMaps[i] = proofIndexMap(StateProofFormats[i])
	}
}

type BeaconDataSource interface { // supported by beacon node API
	GetBlocksFromHead(ctx context.Context, head Header, amount uint64) (Header, []*BlockData, error)
	GetRootsProof(ctx context.Context, block *BlockData) (MultiProof, MultiProof, error)
	GetHistoricRootsProof(ctx context.Context, block *BlockData, period uint64) (MultiProof, error)
}

type HistoricDataSource interface { // supported by ODR
	GetHistoricBlocks(ctx context.Context, head Header, lastSlot, amount uint64) (Header, []*BlockData, MultiProof, error)
	AvailableTailSlots() (uint64, uint64)
}

type execChain interface {
	// GetHeader(common.Hash, uint64) *types.Header   ???
	GetHeaderByHash(common.Hash) *types.Header
}

type Header struct {
	Slot          common.Decimal `json:"slot"` //TODO SignedHead RLP encoding is jo??
	ProposerIndex common.Decimal `json:"proposer_index"`
	ParentRoot    common.Hash    `json:"parent_root"`
	StateRoot     common.Hash    `json:"state_root"`
	BodyRoot      common.Hash    `json:"body_root"`
}

func (bh *Header) Hash() common.Hash {
	var values [8]MerkleValue //TODO ezt lehetne szebben is
	binary.LittleEndian.PutUint64(values[0][:8], uint64(bh.Slot))
	binary.LittleEndian.PutUint64(values[1][:8], uint64(bh.ProposerIndex))
	values[2] = MerkleValue(bh.ParentRoot)
	values[3] = MerkleValue(bh.StateRoot)
	values[4] = MerkleValue(bh.BodyRoot)
	////fmt.Println("hashing full header", bh, values)
	return MultiProof{Format: NewRangeFormat(8, 15, nil), Values: values[:]}.rootHash()
}

type HeaderWithoutState struct {
	Slot                 uint64
	ProposerIndex        uint
	ParentRoot, BodyRoot common.Hash
}

func (bh *HeaderWithoutState) Hash(stateRoot common.Hash) common.Hash {
	return bh.Proof(stateRoot).rootHash()
}

func (bh *HeaderWithoutState) Proof(stateRoot common.Hash) MultiProof {
	var values [8]MerkleValue //TODO ezt lehetne szebben is
	binary.LittleEndian.PutUint64(values[0][:8], bh.Slot)
	binary.LittleEndian.PutUint64(values[1][:8], uint64(bh.ProposerIndex))
	values[2] = MerkleValue(bh.ParentRoot)
	values[3] = MerkleValue(stateRoot)
	values[4] = MerkleValue(bh.BodyRoot)
	return MultiProof{Format: NewRangeFormat(8, 15, nil), Values: values[:]}
}

func (bh *HeaderWithoutState) FullHeader(stateRoot common.Hash) Header {
	return Header{
		Slot:          common.Decimal(bh.Slot),
		ProposerIndex: common.Decimal(bh.ProposerIndex),
		ParentRoot:    bh.ParentRoot,
		StateRoot:     stateRoot,
		BodyRoot:      bh.BodyRoot,
	}
}

type BlockData struct {
	Header         HeaderWithoutState
	StateRoot      common.Hash `rlp:"-"` // calculated by CalculateRoots()
	BlockRoot      common.Hash `rlp:"-"` // calculated by CalculateRoots()
	ProofFormat    byte
	StateProof     MerkleValues
	ParentSlotDiff uint64       // slot-parentSlot; 0 if not initialized
	StateRootDiffs MerkleValues // only valid if ParentSlotDiff is initialized
}

func (block *BlockData) FullHeader() Header {
	return block.Header.FullHeader(block.StateRoot)
}

func (block *BlockData) firstInPeriod() bool {
	newPeriod, oldPeriod := block.Header.Slot>>13, (block.Header.Slot-block.ParentSlotDiff)>>13
	if newPeriod > oldPeriod+1 {
		log.Error("More than an entire period skipped", "oldSlot", block.Header.Slot-block.ParentSlotDiff, "newSlot", block.Header.Slot)
	}
	return newPeriod > oldPeriod
}

func (block *BlockData) firstInEpoch() bool {
	return block.Header.Slot>>5 > (block.Header.Slot-block.ParentSlotDiff)>>5
}

func (block *BlockData) Proof() MultiProof {
	return MultiProof{Format: StateProofFormats[block.ProofFormat], Values: block.StateProof}
}

func (block *BlockData) GetStateValue(index uint64) (MerkleValue, bool) {
	proofIndex, ok := stateProofIndexMaps[block.ProofFormat][index]
	if !ok {
		return MerkleValue{}, false
	}
	return block.StateProof[proofIndex], true
}

func (block *BlockData) mustGetStateValue(index uint64) MerkleValue {
	v, ok := block.GetStateValue(index)
	if !ok {
		panic(nil)
	}
	return v
}

func (block *BlockData) CalculateRoots() {
	block.StateRoot = block.Proof().rootHash()
	block.BlockRoot = block.Header.Hash(block.StateRoot)
}

type BeaconChain struct {
	dataSource     BeaconDataSource
	historicSource HistoricDataSource

	execChain      execChain
	db             ethdb.Database
	failCounter    int
	blockDataCache *lru.Cache // string(dbKey) -> *BlockData  (either blockDataKey, slotByBlockRootKey or blockDataByBlockRootKey)  //TODO use separate cache?
	historicCache  *lru.Cache // string(dbKey) -> MerkleValue (either stateRootsKey or historicRootsKey)

	bellatrixSlot uint64

	execNumberCacheMu sync.RWMutex //TODO ???
	execNumberCache   *lru.Cache   // uint64(execNumber) -> []struct{slot, stateRoot}
	// execNumberIndex fields are locked by chainMu and are always consistent with storedHead, headTree and TailLongTerm
	// These fields are not explicitly stored but initialized at startup based on the first canonical exec number index entry found in the db.
	execNumberIndexHeadPresent bool   // index available for storedHead (== head of storedSection == headTree.HeadBlock)
	execNumberIndexHeadNumber  uint64 // exec block number belonging to storedHead (valid if execNumberIndexHeadPresent is true)
	execNumberIndexTailSlot    uint64 // index available for all slots >= execNumberIndexTailSlot of the canonical chain defined by headTree (valid if execNumberIndexHeadPresent is true)
	execNumberIndexTailNumber  uint64 // exec block number belonging to execNumberIndexTailSlot (valid if execNumberIndexHeadPresent is true)

	chainMu                               sync.RWMutex
	storedHead                            *BlockData
	headTree                              *HistoricTree
	tailShortTerm, tailLongTerm           uint64 // shortTerm >= longTerm
	tailParentHeader                      Header // Slot == tailLongTerm-1 (empty if tailLongTerm == 0)
	blockRoots, stateRoots, historicRoots *merkleListVersion

	historicMu    sync.RWMutex
	historicTrees map[common.Hash]*HistoricTree

	committeeRootCache                   *lru.Cache // uint64(period) -> common.Hash (committee root hash)
	constraintFirst, constraintAfterLast uint64
	lastCommitteeRoot                    common.Hash
	initCallback                         func(genesisData) // called once, set to nil after
	updateCallback                       func()
	callInit, callUpdate                 bool
	callProcessedBeaconHead              common.Hash
	genesisData                          genesisData

	beaconSyncer
}

func NewBeaconChain(dataSource BeaconDataSource, historicSource HistoricDataSource, execChain execChain, db ethdb.Database, forks Forks) *BeaconChain {
	chainDb := rawdb.NewTable(db, "bc-")
	blockDataCache, _ := lru.New(2000)
	historicCache, _ := lru.New(20000)
	execNumberCache, _ := lru.New(2000)
	committeeRootCache, _ := lru.New(200)
	bc := &BeaconChain{
		dataSource:         dataSource,
		historicSource:     historicSource,
		execChain:          execChain,
		db:                 chainDb,
		blockDataCache:     blockDataCache,
		historicCache:      historicCache,
		execNumberCache:    execNumberCache,
		committeeRootCache: committeeRootCache,
	}
	if epoch, ok := forks.epoch("BELLATRIX"); ok {
		bc.bellatrixSlot = epoch << 5
	} else {
		log.Error("Bellatrix fork not found in beacon chain config")
		return nil
	}
	bc.initHistoricStructures()
	if enc, err := bc.db.Get(beaconHeadTailKey); err == nil {
		var ht beaconHeadTailInfo
		if rlp.DecodeBytes(enc, &ht) == nil {
			if bc.storedHead = bc.GetBlockData(ht.HeadSlot, ht.HeadHash, true); bc.storedHead != nil {
				bc.tailShortTerm, bc.tailLongTerm = ht.TailShortTerm, ht.TailLongTerm
				bc.headTree = bc.newHistoricTree(ht.TailHistoric, ht.TailHistoricPeriod, ht.NextHistoricPeriod)
			} else {
				log.Error("Head block data not found in database")
			}
		}
	}
	if bc.storedHead == nil {
		// clear everything if head info is missing to ensure that the chain is not initialized with partially remaining data
		bc.clearDb()
	}
	if bc.storedHead != nil {
		//var tailParent *BlockData
		tailSlot := bc.tailLongTerm
		for {
			if tailBlock := bc.firstCanonicalBlock(tailSlot); tailBlock != nil {
				if tailBlock.Header.Slot == 0 {
					break
				}
				if tailParent := bc.GetParent(tailBlock); tailParent != nil {
					bc.tailParentHeader = tailParent.FullHeader()
					tailSlot = uint64(bc.tailParentHeader.Slot) + 1
					if tailSlot != bc.tailLongTerm {
						log.Error("Beacon chain tail does not match stored tail slot", "stored", bc.tailLongTerm, "found", tailSlot)
						bc.tailLongTerm = tailSlot
						if bc.tailShortTerm < tailSlot {
							bc.tailShortTerm = tailSlot
						}
					}
					break
				}
				tailSlot = tailBlock.Header.Slot + 1
			} else {
				log.Error("Beacon chain tail not found, resetting database")
				bc.clearDb()
				break
			}
		}
		log.Info("Beacon chain initialized", "tailSlot", bc.tailLongTerm, "headSlot", bc.storedHead.Header.Slot)
	}
	bc.initExecNumberIndex()
	return bc
}

func (bc *BeaconChain) clearDb() {
	//fmt.Println("CLEAR DB")
	bc.db.Delete(beaconHeadTailKey) // delete head info first to ensure that next time the chain will not be initialized with partially remaining data
	iter := bc.db.NewIterator(nil, nil)
	for iter.Next() {
		bc.db.Delete(iter.Key())
	}
	iter.Release()
	bc.blockDataCache.Purge()
	bc.historicCache.Purge()
	bc.execNumberCache.Purge()
	bc.callUpdate = true
}

type beaconHeadTailInfo struct {
	HeadSlot                                             uint64
	HeadHash                                             common.Hash
	TailShortTerm, TailLongTerm                          uint64
	TailHistoric, TailHistoricPeriod, NextHistoricPeriod uint64
}

func (bc *BeaconChain) storeHeadTail(batch ethdb.Batch) {
	if bc.storedHead != nil && bc.headTree != nil {
		enc, _ := rlp.EncodeToBytes(&beaconHeadTailInfo{
			HeadSlot:           uint64(bc.storedHead.Header.Slot),
			HeadHash:           bc.storedHead.BlockRoot,
			TailShortTerm:      bc.tailShortTerm,
			TailLongTerm:       bc.tailLongTerm,
			TailHistoric:       bc.headTree.tailSlot,
			TailHistoricPeriod: bc.headTree.tailPeriod,
			NextHistoricPeriod: bc.headTree.nextPeriod,
		})
		batch.Put(beaconHeadTailKey, enc)
	}
}

func getBlockDataKey(slot uint64, root common.Hash, byBlockRoot, addRoot bool) []byte {
	var prefix []byte
	if byBlockRoot {
		prefix = blockDataByBlockRootKey
	} else {
		prefix = blockDataKey
	}
	p := len(prefix)
	keyLen := p + 8
	if addRoot {
		keyLen += 32
	}
	dbKey := make([]byte, keyLen)
	copy(dbKey[:p], prefix)
	binary.BigEndian.PutUint64(dbKey[p:p+8], slot)
	if addRoot {
		copy(dbKey[p+8:], root[:])
	}
	return dbKey
}

func (bc *BeaconChain) GetBlockData(slot uint64, hash common.Hash, byBlockRoot bool) *BlockData {
	//fmt.Println("GetBlockData", slot, hash, byBlockRoot)
	key := getBlockDataKey(slot, hash, byBlockRoot, true)
	if bd, ok := bc.blockDataCache.Get(string(key)); ok {
		//fmt.Println(" cached")
		return bd.(*BlockData)
	}
	var blockData *BlockData

	if byBlockRoot {
		iter := bc.db.NewIterator(getBlockDataKey(slot, common.Hash{}, false, false), nil)
		for iter.Next() {
			blockData = new(BlockData)
			if err := rlp.DecodeBytes(iter.Value(), blockData); err == nil {
				blockData.CalculateRoots()
				if blockData.BlockRoot == hash {
					break
				} else {
					blockData = nil
				}
			} else {
				blockData = nil
				log.Error("Error decoding stored beacon slot data", "slot", slot, "blockRoot", hash, "error", err)
			}
		}
		iter.Release()
	} else {
		if blockDataEnc, err := bc.db.Get(key); err == nil {
			//fmt.Println(" found in db")
			blockData = new(BlockData)
			if err := rlp.DecodeBytes(blockDataEnc, blockData); err == nil {
				blockData.CalculateRoots()
				//fmt.Println(" decoded")
			} else {
				//fmt.Println(" decode err", err)
				blockData = nil
				log.Error("Error decoding stored beacon slot data", "slot", slot, "stateRoot", hash, "error", err)
			}
		} else {
			//fmt.Println(" db err", err)
		}
	}

	bc.blockDataCache.Add(string(key), blockData)
	if blockData != nil {
		if byBlockRoot {
			bc.blockDataCache.Add(string(getBlockDataKey(slot, blockData.StateRoot, false, true)), blockData)
		} else {
			bc.blockDataCache.Add(string(getBlockDataKey(slot, blockData.BlockRoot, true, true)), blockData)
		}
	}
	return blockData
}

func (bc *BeaconChain) GetParent(block *BlockData) *BlockData {
	if block.ParentSlotDiff == 0 {
		return nil
	}
	return bc.GetBlockData(block.Header.Slot-block.ParentSlotDiff, block.Header.ParentRoot, true)
}

func (bc *BeaconChain) storeBlockData(blockData *BlockData) {
	//fmt.Println("storeBlockData", blockData.Header.Slot, blockData.StateRoot)
	key := getBlockDataKey(blockData.Header.Slot, blockData.StateRoot, false, true)
	bc.blockDataCache.Add(string(key), blockData)
	bc.blockDataCache.Add(string(getBlockDataKey(blockData.Header.Slot, blockData.BlockRoot, true, true)), blockData)
	enc, err := rlp.EncodeToBytes(blockData)
	if err != nil {
		//fmt.Println(" encode err", err)
		log.Error("Error encoding beacon slot data for storage", "slot", blockData.Header.Slot, "blockRoot", blockData.BlockRoot, "error", err)
		return
	}
	//fmt.Println(" store err", bc.db.Put(key, enc))
	bc.db.Put(key, enc)
}

func getExecNumberKey(execNumber uint64, stateRoot common.Hash, addRoot bool) []byte {
	p := len(execNumberKey)
	keyLen := p + 8
	if addRoot {
		keyLen += 32
	}
	dbKey := make([]byte, keyLen)
	copy(dbKey[:p], execNumberKey)
	binary.BigEndian.PutUint64(dbKey[p:p+8], execNumber)
	if addRoot {
		copy(dbKey[p+8:], stateRoot[:])
	}
	return dbKey
}

type slotAndStateRoot struct {
	slot      uint64
	stateRoot common.Hash
}

type slotsAndStateRoots []slotAndStateRoot

func (bc *BeaconChain) getSlotsAndStateRoots(execNumber uint64) slotsAndStateRoots {
	//bc.execNumberCacheMu.RLock() //TODO
	if v, ok := bc.execNumberCache.Get(execNumber); ok {
		return v.(slotsAndStateRoots)
	}

	var list slotsAndStateRoots
	prefix := getExecNumberKey(execNumber, common.Hash{}, false)
	prefixLen := len(prefix)
	iter := bc.db.NewIterator(prefix, nil)
	for iter.Next() {
		var entry slotAndStateRoot
		if len(iter.Key()) != prefixLen+32 {
			log.Error("Invalid exec number entry key length", "execNumber", execNumber, "length", len(iter.Key()), "expected", prefixLen+32)
			continue
		}
		copy(entry.stateRoot[:], iter.Key()[prefixLen:])
		if err := rlp.DecodeBytes(iter.Value(), &entry.slot); err != nil {
			log.Error("Error decoding stored exec number entry", "execNumber", execNumber, "error", err)
			continue
		}
		list = append(list, entry)
	}
	iter.Release()
	bc.execNumberCache.Add(execNumber, list)
	return list
}

func (bc *BeaconChain) GetBlockDataByExecNumber(ht *HistoricTree, execNumber uint64) *BlockData {
	//fmt.Println("GetBlockDataByExecNumber", execNumber)
	list := bc.getSlotsAndStateRoots(execNumber)
	//fmt.Println(" list", list)
	for _, entry := range list {
		//fmt.Println("  check", ht.GetStateRoot(entry.slot), entry.stateRoot)
		if ht.GetStateRoot(entry.slot) == entry.stateRoot {
			//fmt.Println("  GetBlockData", bc.GetBlockData(entry.slot, entry.stateRoot, false))
			return bc.GetBlockData(entry.slot, entry.stateRoot, false)
		}
	}
	return nil
}

func (bc *BeaconChain) storeExecNumberIndex(execNumber uint64, blockData *BlockData) {
	bc.execNumberCache.Remove(execNumber)
	slotEnc, _ := rlp.EncodeToBytes(&blockData.Header.Slot)
	bc.db.Put(getExecNumberKey(execNumber, blockData.StateRoot, true), slotEnc)
}

func (bc *BeaconChain) deleteExecNumberIndex(execNumber uint64, stateRoot common.Hash) {
	bc.execNumberCache.Remove(execNumber)
	bc.db.Delete(getExecNumberKey(execNumber, stateRoot, true))
}

func getSlotByBlockRootKey(blockRoot common.Hash) []byte {
	p := len(slotByBlockRootKey)
	dbKey := make([]byte, p+32)
	copy(dbKey[:p], slotByBlockRootKey)
	copy(dbKey[p+8:], blockRoot[:])
	return dbKey
}

func (bc *BeaconChain) GetBlockDataByBlockRoot(blockRoot common.Hash) *BlockData {
	dbKey := getSlotByBlockRootKey(blockRoot)
	var slot uint64
	if enc, err := bc.db.Get(dbKey); err == nil {
		if rlp.DecodeBytes(enc, &slot) != nil {
			return nil //TODO error log
		}
	} else {
		bc.blockDataCache.Add(string(dbKey), nil)
		return nil
	}
	blockData := bc.GetBlockData(slot, blockRoot, true)
	bc.blockDataCache.Add(string(dbKey), blockData)
	return blockData
}

func (bc *BeaconChain) storeSlotByBlockRoot(blockData *BlockData) {
	dbKey := getSlotByBlockRootKey(blockData.BlockRoot)
	enc, _ := rlp.EncodeToBytes(&blockData.Header.Slot)
	bc.db.Put(dbKey, enc)
	bc.blockDataCache.Add(string(dbKey), blockData)
}

func ProofFormatForBlock(block *BlockData) byte {
	format := byte(HspLongTerm + HspShortTerm)
	if block.firstInEpoch() {
		format += HspInitData
	}
	return format
}

// proofFormatForBlock returns the minimal required set of state proof fields for a
// given slot according to the current chain tail values. Stored format equals to or
// is a superset of this.
func (bc *BeaconChain) proofFormatForBlock(block *BlockData) byte {
	if block.ParentSlotDiff == 0 {
		return HspLongTerm + HspShortTerm + HspInitData //???
	}
	format := byte(HspLongTerm)
	if block.Header.Slot >= bc.tailShortTerm {
		format += HspShortTerm
	}
	if block.firstInEpoch() {
		format += HspInitData
	}
	return format
}

func (bc *BeaconChain) GetTailSlots() (longTerm, shortTerm uint64) {
	bc.chainMu.RLock()
	if bc.storedHead != nil {
		longTerm, shortTerm = bc.tailLongTerm, bc.tailShortTerm
	} else {
		longTerm, shortTerm = math.MaxUint64, math.MaxUint64
	}
	bc.chainMu.RUnlock()
	return
}

func (bc *BeaconChain) pruneBlockFormat(block *BlockData) bool {
	if block.ParentSlotDiff == 0 && block.Header.Slot > bc.tailLongTerm {
		return false
	}
	format := bc.proofFormatForBlock(block)
	//fmt.Println("Pruning block at slot", block.Header.Slot, "old format", block.ProofFormat, "pruned format", format, "tailLongTerm", bc.tailLongTerm, "tailShortTerm", bc.tailShortTerm)
	if format == block.ProofFormat {
		return true
	}

	var values MerkleValues
	if _, ok := TraverseProof(block.Proof().Reader(nil), NewMultiProofWriter(StateProofFormats[format], &values, nil)); ok {
		block.ProofFormat, block.StateProof = format, values
		if format&HspShortTerm == 0 {
			block.StateRootDiffs = nil
		}
		return true
	}
	return false
}

func (bc *BeaconChain) PeriodRange() (first, afterFixed, afterLast uint64) {
	bc.chainMu.RLock()
	first, afterFixed, afterLast = bc.constraintFirst, bc.constraintAfterLast, bc.constraintAfterLast
	bc.chainMu.RUnlock()
	return
}

func (bc *BeaconChain) CommitteeRoot(period uint64) (root common.Hash, matchAll bool) {
	bc.chainMu.RLock()
	root = bc.committeeRoot(period)
	bc.chainMu.RUnlock()
	return
}

func (bc *BeaconChain) committeeRoot(period uint64) (root common.Hash) {
	//fmt.Println("committeeRoot", period)
	if bc.headTree == nil {
		return
	}

	if r, ok := bc.committeeRootCache.Get(period); ok {
		return r.(common.Hash)
	}

	periodStart, nextPeriodStart := period<<13, (period+1)<<13
	var start uint64
	if period > 1 {
		start = (period - 1) << 13
	}

	var slotEnc [8]byte
	binary.BigEndian.PutUint64(slotEnc[:], start)
	iter := bc.db.NewIterator(blockDataKey, slotEnc[:])

	//fmt.Println(" committeeRoot iter")
	defer func() {
		iter.Release()
		bc.committeeRootCache.Add(period, root)
	}()

	for iter.Next() {
		block := new(BlockData)
		if err := rlp.DecodeBytes(iter.Value(), block); err == nil {
			//fmt.Println(" committeeRoot block", block.Header.Slot, block.Header.Slot>>13, block.ProofFormat&HspInitData)
			if uint64(block.Header.Slot) >= nextPeriodStart {
				// no more blocks from period-1 and period
				return
			}
			if block.ProofFormat&HspInitData == 0 {
				continue
			}
			block.CalculateRoots()
			if bc.isCanonical(block) {
				if uint64(block.Header.Slot) >= periodStart {
					root = common.Hash(block.mustGetStateValue(BsiSyncCommittee))
				} else {
					root = common.Hash(block.mustGetStateValue(BsiNextSyncCommittee))
				}
				//fmt.Println("  canonical", block.Header.Slot, root)
				return
			}
		} else {
			log.Error("Error decoding beacon block found by iterator", "key", iter.Key(), "value", iter.Value(), "error", err)
		}
	}
	return
}

func (bc *BeaconChain) SetCallbacks(initCallback func(genesisData), updateCallback func()) {
	bc.chainMu.Lock()
	bc.updateCallback = updateCallback
	if genesisData, ok := bc.getGenesisData(); ok {
		bc.chainMu.Unlock()
		initCallback(genesisData)
	} else {
		bc.initCallback = initCallback
		bc.chainMu.Unlock()
	}
}

// chainMu locked
func (bc *BeaconChain) updateConstraints(firstSlot, lastSlot uint64) {
	firstPeriod := firstSlot >> 13
	afterLastPeriod := (lastSlot >> 13) + 2
	oldFirst, oldAfterLast := bc.constraintFirst, bc.constraintAfterLast
	if bc.constraintAfterLast == 0 {
		bc.constraintFirst, bc.constraintAfterLast = firstPeriod, afterLastPeriod
	} else {
		for period := firstPeriod; period < afterLastPeriod; period++ {
			bc.committeeRootCache.Remove(period)
		}
		if firstPeriod < bc.constraintFirst {
			bc.constraintFirst = firstPeriod
			for bc.constraintFirst < bc.constraintAfterLast && bc.constraintFirst < afterLastPeriod && bc.committeeRoot(bc.constraintFirst) == (common.Hash{}) {
				bc.constraintFirst++
			}
		}
		if afterLastPeriod > bc.constraintAfterLast {
			bc.constraintAfterLast = afterLastPeriod
			for bc.constraintAfterLast > bc.constraintFirst && bc.constraintAfterLast > firstPeriod && bc.committeeRoot(bc.constraintAfterLast-1) == (common.Hash{}) {
				bc.constraintAfterLast--
			}
		}
	}
	bc.callUpdate = bc.constraintFirst != oldFirst || bc.constraintAfterLast != oldAfterLast || bc.committeeRoot(bc.constraintAfterLast-1) != bc.lastCommitteeRoot
	bc.lastCommitteeRoot = bc.committeeRoot(bc.constraintAfterLast - 1)
	if bc.initCallback != nil {
		bc.genesisData, bc.callInit = bc.getGenesisData()
	}
	//fmt.Println("updateConstraints  old", oldFirst, oldAfterLast, "update range", firstPeriod, afterLastPeriod, "new", bc.constraintFirst, bc.constraintAfterLast, "callUpdate", bc.callUpdate, "callInit", bc.callInit)
}

func (bc *BeaconChain) getGenesisData() (genesisData, bool) {
	iter := bc.db.NewIterator(blockDataKey, nil)
	for iter.Next() {
		block := new(BlockData)
		if err := rlp.DecodeBytes(iter.Value(), block); err == nil {
			block.CalculateRoots()
			if bc.isCanonical(block) {
				if genesisData, ok := block.getGenesisData(); ok {
					return genesisData, true
				}
			}
		} else {
			log.Error("Error decoding beacon block found by iterator", "key", iter.Key(), "value", iter.Value(), "error", err)
		}
	}
	return genesisData{}, false
}

func (bc *BeaconChain) firstCanonicalBlock(startSlot uint64) *BlockData {
	var slotEnc [8]byte
	binary.BigEndian.PutUint64(slotEnc[:], startSlot)
	iter := bc.db.NewIterator(blockDataKey, slotEnc[:])
	for iter.Next() {
		block := new(BlockData)
		if err := rlp.DecodeBytes(iter.Value(), block); err == nil {
			block.CalculateRoots()
			if bc.isCanonical(block) {
				return block
			}
		} else {
			log.Error("Error decoding beacon block found by iterator", "key", iter.Key(), "value", iter.Value(), "error", err)
		}
	}
	return nil
}

func (bc *BeaconChain) isCanonical(block *BlockData) bool {
	return bc.headTree != nil && bc.headTree.GetStateRoot(uint64(block.Header.Slot)) == block.StateRoot
}

// call when chainMu locked; call returned function after unlocked
func (bc *BeaconChain) constraintCallbacks() func() {
	callInit, callUpdate, initCallback, genesisData := bc.callInit, bc.callUpdate, bc.initCallback, bc.genesisData
	if callInit {
		bc.initCallback = nil
	}
	bc.callInit, bc.callUpdate = false, false
	callProcessedBeaconHead := bc.callProcessedBeaconHead
	bc.callProcessedBeaconHead = common.Hash{}
	return func() {
		//fmt.Println("constraintCallbacks  init", callInit, "update", callUpdate)
		if callInit && initCallback != nil {
			initCallback(genesisData)
		} else if callUpdate && bc.updateCallback != nil { // init updates automatically
			bc.updateCallback()
		}
		if bc.processedCallback != nil && callProcessedBeaconHead != (common.Hash{}) {
			bc.processedCallback(callProcessedBeaconHead)
		}
	}
}

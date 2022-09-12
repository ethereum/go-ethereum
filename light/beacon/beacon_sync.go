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
	//"fmt"
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	firstRollback    = 4
	rollbackMulStep  = 4
	syncWorkerCount  = 1 //TODO 4
	ReverseSyncLimit = 64
	MaxHeaderFetch   = 192
)

// part of BeaconChain, separated for better readability
type beaconSyncer struct {
	storedSection, syncHeadSection *chainSection
	syncHeader                     Header
	newHeadCh                      chan struct{} // closed and replaced when head is changed
	syncedCh                       chan bool
	latestHeadCounter              uint64   // +1 when head is changed
	newHeadReqCancel               []func() // called shortly after head is changed
	nextRollback                   uint64

	processedCallback                               func(common.Hash)
	waitForExecHead                                 bool
	lastProcessedBeaconHead, lastReportedBeaconHead common.Hash
	lastProcessedExecNumber                         uint64
	lastProcessedExecHead, expectedExecHead         common.Hash

	stopSyncCh chan struct{}
	syncWg     sync.WaitGroup
}

func (bc *BeaconChain) SyncToHead(head Header, syncedCh chan bool) {
	bc.chainMu.Lock()
	defer bc.chainMu.Unlock()

	//fmt.Println("SyncToHead", head.Slot)
	if bc.syncedCh != nil {
		bc.syncedCh <- false
	}
	bc.syncedCh = syncedCh
	bc.syncHeader = head
	bc.latestHeadCounter++
	cs := &chainSection{
		headCounter:  bc.latestHeadCounter,
		tailSlot:     uint64(head.Slot) + 1,
		headSlot:     uint64(head.Slot),
		parentHeader: head,
	}
	if bc.syncHeadSection != nil && bc.syncHeadSection.prev != nil {
		bc.syncHeadSection.prev.next = cs
		cs.prev = bc.syncHeadSection.prev
	}
	if cs.prev == nil && bc.storedSection != nil {
		cs.prev = bc.storedSection
		bc.storedSection.next = cs
	}
	bc.syncHeadSection = cs
	bc.cancelRequests(time.Second)
	close(bc.newHeadCh)
	bc.newHeadCh = make(chan struct{})
}

func (bc *BeaconChain) StartSyncing() {
	if bc.storedHead != nil {
		bc.storedSection = &chainSection{headSlot: uint64(bc.storedHead.Header.Slot), tailSlot: bc.tailLongTerm, parentHeader: bc.tailParentHeader}
		bc.updateConstraints(bc.tailLongTerm, uint64(bc.storedHead.Header.Slot))
	}
	bc.newHeadCh = make(chan struct{})
	bc.stopSyncCh = make(chan struct{})
	bc.nextRollback = firstRollback

	bc.syncWg.Add(syncWorkerCount)
	for i := 0; i < syncWorkerCount; i++ {
		go bc.syncWorker()
	}
}

func (bc *BeaconChain) StopSyncing() {
	bc.chainMu.Lock()
	bc.cancelRequests(0)
	bc.chainMu.Unlock()
	close(bc.stopSyncCh)
	bc.syncWg.Wait()
}

func (bc *BeaconChain) syncWorker() {
	defer bc.syncWg.Done()

	bc.chainMu.Lock()
	for {
		bc.debugPrint("before nextRequest")
		if bc.storedSection != nil && bc.storedSection.headSlot < bc.storedSection.tailSlot+10 {
			panic(nil)
		}
		if cs := bc.nextRequest(); cs != nil && bc.syncHeader.StateRoot != (common.Hash{}) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
			bc.newHeadReqCancel = append(bc.newHeadReqCancel, cancel)
			cs.requesting = true
			bc.debugPrint("after nextRequest")
			head := bc.syncHeader
			var (
				parentHeader Header
				blocks       []*BlockData
				proof        MultiProof
				err          error
			)
			newHeadCh := bc.newHeadCh
			cb := bc.constraintCallbacks()
			bc.chainMu.Unlock()
			cb()
			if bc.dataSource != nil && cs.tailSlot+MaxHeaderFetch-1 >= uint64(head.Slot) {
				//fmt.Println("SYNC dataSource request", head.Slot, cs.tailSlot, cs.headSlot)
				parentHeader, blocks, err = bc.dataSource.GetBlocksFromHead(ctx, head, uint64(head.Slot)+1-cs.tailSlot)
			} else if bc.historicSource != nil {
				//fmt.Println("SYNC historicSource request", head.Slot, cs.tailSlot, cs.headSlot)
				parentHeader, blocks, proof, err = bc.historicSource.GetHistoricBlocks(ctx, head, cs.headSlot, cs.headSlot+1-cs.tailSlot)
			} else {
				log.Error("Historic data source not available") //TODO print only once, ?reset chain?
			}
			if err != nil /*&& err != light.ErrNoPeers && err != context.Canceled*/ {
				blocks = nil
				log.Warn("Beacon block data request failed", "tail", cs.tailSlot, "head", cs.headSlot, "error", err)
			}
			if blocks != nil && (len(blocks) == 0 || blocks[0].Header.Slot+1-blocks[0].ParentSlotDiff > cs.tailSlot || blocks[len(blocks)-1].Header.Slot < cs.headSlot) {
				blocks = nil
				log.Error("Retrieved beacon block range insufficient")
			}
			//fmt.Println(" blocks", len(blocks), "err", err)
			if blocks == nil {
				bc.chainMu.Lock()
				if bc.syncedCh != nil {
					bc.syncedCh <- true
					bc.syncedCh = nil
				}
				bc.chainMu.Unlock()
				select {
				case <-newHeadCh:
				case <-bc.stopSyncCh:
					return
				}
			}
			bc.chainMu.Lock()
			if blocks != nil {
				cs.requesting = false
				bc.debugPrint("before addBlocksToSection")
				bc.addBlocksToSection(cs, parentHeader, blocks, proof)
				bc.debugPrint("after addBlocksToSection")
			} else {
				bc.debugPrint("before remove")
				cs.remove()
				bc.debugPrint("after remove")
			}
		} else {
			newHeadCh := bc.newHeadCh
			cb := bc.constraintCallbacks()
			if bc.syncedCh != nil {
				bc.syncedCh <- true
				bc.syncedCh = nil
			}
			bc.chainMu.Unlock()
			cb()
			select {
			case <-newHeadCh:
			case <-bc.stopSyncCh:
				return
			}
			bc.chainMu.Lock()
		}
	}
}

func (bc *BeaconChain) reset() {
	bc.storedSection.remove()
	bc.storedHead, bc.headTree, bc.storedSection = nil, nil, nil
	bc.cancelRequests(0)
	bc.tailShortTerm, bc.tailLongTerm = 0, 0
	bc.initHistoricStructures()
	bc.clearDb()
}

func (bc *BeaconChain) initHistoricStructures() {
	bc.blockRoots = &merkleListVersion{list: &merkleList{db: bc.db, cache: bc.historicCache, dbKey: blockRootsKey, zeroLevel: 13}}
	bc.stateRoots = &merkleListVersion{list: &merkleList{db: bc.db, cache: bc.historicCache, dbKey: stateRootsKey, zeroLevel: 13}}
	bc.historicRoots = &merkleListVersion{list: &merkleList{db: bc.db, cache: bc.historicCache, dbKey: historicRootsKey, zeroLevel: 25}}
	bc.historicTrees = make(map[common.Hash]*HistoricTree)
}

func (bc *BeaconChain) cancelRequests(dt time.Duration) {
	cancelList := bc.newHeadReqCancel
	bc.newHeadReqCancel = nil
	if cancelList != nil {
		time.AfterFunc(dt, func() {
			for _, cancel := range cancelList {
				cancel()
			}
		})
	}
}

func (bc *BeaconChain) rollback(slot uint64) {
	//fmt.Println("bc.rollback", slot)
	if slot >= uint64(bc.storedHead.Header.Slot) {
		log.Error("Cannot roll back beacon chain", "slot", slot, "head", uint64(bc.storedHead.Header.Slot))
	}
	var block *BlockData
	for {
		if block = bc.GetBlockData(slot, bc.headTree.GetStateRoot(slot), false); block != nil {
			break
		}
		slot--
		if slot < bc.tailLongTerm {
			//fmt.Println(" tail reached, reset")
			bc.reset()
			return
		}
	}
	//fmt.Println(" found last non-empty slot", slot, block.Header.Slot)
	headTree := bc.makeChildTree()
	headTree.addRoots(slot, nil, nil, true, 0, MultiProof{})
	headTree.HeadBlock = block
	batch := bc.db.NewBatch()
	bc.commitHistoricTree(batch, headTree)
	bc.storedHead = block
	bc.storedSection.headSlot = slot
	bc.storeHeadTail(batch)
	batch.Write()
	bc.updateTreeMap()
}

func (bc *BeaconChain) blockRootAt(slot uint64) common.Hash {
	if bc.headTree == nil {
		return common.Hash{}
	}
	if slot == bc.headTree.HeadBlock.Header.Slot { //TODO is this special case needed?
		return bc.headTree.HeadBlock.BlockRoot
	}
	if slot+1 == bc.tailLongTerm {
		for s := slot + 1; s <= bc.headTree.HeadBlock.Header.Slot; s++ {
			if block := bc.GetBlockData(s, bc.headTree.GetStateRoot(s), false); block != nil {
				if block.Header.Slot-block.ParentSlotDiff == slot {
					return block.Header.ParentRoot
				}
				log.Error("Could not find parent of tail block")
				return common.Hash{}
			}

		}
	}
	if block := bc.GetBlockData(slot, bc.headTree.GetStateRoot(slot), false); block != nil {
		return block.BlockRoot
	}
	return common.Hash{}
}

type chainSection struct {
	headSlot, tailSlot, headCounter uint64 // tail is parentHeader.Slot+1 or 0 if no parent (genesis)
	requesting                      bool
	blocks                          []*BlockData // nil for stored section, sync head section and sections being requested
	tailProof                       MultiProof   // optional merkle proof including path leading to the first root in the historical_roots structure
	parentHeader                    Header       // empty for stored section, section starting at genesis and sections being requested
	prev, next                      *chainSection
}

func (cs *chainSection) blockIndex(slot uint64) int {
	if slot > cs.headSlot || slot < cs.tailSlot {
		return -1
	}

	min, max := 0, len(cs.blocks)-1
	for min < max {
		mid := (min + max) / 2
		if uint64(cs.blocks[mid].Header.Slot) < slot {
			min = mid + 1
		} else {
			max = mid
		}
	}
	return max
}

// returns empty hash for missed slots and slots outside the parent..head range
func (cs *chainSection) blockRootAt(slot uint64) common.Hash {
	if slot+1 == cs.tailSlot {
		return cs.parentHeader.Hash()
	}
	if index := cs.blockIndex(slot); index != -1 {
		block := cs.blocks[index]
		if uint64(block.Header.Slot) == slot {
			return block.BlockRoot
		}
	}
	return common.Hash{}
}

func (cs *chainSection) blockRange(begin, end uint64) (Header, []*BlockData) {
	b, e := cs.blockIndex(begin), cs.blockIndex(end)
	var parentHeader Header
	if b > 0 {
		parentHeader = cs.blocks[b-1].FullHeader()
	} else {
		parentHeader = cs.parentHeader
	}
	return parentHeader, cs.blocks[b : e+1]
}

func (cs *chainSection) trim(front bool) {
	length := cs.headSlot + 1 - cs.tailSlot
	length /= ((length + MaxHeaderFetch - 1) / MaxHeaderFetch)
	if front {
		cs.headSlot = cs.tailSlot + length - 1
	} else {
		cs.tailSlot = cs.headSlot + 1 - length
	}
}

func (cs *chainSection) remove() {
	if cs.prev != nil {
		cs.prev.next = cs.next
	}
	if cs.next != nil {
		cs.next.prev = cs.prev
	}
}

// chainMu locked
func (bc *BeaconChain) nextRequest() *chainSection {
	if bc.syncHeadSection == nil {
		return nil
	}
	//fmt.Println("SYNC nextRequest")
	origin := bc.storedSection
	if origin == nil {
		origin = bc.syncHeadSection
	}
	cs := origin
	for cs != bc.syncHeadSection && cs.next != nil {
		if cs.next.tailSlot > cs.headSlot+1 {
			req := &chainSection{
				headSlot:    cs.next.tailSlot - 1,
				tailSlot:    cs.headSlot + 1,
				headCounter: bc.latestHeadCounter,
				prev:        cs,
				next:        cs.next,
			}
			cs.next.prev = req
			cs.next = req
			req.trim(true)
			//fmt.Println(" fwd", req.tailSlot, req.headSlot)
			return req
		}
		cs = cs.next
	}
	cs = origin
	for cs.prev != nil {
		if cs.tailSlot > cs.prev.headSlot+1 {
			req := &chainSection{
				headSlot:    cs.tailSlot - 1,
				tailSlot:    cs.prev.headSlot + 1,
				headCounter: bc.latestHeadCounter,
				prev:        cs.prev,
				next:        cs,
			}
			cs.prev.next = req
			cs.prev = req
			req.trim(false)
			//fmt.Println(" rev", req.tailSlot, req.headSlot)
			return req
		}
		cs = cs.prev
	}

	var syncTail uint64
	if bc.historicSource != nil {
		syncTail, _ = bc.historicSource.AvailableTailSlots() //TODO long term, short term
	} else {
		if bc.syncHeadSection.headSlot > 32 {
			syncTail = bc.syncHeadSection.headSlot - 32 //TODO named constant
		}
	}

	if cs.tailSlot > syncTail {
		req := &chainSection{
			headSlot:    cs.tailSlot - 1,
			tailSlot:    syncTail,
			headCounter: bc.latestHeadCounter,
			next:        cs,
		}
		cs.prev = req
		req.trim(false)
		//fmt.Println(" tail", req.tailSlot, req.headSlot)
		return req
	}
	return nil
}

var unknownExecNumber = uint64(math.MaxUint64)

// assumes continuity; overwrite allowed
func (bc *BeaconChain) addCanonicalBlocks(parentHeader Header, blocks []*BlockData, setHead bool, tailProof MultiProof) {
	//fmt.Println("SYNC addCanonicalBlocks", len(blocks), blocks[0].Header.Slot, blocks[len(blocks)-1].Header.Slot, setHead, tailProof.Format != nil)

	lastExecNumber := unknownExecNumber
	if bc.execNumberIndexHeadPresent {
		if setHead {
			lastRemainingNumber := bc.execNumberIndexHeadNumber // last remaining exec number of the old chain
			block := bc.storedHead
			for block != nil && block.Header.Slot > uint64(parentHeader.Slot) {
				block = bc.GetParent(block)
				lastRemainingNumber--
				if lastRemainingNumber < bc.execNumberIndexTailNumber {
					bc.execNumberIndexHeadPresent = false // de-initialize exec number index if rolled back before tail
					block = nil
				}
			}
			if block != nil && block.BlockRoot == parentHeader.Hash() {
				lastExecNumber = lastRemainingNumber + uint64(len(blocks))
			}
		} else {
			if bc.execNumberIndexTailSlot == bc.tailLongTerm && bc.storedSection.parentHeader.Hash() == blocks[len(blocks)-1].BlockRoot && bc.execNumberIndexTailNumber > 0 {
				lastExecNumber = bc.execNumberIndexTailNumber - 1
			}
		}
	}

	firstSlot, blockRoots, stateRoots := blockAndStateRoots(parentHeader, blocks)
	tailSlot := blocks[0].Header.Slot - uint64(len(blocks[0].StateRootDiffs))
	//fmt.Println("addCanonicalBlocks", tailSlot, bc.tailLongTerm)
	if tailSlot < bc.tailLongTerm {
		bc.tailLongTerm = tailSlot
		if bc.storedSection != nil {
			bc.storedSection.tailSlot, bc.storedSection.parentHeader = bc.tailLongTerm, parentHeader
		}
		bc.storeBlockData(&BlockData{
			Header: HeaderWithoutState{
				Slot:          uint64(parentHeader.Slot),
				ProposerIndex: uint(parentHeader.ProposerIndex),
				ParentRoot:    parentHeader.ParentRoot,
				BodyRoot:      parentHeader.BodyRoot,
			},
			ProofFormat: 0,
			StateProof:  MerkleValues{MerkleValue(parentHeader.StateRoot)},
			BlockRoot:   parentHeader.Hash(),
			StateRoot:   parentHeader.StateRoot,
		})
	}

	for i, block := range blocks {
		if !bc.pruneBlockFormat(block) {
			log.Error("fetched state proofs insufficient", "slot", block.Header.Slot)
		}
		bc.storeBlockData(block)
		bc.storeSlotByBlockRoot(block)
		if lastExecNumber != unknownExecNumber && lastExecNumber >= uint64(len(blocks)-1-i) {
			execNumber := lastExecNumber - uint64(len(blocks)-1-i)
			bc.storeExecNumberIndex(execNumber, block)
			if execNumber > bc.execNumberIndexHeadNumber {
				bc.execNumberIndexHeadNumber = execNumber
			}
			if execNumber < bc.execNumberIndexTailNumber {
				bc.execNumberIndexTailNumber = execNumber
			}
		}
	}

	headTree := bc.makeChildTree()
	headTree.addRoots(firstSlot, blockRoots, stateRoots, setHead, blocks[0].Header.Slot-uint64(len(blocks[0].StateRootDiffs)), tailProof)
	lastSlot := blocks[len(blocks)-1].Header.Slot
	if setHead {
		bc.storedHead = blocks[len(blocks)-1]
		if bc.storedSection != nil {
			bc.storedSection.headSlot = bc.storedHead.Header.Slot
		}
		if uint64(bc.storedHead.Header.Slot) > lastSlot {
			lastSlot = uint64(bc.storedHead.Header.Slot)
		}
		headTree.HeadBlock = bc.storedHead
	}
	headTree.verifyRoots()
	bc.updateConstraints(tailSlot, lastSlot)
	batch := bc.db.NewBatch()
	bc.commitHistoricTree(batch, headTree)
	bc.storeHeadTail(batch)
	batch.Write()
	bc.updateTreeMap()

	if setHead {
		if lastExecHash, ok := blocks[len(blocks)-1].GetStateValue(BsiExecHead); ok {
			if lastExecHash != (MerkleValue{}) && bc.processedBeaconHead(blocks[len(blocks)-1].BlockRoot, common.Hash(lastExecHash)) {
				bc.callProcessedBeaconHead = blocks[len(blocks)-1].BlockRoot
			}
		} else {
			// should not happen, backend should check proof format
			log.Error("exec header root not found in beacon state", "slot", blocks[0].Header.Slot)
		}
	}
	log.Info("Inserted beacon blocks", "section tail", tailSlot, "section head", lastSlot, "chain tail", bc.tailLongTerm, "chain head", bc.storedHead.Header.Slot)
}

func (bc *BeaconChain) initWithSection(cs *chainSection) bool { // ha a result true, a chain inicializalva van, storedSection != nil
	bc.debugPrint("before initWithSection")
	defer bc.debugPrint("after initWithSection")

	//fmt.Println("SYNC initWithSection", cs.tailSlot, cs.headSlot)
	if bc.syncHeadSection == nil || cs.next != bc.syncHeadSection || cs.headSlot != uint64(bc.syncHeader.Slot) || cs.blockRootAt(uint64(bc.syncHeader.Slot)) != bc.syncHeader.Hash() {
		return false
	}

	bc.storedHead = cs.blocks[len(cs.blocks)-1]
	bc.tailLongTerm = uint64(bc.storedHead.Header.Slot) + 1
	//bc.tailShortTerm = bc.tailLongTerm
	bc.headTree = bc.newHistoricTree(bc.tailLongTerm, 0, 0)
	if bc.dataSource != nil {
		headTree := bc.makeChildTree()
		ctx, _ := context.WithTimeout(context.Background(), time.Second*2) // API backend should respond immediately so waiting here once during init is acceptable
		if err := headTree.initRecentRoots(ctx, bc.dataSource); err == nil {
			batch := bc.db.NewBatch()
			bc.commitHistoricTree(batch, headTree)
			bc.storeHeadTail(batch)
			batch.Write()
			//bc.updateTreeMap()
		} else {
			log.Error("Error retrieving recent roots from beacon API", "error", err)
			bc.storedHead, bc.headTree = nil, nil
			return false
		}
	}
	bc.addCanonicalBlocks(cs.parentHeader, cs.blocks, true, MultiProof{})
	bc.storedSection = cs
	cs.blocks, cs.tailProof = nil, MultiProof{}
	//fmt.Println(" init success")
	return true
}

// storedSection is expected to exist
func (bc *BeaconChain) mergeWithStoredSection(cs *chainSection) bool { // ha a result true, ezutan cs eldobhato
	bc.debugPrint("before mergeWithStoredSection")
	defer bc.debugPrint("after mergeWithStoredSection")

	//fmt.Println("SYNC mergeWithStoredSection", bc.storedSection.tailSlot, bc.storedSection.headSlot, cs.tailSlot, cs.headSlot)
	if cs.tailSlot > bc.storedSection.headSlot+1 || cs.headSlot+1 < bc.storedSection.tailSlot {
		//fmt.Println(" 1 false")
		return false
	}

	/*if bc.storedSection == nil {
		// try to init the chain with the current section
	}*/

	if cs.tailSlot < bc.storedSection.tailSlot {
		if cs.blockRootAt(bc.storedSection.tailSlot-1) == bc.blockRootAt(bc.storedSection.tailSlot-1) {
			//bc.addToTail(cs.blockRange(cs.tailSlot, bc.storedSection.tailSlot-1), cs.tailProof)
			parentHeader, blocks := cs.blockRange(cs.tailSlot, bc.storedSection.tailSlot-1)
			var tailProof MultiProof
			if cs.headCounter == bc.storedSection.headCounter {
				// only use tail proof if it is rooted in the same head state
				tailProof = cs.tailProof
			}
			bc.addCanonicalBlocks(parentHeader, blocks, false, tailProof)
		} else {
			if cs.headCounter <= bc.storedSection.headCounter {
				//fmt.Println(" 2 true")
				return true
			}
			bc.reset()
			//fmt.Println(" 3 false")
			return false
		}
	}

	if cs.headCounter <= bc.storedSection.headCounter {
		if cs.headSlot > bc.storedSection.headSlot && cs.blockRootAt(bc.storedSection.headSlot) == bc.blockRootAt(bc.storedSection.headSlot) {
			//bc.addToHead(cs.blockRange(bc.storedSection.headSlot+1, cs.headSlot))
			parentHeader, blocks := cs.blockRange(bc.storedSection.headSlot+1, cs.headSlot)
			bc.addCanonicalBlocks(parentHeader, blocks, true, MultiProof{})
			bc.storedSection.headCounter = cs.headCounter
			bc.nextRollback = firstRollback
		}
		//fmt.Println(" 4 true")
		return true
	}

	lastCommon := cs.headSlot
	if bc.storedSection.headSlot < lastCommon {
		lastCommon = bc.storedSection.headSlot
	}
	for cs.blockRootAt(lastCommon) != bc.blockRootAt(lastCommon) {
		if lastCommon == 0 || lastCommon < cs.tailSlot || lastCommon < bc.storedSection.tailSlot {
			rollback := bc.nextRollback
			bc.nextRollback *= rollbackMulStep
			if lastCommon >= bc.storedSection.tailSlot+rollback {
				bc.rollback(lastCommon - rollback)
			} else {
				bc.reset()
			}
			//fmt.Println(" 5 false")
			return false
		}
		lastCommon--
	}
	bc.storedSection.headCounter = cs.headCounter
	if lastCommon < bc.storedSection.headSlot {
		bc.rollback(lastCommon)
	}
	if lastCommon < cs.headSlot {
		//bc.addToHead(cs.blockRange(lastCommon+1, cs.headSlot))
		parentHeader, blocks := cs.blockRange(lastCommon+1, cs.headSlot)
		bc.addCanonicalBlocks(parentHeader, blocks, true, MultiProof{})
		bc.nextRollback = firstRollback
	}
	//fmt.Println(" 6 true")
	return true
}

func (bc *BeaconChain) addBlocksToSection(cs *chainSection, parentHeader Header, blocks []*BlockData, tailProof MultiProof) {
	//fmt.Println("SYNC addBlocksToSection", cs.tailSlot, cs.headSlot, len(blocks), blocks[0].Header.Slot, blocks[len(blocks)-1].Header.Slot, tailProof.Format != nil)
	headSlot, tailSlot := blocks[len(blocks)-1].Header.Slot, blocks[0].Header.Slot+1-blocks[0].ParentSlotDiff
	if headSlot < cs.headSlot || tailSlot > cs.tailSlot {
		panic(nil)
	}
	cs.headSlot, cs.tailSlot = headSlot, tailSlot
	cs.blocks, cs.parentHeader, cs.tailProof = blocks, parentHeader, tailProof

	for {
		if bc.storedSection == nil && (bc.syncHeadSection == nil || bc.syncHeadSection.prev == nil || !bc.initWithSection(bc.syncHeadSection.prev)) {
			return
		}
		if bc.storedSection.next != nil && !bc.storedSection.next.requesting && bc.mergeWithStoredSection(bc.storedSection.next) && bc.storedSection.next != bc.syncHeadSection {
			bc.storedSection.next.remove()
			continue
		}
		if bc.storedSection == nil {
			continue
		}
		if bc.storedSection.prev != nil && !bc.storedSection.prev.requesting && bc.mergeWithStoredSection(bc.storedSection.prev) {
			bc.storedSection.prev.remove()
		}
		if bc.storedSection != nil {
			return
		}
	}
}

func (bc *BeaconChain) SubscribeToProcessedHeads(processedCallback func(common.Hash), waitForExecHead bool) {
	// Note: called during init phase, after that these variables are read-only
	bc.processedCallback = processedCallback
	bc.waitForExecHead = waitForExecHead
}

// called under chainMu
// returns true if cb should be called with beaconHead as parameter after releasing lock
func (bc *BeaconChain) processedBeaconHead(beaconHead, execHead common.Hash) bool {
	//fmt.Println("processedBeaconHead", beaconHead, execHead)
	if !bc.execNumberIndexHeadPresent && bc.lastProcessedExecHead == execHead {
		bc.startExecNumberIndexFromHead(bc.lastProcessedExecNumber)
	}
	if bc.processedCallback == nil {
		return false
	}
	bc.lastProcessedBeaconHead, bc.expectedExecHead = beaconHead, execHead
	if bc.lastReportedBeaconHead != beaconHead && (!bc.waitForExecHead || bc.lastProcessedExecHead == execHead) {
		bc.lastReportedBeaconHead = beaconHead
		return true
	}
	return false
}

func (bc *BeaconChain) ProcessedExecHead(header *types.Header) {
	bc.chainMu.Lock()
	execNumber, execHead := header.Number.Uint64(), header.Hash()
	if !bc.execNumberIndexHeadPresent && bc.expectedExecHead == execHead {
		bc.startExecNumberIndexFromHead(execNumber)
	}
	var reportHead common.Hash
	if bc.processedCallback != nil && bc.waitForExecHead {
		if bc.expectedExecHead == execHead && bc.lastReportedBeaconHead != bc.lastProcessedBeaconHead {
			reportHead = bc.lastProcessedBeaconHead
			bc.lastReportedBeaconHead = reportHead
		}
		bc.lastProcessedExecNumber, bc.lastProcessedExecHead = execNumber, execHead
	}
	bc.chainMu.Unlock()

	if reportHead != (common.Hash{}) {
		bc.processedCallback(reportHead)
	}
}

func (bc *BeaconChain) debugPrint(id string) {
	/*	if bc.storedHead != nil {
			//fmt.Println("***", id, "*** storedHead", bc.storedHead.Header.Slot)
		} else {
			//fmt.Println("***", id, "*** no storedHead")
		}
		cs := bc.syncHeadSection
		for cs != nil {
			var bs, be uint64
			if len(cs.blocks) > 0 {
				bs, be = cs.blocks[0].Header.Slot, cs.blocks[len(cs.blocks)-1].Header.Slot
			}
			//fmt.Println("***", id, "*** cs  headCounter", cs.headCounter, "requesting", cs.requesting, "stored", cs == bc.storedSection, "tail", cs.tailSlot, "head", cs.headSlot, "blocks", len(cs.blocks), bs, be)
			cs = cs.prev
		}
		//fmt.Println("***", id, "*** tail", bc.tailLongTerm)
	*/
}

func (bc *BeaconChain) startExecNumberIndexFromHead(headExecNumber uint64) {
	if bc.storedHead == nil {
		return
	}
	bc.storeExecNumberIndex(headExecNumber, bc.storedHead)
	bc.execNumberIndexTailSlot, bc.execNumberIndexTailNumber = bc.storedHead.Header.Slot, headExecNumber
	bc.execNumberIndexHeadPresent, bc.execNumberIndexHeadNumber = true, headExecNumber
	bc.extendExecNumberIndex(headExecNumber, bc.storedHead)
}

func (bc *BeaconChain) extendExecNumberIndex(execNumber uint64, block *BlockData) {
	bc.syncWg.Add(1)
	go func() {
		bc.chainMu.Lock()
	loop:
		for {
			if bc.execNumberIndexTailSlot <= bc.tailLongTerm || execNumber == 0 {
				bc.chainMu.Unlock()
				log.Info("Finished indexing beacon blocks by exec block number")
				break loop
			}
			if execNumber%1000 == 0 {
				bc.chainMu.Unlock()
				select {
				case <-time.After(time.Millisecond * 10):
				case <-bc.stopSyncCh:
					break loop
				}
				bc.chainMu.Lock()
			}
			if !bc.execNumberIndexHeadPresent {
				// index has been de-initialized by a rollback, next init starts a new goroutine
				bc.chainMu.Unlock()
				break loop
			}
			block = bc.GetParent(block)
			if block == nil {
				bc.chainMu.Unlock()
				log.Error("Missing beacon block after chain tail")
				break loop
			}
			execNumber--
			bc.storeExecNumberIndex(execNumber, block)
			bc.execNumberIndexTailSlot = block.Header.Slot - uint64(len(block.StateRootDiffs))
		}
		bc.syncWg.Done()
	}()
}

func (bc *BeaconChain) initExecNumberIndex() {
	bc.chainMu.Lock()
	defer bc.chainMu.Unlock()

	if bc.headTree == nil {
		return
	}
	tailNumber, tailBlock := bc.firstCanonicalBlockWithExecNumber(0)
	if tailBlock == nil {
		return
	}
	tailSlot := tailBlock.Header.Slot - uint64(len(tailBlock.StateRootDiffs))
	if tailSlot < bc.tailLongTerm {
		log.Warn("Exec number index entry found before beacon chain tail", "exec number index tail", tailSlot, "beacon chain tail", bc.tailLongTerm)
		for tailSlot < bc.tailLongTerm {
			bc.deleteExecNumberIndex(tailNumber, tailBlock.StateRoot)
			tailNumber, tailBlock = bc.firstCanonicalBlockWithExecNumber(0)
			if tailBlock == nil {
				return
			}
			tailSlot = tailBlock.Header.Slot - uint64(len(tailBlock.StateRootDiffs))
		}
	}
	bc.execNumberIndexTailSlot, bc.execNumberIndexTailNumber = tailSlot, tailNumber

	// find entry for head slot
	minNumber, maxNumber := tailNumber, tailNumber+uint64(bc.storedHead.Header.Slot-tailBlock.Header.Slot) // firstCanonicalBlockWithExecNumber ensures that the slot diff is not negative
	for maxNumber > minNumber {
		m := (minNumber + maxNumber + 1) / 2
		if number, block := bc.firstCanonicalBlockWithExecNumber(m); block != nil {
			minNumber = number
		} else {
			maxNumber = m - 1
		}
	}
	bc.execNumberIndexHeadPresent, bc.execNumberIndexHeadNumber = true, maxNumber

	if bc.execNumberIndexTailSlot > bc.tailLongTerm {
		bc.extendExecNumberIndex(tailNumber, tailBlock)
	}
}

func (bc *BeaconChain) firstCanonicalBlockWithExecNumber(start uint64) (execNumber uint64, block *BlockData) {
	if bc.headTree == nil {
		return 0, nil
	}
	var startEnc [8]byte
	binary.BigEndian.PutUint64(startEnc[:], start)
	iter := bc.db.NewIterator(execNumberKey, startEnc[:])
	defer iter.Release()

	p := len(execNumberKey)
	for iter.Next() {
		if len(iter.Key()) != p+40 {
			log.Error("Invalid exec number entry key length", "length", len(iter.Key()), "expected", p+40)
			continue
		}
		var (
			slot      uint64
			stateRoot common.Hash
		)
		if err := rlp.DecodeBytes(iter.Value(), &slot); err != nil {
			log.Error("Error decoding stored exec number entry", "error", err)
			continue
		}
		if slot < bc.tailLongTerm {
			continue
		}
		if slot > bc.headTree.HeadBlock.Header.Slot {
			return 0, nil
		}
		copy(stateRoot[:], iter.Key()[p+8:p+40])
		if bc.headTree.GetStateRoot(slot) == stateRoot {
			if block = bc.GetBlockData(slot, stateRoot, false); block != nil {
				return binary.BigEndian.Uint64(iter.Key()[p : p+8]), block
			}
			log.Error("Canonical beacon block missing", "slot", slot, "stateRoot", stateRoot)
		}
	}
	return 0, nil
}

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

package main

import (
	"encoding/hex"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/beacon/light/api"
	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
)

const (
	maxTestRequestAge = 64
	testStateSubCount = 8
)

type testSyncer struct {
	lock              sync.Mutex
	api               *api.BeaconLightApi
	stateProofVersion int
	subs              []*api.StateProofSub // only for new api
	// maps are nil until initialized by first head
	headers            map[uint64]types.Header // slot -> canonical header
	headSlot, tailSlot uint64                  // headers available between tailSlot..headSlot
	// waitForSig contains the arrival time of canonical headers (also in headers map)
	// which are waiting for a signature (no signed header seen, not timed out yet).
	waitForSig                       map[common.Hash]mclock.AbsTime // blockRoot -> abs time
	headLastSeen                     mclock.AbsTime                 // last time when the current head was confirmed
	headStateCount, recentStateCount int
}

func newTestSyncer(api *api.BeaconLightApi, stateProofVersion int) *testSyncer {
	rand.Seed(time.Now().UnixNano())
	t := &testSyncer{
		api:               api,
		stateProofVersion: stateProofVersion,
	}
	go t.updateLoop()
	return t
}

func (t *testSyncer) createSubs() bool {
	t.subs = make([]*api.StateProofSub, testStateSubCount)
	for i := range t.subs {
		subFormat, subPaths := t.makeTestFormat(3, 0.25/float32(i))
		encFormat, _ := api.EncodeCompactProofFormat(subFormat)
		hexFormat := "0x" + hex.EncodeToString(encFormat)
		if sub, err := t.api.SubscribeStateProof(subFormat, subPaths, 0, 1); err == nil {
			log.Info("Successfully created state subscription", "subscription index", i, "format", hexFormat)
			t.subs[i] = sub
		} else {
			log.Error("Could not create state subscription", "format", hexFormat, "error", err)
			t.subs = nil
			return false
		}
	}
	if err := t.updateHead(true); err != nil {
		log.Error("Error retrieving updated head", "error", err)
		return false
	}
	t.pruneTail(t.headSlot)
	return true
}

func (t *testSyncer) pruneTail(newTail uint64) {
	if newTail > t.tailSlot {
		for slot := t.tailSlot; slot < newTail; slot++ {
			if header, ok := t.headers[slot]; ok {
				delete(t.waitForSig, header.Hash())
				delete(t.headers, slot)
			}
		}
		t.tailSlot = newTail
	}
}

func (t *testSyncer) updateLoop() {
	for {
		time.Sleep(time.Millisecond * 100)
		t.lock.Lock()
		t.checkSignatureTimeouts()
		if err := t.updateHead(false); err == nil {
			t.testHeadProof()
			t.testRecentProof()
			t.lock.Unlock()
		} else {
			t.lock.Unlock()
			log.Warn("Could not retrieve head", "error", err)
			time.Sleep(time.Second)
		}
	}
}

func (t *testSyncer) makeTestFormat(avgIndexCount, stopRatio float32) (merkle.ProofFormat, []string) {
	format := merkle.NewIndexMapFormat()
	var paths []string
	if t.stateProofVersion >= 2 {
		for i := uint64(params.BsiGenesisTime); i <= params.BsiExecPayload; i++ {
			format.AddLeaf(i, nil)
		}
		format.AddLeaf(params.BsiForkVersion, nil)
		format.AddLeaf(params.BsiFinalBlock, nil)
		format.AddLeaf(params.BsiExecHead, nil)
	} else {
		format.AddLeaf(params.BsiFinalBlock, nil)
		format.AddLeaf(params.BsiExecHead, nil)
		paths = []string{
			"[\"finalizedCheckpoint\",\"root\"]",
			"[\"latestExecutionPayloadHeader\",\"blockHash\"]",
		}
	}
	for rand.Float32()*avgIndexCount > 1 {
		srIndex := rand.Intn(0x2000)
		format.AddLeaf(merkle.ChildIndex(params.BsiStateRoots, uint64(0x2000+srIndex)), nil)
		if t.stateProofVersion == 1 {
			paths = append(paths, "[\"stateRoots\","+strconv.Itoa(srIndex)+"]")
		}
	}
	for rand.Float32()*avgIndexCount > 1 {
		brIndex := rand.Intn(0x2000)
		format.AddLeaf(merkle.ChildIndex(params.BsiBlockRoots, uint64(0x2000+brIndex)), nil)
		if t.stateProofVersion == 1 {
			paths = append(paths, "[\"blockRoots\","+strconv.Itoa(brIndex)+"]")
		}
	}
	for rand.Float32()*avgIndexCount > 1 {
		hrIndex := rand.Intn(0x1000000)
		format.AddLeaf(merkle.ChildIndex(params.BsiHistoricRoots, merkle.ChildIndex(2, uint64(0x1000000+hrIndex))), nil)
		if t.stateProofVersion == 1 {
			paths = append(paths, "[\"historicalRoots\","+strconv.Itoa(hrIndex)+"]")
		}
	}
	//TODO sample all lists/vectors?
	return randomSubset(format, stopRatio), paths
}

func randomSubset(format merkle.ProofFormat, stopRatio float32) merkle.ProofFormat {
	subset := merkle.NewIndexMapFormat()
	addRandomSubset(format, subset, 1, stopRatio)
	return subset
}

func addRandomSubset(format merkle.ProofFormat, subset merkle.IndexMapFormat, index uint64, stopRatio float32) {
	left, right := format.Children()
	if left == nil || rand.Float32() < stopRatio {
		subset.AddLeaf(index, nil)
		return
	}
	addRandomSubset(left, subset, index*2, stopRatio)
	addRandomSubset(right, subset, index*2+1, stopRatio)
}

func (t *testSyncer) testHeadProof() {
	format, paths := t.makeTestFormat(3, 0.1)
	proof, err := t.api.GetHeadStateProof(format, paths)
	if err != nil {
		encFormat, _ := api.EncodeCompactProofFormat(format)
		log.Error("Error retrieving head state proof", "format", "0x"+hex.EncodeToString(encFormat), "error", err)
		return
	}
	stateRoot := proof.RootHash()
	if stateRoot == t.headers[t.headSlot].StateRoot {
		t.headStateCount++
		t.headLastSeen = mclock.Now()
		return
	}
	oldHeadSlot := t.headSlot
	if err := t.updateHead(true); err != nil {
		log.Error("Error retrieving updated head", "error", err)
		return
	}
	if header, ok := t.findHeaderWithStateRoot(stateRoot); !ok || header.Slot < oldHeadSlot {
		if ok {
			log.Error("Head state proof request returned proof with old state root", "slot", header.Slot, "head slot before request", oldHeadSlot)
		} else {
			log.Error("Head state proof request returned proof with unknown state root")
		}
		return
	}
	t.headStateCount++
}

func (t *testSyncer) findHeaderWithBlockRoot(blockRoot common.Hash) (types.Header, bool) {
	for _, header := range t.headers {
		if header.Hash() == blockRoot {
			return header, true
		}
	}
	return types.Header{}, false
}

func (t *testSyncer) findHeaderWithStateRoot(stateRoot common.Hash) (types.Header, bool) {
	for _, header := range t.headers {
		if header.StateRoot == stateRoot {
			return header, true
		}
	}
	return types.Header{}, false
}

func (t *testSyncer) testRecentProof() {
	if t.subs == nil && !t.createSubs() {
		return
	}
	maxAge := t.headSlot - t.tailSlot
	if maxAge > maxTestRequestAge {
		maxAge = maxTestRequestAge
		t.pruneTail(t.headSlot - maxTestRequestAge)
	}
	var (
		slot      uint64
		stateRoot common.Hash
	)
	for slot = t.headSlot - uint64(rand.Intn(int(maxAge)+1)); slot <= t.headSlot; slot++ {
		if header, ok := t.headers[slot]; ok {
			stateRoot = header.StateRoot
			break
		}
	}
	subIndex := rand.Intn(testStateSubCount)
	if _, err := t.subs[subIndex].Get(stateRoot); err != nil { // sub.Get checks state root
		log.Error("Error retrieving subscribed state proof", "error", err, "subscription index", subIndex, "requested slot", slot, "head slot", t.headSlot)
		return
	}
	t.recentStateCount++
}

func (t *testSyncer) resetChain(head types.Header) {
	t.headers = make(map[uint64]types.Header)
	t.waitForSig = make(map[common.Hash]mclock.AbsTime)
	t.headSlot, t.tailSlot = head.Slot, head.Slot
	t.headers[head.Slot] = head
	t.headLastSeen = mclock.Now()
	// do not add to waitForSig because we don't know when this head has first appeared
}

func (t *testSyncer) updateHead(force bool) error {
	if !force && t.headers != nil && time.Duration(mclock.Now()-t.headLastSeen) < time.Second {
		return nil
	}
	head, err := t.api.GetHeader(common.Hash{})
	if err != nil {
		return err
	}
	if t.headers == nil {
		t.resetChain(head)
		log.Info("Initialized header chain", "slot", head.Slot)
	}
	oldHeadSeen := t.headLastSeen
	t.headLastSeen = mclock.Now()
	if head == t.headers[t.headSlot] {
		return nil
	}
	if time.Duration(t.headLastSeen-oldHeadSeen) < time.Millisecond*500 {
		t.waitForSig[head.Hash()] = t.headLastSeen
	}
	removeSlot := t.headSlot
	t.headSlot = head.Slot
	for {
		var (
			parent      types.Header
			parentFound bool
		)
		slot := head.Slot
		for slot > t.tailSlot {
			slot--
			if header, ok := t.headers[slot]; ok {
				if header.Hash() == head.ParentRoot {
					parent, parentFound = header, true
				}
				break
			}
		}
		if !parentFound {
			parent, err = t.api.GetHeader(head.ParentRoot)
			if err != nil {
				//cannot trace back to the known chain
				t.resetChain(head)
				return err
			}
		}
		// now parent is always valid
		for slot := parent.Slot + 1; slot <= removeSlot; slot++ {
			if header, ok := t.headers[slot]; ok {
				delete(t.headers, slot)
				delete(t.waitForSig, header.Hash())
			}
		}
		removeSlot = parent.Slot
		t.headers[head.Slot] = head
		head = parent
		if parentFound {
			return nil
		}
		if head.Slot < t.tailSlot {
			t.resetChain(head)
			return nil
		}
	}
}

func (t *testSyncer) newSignedHead(head types.Header) {
	now := mclock.Now()
	go func() {
		var delay interface{}
		hash := head.Hash()
		t.lock.Lock()
		if arrivedAt, ok := t.waitForSig[hash]; ok {
			delay = time.Duration(now - arrivedAt)
			delete(t.waitForSig, hash)
		} else {
			delay = "unknown"
		}
		t.lock.Unlock()
		log.Info("Received new signed head", "slot", head.Slot, "blockRoot", hash, "delay", delay, "head states retrieved", t.headStateCount, "subscribed states retrieved", t.recentStateCount)
	}()
}

func (t *testSyncer) checkSignatureTimeouts() {
	now := mclock.Now()
	for hash, arrivedAt := range t.waitForSig {
		if dt := time.Duration(now - arrivedAt); dt > time.Second*65 {
			header, _ := t.findHeaderWithBlockRoot(hash)
			log.Warn("Wait for header signature timed out", "slot", header.Slot, "blockRoot", hash, "delay", dt)
			delete(t.waitForSig, hash)
		}
	}
}

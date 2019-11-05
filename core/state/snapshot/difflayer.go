// Copyright 2019 The go-ethereum Authors
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

package snapshot

import (
	"encoding/binary"
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/steakknife/bloomfilter"
)

var (
	// aggregatorMemoryLimit is the maximum size of the bottom-most diff layer
	// that aggregates the writes from above until it's flushed into the disk
	// layer.
	//
	// Note, bumping this up might drastically increase the size of the bloom
	// filters that's stored in every diff layer. Don't do that without fully
	// understanding all the implications.
	aggregatorMemoryLimit = uint64(4 * 1024 * 1024)

	// aggregatorItemLimit is an approximate number of items that will end up
	// in the agregator layer before it's flushed out to disk. A plain account
	// weighs around 14B (+hash), a storage slot 32B (+hash), a deleted slot
	// 0B (+hash). Slots are mostly set/unset in lockstep, so thet average at
	// 16B (+hash). All in all, the average entry seems to be 15+32=47B. Use a
	// smaller number to be on the safe side.
	aggregatorItemLimit = aggregatorMemoryLimit / 42

	// bloomTargetError is the target false positive rate when the aggregator
	// layer is at its fullest. The actual value will probably move around up
	// and down from this number, it's mostly a ballpark figure.
	//
	// Note, dropping this down might drastically increase the size of the bloom
	// filters that's stored in every diff layer. Don't do that without fully
	// understanding all the implications.
	bloomTargetError = 0.02

	// bloomSize is the ideal bloom filter size given the maximum number of items
	// it's expected to hold and the target false positive error rate.
	bloomSize = math.Ceil(float64(aggregatorItemLimit) * math.Log(bloomTargetError) / math.Log(1/math.Pow(2, math.Log(2))))

	// bloomFuncs is the ideal number of bits a single entry should set in the
	// bloom filter to keep its size to a minimum (given it's size and maximum
	// entry count).
	bloomFuncs = math.Round((bloomSize / float64(aggregatorItemLimit)) * math.Log(2))

	// bloomHashesOffset is a runtime constant which determines which part of the
	// the account/storage hash the hasher functions looks at, to determine the
	// bloom key for an account/slot. This is randomized at init(), so that the
	// global population of nodes do not all display the exact same behaviour with
	// regards to bloom content
	bloomHasherOffset = 0
)

func init() {
	// Init bloomHasherOffset in the range [0:24] (requires 8 bytes)
	bloomHasherOffset = rand.Intn(25)
}

// diffLayer represents a collection of modifications made to a state snapshot
// after running a block on top. It contains one sorted list for the account trie
// and one-one list for each storage tries.
//
// The goal of a diff layer is to act as a journal, tracking recent modifications
// made to the state, that have not yet graduated into a semi-immutable state.
type diffLayer struct {
	origin *diskLayer // Base disk layer to directly use on bloom misses
	parent snapshot   // Parent snapshot modified by this one, never nil
	memory uint64     // Approximate guess as to how much memory we use

	root  common.Hash // Root hash to which this snapshot diff belongs to
	stale bool        // Signals that the layer became stale (state progressed)

	accountList []common.Hash                          // List of account for iteration. If it exists, it's sorted, otherwise it's nil
	accountData map[common.Hash][]byte                 // Keyed accounts for direct retrival (nil means deleted)
	storageList map[common.Hash][]common.Hash          // List of storage slots for iterated retrievals, one per account. Any existing lists are sorted if non-nil
	storageData map[common.Hash]map[common.Hash][]byte // Keyed storage slots for direct retrival. one per account (nil means deleted)

	diffed *bloomfilter.Filter // Bloom filter tracking all the diffed items up to the disk layer

	lock sync.RWMutex
}

// accountBloomHasher is a wrapper around a common.Hash to satisfy the interface
// API requirements of the bloom library used. It's used to convert an account
// hash into a 64 bit mini hash.
type accountBloomHasher common.Hash

func (h accountBloomHasher) Write(p []byte) (n int, err error) { panic("not implemented") }
func (h accountBloomHasher) Sum(b []byte) []byte               { panic("not implemented") }
func (h accountBloomHasher) Reset()                            { panic("not implemented") }
func (h accountBloomHasher) BlockSize() int                    { panic("not implemented") }
func (h accountBloomHasher) Size() int                         { return 8 }
func (h accountBloomHasher) Sum64() uint64 {
	return binary.BigEndian.Uint64(h[bloomHasherOffset : bloomHasherOffset+8])
}

// storageBloomHasher is a wrapper around a [2]common.Hash to satisfy the interface
// API requirements of the bloom library used. It's used to convert an account
// hash into a 64 bit mini hash.
type storageBloomHasher [2]common.Hash

func (h storageBloomHasher) Write(p []byte) (n int, err error) { panic("not implemented") }
func (h storageBloomHasher) Sum(b []byte) []byte               { panic("not implemented") }
func (h storageBloomHasher) Reset()                            { panic("not implemented") }
func (h storageBloomHasher) BlockSize() int                    { panic("not implemented") }
func (h storageBloomHasher) Size() int                         { return 8 }
func (h storageBloomHasher) Sum64() uint64 {
	return binary.BigEndian.Uint64(h[0][bloomHasherOffset:bloomHasherOffset+8]) ^
		binary.BigEndian.Uint64(h[1][bloomHasherOffset:bloomHasherOffset+8])
}

// newDiffLayer creates a new diff on top of an existing snapshot, whether that's a low
// level persistent database or a hierarchical diff already.
func newDiffLayer(parent snapshot, root common.Hash, accounts map[common.Hash][]byte, storage map[common.Hash]map[common.Hash][]byte) *diffLayer {
	// Create the new layer with some pre-allocated data segments
	dl := &diffLayer{
		parent:      parent,
		root:        root,
		accountData: accounts,
		storageData: storage,
	}
	switch parent := parent.(type) {
	case *diskLayer:
		dl.rebloom(parent)
	case *diffLayer:
		dl.rebloom(parent.origin)
	default:
		panic("unknown parent type")
	}
	// Determine memory size and track the dirty writes
	for _, data := range accounts {
		dl.memory += uint64(common.HashLength + len(data))
		snapshotDirtyAccountWriteMeter.Mark(int64(len(data)))
	}
	// Fill the storage hashes and sort them for the iterator
	dl.storageList = make(map[common.Hash][]common.Hash)

	for accountHash, slots := range storage {
		// If the slots are nil, sanity check that it's a deleted account
		if slots == nil {
			// Ensure that the account was just marked as deleted
			if account, ok := accounts[accountHash]; account != nil || !ok {
				panic(fmt.Sprintf("storage in %#x nil, but account conflicts (%#x, exists: %v)", accountHash, account, ok))
			}
			// Everything ok, store the deletion mark and continue
			dl.storageList[accountHash] = nil
			continue
		}
		// Storage slots are not nil so entire contract was not deleted, ensure the
		// account was just updated.
		if account, ok := accounts[accountHash]; account == nil || !ok {
			log.Error(fmt.Sprintf("storage in %#x exists, but account nil (exists: %v)", accountHash, ok))
		}
		// Determine memory size and track the dirty writes
		for _, data := range slots {
			dl.memory += uint64(common.HashLength + len(data))
			snapshotDirtyStorageWriteMeter.Mark(int64(len(data)))
		}
	}
	dl.memory += uint64(len(dl.storageList) * common.HashLength)
	return dl
}

// rebloom discards the layer's current bloom and rebuilds it from scratch based
// on the parent's and the local diffs.
func (dl *diffLayer) rebloom(origin *diskLayer) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	defer func(start time.Time) {
		snapshotBloomIndexTimer.Update(time.Since(start))
	}(time.Now())

	// Inject the new origin that triggered the rebloom
	dl.origin = origin

	// Retrieve the parent bloom or create a fresh empty one
	if parent, ok := dl.parent.(*diffLayer); ok {
		parent.lock.RLock()
		dl.diffed, _ = parent.diffed.Copy()
		parent.lock.RUnlock()
	} else {
		dl.diffed, _ = bloomfilter.New(uint64(bloomSize), uint64(bloomFuncs))
	}
	// Iterate over all the accounts and storage slots and index them
	for hash := range dl.accountData {
		dl.diffed.Add(accountBloomHasher(hash))
	}
	for accountHash, slots := range dl.storageData {
		for storageHash := range slots {
			dl.diffed.Add(storageBloomHasher{accountHash, storageHash})
		}
	}
	// Calculate the current false positive rate and update the error rate meter.
	// This is a bit cheating because subsequent layers will overwrite it, but it
	// should be fine, we're only interested in ballpark figures.
	k := float64(dl.diffed.K())
	n := float64(dl.diffed.N())
	m := float64(dl.diffed.M())
	snapshotBloomErrorGauge.Update(math.Pow(1.0-math.Exp((-k)*(n+0.5)/(m-1)), k))
}

// Root returns the root hash for which this snapshot was made.
func (dl *diffLayer) Root() common.Hash {
	return dl.root
}

// Stale return whether this layer has become stale (was flattened across) or if
// it's still live.
func (dl *diffLayer) Stale() bool {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return dl.stale
}

// Account directly retrieves the account associated with a particular hash in
// the snapshot slim data format.
func (dl *diffLayer) Account(hash common.Hash) (*Account, error) {
	data, err := dl.AccountRLP(hash)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 { // can be both nil and []byte{}
		return nil, nil
	}
	account := new(Account)
	if err := rlp.DecodeBytes(data, account); err != nil {
		panic(err)
	}
	return account, nil
}

// AccountRLP directly retrieves the account RLP associated with a particular
// hash in the snapshot slim data format.
func (dl *diffLayer) AccountRLP(hash common.Hash) ([]byte, error) {
	// Check the bloom filter first whether there's even a point in reaching into
	// all the maps in all the layers below
	dl.lock.RLock()
	hit := dl.diffed.Contains(accountBloomHasher(hash))
	dl.lock.RUnlock()

	// If the bloom filter misses, don't even bother with traversing the memory
	// diff layers, reach straight into the bottom persistent disk layer
	if !hit {
		snapshotBloomAccountMissMeter.Mark(1)
		return dl.origin.AccountRLP(hash)
	}
	// The bloom filter hit, start poking in the internal maps
	return dl.accountRLP(hash, 0)
}

// accountRLP is an internal version of AccountRLP that skips the bloom filter
// checks and uses the internal maps to try and retrieve the data. It's meant
// to be used if a higher layer's bloom filter hit already.
func (dl *diffLayer) accountRLP(hash common.Hash, depth int) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// If the layer was flattened into, consider it invalid (any live reference to
	// the original should be marked as unusable).
	if dl.stale {
		return nil, ErrSnapshotStale
	}
	// If the account is known locally, return it. Note, a nil account means it was
	// deleted, and is a different notion than an unknown account!
	if data, ok := dl.accountData[hash]; ok {
		snapshotDirtyAccountHitMeter.Mark(1)
		snapshotDirtyAccountHitDepthHist.Update(int64(depth))
		if n := len(data); n > 0 {
			snapshotDirtyAccountReadMeter.Mark(int64(n))
		} else {
			snapshotDirtyAccountInexMeter.Mark(1)
		}
		snapshotBloomAccountTrueHitMeter.Mark(1)
		return data, nil
	}
	// Account unknown to this diff, resolve from parent
	if diff, ok := dl.parent.(*diffLayer); ok {
		return diff.accountRLP(hash, depth+1)
	}
	// Failed to resolve through diff layers, mark a bloom error and use the disk
	snapshotBloomAccountFalseHitMeter.Mark(1)
	return dl.parent.AccountRLP(hash)
}

// Storage directly retrieves the storage data associated with a particular hash,
// within a particular account. If the slot is unknown to this diff, it's parent
// is consulted.
func (dl *diffLayer) Storage(accountHash, storageHash common.Hash) ([]byte, error) {
	// Check the bloom filter first whether there's even a point in reaching into
	// all the maps in all the layers below
	dl.lock.RLock()
	hit := dl.diffed.Contains(storageBloomHasher{accountHash, storageHash})
	dl.lock.RUnlock()

	// If the bloom filter misses, don't even bother with traversing the memory
	// diff layers, reach straight into the bottom persistent disk layer
	if !hit {
		snapshotBloomStorageMissMeter.Mark(1)
		return dl.origin.Storage(accountHash, storageHash)
	}
	// The bloom filter hit, start poking in the internal maps
	return dl.storage(accountHash, storageHash, 0)
}

// storage is an internal version of Storage that skips the bloom filter checks
// and uses the internal maps to try and retrieve the data. It's meant  to be
// used if a higher layer's bloom filter hit already.
func (dl *diffLayer) storage(accountHash, storageHash common.Hash, depth int) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// If the layer was flattened into, consider it invalid (any live reference to
	// the original should be marked as unusable).
	if dl.stale {
		return nil, ErrSnapshotStale
	}
	// If the account is known locally, try to resolve the slot locally. Note, a nil
	// account means it was deleted, and is a different notion than an unknown account!
	if storage, ok := dl.storageData[accountHash]; ok {
		if storage == nil {
			snapshotDirtyStorageHitMeter.Mark(1)
			snapshotDirtyStorageHitDepthHist.Update(int64(depth))
			snapshotDirtyStorageInexMeter.Mark(1)
			snapshotBloomStorageTrueHitMeter.Mark(1)
			return nil, nil
		}
		if data, ok := storage[storageHash]; ok {
			snapshotDirtyStorageHitMeter.Mark(1)
			snapshotDirtyStorageHitDepthHist.Update(int64(depth))
			if n := len(data); n > 0 {
				snapshotDirtyStorageReadMeter.Mark(int64(n))
			} else {
				snapshotDirtyStorageInexMeter.Mark(1)
			}
			snapshotBloomStorageTrueHitMeter.Mark(1)
			return data, nil
		}
	}
	// Storage slot unknown to this diff, resolve from parent
	if diff, ok := dl.parent.(*diffLayer); ok {
		return diff.storage(accountHash, storageHash, depth+1)
	}
	// Failed to resolve through diff layers, mark a bloom error and use the disk
	snapshotBloomStorageFalseHitMeter.Mark(1)
	return dl.parent.Storage(accountHash, storageHash)
}

// Update creates a new layer on top of the existing snapshot diff tree with
// the specified data items.
func (dl *diffLayer) Update(blockRoot common.Hash, accounts map[common.Hash][]byte, storage map[common.Hash]map[common.Hash][]byte) *diffLayer {
	return newDiffLayer(dl, blockRoot, accounts, storage)
}

// flatten pushes all data from this point downwards, flattening everything into
// a single diff at the bottom. Since usually the lowermost diff is the largest,
// the flattening bulds up from there in reverse.
func (dl *diffLayer) flatten() snapshot {
	// If the parent is not diff, we're the first in line, return unmodified
	parent, ok := dl.parent.(*diffLayer)
	if !ok {
		return dl
	}
	// Parent is a diff, flatten it first (note, apart from weird corned cases,
	// flatten will realistically only ever merge 1 layer, so there's no need to
	// be smarter about grouping flattens together).
	parent = parent.flatten().(*diffLayer)

	parent.lock.Lock()
	defer parent.lock.Unlock()

	// Before actually writing all our data to the parent, first ensure that the
	// parent hasn't been 'corrupted' by someone else already flattening into it
	if parent.stale {
		panic("parent diff layer is stale") // we've flattened into the same parent from two children, boo
	}
	parent.stale = true

	// Overwrite all the updated accounts blindly, merge the sorted list
	for hash, data := range dl.accountData {
		parent.accountData[hash] = data
	}
	// Overwrite all the updates storage slots (individually)
	for accountHash, storage := range dl.storageData {
		// If storage didn't exist (or was deleted) in the parent; or if the storage
		// was freshly deleted in the child, overwrite blindly
		if parent.storageData[accountHash] == nil || storage == nil {
			parent.storageData[accountHash] = storage
			continue
		}
		// Storage exists in both parent and child, merge the slots
		comboData := parent.storageData[accountHash]
		for storageHash, data := range storage {
			comboData[storageHash] = data
		}
		parent.storageData[accountHash] = comboData
	}
	// Return the combo parent
	return &diffLayer{
		parent:      parent.parent,
		origin:      parent.origin,
		root:        dl.root,
		storageList: parent.storageList,
		storageData: parent.storageData,
		accountList: parent.accountList,
		accountData: parent.accountData,
		diffed:      dl.diffed,
		memory:      parent.memory + dl.memory,
	}
}

// AccountList returns a sorted list of all accounts in this difflayer.
func (dl *diffLayer) AccountList() []common.Hash {
	dl.lock.Lock()
	defer dl.lock.Unlock()
	if dl.accountList != nil {
		return dl.accountList
	}
	accountList := make([]common.Hash, len(dl.accountData))
	i := 0
	for k, _ := range dl.accountData {
		accountList[i] = k
		i++
		// This would be a pretty good opportunity to also
		// calculate the size, if we want to
	}
	sort.Sort(hashes(accountList))
	dl.accountList = accountList
	return dl.accountList
}

// StorageList returns a sorted list of all storage slot hashes
// in this difflayer for the given account.
func (dl *diffLayer) StorageList(accountHash common.Hash) []common.Hash {
	dl.lock.Lock()
	defer dl.lock.Unlock()
	if dl.storageList[accountHash] != nil {
		return dl.storageList[accountHash]
	}
	accountStorageMap := dl.storageData[accountHash]
	accountStorageList := make([]common.Hash, len(accountStorageMap))
	i := 0
	for k, _ := range accountStorageMap {
		accountStorageList[i] = k
		i++
		// This would be a pretty good opportunity to also
		// calculate the size, if we want to
	}
	sort.Sort(hashes(accountStorageList))
	dl.storageList[accountHash] = accountStorageList
	return accountStorageList
}

type Iterator interface {
	// Next steps the iterator forward one element, and returns false if
	// the iterator is exhausted
	Next() bool
	// Key returns the current key
	Key() common.Hash
	// Seek steps the iterator forward as many elements as needed, so that after
	// calling Next(), the iterator will be at a key higher than the given hash
	Seek(common.Hash)
}

func (dl *diffLayer) newIterator() Iterator {
	dl.AccountList()
	return &dlIterator{dl, -1}
}

type dlIterator struct {
	layer *diffLayer
	index int
}

func (it *dlIterator) Next() bool {
	if it.index < len(it.layer.accountList) {
		it.index++
	}
	return it.index < len(it.layer.accountList)
}

func (it *dlIterator) Key() common.Hash {
	if it.index < len(it.layer.accountList) {
		return it.layer.accountList[it.index]
	}
	return common.Hash{}
}

func (it *dlIterator) Seek(key common.Hash) {
	// Search uses binary search to find and return the smallest index i
	// in [0, n) at which f(i) is true
	size := len(it.layer.accountList)
	index := sort.Search(size,
		func(i int) bool {
			v := it.layer.accountList[i]
			return bytes.Compare(key[:], v[:]) < 0
		})
	it.index = index - 1
}

type binaryIterator struct {
	a     Iterator
	b     Iterator
	aDone bool
	bDone bool
	k     common.Hash
}

func (dl *diffLayer) newBinaryIterator() Iterator {
	parent, ok := dl.parent.(*diffLayer)
	if !ok {
		// parent is the disk layer
		return dl.newIterator()
	}
	l := &binaryIterator{
		a: dl.newIterator(),
		b: parent.newBinaryIterator()}

	l.aDone = !l.a.Next()
	l.bDone = !l.b.Next()
	return l
}

func (it *binaryIterator) Next() bool {

	if it.aDone && it.bDone {
		return false
	}
	nextB := it.b.Key()
first:
	nextA := it.a.Key()
	if it.aDone {
		it.bDone = !it.b.Next()
		it.k = nextB
		return true
	}
	if it.bDone {
		it.aDone = !it.a.Next()
		it.k = nextA
		return true
	}
	if diff := bytes.Compare(nextA[:], nextB[:]); diff < 0 {
		it.aDone = !it.a.Next()
		it.k = nextA
		return true
	} else if diff == 0 {
		// Now we need to advance one of them
		it.aDone = !it.a.Next()
		goto first
	}
	it.bDone = !it.b.Next()
	it.k = nextB
	return true
}

func (it *binaryIterator) Key() common.Hash {
	return it.k
}
func (it *binaryIterator) Seek(key common.Hash) {
	panic("todo: implement")
}

func (dl *diffLayer) iterators() []Iterator {
	if parent, ok := dl.parent.(*diffLayer); ok {
		iterators := parent.iterators()
		return append(iterators, dl.newIterator())
	}
	return []Iterator{dl.newIterator()}
}

// fastIterator is a more optimized multi-layer iterator which maintains a
// direct mapping of all iterators leading down to the bottom layer
type fastIterator struct {
	iterators []Iterator
	initiated bool
}

// Len returns the number of active iterators
func (fi *fastIterator) Len() int {
	return len(fi.iterators)
}

// Less implements sort.Interface
func (fi *fastIterator) Less(i, j int) bool {
	a := fi.iterators[i].Key()
	b := fi.iterators[j].Key()
	return bytes.Compare(a[:], b[:]) < 0
}

// Swap implements sort.Interface
func (fi *fastIterator) Swap(i, j int) {
	fi.iterators[i], fi.iterators[j] = fi.iterators[j], fi.iterators[i]
}

// Next implements the Iterator interface. It returns false if no more elemnts
// can be retrieved (false == exhausted)
func (fi *fastIterator) Next() bool {
	if len(fi.iterators) == 0 {
		return false
	}
	if !fi.initiated {
		// Don't forward first time -- we had to 'Next' once in order to
		// do the sorting already
		fi.initiated = true
		return true
	}
	return fi.innerNext(0)
}

// innerNext handles the next operation internally,
// and should be invoked when we know that two elements in the list may have
// the same value.
// For example, if the list becomes [2,3,5,5,8,9,10], then we should invoke
// innerNext(3), which will call Next on elem 3 (the second '5'). It will continue
// along the list and apply the same operation if needed
func (fi *fastIterator) innerNext(pos int) bool {
	if !fi.iterators[pos].Next() {
		//Exhausted, remove this iterator
		fi.remove(pos)
		if len(fi.iterators) == 0 {
			return false
		}
		return true
	}
	if pos == len(fi.iterators)-1 {
		// Only one iterator left
		return true
	}
	// We next:ed the elem at 'pos'. Now we may have to re-sort that elem
	val, neighbour := fi.iterators[pos].Key(), fi.iterators[pos+1].Key()
	diff := bytes.Compare(val[:], neighbour[:])
	if diff < 0 {
		// It is still in correct place
		return true
	}
	if diff == 0 {
		// It has same value as the neighbour. So still in correct place, but
		// we need to iterate on the neighbour
		fi.innerNext(pos + 1)
		return true
	}
	// At this point, the elem is in the wrong location, but the
	// remaining list is sorted. Find out where to move the elem
	iterationNeeded := false
	index := sort.Search(len(fi.iterators), func(n int) bool {
		if n <= pos {
			// No need to search 'behind' us
			return false
		}
		if n == len(fi.iterators)-1 {
			// Can always place an elem last
			return true
		}
		neighbour := fi.iterators[n+1].Key()
		diff := bytes.Compare(val[:], neighbour[:])
		if diff == 0 {
			// The elem we're placing it next to has the same value,
			// so it's going to need further iteration
			iterationNeeded = true
		}
		return diff < 0
	})
	fi.move(pos, index)
	if iterationNeeded {
		fi.innerNext(index)
	}
	return true
}

// move moves an iterator to another position in the list
func (fi *fastIterator) move(index, newpos int) {
	if newpos > len(fi.iterators)-1 {
		newpos = len(fi.iterators) - 1
	}
	var (
		elem   = fi.iterators[index]
		middle = fi.iterators[index+1 : newpos+1]
		suffix []Iterator
	)
	if newpos < len(fi.iterators)-1 {
		suffix = fi.iterators[newpos+1:]
	}
	fi.iterators = append(fi.iterators[:index], middle...)
	fi.iterators = append(fi.iterators, elem)
	fi.iterators = append(fi.iterators, suffix...)
}

// remove drops an iterator from the list
func (fi *fastIterator) remove(index int) {
	fi.iterators = append(fi.iterators[:index], fi.iterators[index+1:]...)
}

// Key returns the current key
func (fi *fastIterator) Key() common.Hash {
	return fi.iterators[0].Key()
}

func (fi *fastIterator) Seek(key common.Hash) {
	// We need to apply this across all iterators
	var seen = make(map[common.Hash]struct{})

	length := len(fi.iterators)
	for i, it := range fi.iterators {
		it.Seek(key)
		for {
			if !it.Next() {
				// To be removed
				// swap it to the last position for now
				fi.iterators[i], fi.iterators[length-1] = fi.iterators[length-1], fi.iterators[i]
				length--
				break
			}
			v := it.Key()
			if _, exist := seen[v]; !exist {
				seen[v] = struct{}{}
				break
			}
		}
	}
	// Now remove those that were placed in the end
	fi.iterators = fi.iterators[:length]
	// The list is now totally unsorted, need to re-sort the entire list
	sort.Sort(fi)
	fi.initiated = false
}

// The fast iterator does not query parents as much.
func (dl *diffLayer) newFastIterator() Iterator {
	f := &fastIterator{dl.iterators(), false}
	f.Seek(common.Hash{})
	return f
}

// Debug is a convencience helper during testing
func (fi *fastIterator) Debug() {
	for _, it := range fi.iterators {
		fmt.Printf(" %v ", it.Key()[31])
	}
	fmt.Println()
}

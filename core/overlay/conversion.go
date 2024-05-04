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

package overlay

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-verkle"
	"github.com/holiman/uint256"
)

var zeroTreeIndex uint256.Int

// keyValueMigrator is a helper module that collects key-values from the overlay-tree migration for Verkle Trees.
// It assumes that the walk of the base tree is done in address-order, so it exploit that fact to
// collect the key-values in a way that is efficient.
type keyValueMigrator struct {
	// leafData contains the values for the future leaf for a particular VKT branch.
	leafData map[branchKey]*migratedKeyValue

	// When prepare() is called, it will start a background routine that will process the leafData
	// saving the result in newLeaves to be used by migrateCollectedKeyValues(). The background
	// routine signals that it is done by closing processingReady.
	processingReady chan struct{}
	newLeaves       []verkle.LeafNode
	prepareErr      error
}

func newKeyValueMigrator() *keyValueMigrator {
	// We do initialize the VKT config since prepare() might indirectly make multiple GetConfig() calls
	// in different goroutines when we never called GetConfig() before, causing a race considering the way
	// that `config` is designed in go-verkle.
	// TODO: jsign as a fix for this in the PR where we move to a file-less precomp, since it allows safe
	//       concurrent calls to GetConfig(). When that gets merged, we can remove this line.
	_ = verkle.GetConfig()
	return &keyValueMigrator{
		processingReady: make(chan struct{}),
		leafData:        make(map[branchKey]*migratedKeyValue, 10_000),
	}
}

type migratedKeyValue struct {
	branchKey    branchKey
	leafNodeData verkle.BatchNewLeafNodeData
}
type branchKey struct {
	addr      common.Address
	treeIndex uint256.Int
}

func newBranchKey(addr []byte, treeIndex *uint256.Int) branchKey {
	var sk branchKey
	copy(sk.addr[:], addr)
	sk.treeIndex = *treeIndex
	return sk
}

func (kvm *keyValueMigrator) addStorageSlot(addr []byte, slotNumber []byte, slotValue []byte) {
	treeIndex, subIndex := utils.GetTreeKeyStorageSlotTreeIndexes(slotNumber)
	leafNodeData := kvm.getOrInitLeafNodeData(newBranchKey(addr, treeIndex))
	leafNodeData.Values[subIndex] = slotValue
}

func (kvm *keyValueMigrator) addAccount(addr []byte, acc *types.StateAccount) {
	leafNodeData := kvm.getOrInitLeafNodeData(newBranchKey(addr, &zeroTreeIndex))

	var version [verkle.LeafValueSize]byte
	leafNodeData.Values[utils.VersionLeafKey] = version[:]

	var balance [verkle.LeafValueSize]byte
	for i, b := range acc.Balance.Bytes() {
		balance[len(acc.Balance.Bytes())-1-i] = b
	}
	leafNodeData.Values[utils.BalanceLeafKey] = balance[:]

	var nonce [verkle.LeafValueSize]byte
	binary.LittleEndian.PutUint64(nonce[:8], acc.Nonce)
	leafNodeData.Values[utils.NonceLeafKey] = nonce[:]

	leafNodeData.Values[utils.CodeHashLeafKey] = acc.CodeHash[:]
}

func (kvm *keyValueMigrator) addAccountCode(addr []byte, codeSize uint64, chunks []byte) {
	leafNodeData := kvm.getOrInitLeafNodeData(newBranchKey(addr, &zeroTreeIndex))

	// Save the code size.
	var codeSizeBytes [verkle.LeafValueSize]byte
	binary.LittleEndian.PutUint64(codeSizeBytes[:8], codeSize)
	leafNodeData.Values[utils.CodeSizeLeafKey] = codeSizeBytes[:]

	// The first 128 chunks are stored in the account header leaf.
	for i := 0; i < 128 && i < len(chunks)/32; i++ {
		leafNodeData.Values[byte(128+i)] = chunks[32*i : 32*(i+1)]
	}

	// Potential further chunks, have their own leaf nodes.
	for i := 128; i < len(chunks)/32; {
		treeIndex, _ := utils.GetTreeKeyCodeChunkIndices(uint256.NewInt(uint64(i)))
		leafNodeData := kvm.getOrInitLeafNodeData(newBranchKey(addr, treeIndex))

		j := i
		for ; (j-i) < 256 && j < len(chunks)/32; j++ {
			leafNodeData.Values[byte((j-128)%256)] = chunks[32*j : 32*(j+1)]
		}
		i = j
	}
}

func (kvm *keyValueMigrator) getOrInitLeafNodeData(bk branchKey) *verkle.BatchNewLeafNodeData {
	if ld, ok := kvm.leafData[bk]; ok {
		return &ld.leafNodeData
	}
	kvm.leafData[bk] = &migratedKeyValue{
		branchKey: bk,
		leafNodeData: verkle.BatchNewLeafNodeData{
			Stem:   nil, // It will be calculated in the prepare() phase, since it's CPU heavy.
			Values: make(map[byte][]byte, 256),
		},
	}
	return &kvm.leafData[bk].leafNodeData
}

func (kvm *keyValueMigrator) prepare() {
	// We fire a background routine to process the leafData and save the result in newLeaves.
	// The background routine signals that it is done by closing processingReady.
	go func() {
		// Step 1: We split kvm.leafData in numBatches batches, and we process each batch in a separate goroutine.
		//         This fills each leafNodeData.Stem with the correct value.
		leafData := make([]migratedKeyValue, 0, len(kvm.leafData))
		for _, v := range kvm.leafData {
			leafData = append(leafData, *v)
		}
		var wg sync.WaitGroup
		batchNum := runtime.NumCPU()
		batchSize := (len(kvm.leafData) + batchNum - 1) / batchNum
		for i := 0; i < len(kvm.leafData); i += batchSize {
			start := i
			end := i + batchSize
			if end > len(kvm.leafData) {
				end = len(kvm.leafData)
			}
			wg.Add(1)

			batch := leafData[start:end]
			go func() {
				defer wg.Done()
				var currAddr common.Address
				var currPoint *verkle.Point
				for i := range batch {
					if batch[i].branchKey.addr != currAddr || currAddr == (common.Address{}) {
						currAddr = batch[i].branchKey.addr
						currPoint = utils.EvaluateAddressPoint(currAddr[:])
					}
					stem := utils.GetTreeKeyWithEvaluatedAddess(currPoint, &batch[i].branchKey.treeIndex, 0)
					stem = stem[:verkle.StemSize]
					batch[i].leafNodeData.Stem = stem
				}
			}()
		}
		wg.Wait()

		// Step 2: Now that we have all stems (i.e: tree keys) calculated, we can create the new leaves.
		nodeValues := make([]verkle.BatchNewLeafNodeData, len(kvm.leafData))
		for i := range leafData {
			nodeValues[i] = leafData[i].leafNodeData
		}

		// Create all leaves in batch mode so we can optimize cryptography operations.
		kvm.newLeaves, kvm.prepareErr = verkle.BatchNewLeafNode(nodeValues)
		close(kvm.processingReady)
	}()
}

func (kvm *keyValueMigrator) migrateCollectedKeyValues(tree *trie.VerkleTrie) error {
	now := time.Now()
	<-kvm.processingReady
	if kvm.prepareErr != nil {
		return fmt.Errorf("failed to prepare key values: %w", kvm.prepareErr)
	}
	log.Info("Prepared key values from base tree", "duration", time.Since(now))

	// Insert into the tree.
	if err := tree.InsertMigratedLeaves(kvm.newLeaves); err != nil {
		return fmt.Errorf("failed to insert migrated leaves: %w", err)
	}

	return nil
}

// OverlayVerkleTransition contains the overlay conversion logic
func OverlayVerkleTransition(statedb *state.StateDB, root common.Hash, maxMovedCount uint64) error {
	migrdb := statedb.Database()
	migrdb.LockCurrentTransitionState()
	defer migrdb.UnLockCurrentTransitionState()

	// verkle transition: if the conversion process is in progress, move
	// N values from the MPT into the verkle tree.
	if migrdb.InTransition() {
		log.Debug("Processing verkle conversion starting", "account hash", migrdb.GetCurrentAccountHash(), "slot hash", migrdb.GetCurrentSlotHash(), "state root", root)
		var (
			now             = time.Now()
			tt              = statedb.GetTrie().(*trie.TransitionTrie)
			mpt             = tt.Base()
			vkt             = tt.Overlay()
			hasPreimagesBin = false
			preimageSeek    = migrdb.GetCurrentPreimageOffset()
			fpreimages      *bufio.Reader
		)

		// TODO: avoid opening the preimages file here and make it part of, potentially, statedb.Database().
		filePreimages, err := os.Open("preimages.bin")
		if err != nil {
			// fallback on reading the db
			log.Warn("opening preimage file", "error", err)
		} else {
			defer filePreimages.Close()
			if _, err := filePreimages.Seek(preimageSeek, io.SeekStart); err != nil {
				return fmt.Errorf("seeking preimage file: %s", err)
			}
			fpreimages = bufio.NewReader(filePreimages)
			hasPreimagesBin = true
		}

		accIt, err := statedb.Snaps().AccountIterator(mpt.Hash(), migrdb.GetCurrentAccountHash())
		if err != nil {
			return err
		}
		defer accIt.Release()
		accIt.Next()

		// If we're about to start with the migration process, we have to read the first account hash preimage.
		if migrdb.GetCurrentAccountAddress() == nil {
			var addr common.Address
			if hasPreimagesBin {
				if _, err := io.ReadFull(fpreimages, addr[:]); err != nil {
					return fmt.Errorf("reading preimage file: %s", err)
				}
			} else {
				addr = common.BytesToAddress(rawdb.ReadPreimage(migrdb.DiskDB(), accIt.Hash()))
				if len(addr) != 20 {
					return fmt.Errorf("addr len is zero is not 32: %d", len(addr))
				}
			}
			migrdb.SetCurrentAccountAddress(addr)
			if migrdb.GetCurrentAccountHash() != accIt.Hash() {
				return fmt.Errorf("preimage file does not match account hash: %s != %s", crypto.Keccak256Hash(addr[:]), accIt.Hash())
			}
			preimageSeek += int64(len(addr))
		}

		// mkv will be assiting in the collection of up to maxMovedCount key values to be migrated to the VKT.
		// It has internal caches to do efficient MPT->VKT key calculations, which will be discarded after
		// this function.
		mkv := newKeyValueMigrator()
		// move maxCount accounts into the verkle tree, starting with the
		// slots from the previous account.
		count := uint64(0)

		// if less than maxCount slots were moved, move to the next account
		for count < maxMovedCount {
			acc, err := types.FullAccount(accIt.Account())
			if err != nil {
				log.Error("Invalid account encountered during traversal", "error", err)
				return err
			}
			vkt.SetStorageRootConversion(*migrdb.GetCurrentAccountAddress(), acc.Root)

			// Start with processing the storage, because once the account is
			// converted, the `stateRoot` field loses its meaning. Which means
			// that it opens the door to a situation in which the storage isn't
			// converted, but it can not be found since the account was and so
			// there is no way to find the MPT storage from the information found
			// in the verkle account.
			// Note that this issue can still occur if the account gets written
			// to during normal block execution. A mitigation strategy has been
			// introduced with the `*StorageRootConversion` fields in VerkleDB.
			if acc.HasStorage() {
				stIt, err := statedb.Snaps().StorageIterator(mpt.Hash(), accIt.Hash(), migrdb.GetCurrentSlotHash())
				if err != nil {
					return err
				}
				processed := stIt.Next()
				if processed {
					log.Debug("account has storage and a next item")
				} else {
					log.Debug("account has storage and NO next item")
				}

				// fdb.StorageProcessed will be initialized to `true` if the
				// entire storage for an account was not entirely processed
				// by the previous block. This is used as a signal to resume
				// processing the storage for that account where we left off.
				// If the entire storage was processed, then the iterator was
				// created in vain, but it's ok as this will not happen often.
				for ; !migrdb.GetStorageProcessed() && count < maxMovedCount; count++ {
					log.Trace("Processing storage", "count", count, "slot", stIt.Slot(), "storage processed", migrdb.GetStorageProcessed(), "current account", migrdb.GetCurrentAccountAddress(), "current account hash", migrdb.GetCurrentAccountHash())
					var (
						value     []byte   // slot value after RLP decoding
						safeValue [32]byte // 32-byte aligned value
					)
					if err := rlp.DecodeBytes(stIt.Slot(), &value); err != nil {
						return fmt.Errorf("error decoding bytes %x: %w", stIt.Slot(), err)
					}
					copy(safeValue[32-len(value):], value)

					var slotnr []byte
					if hasPreimagesBin {
						var s [32]byte
						slotnr = s[:]
						if _, err := io.ReadFull(fpreimages, slotnr); err != nil {
							return fmt.Errorf("reading preimage file: %s", err)
						}
					} else {
						slotnr = rawdb.ReadPreimage(migrdb.DiskDB(), stIt.Hash())
						if len(slotnr) != 32 {
							return fmt.Errorf("slotnr len is zero is not 32: %d", len(slotnr))
						}
					}
					log.Trace("found slot number", "number", slotnr)
					if crypto.Keccak256Hash(slotnr[:]) != stIt.Hash() {
						return fmt.Errorf("preimage file does not match storage hash: %s!=%s", crypto.Keccak256Hash(slotnr), stIt.Hash())
					}
					preimageSeek += int64(len(slotnr))

					mkv.addStorageSlot(migrdb.GetCurrentAccountAddress().Bytes(), slotnr, safeValue[:])

					// advance the storage iterator
					migrdb.SetStorageProcessed(!stIt.Next())
					if !migrdb.GetStorageProcessed() {
						migrdb.SetCurrentSlotHash(stIt.Hash())
					}
				}
				stIt.Release()
			}

			// If the maximum number of leaves hasn't been reached, then
			// it means that the storage has finished processing (or none
			// was available for this account) and that the account itself
			// can be processed.
			if count < maxMovedCount {
				count++ // count increase for the account itself

				mkv.addAccount(migrdb.GetCurrentAccountAddress().Bytes(), acc)
				vkt.ClearStrorageRootConversion(*migrdb.GetCurrentAccountAddress())

				// Store the account code if present
				if !bytes.Equal(acc.CodeHash, types.EmptyCodeHash[:]) {
					code := rawdb.ReadCode(statedb.Database().DiskDB(), common.BytesToHash(acc.CodeHash))
					chunks := trie.ChunkifyCode(code)

					mkv.addAccountCode(migrdb.GetCurrentAccountAddress().Bytes(), uint64(len(code)), chunks)
				}

				// reset storage iterator marker for next account
				migrdb.SetStorageProcessed(false)
				migrdb.SetCurrentSlotHash(common.Hash{})

				// Move to the next account, if available - or end
				// the transition otherwise.
				if accIt.Next() {
					log.Trace("Found another account to convert", "hash", accIt.Hash())
					var addr common.Address
					if hasPreimagesBin {
						if _, err := io.ReadFull(fpreimages, addr[:]); err != nil {
							return fmt.Errorf("reading preimage file: %s", err)
						}
					} else {
						addr = common.BytesToAddress(rawdb.ReadPreimage(migrdb.DiskDB(), accIt.Hash()))
						if len(addr) != 20 {
							return fmt.Errorf("account address len is zero is not 20: %d", len(addr))
						}
					}
					if crypto.Keccak256Hash(addr[:]) != accIt.Hash() {
						return fmt.Errorf("preimage file does not match account hash: %s != %s", crypto.Keccak256Hash(addr[:]), accIt.Hash())
					}
					log.Trace("Converting account address", "hash", accIt.Hash(), "addr", addr)
					preimageSeek += int64(len(addr))
					migrdb.SetCurrentAccountAddress(addr)
				} else {
					// case when the account iterator has
					// reached the end but count < maxCount
					migrdb.EndVerkleTransition()
					break
				}
			}
		}
		migrdb.SetCurrentPreimageOffset(preimageSeek)

		log.Info("Collected key values from base tree", "count", count, "duration", time.Since(now), "last account hash", statedb.Database().GetCurrentAccountHash(), "last account address", statedb.Database().GetCurrentAccountAddress(), "storage processed", statedb.Database().GetStorageProcessed(), "last storage", statedb.Database().GetCurrentSlotHash())

		// Take all the collected key-values and prepare the new leaf values.
		// This fires a background routine that will start doing the work that
		// migrateCollectedKeyValues() will use to insert into the tree.
		//
		// TODO: Now both prepare() and migrateCollectedKeyValues() are next to each other, but
		//       after we fix an existing bug, we can call prepare() before the block execution and
		//       let it do the work in the background. After the block execution and finalization
		//       finish, we can call migrateCollectedKeyValues() which should already find everything ready.
		mkv.prepare()
		now = time.Now()
		if err := mkv.migrateCollectedKeyValues(tt.Overlay()); err != nil {
			return fmt.Errorf("could not migrate key values: %w", err)
		}
		log.Info("Inserted key values in overlay tree", "count", count, "duration", time.Since(now))
	}

	return nil
}

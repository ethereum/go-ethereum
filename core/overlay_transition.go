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

package core

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// OverlayVerkleTransition contains the overlay conversion logic
func OverlayVerkleTransition(statedb *state.StateDB) error {
	migrdb := statedb.Database()

	// verkle transition: if the conversion process is in progress, move
	// N values from the MPT into the verkle tree.
	if migrdb.InTransition() {
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

		const maxMovedCount = 10000
		// mkv will be assiting in the collection of up to maxMovedCount key values to be migrated to the VKT.
		// It has internal caches to do efficient MPT->VKT key calculations, which will be discarded after
		// this function.
		mkv := newKeyValueMigrator()
		// move maxCount accounts into the verkle tree, starting with the
		// slots from the previous account.
		count := 0

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
				stIt.Next()

				// fdb.StorageProcessed will be initialized to `true` if the
				// entire storage for an account was not entirely processed
				// by the previous block. This is used as a signal to resume
				// processing the storage for that account where we left off.
				// If the entire storage was processed, then the iterator was
				// created in vain, but it's ok as this will not happen often.
				for ; !migrdb.GetStorageProcessed() && count < maxMovedCount; count++ {
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
					// fmt.Printf("account switch: %s != %s\n", crypto.Keccak256Hash(addr[:]), accIt.Hash())
					if crypto.Keccak256Hash(addr[:]) != accIt.Hash() {
						return fmt.Errorf("preimage file does not match account hash: %s != %s", crypto.Keccak256Hash(addr[:]), accIt.Hash())
					}
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

		log.Info("Collected key values from base tree", "count", count, "duration", time.Since(now), "last account", statedb.Database().GetCurrentAccountHash())

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

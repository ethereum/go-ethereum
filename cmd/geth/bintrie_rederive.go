// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/holiman/uint256"
)

// The merkle re-derivation both commands hold themselves to: the converter
// demands the state root it scanned, the importer the root its anchor block
// commits. One implementation, because two would drift, and a drifting
// re-derivation is a check that passes on state nobody has.

// merkleAccountRecordLen is the width of an account record: nonce, balance
// and code hash, in that order.
const merkleAccountRecordLen = 8 + 32 + common.HashLength

// merkleAccountRecord encodes the account fields the re-derivation folds
// back into a trie. The storage root is deliberately absent: it is recomputed
// from the storage records, which is the whole point of the check.
func merkleAccountRecord(nonce uint64, balance *uint256.Int, codeHash common.Hash) []byte {
	record := make([]byte, 0, merkleAccountRecordLen)
	record = binary.BigEndian.AppendUint64(record, nonce)
	balance32 := balance.Bytes32()
	record = append(record, balance32[:]...)
	return append(record, codeHash.Bytes()...)
}

// heldStream wraps a sorted record stream with one record of lookahead, the
// shape a merge-join needs.
type heldStream struct {
	stream     *bintrie.RecordStream
	key, value []byte
	done       bool
}

// current returns the held record, loading the next one if none is held.
func (h *heldStream) current() ([]byte, []byte, error) {
	if h.done || h.key != nil {
		return h.key, h.value, nil
	}
	key, value, err := h.stream.Next()
	if err == io.EOF {
		h.done = true
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	h.key, h.value = key, value
	return key, value, nil
}

// advance drops the held record.
func (h *heldStream) advance() { h.key, h.value = nil, nil }

// rederiveMerkleRoot folds account and storage records back into a
// merkle-patricia trie and returns its root. Both streams must ascend -
// accounts by account hash, storage by account hash then slot hash - which is
// what the sorters guarantee.
//
// Storage under no account errors rather than being skipped: it means the
// record set is short an account the state holds, exactly the loss the root
// comparison exists to catch, and skipping it would hide it behind a root
// that happens to match.
func rederiveMerkleRoot(accounts, slots *bintrie.RecordStream, start time.Time) (common.Hash, error) {
	var (
		slotHeld    = &heldStream{stream: slots}
		accountTrie = trie.NewStackTrie(nil)
		storageTrie = trie.NewStackTrie(nil)
		rederived   uint64
	)
	for {
		akey, avalue, err := accounts.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return common.Hash{}, err
		}
		// The two producers encode this record independently; a short one
		// would otherwise slice out of range mid-fold.
		if len(avalue) != merkleAccountRecordLen {
			return common.Hash{}, fmt.Errorf("account record for %x is %d bytes, want %d", akey, len(avalue), merkleAccountRecordLen)
		}
		storageTrie.Reset()
		storageRoot := types.EmptyRootHash
		hasStorage := false
		for {
			skey, svalue, err := slotHeld.current()
			if err != nil {
				return common.Hash{}, err
			}
			if skey == nil {
				break
			}
			switch bytes.Compare(skey[:common.HashLength], akey) {
			case -1:
				return common.Hash{}, fmt.Errorf("storage under account hash %x, which holds no account", skey[:common.HashLength])
			case 1:
			default:
				if err := storageTrie.Update(skey[common.HashLength:], svalue); err != nil {
					return common.Hash{}, err
				}
				hasStorage = true
				slotHeld.advance()
				continue
			}
			break
		}
		if hasStorage {
			storageRoot = storageTrie.Hash()
		}
		full, err := rlp.EncodeToBytes(&types.StateAccount{
			Nonce:    binary.BigEndian.Uint64(avalue[:8]),
			Balance:  new(uint256.Int).SetBytes(avalue[8:40]),
			Root:     storageRoot,
			CodeHash: common.CopyBytes(avalue[40:merkleAccountRecordLen]),
		})
		if err != nil {
			return common.Hash{}, err
		}
		if err := accountTrie.Update(akey, full); err != nil {
			return common.Hash{}, err
		}
		rederived++
		if rederived%100_000 == 0 {
			log.Info("Re-deriving merkle state", "accounts", rederived,
				"elapsed", common.PrettyDuration(time.Since(start)))
		}
	}
	if skey, _, err := slotHeld.current(); err != nil {
		return common.Hash{}, err
	} else if skey != nil {
		return common.Hash{}, fmt.Errorf("storage under account hash %x, which holds no account", skey[:common.HashLength])
	}
	return accountTrie.Hash(), nil
}

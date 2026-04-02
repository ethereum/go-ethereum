// Copyright 2026 The go-ethereum Authors
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

package snap

import (
	"bytes"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/tracker"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb/database"
)

func handleGetAccountRange(backend Backend, msg Decoder, peer *Peer) error {
	var req GetAccountRangePacket
	if err := msg.Decode(&req); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	// Service the request, potentially returning nothing in case of errors
	accounts, proofs := ServiceGetAccountRangeQuery(backend.Chain(), &req)

	// Send back anything accumulated (or empty in case of errors)
	return p2p.Send(peer.rw, AccountRangeMsg, &AccountRangePacket{
		ID:       req.ID,
		Accounts: accounts,
		Proof:    proofs,
	})
}

// ServiceGetAccountRangeQuery assembles the response to an account range query.
// It is exposed to allow external packages to test protocol behavior.
func ServiceGetAccountRangeQuery(chain *core.BlockChain, req *GetAccountRangePacket) ([]*AccountData, [][]byte) {
	if req.Bytes > softResponseLimit {
		req.Bytes = softResponseLimit
	}
	// Retrieve the requested state and bail out if non existent
	tr, err := trie.New(trie.StateTrieID(req.Root), chain.TrieDB())
	if err != nil {
		return nil, nil
	}
	// Temporary solution: using the snapshot interface for both cases.
	// This can be removed once the hash scheme is deprecated.
	var it snapshot.AccountIterator
	if chain.TrieDB().Scheme() == rawdb.HashScheme {
		// The snapshot is assumed to be available in hash mode if
		// the SNAP protocol is enabled.
		it, err = chain.Snapshots().AccountIterator(req.Root, req.Origin)
	} else {
		it, err = chain.TrieDB().AccountIterator(req.Root, req.Origin)
	}
	if err != nil {
		return nil, nil
	}
	// Iterate over the requested range and pile accounts up
	var (
		accounts []*AccountData
		size     uint64
		last     common.Hash
	)
	for it.Next() {
		hash, account := it.Hash(), common.CopyBytes(it.Account())

		// Track the returned interval for the Merkle proofs
		last = hash

		// Assemble the reply item
		size += uint64(common.HashLength + len(account))
		accounts = append(accounts, &AccountData{
			Hash: hash,
			Body: account,
		})
		// If we've exceeded the request threshold, abort
		if bytes.Compare(hash[:], req.Limit[:]) >= 0 {
			break
		}
		if size > req.Bytes {
			break
		}
	}
	it.Release()

	// Generate the Merkle proofs for the first and last account
	proof := trienode.NewProofSet()
	if err := tr.Prove(req.Origin[:], proof); err != nil {
		log.Warn("Failed to prove account range", "origin", req.Origin, "err", err)
		return nil, nil
	}
	if last != (common.Hash{}) {
		if err := tr.Prove(last[:], proof); err != nil {
			log.Warn("Failed to prove account range", "last", last, "err", err)
			return nil, nil
		}
	}
	return accounts, proof.List()
}

func handleAccountRange(backend Backend, msg Decoder, peer *Peer) error {
	res := new(accountRangeInput)
	if err := msg.Decode(res); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}

	// Check response validity.
	if len := res.Proof.Len(); len > 128 {
		return fmt.Errorf("AccountRange: invalid proof (length %d)", len)
	}
	tresp := tracker.Response{ID: res.ID, MsgCode: AccountRangeMsg, Size: len(res.Accounts.Content())}
	if err := peer.tracker.Fulfil(tresp); err != nil {
		return err
	}

	// Decode.
	accounts, err := res.Accounts.Items()
	if err != nil {
		return fmt.Errorf("AccountRange: invalid accounts list: %v", err)
	}
	proof, err := res.Proof.Items()
	if err != nil {
		return fmt.Errorf("AccountRange: invalid proof: %v", err)
	}

	// Ensure the range is monotonically increasing
	for i := 1; i < len(accounts); i++ {
		if bytes.Compare(accounts[i-1].Hash[:], accounts[i].Hash[:]) >= 0 {
			return fmt.Errorf("accounts not monotonically increasing: #%d [%x] vs #%d [%x]", i-1, accounts[i-1].Hash[:], i, accounts[i].Hash[:])
		}
	}

	return backend.Handle(peer, &AccountRangePacket{res.ID, accounts, proof})
}

func handleGetStorageRanges(backend Backend, msg Decoder, peer *Peer) error {
	var req GetStorageRangesPacket
	if err := msg.Decode(&req); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	// Service the request, potentially returning nothing in case of errors
	slots, proofs := ServiceGetStorageRangesQuery(backend.Chain(), &req)

	// Send back anything accumulated (or empty in case of errors)
	return p2p.Send(peer.rw, StorageRangesMsg, &StorageRangesPacket{
		ID:    req.ID,
		Slots: slots,
		Proof: proofs,
	})
}

func ServiceGetStorageRangesQuery(chain *core.BlockChain, req *GetStorageRangesPacket) ([][]*StorageData, [][]byte) {
	if req.Bytes > softResponseLimit {
		req.Bytes = softResponseLimit
	}
	// TODO(karalabe): Do we want to enforce > 0 accounts and 1 account if origin is set?
	// TODO(karalabe):   - Logging locally is not ideal as remote faults annoy the local user
	// TODO(karalabe):   - Dropping the remote peer is less flexible wrt client bugs (slow is better than non-functional)

	// Calculate the hard limit at which to abort, even if mid storage trie
	hardLimit := uint64(float64(req.Bytes) * (1 + stateLookupSlack))

	// Retrieve storage ranges until the packet limit is reached
	var (
		slots  [][]*StorageData
		proofs [][]byte
		size   uint64
	)
	for _, account := range req.Accounts {
		// If we've exceeded the requested data limit, abort without opening
		// a new storage range (that we'd need to prove due to exceeded size)
		if size >= req.Bytes {
			break
		}
		// The first account might start from a different origin and end sooner
		var origin common.Hash
		if len(req.Origin) > 0 {
			origin, req.Origin = common.BytesToHash(req.Origin), nil
		}
		var limit = common.MaxHash
		if len(req.Limit) > 0 {
			limit, req.Limit = common.BytesToHash(req.Limit), nil
		}
		// Retrieve the requested state and bail out if non existent
		var (
			err error
			it  snapshot.StorageIterator
		)
		// Temporary solution: using the snapshot interface for both cases.
		// This can be removed once the hash scheme is deprecated.
		if chain.TrieDB().Scheme() == rawdb.HashScheme {
			// The snapshot is assumed to be available in hash mode if
			// the SNAP protocol is enabled.
			it, err = chain.Snapshots().StorageIterator(req.Root, account, origin)
		} else {
			it, err = chain.TrieDB().StorageIterator(req.Root, account, origin)
		}
		if err != nil {
			return nil, nil
		}
		// Iterate over the requested range and pile slots up
		var (
			storage []*StorageData
			last    common.Hash
			abort   bool
		)
		for it.Next() {
			if size >= hardLimit {
				abort = true
				break
			}
			hash, slot := it.Hash(), common.CopyBytes(it.Slot())

			// Track the returned interval for the Merkle proofs
			last = hash

			// Assemble the reply item
			size += uint64(common.HashLength + len(slot))
			storage = append(storage, &StorageData{
				Hash: hash,
				Body: slot,
			})
			// If we've exceeded the request threshold, abort
			if bytes.Compare(hash[:], limit[:]) >= 0 {
				break
			}
		}
		if len(storage) > 0 {
			slots = append(slots, storage)
		}
		it.Release()

		// Generate the Merkle proofs for the first and last storage slot, but
		// only if the response was capped. If the entire storage trie included
		// in the response, no need for any proofs.
		if origin != (common.Hash{}) || (abort && len(storage) > 0) {
			// Request started at a non-zero hash or was capped prematurely, add
			// the endpoint Merkle proofs
			accTrie, err := trie.NewStateTrie(trie.StateTrieID(req.Root), chain.TrieDB())
			if err != nil {
				return nil, nil
			}
			acc, err := accTrie.GetAccountByHash(account)
			if err != nil || acc == nil {
				return nil, nil
			}
			id := trie.StorageTrieID(req.Root, account, acc.Root)
			stTrie, err := trie.NewStateTrie(id, chain.TrieDB())
			if err != nil {
				return nil, nil
			}
			proof := trienode.NewProofSet()
			if err := stTrie.Prove(origin[:], proof); err != nil {
				log.Warn("Failed to prove storage range", "origin", req.Origin, "err", err)
				return nil, nil
			}
			if last != (common.Hash{}) {
				if err := stTrie.Prove(last[:], proof); err != nil {
					log.Warn("Failed to prove storage range", "last", last, "err", err)
					return nil, nil
				}
			}
			proofs = append(proofs, proof.List()...)
			// Proof terminates the reply as proofs are only added if a node
			// refuses to serve more data (exception when a contract fetch is
			// finishing, but that's that).
			break
		}
	}
	return slots, proofs
}

func handleStorageRanges(backend Backend, msg Decoder, peer *Peer) error {
	res := new(storageRangesInput)
	if err := msg.Decode(res); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}

	// Check response validity.
	if len := res.Proof.Len(); len > 128 {
		return fmt.Errorf("StorageRangesMsg: invalid proof (length %d)", len)
	}
	tresp := tracker.Response{ID: res.ID, MsgCode: StorageRangesMsg, Size: len(res.Slots.Content())}
	if err := peer.tracker.Fulfil(tresp); err != nil {
		return fmt.Errorf("StorageRangesMsg: %w", err)
	}

	// Decode.
	slotLists, err := res.Slots.Items()
	if err != nil {
		return fmt.Errorf("AccountRange: invalid accounts list: %v", err)
	}
	proof, err := res.Proof.Items()
	if err != nil {
		return fmt.Errorf("AccountRange: invalid proof: %v", err)
	}

	// Ensure the ranges are monotonically increasing
	for i, slots := range slotLists {
		for j := 1; j < len(slots); j++ {
			if bytes.Compare(slots[j-1].Hash[:], slots[j].Hash[:]) >= 0 {
				return fmt.Errorf("storage slots not monotonically increasing for account #%d: #%d [%x] vs #%d [%x]", i, j-1, slots[j-1].Hash[:], j, slots[j].Hash[:])
			}
		}
	}

	return backend.Handle(peer, &StorageRangesPacket{res.ID, slotLists, proof})
}

func handleGetByteCodes(backend Backend, msg Decoder, peer *Peer) error {
	var req GetByteCodesPacket
	if err := msg.Decode(&req); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	// Service the request, potentially returning nothing in case of errors
	codes := ServiceGetByteCodesQuery(backend.Chain(), &req)

	// Send back anything accumulated (or empty in case of errors)
	return p2p.Send(peer.rw, ByteCodesMsg, &ByteCodesPacket{
		ID:    req.ID,
		Codes: codes,
	})
}

// ServiceGetByteCodesQuery assembles the response to a byte codes query.
// It is exposed to allow external packages to test protocol behavior.
func ServiceGetByteCodesQuery(chain *core.BlockChain, req *GetByteCodesPacket) [][]byte {
	if req.Bytes > softResponseLimit {
		req.Bytes = softResponseLimit
	}
	if len(req.Hashes) > maxCodeLookups {
		req.Hashes = req.Hashes[:maxCodeLookups]
	}
	// Retrieve bytecodes until the packet size limit is reached
	var (
		codes [][]byte
		bytes uint64
	)
	for _, hash := range req.Hashes {
		if hash == types.EmptyCodeHash {
			// Peers should not request the empty code, but if they do, at
			// least sent them back a correct response without db lookups
			codes = append(codes, []byte{})
		} else if blob := chain.ContractCodeWithPrefix(hash); len(blob) > 0 {
			codes = append(codes, blob)
			bytes += uint64(len(blob))
		}
		if bytes > req.Bytes {
			break
		}
	}
	return codes
}

func handleByteCodes(backend Backend, msg Decoder, peer *Peer) error {
	res := new(byteCodesInput)
	if err := msg.Decode(res); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}

	length := res.Codes.Len()
	tresp := tracker.Response{ID: res.ID, MsgCode: ByteCodesMsg, Size: length}
	if err := peer.tracker.Fulfil(tresp); err != nil {
		return fmt.Errorf("ByteCodes: %w", err)
	}

	codes, err := res.Codes.Items()
	if err != nil {
		return fmt.Errorf("ByteCodes: %w", err)
	}

	return backend.Handle(peer, &ByteCodesPacket{res.ID, codes})
}

func handleGetTrienodes(backend Backend, msg Decoder, peer *Peer) error {
	var req GetTrieNodesPacket
	if err := msg.Decode(&req); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	// Service the request, potentially returning nothing in case of errors
	nodes, err := ServiceGetTrieNodesQuery(backend.Chain(), &req)
	if err != nil {
		return err
	}
	// Send back anything accumulated (or empty in case of errors)
	return p2p.Send(peer.rw, TrieNodesMsg, &TrieNodesPacket{
		ID:    req.ID,
		Nodes: nodes,
	})
}

func nextBytes(it *rlp.Iterator) []byte {
	if !it.Next() {
		return nil
	}
	content, _, err := rlp.SplitString(it.Value())
	if err != nil {
		return nil
	}
	return content
}

// ServiceGetTrieNodesQuery assembles the response to a trie nodes query.
// It is exposed to allow external packages to test protocol behavior.
func ServiceGetTrieNodesQuery(chain *core.BlockChain, req *GetTrieNodesPacket) ([][]byte, error) {
	start := time.Now()
	if req.Bytes > softResponseLimit {
		req.Bytes = softResponseLimit
	}
	// Make sure we have the state associated with the request
	triedb := chain.TrieDB()

	accTrie, err := trie.NewStateTrie(trie.StateTrieID(req.Root), triedb)
	if err != nil {
		// We don't have the requested state available, bail out
		return nil, nil
	}
	// The 'reader' might be nil, in which case we cannot serve storage slots
	// via snapshot.
	var reader database.StateReader
	if chain.Snapshots() != nil {
		reader = chain.Snapshots().Snapshot(req.Root)
	}
	if reader == nil {
		reader, _ = triedb.StateReader(req.Root)
	}

	// Retrieve trie nodes until the packet size limit is reached
	var (
		outerIt = req.Paths.ContentIterator()
		nodes   [][]byte
		bytes   uint64
		loads   int // Trie hash expansions to count database reads
	)
	for outerIt.Next() {
		innerIt, err := rlp.NewListIterator(outerIt.Value())
		if err != nil {
			return nodes, err
		}

		switch innerIt.Count() {
		case 0:
			// Ensure we penalize invalid requests
			return nil, fmt.Errorf("%w: zero-item pathset requested", errBadRequest)

		case 1:
			// If we're only retrieving an account trie node, fetch it directly
			accKey := nextBytes(&innerIt)
			if accKey == nil {
				return nodes, fmt.Errorf("%w: invalid account node request", errBadRequest)
			}
			blob, resolved, err := accTrie.GetNode(accKey)
			loads += resolved // always account database reads, even for failures
			if err != nil {
				break
			}
			nodes = append(nodes, blob)
			bytes += uint64(len(blob))

		default:
			// Storage slots requested, open the storage trie and retrieve from there
			accKey := nextBytes(&innerIt)
			if accKey == nil {
				return nodes, fmt.Errorf("%w: invalid account storage request", errBadRequest)
			}
			var stRoot common.Hash
			if reader == nil {
				// We don't have the requested state snapshotted yet (or it is stale),
				// but can look up the account via the trie instead.
				account, err := accTrie.GetAccountByHash(common.BytesToHash(accKey))
				loads += 8 // We don't know the exact cost of lookup, this is an estimate
				if err != nil || account == nil {
					break
				}
				stRoot = account.Root
			} else {
				account, err := reader.Account(common.BytesToHash(accKey))
				loads++ // always account database reads, even for failures
				if err != nil || account == nil {
					break
				}
				stRoot = common.BytesToHash(account.Root)
			}

			id := trie.StorageTrieID(req.Root, common.BytesToHash(accKey), stRoot)
			stTrie, err := trie.NewStateTrie(id, triedb)
			loads++ // always account database reads, even for failures
			if err != nil {
				break
			}
			for innerIt.Next() {
				path, _, err := rlp.SplitString(innerIt.Value())
				if err != nil {
					return nil, fmt.Errorf("%w: invalid storage key: %v", errBadRequest, err)
				}
				blob, resolved, err := stTrie.GetNode(path)
				loads += resolved // always account database reads, even for failures
				if err != nil {
					break
				}
				nodes = append(nodes, blob)
				bytes += uint64(len(blob))

				// Sanity check limits to avoid DoS on the store trie loads
				if bytes > req.Bytes || loads > maxTrieNodeLookups || time.Since(start) > maxTrieNodeTimeSpent {
					break
				}
			}
		}
		// Abort request processing if we've exceeded our limits
		if bytes > req.Bytes || loads > maxTrieNodeLookups || time.Since(start) > maxTrieNodeTimeSpent {
			break
		}
	}
	return nodes, nil
}

func handleTrieNodes(backend Backend, msg Decoder, peer *Peer) error {
	res := new(trieNodesInput)
	if err := msg.Decode(res); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}

	tresp := tracker.Response{ID: res.ID, MsgCode: TrieNodesMsg, Size: res.Nodes.Len()}
	if err := peer.tracker.Fulfil(tresp); err != nil {
		return fmt.Errorf("TrieNodes: %w", err)
	}
	nodes, err := res.Nodes.Items()
	if err != nil {
		return fmt.Errorf("TrieNodes: %w", err)
	}

	return backend.Handle(peer, &TrieNodesPacket{res.ID, nodes})
}

// nolint:unused
func handleGetAccessLists(backend Backend, msg Decoder, peer *Peer) error {
	var req GetAccessListsPacket
	if err := msg.Decode(&req); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	bals := ServiceGetAccessListsQuery(backend.Chain(), &req)
	return p2p.Send(peer.rw, AccessListsMsg, &AccessListsPacket{
		ID:          req.ID,
		AccessLists: bals,
	})
}

// ServiceGetAccessListsQuery assembles the response to an access list query.
// It is exposed to allow external packages to test protocol behavior.
func ServiceGetAccessListsQuery(chain *core.BlockChain, req *GetAccessListsPacket) []rlp.RawValue {
	// Cap the number of lookups
	if len(req.Hashes) > maxAccessListLookups {
		req.Hashes = req.Hashes[:maxAccessListLookups]
	}
	var (
		bals  []rlp.RawValue
		bytes uint64
	)
	for _, hash := range req.Hashes {
		if bal := chain.GetAccessListRLP(hash); len(bal) > 0 {
			bals = append(bals, bal)
			bytes += uint64(len(bal))
		} else {
			// Either the block is unknown or the BAL doesn't exist
			bals = append(bals, nil)
		}
		if bytes > softResponseLimit {
			break
		}
	}
	return bals
}

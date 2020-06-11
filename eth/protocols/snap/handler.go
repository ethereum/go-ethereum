// Copyright 2020 The go-ethereum Authors
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

package snap

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	// softResponseLimit is the target maximum size of replies to data retrievals.
	softResponseLimit = 2 * 1024 * 1024

	// maxCodeLookups is the maximum number of bytecodes to serve. This number is
	// there to limit the number of disk lookups.
	maxCodeLookups = 1024

	// stateLookupSlack defines the ratio by how much a state response can exceed
	// the requested limit in order to try and avoid breaking up contracts into
	// multiple packages and proving them.
	stateLookupSlack = 0.1

	// maxTrieNodeLookups is the maximum number of state trie nodes to serve. This
	// number is there to limit the number of disk lookups.
	maxTrieNodeLookups = 1024
)

// Handler is a callback to invoke from an outside runner after the boilerplate
// exchanges have passed.
type Handler func(peer *Peer) error

// Backend defines the data retrieval methods to serve remote requests and the
// calback methods to invoke on remote deliveries.
type Backend interface {
	// Chain retrieves the blockchain object to serve data.
	Chain() *core.BlockChain

	// RunPeer is invoked when a peer joins on the `eth` protocol. The handler
	// should do any peer maintenance work, handshakes and validations. If all
	// is passed, control should be given back to the `handler` to process the
	// inbound messages going forward.
	RunPeer(peer *Peer, handler Handler) error

	// PeerInfo retrieves all known `eth` information about a peer.
	PeerInfo(id enode.ID) interface{}

	// OnAccounts is a callback method to invoke when a range of accounts are
	// received from a remote peer.
	OnAccounts(peer *Peer, id uint64, hashes []common.Hash, accounts [][]byte, proof [][]byte) error

	// OnStorage is a callback method to invoke when ranges of storage slots
	// are received from a remote peer. The data might contain multiple storage
	// tries, but proofs can only be attached for the last one.
	OnStorage(peer *Peer, id uint64, hashes [][]common.Hash, slots [][][]byte, proof [][]byte) error

	// OnByteCodes is a callback method to invoke when a batch of contract
	// bytes codes are received from a remote peer.
	OnByteCodes(peer *Peer, id uint64, bytecodes [][]byte) error

	// OnTrieNodes is a callback method to invoke when a batch of trie nodes
	// are received from a remote peer.
	OnTrieNodes(peer *Peer, id uint64, nodes [][]byte) error
}

// MakeProtocols constructs the P2P protocol definitions for `snap`.
func MakeProtocols(backend Backend, dnsdisc enode.Iterator) []p2p.Protocol {
	protocols := make([]p2p.Protocol, len(protocolVersions))
	for i, version := range protocolVersions {
		version := version // Closure

		protocols[i] = p2p.Protocol{
			Name:    protocolName,
			Version: version,
			Length:  protocolLengths[version],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				return backend.RunPeer(newPeer(version, p, rw), func(peer *Peer) error {
					return handle(backend, peer)
				})
			},
			NodeInfo: func() interface{} {
				return nodeInfo(backend.Chain())
			},
			PeerInfo: func(id enode.ID) interface{} {
				return backend.PeerInfo(id)
			},
			Attributes:     []enr.Entry{&enrEntry{}},
			DialCandidates: dnsdisc,
		}
	}
	return protocols
}

// handle is the callback invoked to manage the life cycle of a `snap` peer.
// When this function terminates, the peer is disconnected.
func handle(backend Backend, peer *Peer) error {
	for {
		if err := handleMessage(backend, peer); err != nil {
			peer.Log().Debug("Message handling failed in `snap`", "err", err)
			return err
		}
	}
}

// handleMessage is invoked whenever an inbound message is received from a
// remote peer on the `spap` protocol. The remote connection is torn down upon
// returning any error.
func handleMessage(backend Backend, peer *Peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := peer.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > maxMessageSize {
		return fmt.Errorf("%w: %v > %v", errMsgTooLarge, msg.Size, maxMessageSize)
	}
	defer msg.Discard()

	// Handle the message depending on its contents
	switch {
	case msg.Code == getAccountRangeMsg:
		// Decode the account retrieval request
		var req getAccountRangeData
		if err := msg.Decode(&req); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		if req.Bytes > softResponseLimit {
			req.Bytes = softResponseLimit
		}
		// Retrieve the requested state and bail out if non existent
		tr, err := trie.New(req.Root, backend.Chain().StateCache().TrieDB())
		if err != nil {
			return p2p.Send(peer.rw, accountRangeMsg, &accountRangeData{ID: req.ID})
		}
		it, err := backend.Chain().Snapshot().AccountIterator(req.Root, req.Origin)
		if err != nil {
			return p2p.Send(peer.rw, accountRangeMsg, &accountRangeData{ID: req.ID})
		}
		defer it.Release()

		// Iterate over the requested range and pile accounts up
		var (
			accounts []*accountData
			bytes    uint64
			last     common.Hash
		)
		for it.Next() && bytes < req.Bytes {
			hash, account := it.Hash(), common.CopyBytes(it.Account())

			// Track the returned interval for the Merkle proofs
			last = hash

			// Assemble the reply item
			bytes += uint64(common.HashLength + len(account))
			accounts = append(accounts, &accountData{
				Hash: hash,
				Body: account,
			})
		}
		// Generate the Merkle proofs for the first and last account
		proof := light.NewNodeSet()
		if err := tr.Prove(req.Origin[:], 0, proof); err != nil {
			log.Warn("Failed to prove account range", "origin", req.Origin, "err", err)
			return p2p.Send(peer.rw, accountRangeMsg, &accountRangeData{ID: req.ID})
		}
		if last != (common.Hash{}) {
			if err := tr.Prove(last[:], 0, proof); err != nil {
				log.Warn("Failed to prove account range", "last", last, "err", err)
				return p2p.Send(peer.rw, accountRangeMsg, &accountRangeData{ID: req.ID})
			}
		}
		var proofs [][]byte
		for _, blob := range proof.NodeList() {
			proofs = append(proofs, blob)
		}
		// Send back anything accumulated
		return p2p.Send(peer.rw, accountRangeMsg, &accountRangeData{
			ID:       req.ID,
			Accounts: accounts,
			Proof:    proofs,
		})

	case msg.Code == accountRangeMsg:
		// A range of accounts arrived to one of our previous requests
		var res accountRangeData
		if err := msg.Decode(&res); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		// Transform the snap slim format into the consensus representation
		var (
			keys = make([]common.Hash, len(res.Accounts))
			vals = make([][]byte, len(res.Accounts))
		)
		for i, acc := range res.Accounts {
			val, err := snapshot.FullAccountRLP(acc.Body)
			if err != nil {
				return fmt.Errorf("invalid account %x: %v", acc.Body, err)
			}
			keys[i] = acc.Hash
			vals[i] = val
		}
		return backend.OnAccounts(peer, res.ID, keys, vals, res.Proof)

	case msg.Code == getStorageRangesMsg:
		// Decode the storage retrieval request
		var req getStorageRangesData
		if err := msg.Decode(&req); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		if req.Bytes > softResponseLimit {
			req.Bytes = softResponseLimit
		}
		// Calculate the hard limit at which to abort, even if mid storage trie
		hardLimit := uint64(float64(req.Bytes) * (1 + stateLookupSlack))

		// Retrieve storage ranges until the packet limit is reached
		var (
			slots  [][]*storageData
			proofs [][]byte
			bytes  uint64
		)
		for _, account := range req.Accounts {
			// If we've exceeded the requested data limit, abort without opening
			// a new storage range (that we'd need to prove due to exceeded size)
			if bytes >= req.Bytes {
				break
			}
			// The first account might start from a different origin
			var origin common.Hash
			if len(req.Origin) > 0 {
				origin, req.Origin = common.BytesToHash(req.Origin), nil
			}
			// Retrieve the requested state and bail out if non existent
			it, err := backend.Chain().Snapshot().StorageIterator(req.Root, account, origin)
			if err != nil {
				return p2p.Send(peer.rw, storageRangesMsg, &storageRangesData{ID: req.ID})
			}
			// Iterate over the requested range and pile slots up
			var (
				storage []*storageData
				last    common.Hash
			)
			for it.Next() && bytes < hardLimit {
				hash, slot := it.Hash(), common.CopyBytes(it.Slot())

				// Track the returned interval for the Merkle proofs
				last = hash

				// Assemble the reply item
				bytes += uint64(common.HashLength + len(slot))
				storage = append(storage, &storageData{
					Hash: hash,
					Body: slot,
				})
			}
			slots = append(slots, storage)
			it.Release()

			// Generate the Merkle proofs for the first and last storage slot, but
			// only if the response was capped. If the entire storage trie included
			// in the response, no need for any proofs.
			if origin != (common.Hash{}) || bytes >= hardLimit {
				// Request started at a non-zero hash or was capped prematurely, add
				// the endpoint Merkle proofs
				accTrie, err := trie.New(req.Root, backend.Chain().StateCache().TrieDB())
				if err != nil {
					return p2p.Send(peer.rw, storageRangesMsg, &storageRangesData{ID: req.ID})
				}
				var acc state.Account
				if err := rlp.DecodeBytes(accTrie.Get(account[:]), &acc); err != nil {
					return p2p.Send(peer.rw, storageRangesMsg, &storageRangesData{ID: req.ID})
				}
				stTrie, err := trie.New(acc.Root, backend.Chain().StateCache().TrieDB())
				if err != nil {
					return p2p.Send(peer.rw, storageRangesMsg, &storageRangesData{ID: req.ID})
				}
				proof := light.NewNodeSet()
				if err := stTrie.Prove(origin[:], 0, proof); err != nil {
					log.Warn("Failed to prove storage range", "origin", req.Origin, "err", err)
					return p2p.Send(peer.rw, storageRangesMsg, &storageRangesData{ID: req.ID})
				}
				if last != (common.Hash{}) {
					if err := stTrie.Prove(last[:], 0, proof); err != nil {
						log.Warn("Failed to prove storage range", "last", last, "err", err)
						return p2p.Send(peer.rw, storageRangesMsg, &storageRangesData{ID: req.ID})
					}
				}
				for _, blob := range proof.NodeList() {
					proofs = append(proofs, blob)
				}
				// Proof terminates the reply as proofs are only added if a node
				// refuses to serve more data (exception when a contract fetch is
				// finishing, but that's that).
				break
			}
		}
		// Send back anything accumulated
		return p2p.Send(peer.rw, storageRangesMsg, &storageRangesData{
			ID:    req.ID,
			Slots: slots,
			Proof: proofs,
		})

	case msg.Code == storageRangesMsg:
		// A range of accounts arrived to one of our previous requests
		var res storageRangesData
		if err := msg.Decode(&res); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		var (
			keys = make([][]common.Hash, len(res.Slots))
			vals = make([][][]byte, len(res.Slots))
		)
		for i, slots := range res.Slots {
			keys[i] = make([]common.Hash, len(slots))
			vals[i] = make([][]byte, len(slots))
			for j, slot := range slots {
				keys[i][j] = slot.Hash
				vals[i][j] = slot.Body
			}
		}
		return backend.OnStorage(peer, res.ID, keys, vals, res.Proof)

	case msg.Code == getByteCodesMsg:
		// Decode bytecode retrieval request
		var req getByteCodesData
		if err := msg.Decode(&req); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
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
			if blob, err := backend.Chain().ByteCode(hash); err == nil {
				codes = append(codes, blob)
				bytes += uint64(len(blob))
			}
			if bytes > req.Bytes {
				break
			}
		}
		// Send back anything accumulated
		return p2p.Send(peer.rw, byteCodesMsg, &byteCodesData{
			ID:    req.ID,
			Codes: codes,
		})

	case msg.Code == byteCodesMsg:
		// A batch of byte codes arrived to one of our previous requests
		var res byteCodesData
		if err := msg.Decode(&res); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		return backend.OnByteCodes(peer, res.ID, res.Codes)

	case msg.Code == getTrieNodesMsg:
		// Decode trie node retrieval request
		var req getTrieNodesData
		if err := msg.Decode(&req); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		if req.Bytes > softResponseLimit {
			req.Bytes = softResponseLimit
		}
		if len(req.Hashes) > maxTrieNodeLookups {
			req.Hashes = req.Hashes[:maxTrieNodeLookups]
		}
		// Retrieve bytecodes until the packet size limit is reached
		var (
			nodes [][]byte
			bytes uint64
		)
		for _, hash := range req.Hashes {
			if blob, err := backend.Chain().TrieNode(hash); err == nil {
				nodes = append(nodes, blob)
				bytes += uint64(len(blob))
			}
			if bytes > req.Bytes {
				break
			}
		}
		// Send back anything accumulated
		return p2p.Send(peer.rw, trieNodesMsg, &trieNodesData{
			ID:    req.ID,
			Nodes: nodes,
		})

	case msg.Code == trieNodesMsg:
		// A batch of trie nodes arrived to one of our previous requests
		var res trieNodesData
		if err := msg.Decode(&res); err != nil {
			return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
		}
		return backend.OnTrieNodes(peer, res.ID, res.Nodes)

	default:
		return fmt.Errorf("%w: %v", errInvalidMsgCode, msg.Code)
	}
}

// NodeInfo represents a short summary of the `snap` sub-protocol metadata
// known about the host peer.
type NodeInfo struct{}

// nodeInfo retrieves some `snap` protocol metadata about the running host node.
func nodeInfo(chain *core.BlockChain) *NodeInfo {
	return &NodeInfo{}
}

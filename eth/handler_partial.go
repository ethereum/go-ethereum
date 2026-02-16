// Copyright 2025 The go-ethereum Authors
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

package eth

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/protocols/snap"
	"github.com/ethereum/go-ethereum/log"
)

const (
	// storageRootQueryTimeout is the time to wait for a single snap account query response.
	storageRootQueryTimeout = 5 * time.Second

	// storageRootMaxRetries is the maximum number of peers to try per unresolved address.
	storageRootMaxRetries = 6

	// storageRootQueryBytes is the soft response size limit for account range queries.
	// We request a single account, so this is generous.
	storageRootQueryBytes = 4096
)

// ResolveStorageRoots queries snap-capable peers for the storage roots of the
// given addresses at the specified state root. This is used by partial state
// nodes to learn the updated storage roots of untracked contracts (whose storage
// tries are not maintained locally).
//
// For each address, the method sends a snap GetAccountRange request scoped to
// exactly that account's hash. The response contains the full StateAccount
// including the storage root. If a peer returns the same root as oldRoots[addr],
// it's considered stale (hasn't processed the block yet) and the next peer is tried.
func (h *handler) ResolveStorageRoots(
	stateRoot common.Hash,
	addrs []common.Address,
	oldRoots map[common.Address]common.Hash,
) (map[common.Address]common.Hash, error) {
	if len(addrs) == 0 {
		return nil, nil
	}

	// Collect snap-capable peers
	allPeers := h.peers.all()
	var snapPeers []*ethPeer
	for _, p := range allPeers {
		if p.snapExt != nil {
			snapPeers = append(snapPeers, p)
		}
	}
	if len(snapPeers) == 0 {
		return nil, fmt.Errorf("no snap-capable peers available")
	}

	resolved := make(map[common.Address]common.Hash)

	for _, addr := range addrs {
		addrHash := crypto.Keccak256Hash(addr.Bytes())

		var found bool
		for attempt := 0; attempt < storageRootMaxRetries && attempt < len(snapPeers)*2; attempt++ {
			peer := snapPeers[attempt%len(snapPeers)]

			root, err := h.queryAccountStorageRoot(peer, stateRoot, addr, addrHash)
			if err != nil {
				log.Trace("Storage root query failed", "addr", addr, "peer", peer.ID(), "err", err)
				continue
			}
			// Check if peer returned a stale root (hasn't processed this block yet)
			if oldRoot, ok := oldRoots[addr]; ok && root == oldRoot {
				log.Trace("Peer returned stale storage root, trying next", "addr", addr, "peer", peer.ID())
				continue
			}
			resolved[addr] = root
			found = true
			log.Debug("Resolved storage root", "addr", addr, "root", root, "peer", peer.ID())
			break
		}
		if !found {
			log.Warn("Failed to resolve storage root", "addr", addr, "attempts", storageRootMaxRetries)
		}
	}
	return resolved, nil
}

// queryAccountStorageRoot sends a snap GetAccountRange request for a single account
// and returns its storage root from the response.
func (h *handler) queryAccountStorageRoot(
	peer *ethPeer,
	stateRoot common.Hash,
	addr common.Address,
	addrHash common.Hash,
) (common.Hash, error) {
	// Generate unique request ID
	reqID := rand.Uint64()

	// Create response channel and register it
	respCh := make(chan *snap.AccountRangePacket, 1)
	h.pendingSnapQueries.Store(reqID, respCh)

	// Clean up on any exit path
	defer h.pendingSnapQueries.Delete(reqID)

	// Send request: origin = limit = addrHash to request exactly this one account
	if err := peer.snapExt.RequestAccountRange(reqID, stateRoot, addrHash, addrHash, storageRootQueryBytes); err != nil {
		return common.Hash{}, fmt.Errorf("request failed: %w", err)
	}

	// Wait for response with timeout
	select {
	case resp := <-respCh:
		if len(resp.Accounts) == 0 {
			return common.Hash{}, fmt.Errorf("empty response for %s", addr.Hex())
		}
		// Find the account matching our address hash
		for _, acc := range resp.Accounts {
			if acc.Hash == addrHash {
				account, err := types.FullAccount(acc.Body)
				if err != nil {
					return common.Hash{}, fmt.Errorf("failed to decode account: %w", err)
				}
				return account.Root, nil
			}
		}
		return common.Hash{}, fmt.Errorf("account %s not found in response", addr.Hex())

	case <-time.After(storageRootQueryTimeout):
		return common.Hash{}, fmt.Errorf("timeout waiting for account %s", addr.Hex())

	case <-h.quitSync:
		return common.Hash{}, fmt.Errorf("handler shutting down")
	}
}

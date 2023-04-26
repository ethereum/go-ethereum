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

package eth

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
    "github.com/ethereum/go-ethereum/log"
    "github.com/ethereum/go-ethereum/loggy"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// ethHandler implements the eth.Backend interface to handle the various network
// packets that are sent as replies or broadcasts.
type ethHandler handler

func (h *ethHandler) Chain() *core.BlockChain { return h.chain }
func (h *ethHandler) TxPool() eth.TxPool      { return h.txpool }

// RunPeer is invoked when a peer joins on the `eth` protocol.
func (h *ethHandler) RunPeer(peer *eth.Peer, hand eth.Handler) error {
	return (*handler)(h).runEthPeer(peer, hand)
}

// PeerInfo retrieves all known `eth` information about a peer.
func (h *ethHandler) PeerInfo(id enode.ID) interface{} {
	if p := h.peers.peer(id.String()); p != nil {
		return p.info()
	}
	return nil
}

// AcceptTxs retrieves whether transaction processing is enabled on the node
// or if inbound transactions should simply be dropped.
func (h *ethHandler) AcceptTxs() bool {
	return h.acceptTxs.Load()
}

// Handle is invoked from a peer's message handler when it receives a new remote
// message that the handler couldn't consume and serve itself.
func (h *ethHandler) Handle(peer *eth.Peer, packet eth.Packet) error {
	// Consume any broadcasts and announces, forwarding the rest to the downloader
	switch packet := packet.(type) {
	case *eth.NewBlockHashesPacket:
		hashes, numbers := packet.Unpack()
		return h.handleBlockAnnounces(peer, hashes, numbers)

	case *eth.NewBlockPacket:
		return h.handleBlockBroadcast(peer, packet.Block, packet.TD)

	case *eth.NewPooledTransactionHashesPacket66:
		// PERI
		recordAllAnnouncement(peer, *packet)
		return h.txFetcher.Notify(peer.ID(), *packet)

    case *eth.NewPooledTransactionHashesPacket68:
        recordAllAnnouncement(peer, packet.Hashes)
        return h.txFetcher.Notify(peer.ID(), packet.Hashes)
/*
	case *eth.NewPooledTransactionHashesPacket66:
		return h.txFetcher.Notify(peer.ID(), *packet)

	case *eth.NewPooledTransactionHashesPacket68:
		return h.txFetcher.Notify(peer.ID(), packet.Hashes)

	case *eth.TransactionsPacket:
		return h.txFetcher.Enqueue(peer.ID(), *packet, false)

	case *eth.PooledTransactionsPacket:
		return h.txFetcher.Enqueue(peer.ID(), *packet, true)
*/
	case *eth.TransactionsPacket:
		// PERI
		recordAllTx(peer, *packet)
		return h.txFetcher.Enqueue(peer.ID(), *packet, false)

	case *eth.PooledTransactionsPacket:
		// PERI
		recordAllPooledTx(peer, *packet)
		return h.txFetcher.Enqueue(peer.ID(), *packet, true)
	default:
		return fmt.Errorf("unexpected eth packet type: %T", packet)
	}
}

// handleBlockAnnounces is invoked from a peer's message handler when it transmits a
// batch of block announcements for the local node to process.
func (h *ethHandler) handleBlockAnnounces(peer *eth.Peer, hashes []common.Hash, numbers []uint64) error {
	// Drop all incoming block announces from the p2p network if
	// the chain already entered the pos stage and disconnect the
	// remote peer.
	if h.merger.PoSFinalized() {
		// TODO (MariusVanDerWijden) drop non-updated peers after the merge
		return nil
		// return errors.New("unexpected block announces")
	}
	// Schedule all the unknown hashes for retrieval
	var (
		unknownHashes  = make([]common.Hash, 0, len(hashes))
		unknownNumbers = make([]uint64, 0, len(numbers))
	)
	for i := 0; i < len(hashes); i++ {
		if !h.chain.HasBlock(hashes[i], numbers[i]) {
			unknownHashes = append(unknownHashes, hashes[i])
			unknownNumbers = append(unknownNumbers, numbers[i])
		}
	}
	for i := 0; i < len(unknownHashes); i++ {
		h.blockFetcher.Notify(peer.ID(), unknownHashes[i], unknownNumbers[i], time.Now(), peer.RequestOneHeader, peer.RequestBodies)
	}
	return nil
}

// handleBlockBroadcast is invoked from a peer's message handler when it transmits a
// block broadcast for the local node to process.
func (h *ethHandler) handleBlockBroadcast(peer *eth.Peer, block *types.Block, td *big.Int) error {
	// Drop all incoming block announces from the p2p network if
	// the chain already entered the pos stage and disconnect the
	// remote peer.
	if h.merger.PoSFinalized() {
		// TODO (MariusVanDerWijden) drop non-updated peers after the merge
		return nil
		// return errors.New("unexpected block announces")
	}
	// Schedule the block for import
	h.blockFetcher.Enqueue(peer.ID(), block)

	// Assuming the block is importable by the peer, but possibly not yet done so,
	// calculate the head hash and TD that the peer truly must have.
	var (
		trueHead = block.ParentHash()
		trueTD   = new(big.Int).Sub(td, block.Difficulty())
	)
	// Update the peer's total difficulty if better than the previous
	if _, td := peer.Head(); trueTD.Cmp(td) > 0 {
		peer.SetHead(trueHead, trueTD)
		h.chainSync.handlePeerEvent(peer)
	}
	return nil
}

// PERI
// Get transactions from a transaction packet and record their timestamps
func recordAllTx(peer *eth.Peer, txs []*types.Transaction) {
	timestamp := time.Now().UnixNano()
	for _, tx := range txs {
		recordArrival(tx, peer, timestamp, false)
	}
}

// PERI
// Get transactions from a pooled transaction packet and record their timestamps
func recordAllPooledTx(peer *eth.Peer, txs []*types.Transaction) {
	timestamp := time.Now().UnixNano()
	for _, tx := range txs {
		recordArrival(tx, peer, timestamp, true)
	}
}

// PERI
// Get transactions from a transaction hash packet and record their timestamps
func recordAllAnnouncement(peer *eth.Peer, hashes []common.Hash) {
	timestamp := time.Now().UnixNano()
	for _, hash := range hashes {
		recordAnnouncement(hash, peer, timestamp)
	}
}

// PERI
// Record an announcement of a single transaction
func recordAnnouncement(tx common.Hash, peer *eth.Peer, time int64) {
	mapsMutex.Lock()
	defer mapsMutex.Unlock()

	peerID, enode := peer.ID(), peer.Peer.Node().URLv4()

	// Skip the transaction if it is not sampled by hash division (disabled for targeted latency measurement)
	if !isSampledByHashDivision(tx) {
		return
	}

	// Check if the peer forwards a stale tx
	if _, stale := oldArrivals[tx]; stale {
		return
	}

	arrivalTime, arrived := arrivals[tx]
	if !arrived {
		arrivals[tx] = time
		arrivalPerPeer[tx] = map[string]int64{peerID: time}
	} else {
		if arrivalTime >= time {
			arrivals[tx] = time
		}
		arrivalPerPeer[tx][peerID] = time
	}

	// if all transactions are relevant, record them immediately; otherwise, record them at the end of a period
	if !PeriConfig.Targeted {
		loggy.ObserveAll(tx, enode, time)
	}

	if PeriConfig.ShowTxDelivery {
		log.Warn(fmt.Sprintf("Transaction 0x%x received from peer %s", tx, peerID))
	}
}

// PERI
// Record an announcement of a single transaction
func recordArrival(tx *types.Transaction, peer *eth.Peer, time int64, pooled bool) {
	mapsMutex.Lock()
	defer mapsMutex.Unlock()

	txHash, peerID, enode := tx.Hash(), peer.ID(), peer.Peer.Node().URLv4()

	// Skip the transaction if it is not sampled by hash division (disabled for targeted latency measurement)
	if !isSampledByHashDivision(txHash) {
		return
	}

	// Check if the peer forwards a stale tx
	if _, stale := oldArrivals[txHash]; stale {
		return
	}

	arrivalTime, arrived := arrivals[txHash]
	if !arrived {
		arrivals[txHash] = time
		arrivalPerPeer[txHash] = map[string]int64{peerID: time}
	} else {
		if arrivalTime >= time {
			arrivals[txHash] = time
		}
		arrivalPerPeer[txHash][peerID] = time
	}

	if !PeriConfig.Targeted { // Directly record it for global latency measurement
		loggy.ObserveAll(txHash, enode, time)
	} else if isRelevant(tx) { // For targeted latency, print info on the command line without doing anything yet
		_, recorded := targetTx[txHash]

		if !recorded {
			targetTx[txHash] = true
			if pooled {
				log.Warn(fmt.Sprintf("Got full target transaction 0x%x from peer %s", txHash, peerID))
			} else {
				log.Warn(fmt.Sprintf("Got full POOLED target transaction 0x%x from peer %s", txHash, peerID))
			}
		}
	}

	if PeriConfig.ShowTxDelivery {
		log.Warn(fmt.Sprintf("Transaction 0x%x received from peer %s", txHash, peerID))
	}
}

// PERI
// predicate of whether a transaction is relevant (signed by a victim account)
func isRelevant(tx *types.Transaction) bool {
	signers := [...]types.Signer{types.NewLondonSigner(tx.ChainId()), types.NewEIP2930Signer(tx.ChainId()), types.NewEIP155Signer(tx.ChainId()), types.HomesteadSigner{}}
	for _, signer := range signers {
		if sender, err := types.Sender(signer, tx); err == nil {
			if isVictimAccount(sender) {
				return true
			}
		}
	}
	return false
}

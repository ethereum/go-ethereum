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

package txpool

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/event"
)

// Transaction is a helper struct to group together a canonical transaction with
// satellite data items that are needed by the pool but are not part of the chain.
type Transaction struct {
	Tx *types.Transaction // Canonical transaction

	BlobTxBlobs   []kzg4844.Blob       // Blobs needed by the blob pool
	BlobTxCommits []kzg4844.Commitment // Commitments needed by the blob pool
	BlobTxProofs  []kzg4844.Proof      // Proofs needed by the blob pool
}

// SubPool represents a specialized transaction pool that lives on its own (e.g.
// blob pool). Since independent of how many specialized pools we have, they do
// need to be updated in lockstep and assemble into one coherent view for block
// production, this interface defines the common methods that allow the primary
// transaction pool to manage the subpools.
type SubPool interface {
	// Filter is a selector used to decide whether a transaction whould be added
	// to this particular subpool.
	Filter(tx *types.Transaction) bool

	// Init sets the base parameters of the subpool, allowing it to load any saved
	// transactions from disk and also permitting internal maintenance routines to
	// start up.
	//
	// These should not be passed as a constructor argument - nor should the pools
	// start by themselves - in order to keep multiple subpools in lockstep with
	// one another.
	Init(gasTip *big.Int, head *types.Header) error

	// Close terminates any background processing threads and releases any held
	// resources.
	Close() error

	// Reset retrieves the current state of the blockchain and ensures the content
	// of the transaction pool is valid with regard to the chain state.
	Reset(oldHead, newHead *types.Header)

	// SetGasTip updates the minimum price required by the subpool for a new
	// transaction, and drops all transactions below this threshold.
	SetGasTip(tip *big.Int)

	// Has returns an indicator whether subpool has a transaction cached with the
	// given hash.
	Has(hash common.Hash) bool

	// Get returns a transaction if it is contained in the pool, or nil otherwise.
	Get(hash common.Hash) *Transaction

	// Add enqueues a batch of transactions into the pool if they are valid. Due
	// to the large transaction churn, add may postpone fully integrating the tx
	// to a later point to batch multiple ones together.
	Add(txs []*Transaction, local bool, sync bool) []error

	// Pending retrieves all currently processable transactions, grouped by origin
	// account and sorted by nonce.
	Pending(enforceTips bool) map[common.Address][]*types.Transaction

	// SubscribeTransactions subscribes to new transaction events.
	SubscribeTransactions(ch chan<- core.NewTxsEvent) event.Subscription

	// Nonce returns the next nonce of an account, with all transactions executable
	// by the pool already applied on top.
	Nonce(addr common.Address) uint64

	// Stats retrieves the current pool stats, namely the number of pending and the
	// number of queued (non-executable) transactions.
	Stats() (int, int)

	// Content retrieves the data content of the transaction pool, returning all the
	// pending as well as queued transactions, grouped by account and sorted by nonce.
	Content() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction)

	// ContentFrom retrieves the data content of the transaction pool, returning the
	// pending as well as queued transactions of this address, grouped by nonce.
	ContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction)

	// Locals retrieves the accounts currently considered local by the pool.
	Locals() []common.Address

	// Status returns the known status (unknown/pending/queued) of a transaction
	// identified by their hashes.
	Status(hash common.Hash) TxStatus
}

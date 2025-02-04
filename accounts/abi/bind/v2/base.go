// Copyright 2015 The go-ethereum Authors
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

package bind

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/event"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const basefeeWiggleMultiplier = 2

var (
	errNoEventSignature       = errors.New("no event signature")
	errEventSignatureMismatch = errors.New("event signature mismatch")
)

// SignerFn is a signer function callback when a contract requires a method to
// sign the transaction before submission.
type SignerFn func(common.Address, *types.Transaction) (*types.Transaction, error)

// CallOpts is the collection of options to fine tune a contract call request.
type CallOpts struct {
	Pending     bool            // Whether to operate on the pending state or the last known one
	From        common.Address  // Optional the sender address, otherwise the first account is used
	BlockNumber *big.Int        // Optional the block number on which the call should be performed
	BlockHash   common.Hash     // Optional the block hash on which the call should be performed
	Context     context.Context // Network context to support cancellation and timeouts (nil = no timeout)
}

// TransactOpts is the collection of authorization data required to create a
// valid Ethereum transaction.
type TransactOpts struct {
	From   common.Address // Ethereum account to send the transaction from
	Nonce  *big.Int       // Nonce to use for the transaction execution (nil = use pending state)
	Signer SignerFn       // Method to use for signing the transaction (mandatory)

	Value      *big.Int         // Funds to transfer along the transaction (nil = 0 = no funds)
	GasPrice   *big.Int         // Gas price to use for the transaction execution (nil = gas price oracle)
	GasFeeCap  *big.Int         // Gas fee cap to use for the 1559 transaction execution (nil = gas price oracle)
	GasTipCap  *big.Int         // Gas priority fee cap to use for the 1559 transaction execution (nil = gas price oracle)
	GasLimit   uint64           // Gas limit to set for the transaction execution (0 = estimate)
	AccessList types.AccessList // Access list to set for the transaction execution (nil = no access list)

	Context context.Context // Network context to support cancellation and timeouts (nil = no timeout)

	NoSend bool // Do all transact steps but do not send the transaction
}

// FilterOpts is the collection of options to fine tune filtering for events
// within a bound contract.
type FilterOpts struct {
	Start uint64  // Start of the queried range
	End   *uint64 // End of the range (nil = latest)

	Context context.Context // Network context to support cancellation and timeouts (nil = no timeout)
}

// WatchOpts is the collection of options to fine tune subscribing for events
// within a bound contract.
type WatchOpts struct {
	Start   *uint64         // Start of the queried range (nil = latest)
	Context context.Context // Network context to support cancellation and timeouts (nil = no timeout)
}

// MetaData collects all metadata for a bound contract.
type MetaData struct {
	Bin  string      // runtime bytecode (as a hex string)
	ABI  string      // the raw ABI definition (JSON)
	Deps []*MetaData // library dependencies of the contract

	// For bindings that were compiled from combined-json ID is the Solidity library pattern: a 34 character prefix
	// of the hex encoding of the keccak256
	// hash of the fully qualified 'library name', i.e. the path of the source file.
	//
	// For contracts compiled from the ABI definition alone, this is the type name of the contract (as specified
	// in the ABI definition or overridden via the --type flag).
	//
	// This is a unique identifier of a contract within a compilation unit. When used as part of a multi-contract
	// deployment with library dependencies, the ID is used to link
	// contracts during deployment using
	// [LinkAndDeploy].
	ID string

	mu        sync.Mutex
	parsedABI *abi.ABI
}

// ParseABI returns the parsed ABI specification.
func (m *MetaData) ParseABI() (*abi.ABI, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.parsedABI != nil {
		return m.parsedABI, nil
	}
	if parsed, err := abi.JSON(strings.NewReader(m.ABI)); err != nil {
		return nil, err
	} else {
		m.parsedABI = &parsed
	}
	return m.parsedABI, nil
}

// BoundContract represents a contract deployed on-chain.  It does not export any methods, and is used in the low-level
// contract interaction API methods provided in the v2 package.
type BoundContract interface {
	filterLogs(opts *FilterOpts, name string, query ...[]any) (chan types.Log, event.Subscription, error)
	watchLogs(opts *WatchOpts, name string, query ...[]any) (chan types.Log, event.Subscription, error)
	rawCreationTransact(opts *TransactOpts, calldata []byte) (*types.Transaction, error)
	call(opts *CallOpts, input []byte) ([]byte, error)
	transact(opts *TransactOpts, contract *common.Address, input []byte) (*types.Transaction, error)
	addr() common.Address
}

// NewBoundContract creates a new BoundContract instance.
func NewBoundContract(backend ContractBackend, address common.Address, abi abi.ABI) BoundContract {
	return NewBoundContractV1(address, abi, backend, backend, backend)
}

func (c *BoundContractV1) filterLogs(opts *FilterOpts, name string, query ...[]any) (chan types.Log, event.Subscription, error) {
	return c.FilterLogs(opts, name, query...)
}

func (c *BoundContractV1) watchLogs(opts *WatchOpts, name string, query ...[]any) (chan types.Log, event.Subscription, error) {
	return c.WatchLogs(opts, name, query...)
}

func (c *BoundContractV1) rawCreationTransact(opts *TransactOpts, calldata []byte) (*types.Transaction, error) {
	return c.RawCreationTransact(opts, calldata)
}

func (c *BoundContractV1) addr() common.Address {
	return c.address
}

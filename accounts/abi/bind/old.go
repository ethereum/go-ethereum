// Copyright 2016 The go-ethereum Authors
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

// Package bind is the runtime for abigen v1 generated contract bindings.
// Deprecated: please use github.com/ethereum/go-ethereum/bind/v2
package bind

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"io"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/abigen"
	bind2 "github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/accounts/external"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

// Bind generates a v1 contract binding.
// Deprecated: binding generation has moved to github.com/ethereum/go-ethereum/accounts/abi/abigen
func Bind(types []string, abis []string, bytecodes []string, fsigs []map[string]string, pkg string, libs map[string]string, aliases map[string]string) (string, error) {
	return abigen.Bind(types, abis, bytecodes, fsigs, pkg, libs, aliases)
}

// auth.go

// ErrNoChainID is returned whenever the user failed to specify a chain id.
var ErrNoChainID = errors.New("no chain id specified")

// ErrNotAuthorized is returned when an account is not properly unlocked.
var ErrNotAuthorized = bind2.ErrNotAuthorized

// NewTransactor is a utility method to easily create a transaction signer from
// an encrypted json key stream and the associated passphrase.
//
// Deprecated: Use NewTransactorWithChainID instead.
func NewTransactor(keyin io.Reader, passphrase string) (*TransactOpts, error) {
	log.Warn("WARNING: NewTransactor has been deprecated in favour of NewTransactorWithChainID")
	json, err := io.ReadAll(keyin)
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(json, passphrase)
	if err != nil {
		return nil, err
	}
	return NewKeyedTransactor(key.PrivateKey), nil
}

// NewKeyStoreTransactor is a utility method to easily create a transaction signer from
// a decrypted key from a keystore.
//
// Deprecated: Use NewKeyStoreTransactorWithChainID instead.
func NewKeyStoreTransactor(keystore *keystore.KeyStore, account accounts.Account) (*TransactOpts, error) {
	log.Warn("WARNING: NewKeyStoreTransactor has been deprecated in favour of NewTransactorWithChainID")
	signer := types.HomesteadSigner{}
	return &TransactOpts{
		From: account.Address,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != account.Address {
				return nil, ErrNotAuthorized
			}
			signature, err := keystore.SignHash(account, signer.Hash(tx).Bytes())
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},
		Context: context.Background(),
	}, nil
}

// NewKeyedTransactor is a utility method to easily create a transaction signer
// from a single private key.
//
// Deprecated: Use NewKeyedTransactorWithChainID instead.
func NewKeyedTransactor(key *ecdsa.PrivateKey) *TransactOpts {
	log.Warn("WARNING: NewKeyedTransactor has been deprecated in favour of NewKeyedTransactorWithChainID")
	keyAddr := crypto.PubkeyToAddress(key.PublicKey)
	signer := types.HomesteadSigner{}
	return &TransactOpts{
		From: keyAddr,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != keyAddr {
				return nil, ErrNotAuthorized
			}
			signature, err := crypto.Sign(signer.Hash(tx).Bytes(), key)
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},
		Context: context.Background(),
	}
}

// NewTransactorWithChainID is a utility method to easily create a transaction signer from
// an encrypted json key stream and the associated passphrase.
func NewTransactorWithChainID(keyin io.Reader, passphrase string, chainID *big.Int) (*TransactOpts, error) {
	json, err := io.ReadAll(keyin)
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(json, passphrase)
	if err != nil {
		return nil, err
	}
	return NewKeyedTransactorWithChainID(key.PrivateKey, chainID)
}

// NewKeyStoreTransactorWithChainID is a utility method to easily create a transaction signer from
// a decrypted key from a keystore.
func NewKeyStoreTransactorWithChainID(keystore *keystore.KeyStore, account accounts.Account, chainID *big.Int) (*TransactOpts, error) {
	// New version panics for chainID == nil, catch it here.
	if chainID == nil {
		return nil, ErrNoChainID
	}
	return bind2.NewKeyStoreTransactor(keystore, account, chainID), nil
}

// NewKeyedTransactorWithChainID is a utility method to easily create a transaction signer
// from a single private key.
func NewKeyedTransactorWithChainID(key *ecdsa.PrivateKey, chainID *big.Int) (*TransactOpts, error) {
	// New version panics for chainID == nil, catch it here.
	if chainID == nil {
		return nil, ErrNoChainID
	}
	return bind2.NewKeyedTransactor(key, chainID), nil
}

// NewClefTransactor is a utility method to easily create a transaction signer
// with a clef backend.
func NewClefTransactor(clef *external.ExternalSigner, account accounts.Account) *TransactOpts {
	return bind2.NewClefTransactor(clef, account)
}

// backend.go

var (
	// ErrNoCode is returned by call and transact operations for which the requested
	// recipient contract to operate on does not exist in the state db or does not
	// have any code associated with it (i.e. self-destructed).
	ErrNoCode = bind2.ErrNoCode

	// ErrNoPendingState is raised when attempting to perform a pending state action
	// on a backend that doesn't implement PendingContractCaller.
	ErrNoPendingState = bind2.ErrNoPendingState

	// ErrNoBlockHashState is raised when attempting to perform a block hash action
	// on a backend that doesn't implement BlockHashContractCaller.
	ErrNoBlockHashState = bind2.ErrNoBlockHashState

	// ErrNoCodeAfterDeploy is returned by WaitDeployed if contract creation leaves
	// an empty contract behind.
	ErrNoCodeAfterDeploy = bind2.ErrNoCodeAfterDeploy
)

// ContractCaller defines the methods needed to allow operating with a contract on a read
// only basis.
type ContractCaller = bind2.ContractCaller

// PendingContractCaller defines methods to perform contract calls on the pending state.
// Call will try to discover this interface when access to the pending state is requested.
// If the backend does not support the pending state, Call returns ErrNoPendingState.
type PendingContractCaller = bind2.PendingContractCaller

// BlockHashContractCaller defines methods to perform contract calls on a specific block hash.
// Call will try to discover this interface when access to a block by hash is requested.
// If the backend does not support the block hash state, Call returns ErrNoBlockHashState.
type BlockHashContractCaller = bind2.BlockHashContractCaller

// ContractTransactor defines the methods needed to allow operating with a contract
// on a write only basis. Besides the transacting method, the remainder are helpers
// used when the user does not provide some needed values, but rather leaves it up
// to the transactor to decide.
type ContractTransactor = bind2.ContractTransactor

// DeployBackend wraps the operations needed by WaitMined and WaitDeployed.
type DeployBackend = bind2.DeployBackend

// ContractFilterer defines the methods needed to access log events using one-off
// queries or continuous event subscriptions.
type ContractFilterer = bind2.ContractFilterer

// ContractBackend defines the methods needed to work with contracts on a read-write basis.
type ContractBackend = bind2.ContractBackend

// base.go

type SignerFn = bind2.SignerFn

type CallOpts = bind2.CallOpts

type TransactOpts = bind2.TransactOpts

type FilterOpts = bind2.FilterOpts

type WatchOpts = bind2.WatchOpts

type BoundContract = bind2.BoundContract

func NewBoundContract(address common.Address, abi abi.ABI, caller ContractCaller, transactor ContractTransactor, filterer ContractFilterer) *BoundContract {
	return bind2.NewBoundContract(address, abi, caller, transactor, filterer)
}

func DeployContract(opts *TransactOpts, abi abi.ABI, bytecode []byte, backend ContractBackend, params ...interface{}) (common.Address, *types.Transaction, *BoundContract, error) {
	packed, err := abi.Pack("", params...)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	addr, tx, err := bind2.DeployContract(opts, bytecode, backend, packed)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	contract := NewBoundContract(addr, abi, backend, backend, backend)
	return addr, tx, contract, nil
}

// MetaData collects all metadata for a bound contract.
type MetaData struct {
	Bin       string            // runtime bytecode (as a hex string)
	ABI       string            // the raw ABI definition (JSON)
	Sigs      map[string]string // 4byte identifier -> function signature
	mu        sync.Mutex
	parsedABI *abi.ABI
}

// GetAbi returns the parsed ABI definition.
func (m *MetaData) GetAbi() (*abi.ABI, error) {
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

// util.go

// WaitMined waits for tx to be mined on the blockchain.
// It stops waiting when the context is canceled.
func WaitMined(ctx context.Context, b DeployBackend, tx *types.Transaction) (*types.Receipt, error) {
	return bind2.WaitMined(ctx, b, tx.Hash())
}

// WaitMinedHash waits for a transaction with the provided hash to be mined on the blockchain.
// It stops waiting when the context is canceled.
func WaitMinedHash(ctx context.Context, b DeployBackend, hash common.Hash) (*types.Receipt, error) {
	return bind2.WaitMined(ctx, b, hash)
}

// WaitDeployed waits for a contract deployment transaction and returns the on-chain
// contract address when it is mined. It stops waiting when ctx is canceled.
func WaitDeployed(ctx context.Context, b DeployBackend, tx *types.Transaction) (common.Address, error) {
	if tx.To() != nil {
		return common.Address{}, errors.New("tx is not contract creation")
	}
	return bind2.WaitDeployed(ctx, b, tx.Hash())
}

// WaitDeployedHash waits for a contract deployment transaction with the provided hash and returns the on-chain
// contract address when it is mined. It stops waiting when ctx is canceled.
func WaitDeployedHash(ctx context.Context, b DeployBackend, hash common.Hash) (common.Address, error) {
	return bind2.WaitDeployed(ctx, b, hash)
}

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

// Contains all the wrappers from the bind package.

package geth

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Signer is an interface defining the callback when a contract requires a
// method to sign the transaction before submission.
type Signer interface {
	Sign(addr *Address, unsignedTx *Transaction) (tx *Transaction, _ error)
}

type MobileSigner struct {
	sign bind.SignerFn
}

func (s *MobileSigner) Sign(addr *Address, unsignedTx *Transaction) (signedTx *Transaction, _ error) {
	sig, err := s.sign(addr.address, unsignedTx.tx)
	if err != nil {
		return nil, err
	}
	return &Transaction{sig}, nil
}

// CallOpts is the collection of options to fine tune a contract call request.
type CallOpts struct {
	opts bind.CallOpts
}

// NewCallOpts creates a new option set for contract calls.
func NewCallOpts() *CallOpts {
	return new(CallOpts)
}

func (opts *CallOpts) IsPending() bool    { return opts.opts.Pending }
func (opts *CallOpts) GetGasLimit() int64 { return 0 /* TODO(karalabe) */ }

// GetContext cannot be reliably implemented without identity preservation (https://github.com/golang/go/issues/16876)
// Even then it's awkward to unpack the subtleties of a Go context out to Java.
// func (opts *CallOpts) GetContext() *Context { return &Context{opts.opts.Context} }

func (opts *CallOpts) SetPending(pending bool)     { opts.opts.Pending = pending }
func (opts *CallOpts) SetGasLimit(limit int64)     { /* TODO(karalabe) */ }
func (opts *CallOpts) SetContext(context *Context) { opts.opts.Context = context.context }
func (opts *CallOpts) SetFrom(addr *Address)       { opts.opts.From = addr.address }

// TransactOpts is the collection of authorization data required to create a
// valid Ethereum transaction.
type TransactOpts struct {
	opts bind.TransactOpts
}

// NewTransactOpts creates a new option set for contract transaction.
func NewTransactOpts() *TransactOpts {
	return new(TransactOpts)
}

// NewKeyedTransactOpts is a utility method to easily create a transaction signer
// from a single private key.
func NewKeyedTransactOpts(keyJson []byte, passphrase string, chainID *big.Int) (*TransactOpts, error) {
	key, err := keystore.DecryptKey(keyJson, passphrase)
	if err != nil {
		return nil, err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(key.PrivateKey, chainID)
	if err != nil {
		return nil, err
	}
	return &TransactOpts{*auth}, nil
}

func (opts *TransactOpts) GetFrom() *Address    { return &Address{opts.opts.From} }
func (opts *TransactOpts) GetNonce() int64      { return opts.opts.Nonce.Int64() }
func (opts *TransactOpts) GetValue() *BigInt    { return &BigInt{opts.opts.Value} }
func (opts *TransactOpts) GetGasPrice() *BigInt { return &BigInt{opts.opts.GasPrice} }
func (opts *TransactOpts) GetGasLimit() int64   { return int64(opts.opts.GasLimit) }

// GetSigner cannot be reliably implemented without identity preservation (https://github.com/golang/go/issues/16876)
// func (opts *TransactOpts) GetSigner() Signer { return &signer{opts.opts.Signer} }

// GetContext cannot be reliably implemented without identity preservation (https://github.com/golang/go/issues/16876)
// Even then it's awkward to unpack the subtleties of a Go context out to Java.
//func (opts *TransactOpts) GetContext() *Context { return &Context{opts.opts.Context} }

func (opts *TransactOpts) SetFrom(from *Address) { opts.opts.From = from.address }
func (opts *TransactOpts) SetNonce(nonce int64)  { opts.opts.Nonce = big.NewInt(nonce) }
func (opts *TransactOpts) SetSigner(s Signer) {
	opts.opts.Signer = func(addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
		sig, err := s.Sign(&Address{addr}, &Transaction{tx})
		if err != nil {
			return nil, err
		}
		return sig.tx, nil
	}
}
func (opts *TransactOpts) SetValue(value *BigInt)      { opts.opts.Value = value.bigint }
func (opts *TransactOpts) SetGasPrice(price *BigInt)   { opts.opts.GasPrice = price.bigint }
func (opts *TransactOpts) SetGasLimit(limit int64)     { opts.opts.GasLimit = uint64(limit) }
func (opts *TransactOpts) SetContext(context *Context) { opts.opts.Context = context.context }

// BoundContract is the base wrapper object that reflects a contract on the
// Ethereum network. It contains a collection of methods that are used by the
// higher level contract bindings to operate.
type BoundContract struct {
	contract *bind.BoundContract
	address  common.Address
	deployer *types.Transaction
}

// DeployContract deploys a contract onto the Ethereum blockchain and binds the
// deployment address with a wrapper.
func DeployContract(opts *TransactOpts, abiJSON string, bytecode []byte, client *EthereumClient, args *Interfaces) (contract *BoundContract, _ error) {
	// Deploy the contract to the network
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, err
	}
	addr, tx, bound, err := bind.DeployContract(&opts.opts, parsed, common.CopyBytes(bytecode), client.client, args.objects...)
	if err != nil {
		return nil, err
	}
	return &BoundContract{
		contract: bound,
		address:  addr,
		deployer: tx,
	}, nil
}

// BindContract creates a low level contract interface through which calls and
// transactions may be made through.
func BindContract(address *Address, abiJSON string, client *EthereumClient) (contract *BoundContract, _ error) {
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, err
	}
	return &BoundContract{
		contract: bind.NewBoundContract(address.address, parsed, client.client, client.client, client.client),
		address:  address.address,
	}, nil
}

func (c *BoundContract) GetAddress() *Address { return &Address{c.address} }
func (c *BoundContract) GetDeployer() *Transaction {
	if c.deployer == nil {
		return nil
	}
	return &Transaction{c.deployer}
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result.
func (c *BoundContract) Call(opts *CallOpts, out *Interfaces, method string, args *Interfaces) error {
	results := make([]interface{}, len(out.objects))
	copy(results, out.objects)
	if err := c.contract.Call(&opts.opts, &results, method, args.objects...); err != nil {
		return err
	}
	copy(out.objects, results)
	return nil
}

// Transact invokes the (paid) contract method with params as input values.
func (c *BoundContract) Transact(opts *TransactOpts, method string, args *Interfaces) (tx *Transaction, _ error) {
	rawTx, err := c.contract.Transact(&opts.opts, method, args.objects...)
	if err != nil {
		return nil, err
	}
	return &Transaction{rawTx}, nil
}

// RawTransact invokes the (paid) contract method with raw calldata as input values.
func (c *BoundContract) RawTransact(opts *TransactOpts, calldata []byte) (tx *Transaction, _ error) {
	rawTx, err := c.contract.RawTransact(&opts.opts, calldata)
	if err != nil {
		return nil, err
	}
	return &Transaction{rawTx}, nil
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (c *BoundContract) Transfer(opts *TransactOpts) (tx *Transaction, _ error) {
	rawTx, err := c.contract.Transfer(&opts.opts)
	if err != nil {
		return nil, err
	}
	return &Transaction{rawTx}, nil
}

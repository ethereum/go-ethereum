// Copyright 2024 The go-ethereum Authors
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

// Package bind implements utilities for interacting with Solidity contracts.
// This is the 'runtime' for contract bindings generated with the abigen command.
// It includes methods for calling/transacting, filtering chain history for
// specific custom Solidity event types, and creating event subscriptions to monitor the
// chain for event occurrences.
//
// Two methods for contract deployment are provided:
//   - [DeployContract] is intended to be used for deployment of a single contract.
//   - [LinkAndDeploy] is intended for the deployment of multiple
//     contracts, potentially with library dependencies.
package bind

import (
	"errors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
)

// ContractEvent is a type constraint for ABI event types.
type ContractEvent interface {
	ContractEventName() string
}

// FilterEvents filters a historical block range for instances of emission of a
// specific event type from a specified contract.  It returns an error if the
// provided filter opts are invalid or the backend is closed.
//
// FilterEvents is intended to be used with contract event unpack methods in
// bindings generated with the abigen --v2 flag. It should be
// preferred over BoundContract.FilterLogs.
func FilterEvents[Ev ContractEvent](c *BoundContract, opts *FilterOpts, unpack func(*types.Log) (*Ev, error), topics ...[]any) (*EventIterator[Ev], error) {
	var e Ev
	logs, sub, err := c.FilterLogs(opts, e.ContractEventName(), topics...)
	if err != nil {
		return nil, err
	}
	return &EventIterator[Ev]{unpack: unpack, logs: logs, sub: sub}, nil
}

// WatchEvents creates an event subscription to notify when logs of the
// specified event type are emitted from the given contract. Received logs are
// unpacked and forwarded to sink.  If topics are specified, only events are
// forwarded which match the topics.
//
// WatchEvents returns a subscription or an error if the provided WatchOpts are
// invalid or the backend is closed.
//
// WatchEvents is intended to be used with contract event unpack methods in
// bindings generated with the abigen --v2 flag. It should be
// preferred over BoundContract.WatchLogs.
func WatchEvents[Ev ContractEvent](c *BoundContract, opts *WatchOpts, unpack func(*types.Log) (*Ev, error), sink chan<- *Ev, topics ...[]any) (event.Subscription, error) {
	var e Ev
	logs, sub, err := c.WatchLogs(opts, e.ContractEventName(), topics...)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				ev, err := unpack(&log)
				if err != nil {
					return err
				}

				select {
				case sink <- ev:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// EventIterator is an object for iterating over the results of a event log
// filter call.
type EventIterator[T any] struct {
	current *T
	unpack  func(*types.Log) (*T, error)
	logs    <-chan types.Log
	sub     ethereum.Subscription
	fail    error // error to hold reason for iteration failure
	closed  bool  // true if Close has been called
}

// Value returns the current value of the iterator, or nil if there isn't one.
func (it *EventIterator[T]) Value() *T {
	return it.current
}

// Next advances the iterator to the subsequent event (if there is one),
// returning true if the iterator advanced.
//
// If the attempt to convert the raw log object to an instance of T using the
// unpack function provided via FilterEvents returns an error: that error is
// returned and subsequent calls to Next will not advance the iterator.
func (it *EventIterator[T]) Next() (advanced bool) {
	// If the iterator failed with an error, don't proceed
	if it.fail != nil || it.closed {
		return false
	}
	// if the iterator is still active, block until a log is received or the
	// underlying subscription terminates.
	select {
	case log := <-it.logs:
		res, err := it.unpack(&log)
		if err != nil {
			it.fail = err
			return false
		}
		it.current = res
		return true
	case <-it.sub.Err():
		// regardless of how the subscription ends, still be able to iterate
		// over any unread logs.
		select {
		case log := <-it.logs:
			res, err := it.unpack(&log)
			if err != nil {
				it.fail = err
				return false
			}
			it.current = res
			return true
		default:
			return false
		}
	}
}

// Error returns an error if iteration has failed.
func (it *EventIterator[T]) Error() error {
	return it.fail
}

// Close releases any pending underlying resources.  Any subsequent calls to
// Next will not advance the iterator, but the current value remains accessible.
func (it *EventIterator[T]) Close() error {
	it.closed = true
	it.sub.Unsubscribe()
	return nil
}

// Call performs an eth_call to a contract with optional call data.
//
// To call a function that doesn't return any output, pass nil as the unpack
// function. This can be useful if you just want to check that the function
// doesn't revert.
//
// Call is intended to be used with contract method unpack methods in
// bindings generated with the abigen --v2 flag. It should be
// preferred over BoundContract.Call
func Call[T any](c *BoundContract, opts *CallOpts, calldata []byte, unpack func([]byte) (T, error)) (T, error) {
	var defaultResult T
	packedOutput, err := c.CallRaw(opts, calldata)
	if err != nil {
		return defaultResult, err
	}
	if unpack == nil {
		if len(packedOutput) > 0 {
			return defaultResult, errors.New("contract returned data, but no unpack function was given")
		}
		return defaultResult, nil
	}
	res, err := unpack(packedOutput)
	if err != nil {
		return defaultResult, err
	}
	return res, err
}

// Transact creates and submits a transaction to a contract with optional input
// data.
//
// Transact is identical to BoundContract.RawTransact, and is provided as a
// package-level method so that interactions with contracts whose bindings were
// generated with the abigen --v2 flag are consistent (they do not require
// calling methods on the BoundContract instance).
func Transact(c *BoundContract, opt *TransactOpts, data []byte) (*types.Transaction, error) {
	return c.RawTransact(opt, data)
}

// DeployContract creates and submits a deployment transaction based on the
// deployer bytecode and optional ABI-encoded constructor input.  It returns
// the address and creation transaction of the pending contract, or an error
// if the creation failed.
//
// To initiate the deployment of multiple contracts with one method call, see the
// [LinkAndDeploy] method.
func DeployContract(opts *TransactOpts, bytecode []byte, backend ContractBackend, constructorInput []byte) (common.Address, *types.Transaction, error) {
	c := NewBoundContract(common.Address{}, abi.ABI{}, backend, backend, backend)

	tx, err := c.RawCreationTransact(opts, append(bytecode, constructorInput...))
	if err != nil {
		return common.Address{}, nil, err
	}
	return crypto.CreateAddress(opts.From, tx.Nonce()), tx, nil
}

// DefaultDeployer returns a DeployFn that signs and submits creation transactions
// using the given signer.
//
// The DeployFn returned by DefaultDeployer should be used by LinkAndDeploy in
// almost all cases, unless a custom DeployFn implementation is needed.
func DefaultDeployer(opts *TransactOpts, backend ContractBackend) DeployFn {
	return func(input []byte, deployer []byte) (common.Address, *types.Transaction, error) {
		addr, tx, err := DeployContract(opts, deployer, backend, input)
		if err != nil {
			return common.Address{}, nil, err
		}
		return addr, tx, nil
	}
}

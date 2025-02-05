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

package bind

import (
	"context"
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

// FilterEvents returns an EventIterator instance for filtering historical events based on the event id and a block range.
func FilterEvents[Ev ContractEvent](c BoundContract, opts *FilterOpts, unpack func(*types.Log) (*Ev, error), topics ...[]any) (*EventIterator[Ev], error) {
	var e Ev
	logs, sub, err := c.filterLogs(opts, e.ContractEventName(), topics...)
	if err != nil {
		return nil, err
	}
	return &EventIterator[Ev]{unpack: unpack, logs: logs, sub: sub}, nil
}

// WatchEvents causes logs emitted with a given event id from a specified
// contract to be intercepted, unpacked, and forwarded to sink.  If
// unpack returns an error, the returned subscription is closed with the
// error.
func WatchEvents[Ev ContractEvent](c BoundContract, opts *WatchOpts, unpack func(*types.Log) (*Ev, error), sink chan<- *Ev, topics ...[]any) (event.Subscription, error) {
	var e Ev
	logs, sub, err := c.watchLogs(opts, e.ContractEventName(), topics...)
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

// EventIterator is returned from FilterLogs and is used to iterate over the raw logs and unpacked data for events.
type EventIterator[T any] struct {
	event *T // event containing the contract specifics and raw log

	unpack func(*types.Log) (*T, error) // Unpack function for the event

	logs <-chan types.Log      // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for solc_errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Value returns the current value of the iterator, or nil if there isn't one.
func (it *EventIterator[T]) Value() *T {
	return it.event
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *EventIterator[T]) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			res, err := it.unpack(&log)
			if err != nil {
				it.fail = err
				return false
			}
			it.event = res
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		res, err := it.unpack(&log)
		if err != nil {
			it.fail = err
			return false
		}
		it.event = res
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *EventIterator[T]) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EventIterator[T]) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Call performs an eth_call on the given bound contract instance, using the provided
// ABI-encoded input.
//
// To call a function that doesn't return any output, pass nil as the unpack function.
// This can be useful if you just want to check that the function doesn't revert.
func Call[T any](c BoundContract, opts *CallOpts, packedInput []byte, unpack func([]byte) (T, error)) (T, error) {
	var defaultResult T
	packedOutput, err := c.call(opts, packedInput)
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

// Transact initiates a transaction with the given raw calldata as the input.
func Transact(c BoundContract, opt *TransactOpts, packedInput []byte) (*types.Transaction, error) {
	addr := c.addr()
	return c.transact(opt, &addr, packedInput)
}

// DeployContract deploys a contract onto the Ethereum blockchain and binds the
// deployment address with a Go wrapper.  It expects its parameters to be abi-encoded
// bytes.
func DeployContract(opts *TransactOpts, bytecode []byte, backend ContractBackend, packedParams []byte) (common.Address, *types.Transaction, error) {
	c := NewBoundContractV1(common.Address{}, abi.ABI{}, backend, backend, backend)
	tx, err := c.RawCreationTransact(opts, append(bytecode, packedParams...))
	if err != nil {
		return common.Address{}, nil, err
	}
	address := crypto.CreateAddress(opts.From, tx.Nonce())
	return address, tx, nil
}

// DefaultDeployer returns a DeployFn that signs and submits creation transactions using the given signer.
func DefaultDeployer(ctx context.Context, from common.Address, backend ContractBackend, signer SignerFn) DeployFn {
	opts := &TransactOpts{
		From:    from,
		Nonce:   nil,
		Signer:  signer,
		Context: ctx,
	}
	return func(input []byte, deployer []byte) (common.Address, *types.Transaction, error) {
		addr, tx, err := DeployContract(opts, deployer, backend, input)
		if err != nil {
			return common.Address{}, nil, err
		}
		return addr, tx, nil
	}
}

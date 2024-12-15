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

package v2

import (
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// ContractInstance represents a contract deployed on-chain that can be interacted with (filter for past logs, watch
// for new logs, call, transact).
type ContractInstance struct {
	Address common.Address
	Backend bind.ContractBackend
}

// FilterEvents returns an EventIterator instance for filtering historical events based on the event id and a block range.
func FilterEvents[T any](instance *ContractInstance, opts *bind.FilterOpts, eventID common.Hash, unpack func(*types.Log) (*T, error), topics ...[]any) (*EventIterator[T], error) {
	backend := instance.Backend
	c := bind.NewBoundContract(instance.Address, abi.ABI{}, backend, backend, backend)
	logs, sub, err := c.FilterLogsById(opts, eventID, topics...)
	if err != nil {
		return nil, err
	}
	return &EventIterator[T]{unpack: unpack, logs: logs, sub: sub}, nil
}

// WatchEvents causes logs emitted with a given event id from a specified
// contract to be intercepted, unpacked, and forwarded to sink.  If
// unpack returns an error, the returned subscription is closed with the
// error.
func WatchEvents[T any](instance *ContractInstance, opts *bind.WatchOpts, eventID common.Hash, unpack func(*types.Log) (*T, error), sink chan<- *T, topics ...[]any) (event.Subscription, error) {
	backend := instance.Backend
	c := bind.NewBoundContract(instance.Address, abi.ABI{}, backend, backend, backend)
	logs, sub, err := c.WatchLogsForId(opts, eventID, topics...)
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

// Transact creates and submits a transaction to the bound contract instance
// using the provided abi-encoded input (or nil).
func Transact(instance *ContractInstance, opts *bind.TransactOpts, input []byte) (*types.Transaction, error) {
	var (
		addr    = instance.Address
		backend = instance.Backend
	)
	c := bind.NewBoundContract(addr, abi.ABI{}, backend, backend, backend)
	return c.RawTransact(opts, input)
}

// Call performs an eth_call on the given bound contract instance, using the
// provided abi-encoded input (or nil).
func Call[T any](instance *ContractInstance, opts *bind.CallOpts, packedInput []byte, unpack func([]byte) (*T, error)) (*T, error) {
	backend := instance.Backend
	c := bind.NewBoundContract(instance.Address, abi.ABI{}, backend, backend, backend)
	packedOutput, err := c.CallRaw(opts, packedInput)
	if err != nil {
		return nil, err
	}
	return unpack(packedOutput)
}

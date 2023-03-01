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

package bind

import (
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
)

// ContractInstance provides means to interact with
// a deployed contract.
type ContractInstance interface {
	Address() common.Address
	Backend() ContractBackend
}

func DeployContract2(opts *TransactOpts, bytecode []byte, input []byte, backend ContractBackend) (common.Address, *types.Transaction, error) {
	c := NewBoundContract(common.Address{}, abi.ABI{}, backend, backend, backend)
	tx, err := c.transact(opts, nil, append(bytecode, input...))
	if err != nil {
		return common.Address{}, nil, err
	}
	address := crypto.CreateAddress(opts.From, tx.Nonce())
	return address, tx, nil
}

func Call2[T any](instance ContractInstance, opts *CallOpts, input []byte, unpack func([]byte) (T, error)) (arg T, err error) {
	var data []byte
	data, err = CallRaw(instance, opts, input)
	if err != nil {
		return
	}
	return unpack(data)
}

func CallRaw(instance ContractInstance, opts *CallOpts, input []byte) ([]byte, error) {
	backend := instance.Backend()
	c := NewBoundContract(instance.Address(), abi.ABI{}, backend, backend, backend)
	return c.call(opts, input)
}

func Transact2(instance ContractInstance, opts *TransactOpts, input []byte) (*types.Transaction, error) {
	var (
		addr    = instance.Address()
		backend = instance.Backend()
	)
	c := NewBoundContract(addr, abi.ABI{}, backend, backend, backend)
	return c.transact(opts, &addr, input)
}

func Transfer2(instance ContractInstance, opts *TransactOpts) (*types.Transaction, error) {
	backend := instance.Backend()
	c := NewBoundContract(instance.Address(), abi.ABI{}, backend, backend, backend)
	return c.Transfer(opts)
}

func FilterLogs[T any](instance ContractInstance, opts *FilterOpts, eventID common.Hash, unpack func(types.Log) (*T, error), topics ...[]any) (*EventIterator[T], error) {
	backend := instance.Backend()
	c := NewBoundContract(instance.Address(), abi.ABI{}, backend, backend, backend)
	logs, sub, err := c.filterLogs(opts, eventID, topics...)
	if err != nil {
		return nil, err
	}
	return &EventIterator[T]{unpack: unpack, logs: logs, sub: sub}, nil
}

func WatchLogs[T any](instance ContractInstance, opts *WatchOpts, eventID common.Hash, unpack func(types.Log) (*T, error), sink chan<- *T, topics ...[]any) (event.Subscription, error) {
	backend := instance.Backend()
	c := NewBoundContract(instance.Address(), abi.ABI{}, backend, backend, backend)
	logs, sub, err := c.watchLogs(opts, eventID, topics...)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				ev, err := unpack(log)
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
	Event *T // Event containing the contract specifics and raw log

	unpack func(types.Log) (*T, error) // Unpack function for the event

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
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
			res, err := it.unpack(log)
			if err != nil {
				it.fail = err
				return false
			}
			it.Event = res
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		res, err := it.unpack(log)
		if err != nil {
			it.fail = err
			return false
		}
		it.Event = res
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

package v2

import (
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"regexp"
	"strings"
)

type ContractInstance struct {
	Address common.Address
	Backend bind.ContractBackend
}

func DeployContracts(auth *bind.TransactOpts, backend bind.ContractBackend, constructorInput []byte, contracts map[string]*bind.MetaData) {
	// match if the contract has dynamic libraries that need to be linked
	hasDepsMatcher, err := regexp.Compile("__\\$.*\\$__")
	if err != nil {
		panic(err)
	}

	// deps we are linking
	wipDeps := make(map[string]string)
	for id, metadata := range contracts {
		wipDeps[id] = metadata.Bin
	}

	// nested iteration:  find contracts without library dependencies first,
	// deploy them, link them into any other contracts that depend on them.
	// repeat this until there are no more contracts to link/deploy
	for {
		for id, contractBin := range wipDeps {
			if !hasDepsMatcher.MatchString(contractBin) {
				// this library/contract doesn't depend on any others
				// it can be deployed as-is.
				abi, err := contracts[id].GetAbi()
				if err != nil {
					panic(err)
				}
				addr, _, _, err := bind.DeployContractRaw(auth, *abi, []byte(contractBin), backend, constructorInput)
				if err != nil {
					panic(err)
				}
				delete(wipDeps, id)

				// embed the address of the deployed contract into any
				// libraries/contracts that depend on it.
				for id, contractBin := range wipDeps {
					contractBin = strings.ReplaceAll(contractBin, fmt.Sprintf("__$%s%__", id), fmt.Sprintf("__$%s$__", addr.String()))
					wipDeps[id] = contractBin
				}
			}
		}
		if len(wipDeps) == 0 {
			break
		}
	}
}

func FilterLogs[T any](instance *ContractInstance, opts *bind.FilterOpts, eventID common.Hash, unpack func(*types.Log) (*T, error), topics ...[]any) (*EventIterator[T], error) {
	backend := instance.Backend
	c := bind.NewBoundContract(instance.Address, abi.ABI{}, backend, backend, backend)
	logs, sub, err := c.FilterLogs(opts, eventID.String(), topics...)
	if err != nil {
		return nil, err
	}
	return &EventIterator[T]{unpack: unpack, logs: logs, sub: sub}, nil
}

func WatchLogs[T any](instance *ContractInstance, opts *bind.WatchOpts, eventID common.Hash, unpack func(*types.Log) (*T, error), sink chan<- *T, topics ...[]any) (event.Subscription, error) {
	backend := instance.Backend
	c := bind.NewBoundContract(instance.Address, abi.ABI{}, backend, backend, backend)
	logs, sub, err := c.WatchLogs(opts, eventID.String(), topics...)
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
	Event *T // Event containing the contract specifics and raw log

	unpack func(*types.Log) (*T, error) // Unpack function for the event

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
			res, err := it.unpack(&log)
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
		res, err := it.unpack(&log)
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

func Transact(instance bind.ContractInstance, opts *bind.TransactOpts, input []byte) (*types.Transaction, error) {
	var (
		addr    = instance.Address()
		backend = instance.Backend()
	)
	c := bind.NewBoundContract(addr, abi.ABI{}, backend, backend, backend)
	return c.RawTransact(opts, input)
}

func Transfer(instance bind.ContractInstance, opts *bind.TransactOpts) (*types.Transaction, error) {
	backend := instance.Backend()
	c := bind.NewBoundContract(instance.Address(), abi.ABI{}, backend, backend, backend)
	return c.Transfer(opts)
}

func CallRaw(instance bind.ContractInstance, opts *bind.CallOpts, input []byte) ([]byte, error) {
	backend := instance.Backend()
	c := bind.NewBoundContract(instance.Address(), abi.ABI{}, backend, backend, backend)
	return c.CallRaw(opts, input)
}

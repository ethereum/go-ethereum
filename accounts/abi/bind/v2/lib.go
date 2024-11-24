package v2

import (
	"encoding/hex"
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

func deployContract(backend bind.ContractBackend, auth *bind.TransactOpts, constructor []byte, contract string) (deploymentTx *types.Transaction, deploymentAddr common.Address, err error) {
	contractBinBytes, err := hex.DecodeString(contract[2:])
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("contract bytecode is not a hex string: %s", contractBinBytes[2:])
	}
	addr, tx, _, err := bind.DeployContractRaw(auth, contractBinBytes, backend, constructor)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to deploy contract: %v", err)
	}
	return tx, addr, nil
}

func deployLibs(backend bind.ContractBackend, auth *bind.TransactOpts, contracts map[string]string) (deploymentTxs map[common.Address]*types.Transaction, deployAddrs map[common.Address]struct{}, err error) {
	for _, contractBin := range contracts {
		contractBinBytes, err := hex.DecodeString(contractBin[2:])
		if err != nil {
			return deploymentTxs, deployAddrs, fmt.Errorf("contract bytecode is not a hex string: %s", contractBin[2:])
		}
		// TODO: can pass nil for constructor?
		addr, tx, _, err := bind.DeployContractRaw(auth, contractBinBytes, backend, []byte{})
		if err != nil {
			return deploymentTxs, deployAddrs, fmt.Errorf("failed to deploy contract: %v", err)
		}
		deploymentTxs[addr] = tx
		deployAddrs[addr] = struct{}{}
	}

	return deploymentTxs, deployAddrs, nil
}

func linkContract(contract string, linkedLibs map[string]common.Address) (deployableContract string, err error) {
	reMatchSpecificPattern, err := regexp.Compile("__\\$([a-f0-9]+)\\$__")
	if err != nil {
		return "", err
	}

	// link in any library the contract depends on
	for _, match := range reMatchSpecificPattern.FindAllStringSubmatch(contract, -1) {
		matchingPattern := match[1]
		addr := linkedLibs[matchingPattern]
		contract = strings.ReplaceAll(contract, matchingPattern, addr.String())
	}
	return contract, nil
}

func linkLibs(deps *map[string]string, linked *map[string]common.Address) (deployableDeps map[string]string) {
	reMatchSpecificPattern, err := regexp.Compile("__\\$([a-f0-9]+)\\$__")
	if err != nil {
		panic(err)
	}
	reMatchAnyPattern, err := regexp.Compile("__\\$.*\\$__")
	if err != nil {
		panic(err)
	}
	deployableDeps = make(map[string]string)

	for pattern, dep := range *deps {
		// attempt to replace references to every single linked dep
		for _, match := range reMatchSpecificPattern.FindAllStringSubmatch(dep, -1) {
			matchingPattern := match[1]
			addr, ok := (*linked)[matchingPattern]
			if !ok {
				continue
			}
			(*deps)[pattern] = strings.ReplaceAll(dep, matchingPattern, addr.String())
		}
		// if we linked something into this dep, see if it can be deployed
		if !reMatchAnyPattern.MatchString((*deps)[pattern]) {
			deployableDeps[pattern] = (*deps)[pattern]
			delete(*deps, pattern)
		}
	}

	return deployableDeps
}

func LinkAndDeployContractWithOverrides(auth *bind.TransactOpts, backend bind.ContractBackend, constructorInputs []byte, contract *bind.MetaData, libs map[string]string, overrides map[string]common.Address) (allDeployTxs map[common.Address]*types.Transaction, allDeployAddrs map[common.Address]struct{}, err error) {
	// initialize the set of already-deployed contracts with given override addresses
	linked := make(map[string]common.Address)
	for pattern, deployAddr := range overrides {
		linked[pattern] = deployAddr
		if _, ok := libs[pattern]; ok {
			delete(libs, pattern)
		}
	}

	// link and deploy dynamic libraries
	for {
		deployableDeps := linkLibs(&libs, &linked)
		if len(deployableDeps) == 0 {
			break
		}
		deployTxs, deployAddrs, err := deployLibs(backend, auth, deployableDeps)
		for addr, _ := range deployAddrs {
			allDeployAddrs[addr] = struct{}{}
		}
		for addr, tx := range deployTxs {
			allDeployTxs[addr] = tx
		}
		if err != nil {
			return deployTxs, allDeployAddrs, err
		}
	}
	linkedContract, err := linkContract(contract.Bin, linked)
	if err != nil {
		return allDeployTxs, allDeployAddrs, err
	}
	// link and deploy the contracts
	contractTx, contractAddr, err := deployContract(backend, auth, constructorInputs, linkedContract)
	allDeployAddrs[contractAddr] = struct{}{}
	allDeployTxs[contractAddr] = contractTx
	return allDeployTxs, allDeployAddrs, err
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

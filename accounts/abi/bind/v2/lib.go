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

// deployContract deploys a hex-encoded contract with the given constructor
// input.  It returns the deployment transaction, address on success.
func deployContract(constructor []byte, contract string, deploy func(input, deployer []byte) (common.Address, *types.Transaction, error)) (deploymentAddr common.Address, deploymentTx *types.Transaction, err error) {
	contractBinBytes, err := hex.DecodeString(contract[2:])
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("contract bytecode is not a hex string: %s", contractBinBytes[2:])
	}
	addr, tx, err := deploy(constructor, contractBinBytes)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to deploy contract: %v", err)
	}
	return addr, tx, nil
}

// deployLibs iterates the set contracts (map of pattern to hex-encoded
// contract deployer code). Each contract is deployed, and the
// resulting addresses/deployment-txs are returned on success.
func deployLibs(contracts map[string]string, deploy func(input, deployer []byte) (common.Address, *types.Transaction, error)) (deploymentTxs map[common.Address]*types.Transaction, deployAddrs map[string]common.Address, err error) {
	deploymentTxs = make(map[common.Address]*types.Transaction)
	deployAddrs = make(map[string]common.Address)

	for pattern, contractBin := range contracts {
		contractDeployer, err := hex.DecodeString(contractBin[2:])
		if err != nil {
			return deploymentTxs, deployAddrs, fmt.Errorf("contract bytecode is not a hex string: %s", contractBin[2:])
		}
		addr, tx, err := deploy([]byte{}, contractDeployer)
		if err != nil {
			return deploymentTxs, deployAddrs, fmt.Errorf("failed to deploy contract: %v", err)
		}
		deploymentTxs[addr] = tx
		deployAddrs[pattern] = addr
	}

	return deploymentTxs, deployAddrs, nil
}

// linkContract takes an unlinked contract deployer hex-encoded code, a map of
// already-deployed library dependencies, replaces references to deployed library
// dependencies in the contract code, and returns the contract deployment bytecode on
// success.
func linkContract(contract string, linkedLibs map[string]common.Address) (deployableContract string, err error) {
	reMatchSpecificPattern, err := regexp.Compile("__\\$([a-f0-9]+)\\$__")
	if err != nil {
		return "", err
	}
	// link in any library the contract depends on
	for _, match := range reMatchSpecificPattern.FindAllStringSubmatch(contract, -1) {
		matchingPattern := match[1]
		addr := linkedLibs[matchingPattern]
		contract = strings.ReplaceAll(contract, "__$"+matchingPattern+"$__", addr.String()[2:])
	}
	return contract, nil
}

// linkLibs iterates the set of dependencies that have yet to be
// linked/deployed (pending), replacing references to library dependencies
// (i.e. mutating pending) if those dependencies are fully linked/deployed
// (in 'linked').
//
// contracts that have become fully linked in the current invocation are
// returned.
func linkLibs(pending map[string]string, linked map[string]common.Address) (newPending map[string]string, deployableDeps map[string]string) {
	newPending = make(map[string]string)
	reMatchSpecificPattern, err := regexp.Compile("__\\$([a-f0-9]+)\\$__")
	if err != nil {
		panic(err)
	}
	reMatchAnyPattern, err := regexp.Compile("__\\$.*\\$__")
	if err != nil {
		panic(err)
	}
	deployableDeps = make(map[string]string)

	for pattern, dep := range pending {
		newPending[pattern] = dep
		// link references to dependent libraries that have been deployed
		for _, match := range reMatchSpecificPattern.FindAllStringSubmatch(newPending[pattern], -1) {
			matchingPattern := match[1]
			addr, ok := linked[matchingPattern]
			if !ok {
				continue
			}

			newPending[pattern] = strings.ReplaceAll(newPending[pattern], "__$"+matchingPattern+"$__", addr.String()[2:])
		}
		// if the library code became fully linked, move it from pending->linked.
		if !reMatchAnyPattern.MatchString(newPending[pattern]) {
			deployableDeps[pattern] = newPending[pattern]
			delete(newPending, pattern)
		}
	}
	return newPending, deployableDeps
}

// ContractDeployParams represents state needed to deploy a contract:
// the metdata and constructor input (which can be nil if no input is specified).
type ContractDeployParams struct {
	Meta *bind.MetaData
	// Input is the ABI-encoded constructor input for the contract deployment.
	Input []byte
}

// DeploymentParams represents parameters needed to deploy a
// set of contracts, their dependency libraries.  It takes an optional override
// list to specify libraries that have already been deployed on-chain.
type DeploymentParams struct {
	// Contracts is the set of contract deployment parameters for contracts
	// that are about to be deployed.
	Contracts []ContractDeployParams
	// Libraries is a map of pattern to metadata for library contracts that
	// are to be deployed.
	Libraries []*bind.MetaData
	// Overrides is an optional map of pattern to deployment address.
	// Contracts/libraries that refer to dependencies in the override
	// set are linked to the provided address (an already-deployed contract).
	Overrides map[string]common.Address
}

// DeploymentResult contains the relevant information from the deployment of
// multiple contracts:  their deployment txs and addresses.
type DeploymentResult struct {
	// map of contract library pattern -> deploy transaction
	Txs map[string]*types.Transaction
	// map of contract library pattern -> deployed address
	Addrs map[string]common.Address
}

func (d *DeploymentResult) Accumulate(other *DeploymentResult) {
	for pattern, tx := range other.Txs {
		d.Txs[pattern] = tx
	}
	for pattern, addr := range other.Addrs {
		d.Addrs[pattern] = addr
	}
}

type ContractDeployer interface {
	DeployContract(input []byte, deployer []byte) (common.Address, *types.Transaction, error)
}

type depTreeBuilder struct {
	overrides map[string]common.Address
}
type depTreeNode struct {
	pattern      string
	unlinkedCode string
	nodes        []*depTreeNode
}

func (d *depTreeBuilder) buildDepTree(contractBin string, libs map[string]string) *depTreeNode {
	node := depTreeNode{contractBin, contractBin, nil}

	reMatchSpecificPattern, err := regexp.Compile("__\\$([a-f0-9]+)\\$__")
	if err != nil {
		panic(err)
	}
	for _, match := range reMatchSpecificPattern.FindAllStringSubmatch(contractBin, -1) {
		pattern := match[1]
		if _, ok := d.overrides[pattern]; ok {
			continue
		}
		node.nodes = append(node.nodes, d.buildDepTree(libs[pattern], libs))
	}
	return &node
}

type treeDeployer struct {
	deployedAddrs map[string]common.Address
	deployerTxs   map[string]*types.Transaction
	inputs        map[string][]byte
	deploy        func(input, deployer []byte) (common.Address, *types.Transaction, error)
	err           error
}

func (d *treeDeployer) linkAndDeploy(node *depTreeNode) {
	for _, childNode := range node.nodes {
		d.linkAndDeploy(childNode)
	}
	// link in all node dependencies and produce the deployer bytecode
	deployerCode := node.unlinkedCode
	for _, child := range node.nodes {
		deployerCode = strings.ReplaceAll(deployerCode, child.pattern, d.deployedAddrs[child.pattern].String()[2:])
	}
	// deploy the contract.
	addr, tx, err := d.deploy(d.inputs[node.pattern], common.Hex2Bytes(deployerCode))
	if err != nil {
		d.err = err
	} else {
		d.deployedAddrs[node.pattern] = addr
		d.deployerTxs[node.pattern] = tx
	}
}

func (d *treeDeployer) Result() (*DeploymentResult, error) {
	if d.err != nil {
		return nil, d.err
	}
	return &DeploymentResult{
		Txs:   d.deployerTxs,
		Addrs: d.deployedAddrs,
	}, nil
}

// LinkAndDeploy deploys a specified set of contracts and their dependent
// libraries.  If an error occurs, only contracts which were successfully
// deployed are returned in the result.
func LinkAndDeploy(deployParams DeploymentParams, deploy func(input, deployer []byte) (common.Address, *types.Transaction, error)) (res *DeploymentResult, err error) {
	// re-express libraries as a map of pattern -> pre-link binary
	unlinkedLibs := make(map[string]string)
	for _, meta := range deployParams.Libraries {
		unlinkedLibs[meta.Pattern] = meta.Bin
	}
	accumRes := &DeploymentResult{
		Txs:   make(map[string]*types.Transaction),
		Addrs: make(map[string]common.Address),
	}
	for _, contract := range deployParams.Contracts {
		treeBuilder := depTreeBuilder{deployParams.Overrides}
		tree := treeBuilder.buildDepTree(contract.Meta.Bin, unlinkedLibs)
		deployer := treeDeployer{deploy: deploy}
		deployer.linkAndDeploy(tree)
		res, err := deployer.Result()
		if err != nil {
			return accumRes, err
		}
		accumRes.Accumulate(res)

	}
	return accumRes, nil
}

// TODO: this will be generated as part of the bindings, contain the ABI (or metadata object?) and errors
type ContractInstance struct {
	Address common.Address
	Backend bind.ContractBackend
}

// TODO: adding docs soon (jwasinger)
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

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

package bind

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sync/atomic"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/net/context"
)

// SignerFn is a signer function callback when a contract requires a method to
// sign the transaction before submission.
type SignerFn func(common.Address, *types.Transaction) (*types.Transaction, error)

// CallOpts is the collection of options to fine tune a contract call request.
type CallOpts struct {
	Pending bool // Whether to operate on the pending state or the last known one

	Context context.Context // Network context to support cancellation and timeouts (nil = no timeout)
}

// TransactOpts is the collection of authorization data required to create a
// valid Ethereum transaction.
type TransactOpts struct {
	From   common.Address // Ethereum account to send the transaction from
	Nonce  *big.Int       // Nonce to use for the transaction execution (nil = use pending state)
	Signer SignerFn       // Method to use for signing the transaction (mandatory)

	Value    *big.Int // Funds to transfer along along the transaction (nil = 0 = no funds)
	GasPrice *big.Int // Gas price to use for the transaction execution (nil = gas price oracle)
	GasLimit *big.Int // Gas limit to set for the transaction execution (nil = estimate + 10%)

	Context context.Context // Network context to support cancellation and timeouts (nil = no timeout)
}

// SubscribeOpts is the collection of options to fine tune the contract subscription
// and unsubscription requests.
type SubscribeOpts struct {
	History   bool     // Whether to also retrieve events from the past
	FromBlock *big.Int // Block at which we want to start retrieving past events

	Context context.Context // Network context to support cancellation and timeouts (nil = no timeout)
}

// EventOpts is a collection of options to fine tune the retrieval of events that
// happened on a contract.
type EventOpts struct {
	FromBlock *big.Int
	ToBlock   *big.Int

	Context context.Context
}

// BoundContract is the base wrapper object that reflects a contract on the
// Ethereum network. It contains a collection of methods that are used by the
// higher level contract bindings to operate.
type BoundContract struct {
	address common.Address // Deployment address of the contract on the Ethereum blockchain
	abi     abi.ABI        // Reflect based ABI to access the correct Ethereum methods

	backend ContractBackend

	latestHasCode  uint32 // Cached verification that the latest state contains code for this contract
	pendingHasCode uint32 // Cached verification that the pending state contains code for this contract

	subscriptions map[chan<- vm.Log]ethereum.Subscription
	errors        map[chan<- vm.Log]error
}

// NewBoundContract creates a low level contract interface through which calls
// and transactions may be made through.
func NewBoundContract(address common.Address, abi abi.ABI, backend ContractBackend) *BoundContract {
	return &BoundContract{
		address: address,
		abi:     abi,
		backend: backend,
	}
}

// DeployContract deploys a contract onto the Ethereum blockchain and binds the
// deployment address with a Go wrapper.
func DeployContract(opts *TransactOpts, abi abi.ABI, bytecode []byte, backend ContractBackend, params ...interface{}) (common.Address, *types.Transaction, *BoundContract, error) {
	// Otherwise try to deploy the contract
	c := NewBoundContract(common.Address{}, abi, backend)

	input, err := c.abi.Pack("", params...)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	tx, err := c.transact(opts, nil, append(bytecode, input...))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	c.address = crypto.CreateAddress(opts.From, tx.Nonce())
	return c.address, tx, c, nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (c *BoundContract) Call(opts *CallOpts, result interface{}, method string, params ...interface{}) error {
	// Don't crash on a lazy user
	if opts == nil {
		opts = new(CallOpts)
	}
	// Make sure we have a contract to operate on, and bail out otherwise
	var code []byte
	var err error
	if opts.Pending && atomic.LoadUint32(&c.pendingHasCode) == 0 {
		code, err = c.backend.PendingCodeAt(opts.Context, c.address)
	} else if !opts.Pending && atomic.LoadUint32(&c.latestHasCode) == 0 {
		code, err = c.backend.CodeAt(opts.Context, c.address, nil)
	}
	if err != nil {
		return err
	} else if len(code) == 0 {
		return ErrNoCode
	}
	if opts.Pending {
		atomic.StoreUint32(&c.pendingHasCode, 1)
	} else {
		atomic.StoreUint32(&c.latestHasCode, 1)
	}
	// Pack the input, call and unpack the results
	input, err := c.abi.Pack(method, params...)
	if err != nil {
		return err
	}
	// Create the call message we use to make the call
	callMsg := ethereum.CallMsg{
		To:   c.address,
		Data: input,
	}
	var output []byte
	// Call pending or not depending on options
	if opts.Pending {
		output, err = c.backend.PendingCallContract(opts.Context, callMsg)
	} else {
		output, err = c.backend.CallContract(opts.Context, callMsg, nil)
	}
	if err != nil {
		return err
	}
	return c.abi.Unpack(result, method, output)
}

// Transact invokes the (paid) contract method with params as input values.
func (c *BoundContract) Transact(opts *TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	// Otherwise pack up the parameters and invoke the contract
	input, err := c.abi.Pack(method, params...)
	if err != nil {
		return nil, err
	}
	return c.transact(opts, &c.address, input)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (c *BoundContract) Transfer(opts *TransactOpts) (*types.Transaction, error) {
	return c.transact(opts, &c.address, nil)
}

// transact executes an actual transaction invocation, first deriving any missing
// authorization fields, and then scheduling the transaction for execution.
func (c *BoundContract) transact(opts *TransactOpts, contract *common.Address, input []byte) (*types.Transaction, error) {
	var err error

	// Ensure a valid value field and resolve the account nonce
	value := opts.Value
	if value == nil {
		value = new(big.Int)
	}
	nonce := uint64(0)
	if opts.Nonce == nil {
		nonce, err = c.backend.PendingNonceAt(opts.Context, opts.From)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve account nonce: %v", err)
		}
	} else {
		nonce = opts.Nonce.Uint64()
	}
	// Figure out the gas allowance and gas price values
	gasPrice := opts.GasPrice
	if gasPrice == nil {
		gasPrice, err = c.backend.SuggestGasPrice(opts.Context)
		if err != nil {
			return nil, fmt.Errorf("failed to suggest gas price: %v", err)
		}
	}
	gasLimit := opts.GasLimit
	if gasLimit == nil {
		// Gas estimation cannot succeed without code for method invocations
		if contract != nil && atomic.LoadUint32(&c.pendingHasCode) == 0 {
			var code []byte
			if code, err = c.backend.CodeAt(opts.Context, c.address, nil); err != nil {
				return nil, err
			} else if len(code) == 0 {
				return nil, ErrNoCode
			}
			atomic.StoreUint32(&c.pendingHasCode, 1)
		}
		// If the contract surely has code (or code is not needed), estimate the transaction
		callMsg := ethereum.CallMsg{
			From:     opts.From,
			To:       *contract,
			GasPrice: gasPrice,
			Value:    value,
			Data:     input,
		}
		gasLimit, err = c.backend.EstimateGas(opts.Context, callMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas needed: %v", err)
		}
	}
	// Create the transaction, sign it and schedule it for execution
	var rawTx *types.Transaction
	if contract == nil {
		rawTx = types.NewContractCreation(nonce, value, gasLimit, gasPrice, input)
	} else {
		rawTx = types.NewTransaction(nonce, c.address, value, gasLimit, gasPrice, input)
	}
	if opts.Signer == nil {
		return nil, errors.New("no signer to authorize the transaction with")
	}
	signedTx, err := opts.Signer(opts.From, rawTx)
	if err != nil {
		return nil, err
	}
	if err := c.backend.SendTransaction(opts.Context, signedTx); err != nil {
		return nil, err
	}
	return signedTx, nil
}

// Events will return a list of events for this contract for the given topics and
// the given start & end blocks.
func (c *BoundContract) Events(opts *EventOpts, name string, output interface{}, topics ...[]common.Hash) error {

	// get the event so we can encode the name into the first topic
	event, ok := c.abi.Events[name]
	if !ok {
		return fmt.Errorf("unknown event name: %v", name)
	}
	names := []common.Hash{event.Id()}
	topics = append([][]common.Hash{names}, topics...)

	// check that output is a pointer
	ptr := reflect.ValueOf(output)
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("need pointer to slice as output, have %T", output)
	}

	// check that it points to a slice
	slice := ptr.Elem()
	if slice.Kind() != reflect.Slice {
		return fmt.Errorf("need pointer to slice as output, have %T", output)
	}

	// create the filter query and retrieve the events through the backend
	filterQuery := ethereum.FilterQuery{
		FromBlock: opts.FromBlock,
		ToBlock:   opts.ToBlock,
		Addresses: []common.Address{c.address},
		Topics:    topics,
	}
	logs, err := c.backend.FilterLogs(opts.Context, filterQuery)
	if err != nil {
		return fmt.Errorf("could not retrieve events (%v)", err)
	}

	// for each log entry, create an event, unpack the data and append it
	item := slice.Elem()
	events := reflect.MakeSlice(slice.Type(), 0, len(logs))
	for _, log := range logs {
		event := reflect.New(item.Type())
		err = c.abi.Unpack(item.Interface(), name, log.Data)
		if err != nil {
			return fmt.Errorf("could not unpack event data (%v)", err)
		}
		if len(log.Topics) > 1 {
			// TODO: extract indexed parameters from topics 1-3
		}
		events = reflect.Append(events, event)
	}

	// set the output slice to our slice of events
	slice.Set(events)
	return nil
}

// Subscribe subscribes to the contract for the given topics. Subscription events
// will be submitted to the provided channel. Upon cancelation of the subscription,
// the channel will be closed. A channel should only be used for ones subscription.
func (c *BoundContract) Subscribe(opts *SubscribeOpts, name string, channel chan<- vm.Log, topics ...[]common.Hash) error {

	// check if we already have a subscription on this channel
	_, ok := c.subscriptions[channel]
	if ok {
		return fmt.Errorf("cannot reuse channel for multiple subscriptions")
	}

	// Build the filter query with our desired options and topics
	filterQuery := ethereum.FilterQuery{
		FromBlock: opts.FromBlock,
		ToBlock:   nil,
		Addresses: []common.Address{c.address},
		Topics:    topics,
	}

	// If we don't want to miss any real-time event logs, we need to subscribe right away
	tempChannel := make(chan vm.Log)
	tempSub, err := c.backend.SubscribeFilterLogs(opts.Context, filterQuery, tempChannel)
	if err != nil {
		return fmt.Errorf("could not create subscription (%v)", err)
	}

	// Get the desired historical data and feed it into the subscription channel
	if opts.History {
		var logs []vm.Log
		logs, err = c.backend.FilterLogs(opts.Context, filterQuery)
		if err != nil {
			return fmt.Errorf("could not retrieve history of subscription events")
		}
		for _, log := range logs {
			channel <- log
		}
	}

	// Feed the real-time events that happenend in the meantime to the subscription channel
	// and create the new subscription as soon as none are left
	var subscription ethereum.Subscription
Feed:
	for {
		select {
		case log := <-tempChannel:
			channel <- log
		default:
			subscription, err = c.backend.SubscribeFilterLogs(opts.Context, filterQuery, channel)
			tempSub.Unsubscribe()
			for _ = range tempSub.Err() {
			}
		Drain:
			for {
				select {
				case <-tempChannel:
				default:
					break Drain
				}
			}
			close(tempChannel)
			break Feed
		}
	}

	// check if the subscription succeeded
	if err != nil {
		return fmt.Errorf("could not create subscription (%v)", err)
	}
	c.subscriptions[channel] = subscription

	// Read from the error channel in the background
	// When we receive an error, we save it with the channel index
	// When the error channel is closed, it indicates an ended subscription
	// We then close the subscription channel to notify the consumer and remove
	// the subscription from our map
	go func(errChannel <-chan error) {
		for err := range errChannel {
			c.errors[channel] = err
		}
		close(channel)
		delete(c.subscriptions, channel)
	}(subscription.Err())
	return nil
}

// Error returns the error that was encountered for the subscription on the given
// channel. It will reset the error upon retrieval.
func (c *BoundContract) Error(channel chan<- vm.Log) error {
	err := c.errors[channel]
	delete(c.errors, channel)
	return err
}

// Unsubscribe cancels the subscription if one exists on the given channel. The
// subscription channel will be closed once the subscription was successfully
// canceled.
func (c *BoundContract) Unsubscribe(channel chan<- vm.Log) error {
	subscription, ok := c.subscriptions[channel]
	if !ok {
		return fmt.Errorf("subscription doesn't exist or already closed")
	}
	subscription.Unsubscribe()
	return nil
}

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

	"github.com/barakmich/glog"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
)

// GasOracleFn is a gas price oracle function callback to request the suggestion
// of an estimated gas price that should be used to execute a paid transaction.
type GasOracleFn func() *big.Int

// MinerStateFn is a callback method to retrieve the currently pending state
// according ti our local mining instance.
type MinerStateFn func() (*types.Block, *state.StateDB)

// TxScheduleFn is a callback method to schedule a transaction for execution.
type TxScheduleFn func(*types.Transaction) error

// ContractOpts is the set of contract parameters that can be used to fine tune
// behavior tailoring to a specific use case.
type ContractOpts struct {
	Database   ethdb.Database // Chain and state database needed to access past logs
	EventMux   *event.TypeMux // Event multiplexer to publish log events merged with others
	GasOracle  GasOracleFn    // Gas price oracle to allow not specifying transaction prices
	MinerState MinerStateFn   // Pending state retriever to allow nonce and gas limit estimation
	TxSchedule TxScheduleFn   // Transaction scheduler to inject a transaction into the pool
}

// SignerFn is a signer function callback when a contract requires a method to
// sign the transaction before submission.
type SignerFn func(common.Address, *types.Transaction) (*types.Transaction, error)

// AuthOpts is the authorization data required to create a valid Ethereum transaction.
type AuthOpts struct {
	Account common.Address // Ethereum account to send the transaction from
	Nonce   *big.Int       // Nonce to use for the transaction execution (nil = use pending state)
	Signer  SignerFn       // Method to use for signing the transaction (mandatory)

	Value    *big.Int // Funds to transfer along along the transaction (nil = 0 = no funds)
	GasPrice *big.Int // Gas price to use for the transaction execution (nil = gas price oracle)
	GasLimit *big.Int // Gas limit to set for the transaction execution (nil = estimate + 10%)
}

// BoundContract is the base wrapper object that reflects a contract on the
// Ethereum network. It contains a collection of methods that are used by the
// higher level contract bindings to operate.
type BoundContract struct {
	address common.Address // Deployment address of the contract on the Ethereum blockchain
	abi     abi.ABI        // Reflect based ABI to access the correct Ethereum methods

	blockchain *core.BlockChain      // Ethereum blockchain to use for state retrieval
	options    *ContractOpts         // Options fine tuning contract behaviour
	filters    *filters.FilterSystem // Filter system to handle the contract events
}

// NewBoundContract initialises a new ABI and returns the contract. It does not
// deploy the contract, hence the name.
func NewBoundContract(address common.Address, abi abi.ABI, blockchain *core.BlockChain, opts ContractOpts) *BoundContract {
	// Initialize any needed values for the contract options
	if opts.EventMux == nil {
		opts.EventMux = new(event.TypeMux)
	}
	// Create and return the contract base
	return &BoundContract{
		address:    address,
		abi:        abi,
		blockchain: blockchain,
		options:    &opts,
		filters:    filters.NewFilterSystem(opts.EventMux),
	}
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (c *BoundContract) Call(result interface{}, method string, params ...interface{}) error {
	return c.abi.Call(c.execute, result, method, params...)
}

// execute runs the contract code for the given input value and returns the output.
func (c *BoundContract) execute(input []byte) []byte {
	state, _ := c.blockchain.State()

	output, err := runtime.Call(c.address, input, &runtime.Config{
		GetHashFn: core.GetHashFn(c.blockchain.CurrentBlock().ParentHash(), c.blockchain),
		State:     state,
	})
	if err != nil {
		glog.V(logger.Warn).Infof("contract call failed: %v", err)
		return nil
	}
	return output
}

// Transact invokes the (paid) contract method with params as input values and
// value as the fund transfer to the contract.
func (c *BoundContract) Transact(opts *AuthOpts, method string, params ...interface{}) (*types.Transaction, error) {
	// Pack up the method and arguments into an input data blob
	input, err := c.abi.Pack(method, params...)
	if err != nil {
		return nil, err
	}
	// Ensure a valid value field and resolve the account nonce
	value := opts.Value
	if value == nil {
		value = new(big.Int)
	}
	nonce := opts.Nonce
	if nonce == nil {
		if c.options.MinerState == nil {
			return nil, errors.New("account nonce nil and no miner state retriever specified to estimate")
		}
		_, statedb := c.options.MinerState()
		if statedb == nil {
			return nil, errors.New("miner state retriever returned nil")
		}
		nonce = new(big.Int).SetUint64(statedb.GetNonce(opts.Account))
	}
	// Figure out the gas allowance and gas price values
	gasPrice := opts.GasPrice
	if gasPrice == nil {
		if c.options.GasOracle == nil {
			return nil, errors.New("gas price nil and no price oracle set")
		}
		if gasPrice = c.options.GasOracle(); gasPrice == nil {
			return nil, errors.New("gas oracle suggested nil price")
		}
	}
	gasLimit := opts.GasLimit
	if gasLimit == nil {
		limit, err := c.estimate(opts.Account, value, gasPrice, input)
		if err != nil {
			return nil, fmt.Errorf("gas estimation failed: %v", err)
		}
		if gasLimit = limit; gasLimit == nil {
			return nil, errors.New("gas estimator suggested nil limit")
		}
	}
	// Create the transaction, sign it and schedule it for execution
	rawTx := types.NewTransaction(nonce.Uint64(), c.address, value, gasLimit, gasPrice, input)
	if opts.Signer == nil {
		return nil, errors.New("no signer to authorize the transaction with")
	}
	signedTx, err := opts.Signer(opts.Account, rawTx)
	if err != nil {
		return nil, err
	}
	if c.options.TxSchedule == nil {
		return nil, errors.New("no transaction scheduler configured")
	}
	if err := c.options.TxSchedule(signedTx); err != nil {
		return nil, err
	}
	return signedTx, nil
}

// estimate tries to calculate the approximate gas required by a transaction.
func (c *BoundContract) estimate(sender common.Address, value, price *big.Int, input []byte) (*big.Int, error) {
	// Create a copy of the current state db to screw around with
	if c.options.MinerState == nil {
		return nil, errors.New("no miner state retriever configured")
	}
	block, statedb := c.options.MinerState()
	if block == nil || statedb == nil {
		return nil, errors.New("pending miner state nil")
	}
	statedb = statedb.Copy()

	// Set infinite balance to the sender account
	from := statedb.GetOrNewStateObject(sender)
	from.SetBalance(common.MaxBig)

	// Assemble the call invocation to measure the gas usage
	msg := callmsg{
		from:     from,
		to:       &c.address,
		gasLimit: block.GasLimit(),
		gasPrice: price,
		value:    value,
		data:     input,
	}
	// Execute the call and return
	vmenv := core.NewEnv(statedb, c.blockchain, msg, block.Header())
	gaspool := new(core.GasPool).AddGas(common.MaxBig)

	_, gas, err := core.ApplyMessage(vmenv, msg, gaspool)
	return gas, err
}

// callmsg implements core.Message to allow passing it as a transaction simulator.
type callmsg struct {
	from     *state.StateObject
	to       *common.Address
	gasLimit *big.Int
	gasPrice *big.Int
	value    *big.Int
	data     []byte
}

func (m callmsg) From() (common.Address, error)         { return m.from.Address(), nil }
func (m callmsg) FromFrontier() (common.Address, error) { return m.from.Address(), nil }
func (m callmsg) Nonce() uint64                         { return m.from.Nonce() }
func (m callmsg) To() *common.Address                   { return m.to }
func (m callmsg) GasPrice() *big.Int                    { return m.gasPrice }
func (m callmsg) Gas() *big.Int                         { return m.gasLimit }
func (m callmsg) Value() *big.Int                       { return m.value }
func (m callmsg) Data() []byte                          { return m.data }

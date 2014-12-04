package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/vm"
)

/*
 * The State transitioning model
 *
 * A state transition is a change made when a transaction is applied to the current world state
 * The state transitioning model does all all the necessary work to work out a valid new state root.
 * 1) Nonce handling
 * 2) Pre pay / buy gas of the coinbase (miner)
 * 3) Create a new state object if the recipient is \0*32
 * 4) Value transfer
 * == If contract creation ==
 * 4a) Attempt to run transaction data
 * 4b) If valid, use result as code for the new state object
 * == end ==
 * 5) Run Script section
 * 6) Derive new state root
 */
type StateTransition struct {
	coinbase, receiver []byte
	tx                 *types.Transaction
	gas, gasPrice      *big.Int
	value              *big.Int
	data               []byte
	state              *state.StateDB
	block              *types.Block

	cb, rec, sen *state.StateObject
}

func NewStateTransition(coinbase *state.StateObject, tx *types.Transaction, state *state.StateDB, block *types.Block) *StateTransition {
	return &StateTransition{coinbase.Address(), tx.Recipient, tx, new(big.Int), new(big.Int).Set(tx.GasPrice), tx.Value, tx.Data, state, block, coinbase, nil, nil}
}

func (self *StateTransition) Coinbase() *state.StateObject {
	if self.cb != nil {
		return self.cb
	}

	self.cb = self.state.GetOrNewStateObject(self.coinbase)
	return self.cb
}
func (self *StateTransition) Sender() *state.StateObject {
	if self.sen != nil {
		return self.sen
	}

	self.sen = self.state.GetOrNewStateObject(self.tx.Sender())

	return self.sen
}
func (self *StateTransition) Receiver() *state.StateObject {
	if self.tx != nil && self.tx.CreatesContract() {
		return nil
	}

	if self.rec != nil {
		return self.rec
	}

	self.rec = self.state.GetOrNewStateObject(self.tx.Recipient)
	return self.rec
}

func (self *StateTransition) UseGas(amount *big.Int) error {
	if self.gas.Cmp(amount) < 0 {
		return OutOfGasError()
	}
	self.gas.Sub(self.gas, amount)

	return nil
}

func (self *StateTransition) AddGas(amount *big.Int) {
	self.gas.Add(self.gas, amount)
}

func (self *StateTransition) BuyGas() error {
	var err error

	sender := self.Sender()
	if sender.Balance().Cmp(self.tx.GasValue()) < 0 {
		return fmt.Errorf("Insufficient funds to pre-pay gas. Req %v, has %v", self.tx.GasValue(), sender.Balance())
	}

	coinbase := self.Coinbase()
	err = coinbase.BuyGas(self.tx.Gas, self.tx.GasPrice)
	if err != nil {
		return err
	}

	self.AddGas(self.tx.Gas)
	sender.SubAmount(self.tx.GasValue())

	return nil
}

func (self *StateTransition) RefundGas() {
	coinbase, sender := self.Coinbase(), self.Sender()
	coinbase.RefundGas(self.gas, self.tx.GasPrice)

	// Return remaining gas
	remaining := new(big.Int).Mul(self.gas, self.tx.GasPrice)
	sender.AddAmount(remaining)
}

func (self *StateTransition) preCheck() (err error) {
	var (
		tx     = self.tx
		sender = self.Sender()
	)

	// Make sure this transaction's nonce is correct
	if sender.Nonce != tx.Nonce {
		return NonceError(tx.Nonce, sender.Nonce)
	}

	// Pre-pay gas / Buy gas of the coinbase account
	if err = self.BuyGas(); err != nil {
		return err
	}

	return nil
}

func (self *StateTransition) TransitionState() (err error) {
	statelogger.Debugf("(~) %x\n", self.tx.Hash())

	// XXX Transactions after this point are considered valid.
	if err = self.preCheck(); err != nil {
		return
	}

	var (
		tx     = self.tx
		sender = self.Sender()
	)

	defer self.RefundGas()

	// Increment the nonce for the next transaction
	sender.Nonce += 1

	// Transaction gas
	if err = self.UseGas(vm.GasTx); err != nil {
		return
	}

	// Pay data gas
	var dgas int64
	for _, byt := range self.data {
		if byt != 0 {
			dgas += vm.GasData.Int64()
		} else {
			dgas += 1 // This is 1/5. If GasData changes this fails
		}
	}
	if err = self.UseGas(big.NewInt(dgas)); err != nil {
		return
	}

	var ret []byte
	vmenv := NewEnv(self.state, self.tx, self.block)
	var ref vm.ClosureRef
	if tx.CreatesContract() {
		self.rec = MakeContract(tx, self.state)

		ret, err, ref = vmenv.Create(sender, self.rec.Address(), self.tx.Data, self.gas, self.gasPrice, self.value)
		ref.SetCode(ret)
	} else {
		ret, err = vmenv.Call(self.Sender(), self.Receiver().Address(), self.tx.Data, self.gas, self.gasPrice, self.value)
	}
	if err != nil {
		statelogger.Debugln(err)
	}

	return
}

// Converts an transaction in to a state object
func MakeContract(tx *types.Transaction, state *state.StateDB) *state.StateObject {
	addr := tx.CreationAddress(state)

	contract := state.GetOrNewStateObject(addr)
	contract.InitCode = tx.Data

	return contract
}

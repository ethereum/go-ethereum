package ethchain

import (
	"fmt"
	"math/big"

	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethtrie"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethvm"
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
	tx                 *Transaction
	gas, gasPrice      *big.Int
	value              *big.Int
	data               []byte
	state              *ethstate.State
	block              *Block

	cb, rec, sen *ethstate.StateObject
}

func NewStateTransition(coinbase *ethstate.StateObject, tx *Transaction, state *ethstate.State, block *Block) *StateTransition {
	return &StateTransition{coinbase.Address(), tx.Recipient, tx, new(big.Int), new(big.Int).Set(tx.GasPrice), tx.Value, tx.Data, state, block, coinbase, nil, nil}
}

func (self *StateTransition) Coinbase() *ethstate.StateObject {
	if self.cb != nil {
		return self.cb
	}

	self.cb = self.state.GetOrNewStateObject(self.coinbase)
	return self.cb
}
func (self *StateTransition) Sender() *ethstate.StateObject {
	if self.sen != nil {
		return self.sen
	}

	self.sen = self.state.GetOrNewStateObject(self.tx.Sender())

	return self.sen
}
func (self *StateTransition) Receiver() *ethstate.StateObject {
	if self.tx != nil && self.tx.CreatesContract() {
		return nil
	}

	if self.rec != nil {
		return self.rec
	}

	self.rec = self.state.GetOrNewStateObject(self.tx.Recipient)
	return self.rec
}

func (self *StateTransition) MakeStateObject(state *ethstate.State, tx *Transaction) *ethstate.StateObject {
	contract := MakeContract(tx, state)

	return contract
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
	if sender.Balance.Cmp(self.tx.GasValue()) < 0 {
		return fmt.Errorf("Insufficient funds to pre-pay gas. Req %v, has %v", self.tx.GasValue(), sender.Balance)
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

	defer func() {
		if r := recover(); r != nil {
			statelogger.Infoln(r)
			err = fmt.Errorf("state transition err %v", r)
		}
	}()

	// XXX Transactions after this point are considered valid.
	if err = self.preCheck(); err != nil {
		return
	}

	var (
		tx       = self.tx
		sender   = self.Sender()
		receiver *ethstate.StateObject
	)

	defer self.RefundGas()

	// Increment the nonce for the next transaction
	sender.Nonce += 1

	// Transaction gas
	if err = self.UseGas(ethvm.GasTx); err != nil {
		return
	}

	// Pay data gas
	dataPrice := big.NewInt(int64(len(self.data)))
	dataPrice.Mul(dataPrice, ethvm.GasData)
	if err = self.UseGas(dataPrice); err != nil {
		return
	}

	if sender.Balance.Cmp(self.value) < 0 {
		return fmt.Errorf("Insufficient funds to transfer value. Req %v, has %v", self.value, sender.Balance)
	}

	var snapshot *ethstate.State
	// If the receiver is nil it's a contract (\0*32).
	if tx.CreatesContract() {
		// Subtract the (irreversible) amount from the senders account
		sender.SubAmount(self.value)

		snapshot = self.state.Copy()

		// Create a new state object for the contract
		receiver = self.MakeStateObject(self.state, tx)
		self.rec = receiver
		if receiver == nil {
			return fmt.Errorf("Unable to create contract")
		}

		// Add the amount to receivers account which should conclude this transaction
		receiver.AddAmount(self.value)
	} else {
		receiver = self.Receiver()

		// Subtract the amount from the senders account
		sender.SubAmount(self.value)
		// Add the amount to receivers account which should conclude this transaction
		receiver.AddAmount(self.value)

		snapshot = self.state.Copy()
	}

	msg := self.state.Manifest().AddMessage(&ethstate.Message{
		To: receiver.Address(), From: sender.Address(),
		Input:  self.tx.Data,
		Origin: sender.Address(),
		Block:  self.block.Hash(), Timestamp: self.block.Time, Coinbase: self.block.Coinbase, Number: self.block.Number,
		Value: self.value,
	})

	// Process the init code and create 'valid' contract
	if IsContractAddr(self.receiver) {
		// Evaluate the initialization script
		// and use the return value as the
		// script section for the state object.
		self.data = nil

		code, err := self.Eval(msg, receiver.Init(), receiver, "init")
		if err != nil {
			self.state.Set(snapshot)

			return fmt.Errorf("Error during init execution %v", err)
		}

		receiver.Code = code
		msg.Output = code
	} else {
		if len(receiver.Code) > 0 {
			ret, err := self.Eval(msg, receiver.Code, receiver, "code")
			if err != nil {
				self.state.Set(snapshot)

				return fmt.Errorf("Error during code execution %v", err)
			}

			msg.Output = ret
		}
	}

	return
}

func (self *StateTransition) transferValue(sender, receiver *ethstate.StateObject) error {
	if sender.Balance.Cmp(self.value) < 0 {
		return fmt.Errorf("Insufficient funds to transfer value. Req %v, has %v", self.value, sender.Balance)
	}

	// Subtract the amount from the senders account
	sender.SubAmount(self.value)
	// Add the amount to receivers account which should conclude this transaction
	receiver.AddAmount(self.value)

	return nil
}

func (self *StateTransition) Eval(msg *ethstate.Message, script []byte, context *ethstate.StateObject, typ string) (ret []byte, err error) {
	var (
		transactor    = self.Sender()
		state         = self.state
		env           = NewEnv(state, self.tx, self.block)
		callerClosure = ethvm.NewClosure(msg, transactor, context, script, self.gas, self.gasPrice)
	)

	vm := ethvm.New(env)
	vm.Verbose = true
	vm.Fn = typ

	ret, _, err = callerClosure.Call(vm, self.tx.Data)

	if err == nil {
		// Execute POSTs
		for e := vm.Queue().Front(); e != nil; e = e.Next() {
			msg := e.Value.(*ethvm.Message)

			msg.Exec(msg.Addr(), transactor)
		}
	}

	return
}

// Converts an transaction in to a state object
func MakeContract(tx *Transaction, state *ethstate.State) *ethstate.StateObject {
	// Create contract if there's no recipient
	if tx.IsContract() {
		addr := tx.CreationAddress(state)

		contract := state.GetOrNewStateObject(addr)
		contract.InitCode = tx.Data
		contract.State = ethstate.New(ethtrie.New(ethutil.Config.Db, ""))

		return contract
	}

	return nil
}

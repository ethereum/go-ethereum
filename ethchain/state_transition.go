package ethchain

import (
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
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
	coinbase []byte
	tx       *Transaction
	gas      *big.Int
	state    *State
	block    *Block

	cb, rec, sen *StateObject
}

func NewStateTransition(coinbase []byte, tx *Transaction, state *State, block *Block) *StateTransition {
	return &StateTransition{coinbase, tx, new(big.Int), state, block, nil, nil, nil}
}

func (self *StateTransition) Coinbase() *StateObject {
	if self.cb != nil {
		return self.cb
	}

	self.cb = self.state.GetAccount(self.coinbase)
	return self.cb
}
func (self *StateTransition) Sender() *StateObject {
	if self.sen != nil {
		return self.sen
	}

	self.sen = self.state.GetAccount(self.tx.Sender())
	return self.sen
}
func (self *StateTransition) Receiver() *StateObject {
	if self.tx.CreatesContract() {
		return nil
	}

	if self.rec != nil {
		return self.rec
	}

	self.rec = self.state.GetAccount(self.tx.Recipient)
	return self.rec
}

func (self *StateTransition) MakeStateObject(state *State, tx *Transaction) *StateObject {
	contract := MakeContract(tx, state)
	if contract != nil {
		state.states[string(tx.CreationAddress())] = contract.state

		return contract
	}

	return nil
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
	if sender.Amount.Cmp(self.tx.GasValue()) < 0 {
		return fmt.Errorf("Insufficient funds to pre-pay gas. Req %v, has %v", self.tx.GasValue(), self.tx.Value)
	}

	coinbase := self.Coinbase()
	err = coinbase.BuyGas(self.tx.Gas, self.tx.GasPrice)
	if err != nil {
		return err
	}
	self.state.UpdateStateObject(coinbase)

	self.AddGas(self.tx.Gas)
	sender.SubAmount(self.tx.GasValue())

	return nil
}

func (self *StateTransition) TransitionState() (err error) {
	//snapshot := st.state.Snapshot()

	defer func() {
		if r := recover(); r != nil {
			ethutil.Config.Log.Infoln(r)
			err = fmt.Errorf("state transition err %v", r)
		}
	}()

	var (
		tx       = self.tx
		sender   = self.Sender()
		receiver *StateObject
	)

	// Make sure this transaction's nonce is correct
	if sender.Nonce != tx.Nonce {
		return NonceError(tx.Nonce, sender.Nonce)
	}

	// Increment the nonce for the next transaction
	sender.Nonce += 1

	// Pre-pay gas / Buy gas of the coinbase account
	if err = self.BuyGas(); err != nil {
		return err
	}

	// Get the receiver (TODO fix this, if coinbase is the receiver we need to save/retrieve)
	receiver = self.Receiver()

	// Transaction gas
	if err = self.UseGas(GasTx); err != nil {
		return err
	}

	// Pay data gas
	dataPrice := big.NewInt(int64(len(tx.Data)))
	dataPrice.Mul(dataPrice, GasData)
	if err = self.UseGas(dataPrice); err != nil {
		return err
	}

	// If the receiver is nil it's a contract (\0*32).
	if receiver == nil {
		// Create a new state object for the contract
		receiver = self.MakeStateObject(self.state, tx)
		if receiver == nil {
			return fmt.Errorf("ERR. Unable to create contract with transaction %v", tx)
		}
	}

	// Transfer value from sender to receiver
	if err = self.transferValue(sender, receiver); err != nil {
		return err
	}

	// Process the init code and create 'valid' contract
	if tx.CreatesContract() {
		//fmt.Println(Disassemble(receiver.Init()))
		// Evaluate the initialization script
		// and use the return value as the
		// script section for the state object.
		//script, gas, err = sm.Eval(state, contract.Init(), contract, tx, block)
		code, err := self.Eval(receiver.Init(), receiver)
		if err != nil {
			return fmt.Errorf("Error during init script run %v", err)
		}

		receiver.script = code
	}

	// Return remaining gas
	remaining := new(big.Int).Mul(self.gas, tx.GasPrice)
	sender.AddAmount(remaining)

	self.state.UpdateStateObject(sender)
	self.state.UpdateStateObject(receiver)

	return nil
}

func (self *StateTransition) transferValue(sender, receiver *StateObject) error {
	if sender.Amount.Cmp(self.tx.Value) < 0 {
		return fmt.Errorf("Insufficient funds to transfer value. Req %v, has %v", self.tx.Value, sender.Amount)
	}

	if self.tx.Value.Cmp(ethutil.Big0) > 0 {
		// Subtract the amount from the senders account
		sender.SubAmount(self.tx.Value)
		// Add the amount to receivers account which should conclude this transaction
		receiver.AddAmount(self.tx.Value)

		ethutil.Config.Log.Debugf("%x => %x (%v) %x\n", sender.Address()[:4], receiver.Address()[:4], self.tx.Value, self.tx.Hash())
	}

	return nil
}

func (self *StateTransition) Eval(script []byte, context *StateObject) (ret []byte, err error) {
	var (
		tx        = self.tx
		block     = self.block
		initiator = self.Sender()
		state     = self.state
	)

	closure := NewClosure(initiator, context, script, state, self.gas, tx.GasPrice)
	vm := NewVm(state, nil, RuntimeVars{
		Origin:      initiator.Address(),
		BlockNumber: block.BlockInfo().Number,
		PrevHash:    block.PrevHash,
		Coinbase:    block.Coinbase,
		Time:        block.Time,
		Diff:        block.Difficulty,
		Value:       tx.Value,
	})
	ret, _, err = closure.Call(vm, tx.Data, nil)

	return
}

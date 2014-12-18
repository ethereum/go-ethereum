package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
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
	msg                Message
	gas, gasPrice      *big.Int
	value              *big.Int
	data               []byte
	state              *state.StateDB
	block              *types.Block

	cb, rec, sen *state.StateObject

	Env vm.Environment
}

type Message interface {
	Hash() []byte

	From() []byte
	To() []byte

	GasValue() *big.Int
	GasPrice() *big.Int
	Gas() *big.Int
	Value() *big.Int

	Nonce() uint64
	Data() []byte
}

func AddressFromMessage(msg Message) []byte {
	// Generate a new address
	return crypto.Sha3(ethutil.NewValue([]interface{}{msg.From(), msg.Nonce()}).Encode())[12:]
}

func MessageCreatesContract(msg Message) bool {
	return len(msg.To()) == 0
}

func NewStateTransition(coinbase *state.StateObject, msg Message, state *state.StateDB, block *types.Block) *StateTransition {
	return &StateTransition{coinbase.Address(), msg.To(), msg, new(big.Int), new(big.Int).Set(msg.GasPrice()), msg.Value(), msg.Data(), state, block, coinbase, nil, nil, nil}
}

func (self *StateTransition) VmEnv() vm.Environment {
	if self.Env == nil {
		self.Env = NewEnv(self.state, self.msg, self.block)
	}

	return self.Env
}

func (self *StateTransition) Coinbase() *state.StateObject {
	if self.cb != nil {
		return self.cb
	}

	self.cb = self.state.GetOrNewStateObject(self.coinbase)
	return self.cb
}
func (self *StateTransition) From() *state.StateObject {
	if self.sen != nil {
		return self.sen
	}

	self.sen = self.state.GetOrNewStateObject(self.msg.From())

	return self.sen
}
func (self *StateTransition) To() *state.StateObject {
	if self.msg != nil && MessageCreatesContract(self.msg) {
		return nil
	}

	if self.rec != nil {
		return self.rec
	}

	self.rec = self.state.GetOrNewStateObject(self.msg.To())
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

	sender := self.From()
	if sender.Balance().Cmp(self.msg.GasValue()) < 0 {
		return fmt.Errorf("Insufficient funds to pre-pay gas. Req %v, has %v", self.msg.GasValue(), sender.Balance())
	}

	coinbase := self.Coinbase()
	err = coinbase.BuyGas(self.msg.Gas(), self.msg.GasPrice())
	if err != nil {
		return err
	}

	self.AddGas(self.msg.Gas())
	sender.SubAmount(self.msg.GasValue())

	return nil
}

func (self *StateTransition) RefundGas() {
	coinbase, sender := self.Coinbase(), self.From()
	coinbase.RefundGas(self.gas, self.msg.GasPrice())

	// Return remaining gas
	remaining := new(big.Int).Mul(self.gas, self.msg.GasPrice())
	sender.AddAmount(remaining)
}

func (self *StateTransition) preCheck() (err error) {
	var (
		msg    = self.msg
		sender = self.From()
	)

	// Make sure this transaction's nonce is correct
	if sender.Nonce != msg.Nonce() {
		return NonceError(msg.Nonce(), sender.Nonce)
	}

	// Pre-pay gas / Buy gas of the coinbase account
	if err = self.BuyGas(); err != nil {
		return err
	}

	return nil
}

func (self *StateTransition) TransitionState() (err error) {
	statelogger.Debugf("(~) %x\n", self.msg.Hash())

	// XXX Transactions after this point are considered valid.
	if err = self.preCheck(); err != nil {
		return
	}

	var (
		msg    = self.msg
		sender = self.From()
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
	vmenv := self.VmEnv()
	var ref vm.ClosureRef
	if MessageCreatesContract(msg) {
		self.rec = MakeContract(msg, self.state)

		ret, err, ref = vmenv.Create(sender, self.rec.Address(), self.msg.Data(), self.gas, self.gasPrice, self.value)
		ref.SetCode(ret)
	} else {
		ret, err = vmenv.Call(self.From(), self.To().Address(), self.msg.Data(), self.gas, self.gasPrice, self.value)
	}
	if err != nil {
		statelogger.Debugln(err)
	}

	return
}

// Converts an transaction in to a state object
func MakeContract(msg Message, state *state.StateDB) *state.StateObject {
	addr := AddressFromMessage(msg)

	contract := state.GetOrNewStateObject(addr)
	contract.InitCode = msg.Data()

	return contract
}

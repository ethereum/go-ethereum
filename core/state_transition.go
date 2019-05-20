package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/vm"
)

const tryJit = false

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
	coinbase      []byte
	msg           Message
	gas, gasPrice *big.Int
	initialGas    *big.Int
	value         *big.Int
	data          []byte
	state         *state.StateDB

	cb, rec, sen *state.StateObject

	env vm.Environment
}

type Message interface {
	Hash() []byte

	From() []byte
	To() []byte

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

func MessageGasValue(msg Message) *big.Int {
	return new(big.Int).Mul(msg.Gas(), msg.GasPrice())
}

func NewStateTransition(env vm.Environment, msg Message, coinbase *state.StateObject) *StateTransition {
	return &StateTransition{
		coinbase:   coinbase.Address(),
		env:        env,
		msg:        msg,
		gas:        new(big.Int),
		gasPrice:   new(big.Int).Set(msg.GasPrice()),
		initialGas: new(big.Int),
		value:      msg.Value(),
		data:       msg.Data(),
		state:      env.State(),
		cb:         coinbase,
	}
}

func (self *StateTransition) Coinbase() *state.StateObject {
	return self.state.GetOrNewStateObject(self.coinbase)
}
func (self *StateTransition) From() *state.StateObject {
	return self.state.GetOrNewStateObject(self.msg.From())
}
func (self *StateTransition) To() *state.StateObject {
	if self.msg != nil && MessageCreatesContract(self.msg) {
		return nil
	}
	return self.state.GetOrNewStateObject(self.msg.To())
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
	if sender.Balance().Cmp(MessageGasValue(self.msg)) < 0 {
		return fmt.Errorf("insufficient ETH for gas (%x). Req %v, has %v", sender.Address()[:4], MessageGasValue(self.msg), sender.Balance())
	}

	coinbase := self.Coinbase()
	err = coinbase.BuyGas(self.msg.Gas(), self.msg.GasPrice())
	if err != nil {
		return err
	}

	self.AddGas(self.msg.Gas())
	self.initialGas.Set(self.msg.Gas())
	sender.SubAmount(MessageGasValue(self.msg))

	return nil
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

func (self *StateTransition) TransitionState() (ret []byte, err error) {
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

	//stateCopy := self.env.State().Copy()
	vmenv := self.env
	var ref vm.ContextRef
	if MessageCreatesContract(msg) {
		contract := MakeContract(msg, self.state)
		ret, err, ref = vmenv.Create(sender, contract.Address(), self.msg.Data(), self.gas, self.gasPrice, self.value)
		if err == nil {
			dataGas := big.NewInt(int64(len(ret)))
			dataGas.Mul(dataGas, vm.GasCreateByte)
			if err := self.UseGas(dataGas); err == nil {
				ref.SetCode(ret)
			}
		}

		/*
			if vmenv, ok := vmenv.(*VMEnv); ok && tryJit {
				statelogger.Infof("CREATE: re-running using JIT (PH=%x)\n", stateCopy.Root()[:4])
				// re-run using the JIT (validation for the JIT)
				goodState := vmenv.State().Copy()
				vmenv.state = stateCopy
				vmenv.SetVmType(vm.JitVmTy)
				vmenv.Create(sender, contract.Address(), self.msg.Data(), self.gas, self.gasPrice, self.value)
				statelogger.Infof("DONE PH=%x STD_H=%x JIT_H=%x\n", stateCopy.Root()[:4], goodState.Root()[:4], vmenv.State().Root()[:4])
				self.state.Set(goodState)
			}
		*/
	} else {
		ret, err = vmenv.Call(self.From(), self.To().Address(), self.msg.Data(), self.gas, self.gasPrice, self.value)

		/*
			if vmenv, ok := vmenv.(*VMEnv); ok && tryJit {
				statelogger.Infof("CALL: re-running using JIT (PH=%x)\n", stateCopy.Root()[:4])
				// re-run using the JIT (validation for the JIT)
				goodState := vmenv.State().Copy()
				vmenv.state = stateCopy
				vmenv.SetVmType(vm.JitVmTy)
				vmenv.Call(self.From(), self.To().Address(), self.msg.Data(), self.gas, self.gasPrice, self.value)
				statelogger.Infof("DONE PH=%x STD_H=%x JIT_H=%x\n", stateCopy.Root()[:4], goodState.Root()[:4], vmenv.State().Root()[:4])
				self.state.Set(goodState)
			}
		*/
	}

	if err != nil {
		self.UseGas(self.gas)
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

func (self *StateTransition) RefundGas() {
	coinbase, sender := self.Coinbase(), self.From()
	// Return remaining gas
	remaining := new(big.Int).Mul(self.gas, self.msg.GasPrice())
	sender.AddAmount(remaining)

	uhalf := new(big.Int).Div(self.GasUsed(), ethutil.Big2)
	for addr, ref := range self.state.Refunds() {
		refund := ethutil.BigMin(uhalf, ref)
		self.gas.Add(self.gas, refund)
		self.state.AddBalance([]byte(addr), refund.Mul(refund, self.msg.GasPrice()))
	}

	coinbase.RefundGas(self.gas, self.msg.GasPrice())
}

func (self *StateTransition) GasUsed() *big.Int {
	return new(big.Int).Sub(self.initialGas, self.gas)
}

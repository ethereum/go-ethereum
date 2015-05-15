package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
)

const tryJit = false

var ()

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
	coinbase      common.Address
	msg           Message
	gas, gasPrice *big.Int
	initialGas    *big.Int
	value         *big.Int
	data          []byte
	state         *state.StateDB

	cb, rec, sen *state.StateObject

	env vm.Environment
}

// Message represents a message sent to a contract.
type Message interface {
	From() (common.Address, error)
	To() *common.Address

	GasPrice() *big.Int
	Gas() *big.Int
	Value() *big.Int

	Nonce() uint64
	Data() []byte
}

func AddressFromMessage(msg Message) common.Address {
	from, _ := msg.From()

	return crypto.CreateAddress(from, msg.Nonce())
}

func MessageCreatesContract(msg Message) bool {
	return msg.To() == nil
}

func MessageGasValue(msg Message) *big.Int {
	return new(big.Int).Mul(msg.Gas(), msg.GasPrice())
}

func IntrinsicGas(msg Message) *big.Int {
	igas := new(big.Int).Set(params.TxGas)
	for _, byt := range msg.Data() {
		if byt != 0 {
			igas.Add(igas, params.TxDataNonZeroGas)
		} else {
			igas.Add(igas, params.TxDataZeroGas)
		}
	}

	return igas
}

func ApplyMessage(env vm.Environment, msg Message, coinbase *state.StateObject) ([]byte, *big.Int, error) {
	return NewStateTransition(env, msg, coinbase).transitionState()
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
	f, _ := self.msg.From()
	return self.state.GetOrNewStateObject(f)
}
func (self *StateTransition) To() *state.StateObject {
	if self.msg == nil {
		return nil
	}
	to := self.msg.To()
	if to == nil {
		return nil // contract creation
	}
	return self.state.GetOrNewStateObject(*to)
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
		return fmt.Errorf("insufficient ETH for gas (%x). Req %v, has %v", sender.Address().Bytes()[:4], MessageGasValue(self.msg), sender.Balance())
	}

	coinbase := self.Coinbase()
	err = coinbase.BuyGas(self.msg.Gas(), self.msg.GasPrice())
	if err != nil {
		return err
	}

	self.AddGas(self.msg.Gas())
	self.initialGas.Set(self.msg.Gas())
	sender.SubBalance(MessageGasValue(self.msg))

	return nil
}

func (self *StateTransition) preCheck() (err error) {
	var (
		msg    = self.msg
		sender = self.From()
	)

	// Make sure this transaction's nonce is correct
	if sender.Nonce() != msg.Nonce() {
		return NonceError(msg.Nonce(), sender.Nonce())
	}

	// Pre-pay gas / Buy gas of the coinbase account
	if err = self.BuyGas(); err != nil {
		if state.IsGasLimitErr(err) {
			return err
		}
		return InvalidTxError(err)
	}

	return nil
}

func (self *StateTransition) transitionState() (ret []byte, usedGas *big.Int, err error) {
	if err = self.preCheck(); err != nil {
		return
	}

	var (
		msg    = self.msg
		sender = self.From()
	)

	// Pay intrinsic gas
	if err = self.UseGas(IntrinsicGas(self.msg)); err != nil {
		return nil, nil, InvalidTxError(err)
	}

	vmenv := self.env
	var ref vm.ContextRef
	if MessageCreatesContract(msg) {
		ret, err, ref = vmenv.Create(sender, self.msg.Data(), self.gas, self.gasPrice, self.value)
		if err == nil {
			dataGas := big.NewInt(int64(len(ret)))
			dataGas.Mul(dataGas, params.CreateDataGas)
			if err := self.UseGas(dataGas); err == nil {
				ref.SetCode(ret)
			} else {
				ret = nil // does not affect consensus but useful for StateTests validations
				glog.V(logger.Core).Infoln("Insufficient gas for creating code. Require", dataGas, "and have", self.gas)
			}
		}
	} else {
		// Increment the nonce for the next transaction
		self.state.SetNonce(sender.Address(), sender.Nonce()+1)
		ret, err = vmenv.Call(self.From(), self.To().Address(), self.msg.Data(), self.gas, self.gasPrice, self.value)
	}

	if err != nil && IsValueTransferErr(err) {
		return nil, nil, InvalidTxError(err)
	}

	self.refundGas()
	self.state.AddBalance(self.coinbase, new(big.Int).Mul(self.gasUsed(), self.gasPrice))

	return ret, self.gasUsed(), err
}

func (self *StateTransition) refundGas() {
	coinbase, sender := self.Coinbase(), self.From()
	// Return remaining gas
	remaining := new(big.Int).Mul(self.gas, self.msg.GasPrice())
	sender.AddBalance(remaining)

	uhalf := new(big.Int).Div(self.gasUsed(), common.Big2)
	for addr, ref := range self.state.Refunds() {
		refund := common.BigMin(uhalf, ref)
		self.gas.Add(self.gas, refund)
		self.state.AddBalance(common.StringToAddress(addr), refund.Mul(refund, self.msg.GasPrice()))
	}

	coinbase.RefundGas(self.gas, self.msg.GasPrice())
}

func (self *StateTransition) gasUsed() *big.Int {
	return new(big.Int).Sub(self.initialGas, self.gas)
}

// Converts an message in to a state object
func makeContract(msg Message, state *state.StateDB) *state.StateObject {
	faddr, _ := msg.From()
	addr := crypto.CreateAddress(faddr, msg.Nonce())

	contract := state.GetOrNewStateObject(addr)
	contract.SetInitCode(msg.Data())

	return contract
}

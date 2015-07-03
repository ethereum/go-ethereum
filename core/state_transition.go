package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
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

func MessageCreatesContract(msg Message) bool {
	return msg.To() == nil
}

// IntrinsicGas computes the 'intrisic gas' for a message
// with the given data.
func IntrinsicGas(data []byte) *big.Int {
	igas := new(big.Int).Set(params.TxGas)
	if len(data) > 0 {
		var nz int64
		for _, byt := range data {
			if byt != 0 {
				nz++
			}
		}
		m := big.NewInt(nz)
		m.Mul(m, params.TxDataNonZeroGas)
		igas.Add(igas, m)
		m.SetInt64(int64(len(data)) - nz)
		m.Mul(m, params.TxDataZeroGas)
		igas.Add(igas, m)
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
		gasPrice:   msg.GasPrice(),
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
func (self *StateTransition) From() (*state.StateObject, error) {
	f, err := self.msg.From()
	if err != nil {
		return nil, err
	}
	return self.state.GetOrNewStateObject(f), nil
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
	mgas := self.msg.Gas()
	mgval := new(big.Int).Mul(mgas, self.gasPrice)

	sender, err := self.From()
	if err != nil {
		return err
	}
	if sender.Balance().Cmp(mgval) < 0 {
		return fmt.Errorf("insufficient ETH for gas (%x). Req %v, has %v", sender.Address().Bytes()[:4], mgval, sender.Balance())
	}
	if err = self.Coinbase().SubGas(mgas, self.gasPrice); err != nil {
		return err
	}
	self.AddGas(mgas)
	self.initialGas.Set(mgas)
	sender.SubBalance(mgval)
	return nil
}

func (self *StateTransition) preCheck() (err error) {
	msg := self.msg
	sender, err := self.From()
	if err != nil {
		return err
	}

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

	msg := self.msg
	sender, _ := self.From() // err checked in preCheck

	// Pay intrinsic gas
	if err = self.UseGas(IntrinsicGas(self.data)); err != nil {
		return nil, nil, InvalidTxError(err)
	}

	vmenv := self.env
	var ref vm.ContextRef
	if MessageCreatesContract(msg) {
		ret, err, ref = vmenv.Create(sender, self.data, self.gas, self.gasPrice, self.value)
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
		ret, err = vmenv.Call(sender, self.To().Address(), self.data, self.gas, self.gasPrice, self.value)
	}

	if err != nil && IsValueTransferErr(err) {
		return nil, nil, InvalidTxError(err)
	}

	if vm.Debug {
		vm.StdErrFormat(vmenv.StructLogs())
	}

	self.refundGas()
	self.state.AddBalance(self.coinbase, new(big.Int).Mul(self.gasUsed(), self.gasPrice))

	return ret, self.gasUsed(), err
}

func (self *StateTransition) refundGas() {
	coinbase := self.Coinbase()
	sender, _ := self.From() // err already checked
	// Return remaining gas
	remaining := new(big.Int).Mul(self.gas, self.gasPrice)
	sender.AddBalance(remaining)

	uhalf := remaining.Div(self.gasUsed(), common.Big2)
	refund := common.BigMin(uhalf, self.state.Refunds())
	self.gas.Add(self.gas, refund)
	self.state.AddBalance(sender.Address(), refund.Mul(refund, self.gasPrice))

	coinbase.AddGas(self.gas, self.gasPrice)
}

func (self *StateTransition) gasUsed() *big.Int {
	return new(big.Int).Sub(self.initialGas, self.gas)
}

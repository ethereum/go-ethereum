package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/vm"
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

type Message interface {
	From() common.Address
	To() common.Address

	GasPrice() *big.Int
	Gas() *big.Int
	Value() *big.Int

	Nonce() uint64
	Data() []byte
}

func AddressFromMessage(msg Message) []byte {
	// Generate a new address
	return crypto.Sha3(common.NewValue([]interface{}{msg.From(), msg.Nonce()}).Encode())[12:]
}

func MessageCreatesContract(msg Message) bool {
	return len(msg.To()) == 0
}

func MessageGasValue(msg Message) *big.Int {
	return new(big.Int).Mul(msg.Gas(), msg.GasPrice())
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
	// statelogger.Debugf("(~) %x\n", self.msg.Hash())

	// XXX Transactions after this point are considered valid.
	if err = self.preCheck(); err != nil {
		return
	}

	var (
		msg    = self.msg
		sender = self.From()
	)

	// Transaction gas
	if err = self.UseGas(vm.GasTx); err != nil {
		return nil, nil, InvalidTxError(err)
	}

	// Increment the nonce for the next transaction
	self.state.SetNonce(sender.Address(), sender.Nonce()+1)
	//sender.Nonce += 1

	// Pay data gas
	var dgas int64
	for _, byt := range self.data {
		if byt != 0 {
			dgas += vm.GasTxDataNonzeroByte.Int64()
		} else {
			dgas += vm.GasTxDataZeroByte.Int64()
		}
	}
	if err = self.UseGas(big.NewInt(dgas)); err != nil {
		return nil, nil, InvalidTxError(err)
	}

	vmenv := self.env
	var ref vm.ContextRef
	if MessageCreatesContract(msg) {
		//contract := makeContract(msg, self.state)
		//addr := contract.Address()
		ret, err, ref = vmenv.Create(sender, self.msg.Data(), self.gas, self.gasPrice, self.value)
		if err == nil {
			dataGas := big.NewInt(int64(len(ret)))
			dataGas.Mul(dataGas, vm.GasCreateByte)
			if err := self.UseGas(dataGas); err == nil {
				ref.SetCode(ret)
			} else {
				statelogger.Infoln("Insufficient gas for creating code. Require", dataGas, "and have", self.gas)
			}
		}
	} else {
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
	/*
		addr := AddressFromMessage(msg)

		contract := state.GetOrNewStateObject(addr)
		contract.SetInitCode(msg.Data())

		return contract
	*/
	return nil
}

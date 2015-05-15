package helper

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

type Env struct {
	depth        int
	state        *state.StateDB
	skipTransfer bool
	initial      bool
	Gas          *big.Int

	origin common.Address
	//parent   common.Hash
	coinbase common.Address

	number     *big.Int
	time       int64
	difficulty *big.Int
	gasLimit   *big.Int

	logs state.Logs

	vmTest bool
}

func NewEnv(state *state.StateDB) *Env {
	return &Env{
		state: state,
	}
}

func NewEnvFromMap(state *state.StateDB, envValues map[string]string, exeValues map[string]string) *Env {
	env := NewEnv(state)

	env.origin = common.HexToAddress(exeValues["caller"])
	//env.parent = common.Hex2Bytes(envValues["previousHash"])
	env.coinbase = common.HexToAddress(envValues["currentCoinbase"])
	env.number = common.Big(envValues["currentNumber"])
	env.time = common.Big(envValues["currentTimestamp"]).Int64()
	env.difficulty = common.Big(envValues["currentDifficulty"])
	env.gasLimit = common.Big(envValues["currentGasLimit"])
	env.Gas = new(big.Int)

	return env
}

func (self *Env) Origin() common.Address { return self.origin }
func (self *Env) BlockNumber() *big.Int  { return self.number }

//func (self *Env) PrevHash() []byte      { return self.parent }
func (self *Env) Coinbase() common.Address { return self.coinbase }
func (self *Env) Time() int64              { return self.time }
func (self *Env) Difficulty() *big.Int     { return self.difficulty }
func (self *Env) State() *state.StateDB    { return self.state }
func (self *Env) GasLimit() *big.Int       { return self.gasLimit }
func (self *Env) VmType() vm.Type          { return vm.StdVmTy }
func (self *Env) GetHash(n uint64) common.Hash {
	return common.BytesToHash(crypto.Sha3([]byte(big.NewInt(int64(n)).String())))
}
func (self *Env) AddLog(log *state.Log) {
	self.logs = append(self.logs, log)
}
func (self *Env) Depth() int     { return self.depth }
func (self *Env) SetDepth(i int) { self.depth = i }
func (self *Env) Transfer(from, to vm.Account, amount *big.Int) error {
	if self.skipTransfer {
		// ugly hack
		if self.initial {
			self.initial = false
			return nil
		}

		if from.Balance().Cmp(amount) < 0 {
			return errors.New("Insufficient balance in account")
		}

		return nil
	}
	return vm.Transfer(from, to, amount)
}

func (self *Env) vm(addr *common.Address, data []byte, gas, price, value *big.Int) *core.Execution {
	exec := core.NewExecution(self, addr, data, gas, price, value)

	return exec
}

func (self *Env) Call(caller vm.ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	if self.vmTest && self.depth > 0 {
		caller.ReturnGas(gas, price)

		return nil, nil
	}
	exe := self.vm(&addr, data, gas, price, value)
	ret, err := exe.Call(addr, caller)
	self.Gas = exe.Gas

	return ret, err

}
func (self *Env) CallCode(caller vm.ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	if self.vmTest && self.depth > 0 {
		caller.ReturnGas(gas, price)

		return nil, nil
	}

	caddr := caller.Address()
	exe := self.vm(&caddr, data, gas, price, value)
	return exe.Call(addr, caller)
}

func (self *Env) Create(caller vm.ContextRef, data []byte, gas, price, value *big.Int) ([]byte, error, vm.ContextRef) {
	exe := self.vm(nil, data, gas, price, value)
	if self.vmTest {
		caller.ReturnGas(gas, price)

		nonce := self.state.GetNonce(caller.Address())
		obj := self.state.GetOrNewStateObject(crypto.CreateAddress(caller.Address(), nonce))

		return nil, nil, obj
	} else {
		return exe.Create(caller)
	}
}

func RunVm(state *state.StateDB, env, exec map[string]string) ([]byte, state.Logs, *big.Int, error) {
	var (
		to    = common.HexToAddress(exec["address"])
		from  = common.HexToAddress(exec["caller"])
		data  = FromHex(exec["data"])
		gas   = common.Big(exec["gas"])
		price = common.Big(exec["gasPrice"])
		value = common.Big(exec["value"])
	)
	// Reset the pre-compiled contracts for VM tests.
	vm.Precompiled = make(map[string]*vm.PrecompiledAccount)

	caller := state.GetOrNewStateObject(from)

	vmenv := NewEnvFromMap(state, env, exec)
	vmenv.vmTest = true
	vmenv.skipTransfer = true
	vmenv.initial = true
	ret, err := vmenv.Call(caller, to, data, gas, price, value)

	return ret, vmenv.logs, vmenv.Gas, err
}

func RunState(statedb *state.StateDB, env, tx map[string]string) ([]byte, state.Logs, *big.Int, error) {
	var (
		keyPair, _ = crypto.NewKeyPairFromSec([]byte(common.Hex2Bytes(tx["secretKey"])))
		data       = FromHex(tx["data"])
		gas        = common.Big(tx["gasLimit"])
		price      = common.Big(tx["gasPrice"])
		value      = common.Big(tx["value"])
		nonce      = common.Big(tx["nonce"]).Uint64()
		caddr      = common.HexToAddress(env["currentCoinbase"])
	)

	var to *common.Address
	if len(tx["to"]) > 2 {
		t := common.HexToAddress(tx["to"])
		to = &t
	}
	// Set pre compiled contracts
	vm.Precompiled = vm.PrecompiledContracts()

	snapshot := statedb.Copy()
	coinbase := statedb.GetOrNewStateObject(caddr)
	coinbase.SetGasPool(common.Big(env["currentGasLimit"]))

	message := NewMessage(common.BytesToAddress(keyPair.Address()), to, data, value, gas, price, nonce)
	vmenv := NewEnvFromMap(statedb, env, tx)
	vmenv.origin = common.BytesToAddress(keyPair.Address())
	ret, _, err := core.ApplyMessage(vmenv, message, coinbase)
	if core.IsNonceErr(err) || core.IsInvalidTxErr(err) {
		statedb.Set(snapshot)
	}
	statedb.Update()

	return ret, vmenv.logs, vmenv.Gas, err
}

type Message struct {
	from              common.Address
	to                *common.Address
	value, gas, price *big.Int
	data              []byte
	nonce             uint64
}

func NewMessage(from common.Address, to *common.Address, data []byte, value, gas, price *big.Int, nonce uint64) Message {
	return Message{from, to, value, gas, price, data, nonce}
}

func (self Message) Hash() []byte                  { return nil }
func (self Message) From() (common.Address, error) { return self.from, nil }
func (self Message) To() *common.Address           { return self.to }
func (self Message) GasPrice() *big.Int            { return self.price }
func (self Message) Gas() *big.Int                 { return self.gas }
func (self Message) Value() *big.Int               { return self.value }
func (self Message) Nonce() uint64                 { return self.nonce }
func (self Message) Data() []byte                  { return self.data }

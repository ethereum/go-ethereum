package helper

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/vm"
)

type Env struct {
	depth        int
	state        *state.StateDB
	skipTransfer bool
	Gas          *big.Int

	origin   []byte
	parent   []byte
	coinbase []byte

	number     *big.Int
	time       int64
	difficulty *big.Int
	gasLimit   *big.Int

	logs state.Logs
}

func NewEnv(state *state.StateDB) *Env {
	return &Env{
		state: state,
	}
}

func NewEnvFromMap(state *state.StateDB, envValues map[string]string, exeValues map[string]string) *Env {
	env := NewEnv(state)

	env.origin = ethutil.Hex2Bytes(exeValues["caller"])
	env.parent = ethutil.Hex2Bytes(envValues["previousHash"])
	env.coinbase = ethutil.Hex2Bytes(envValues["currentCoinbase"])
	env.number = ethutil.Big(envValues["currentNumber"])
	env.time = ethutil.Big(envValues["currentTimestamp"]).Int64()
	env.difficulty = ethutil.Big(envValues["currentDifficulty"])
	env.gasLimit = ethutil.Big(envValues["currentGasLimit"])
	env.Gas = new(big.Int)

	return env
}

func (self *Env) Origin() []byte        { return self.origin }
func (self *Env) BlockNumber() *big.Int { return self.number }
func (self *Env) PrevHash() []byte      { return self.parent }
func (self *Env) Coinbase() []byte      { return self.coinbase }
func (self *Env) Time() int64           { return self.time }
func (self *Env) Difficulty() *big.Int  { return self.difficulty }
func (self *Env) BlockHash() []byte     { return nil }
func (self *Env) State() *state.StateDB { return self.state }
func (self *Env) GasLimit() *big.Int    { return self.gasLimit }
func (self *Env) AddLog(log state.Log) {
	self.logs = append(self.logs, log)
}
func (self *Env) Depth() int     { return self.depth }
func (self *Env) SetDepth(i int) { self.depth = i }
func (self *Env) Transfer(from, to vm.Account, amount *big.Int) error {
	return vm.Transfer(from, to, amount)
}

func (self *Env) vm(addr, data []byte, gas, price, value *big.Int) *core.Execution {
	exec := core.NewExecution(self, addr, data, gas, price, value)
	exec.SkipTransfer = self.skipTransfer

	return exec
}

func (self *Env) Call(caller vm.ClosureRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error) {
	exe := self.vm(addr, data, gas, price, value)
	ret, err := exe.Call(addr, caller)
	self.Gas = exe.Gas

	return ret, err
}
func (self *Env) CallCode(caller vm.ClosureRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error) {
	exe := self.vm(caller.Address(), data, gas, price, value)
	return exe.Call(addr, caller)
}

func (self *Env) Create(caller vm.ClosureRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error, vm.ClosureRef) {
	exe := self.vm(addr, data, gas, price, value)
	return exe.Create(caller)
}

func RunVm(state *state.StateDB, env, exec map[string]string) ([]byte, state.Logs, *big.Int, error) {
	var (
		to    = FromHex(exec["address"])
		from  = FromHex(exec["caller"])
		data  = FromHex(exec["data"])
		gas   = ethutil.Big(exec["gas"])
		price = ethutil.Big(exec["gasPrice"])
		value = ethutil.Big(exec["value"])
	)

	caller := state.GetOrNewStateObject(from)

	vmenv := NewEnvFromMap(state, env, exec)
	vmenv.skipTransfer = true
	ret, err := vmenv.Call(caller, to, data, gas, price, value)

	return ret, vmenv.logs, vmenv.Gas, err
}

func RunState(statedb *state.StateDB, env, tx map[string]string) ([]byte, state.Logs, *big.Int, error) {
	var (
		keyPair, _ = crypto.NewKeyPairFromSec([]byte(ethutil.Hex2Bytes(tx["secretKey"])))
		to         = FromHex(tx["to"])
		data       = FromHex(tx["data"])
		gas        = ethutil.Big(tx["gasLimit"])
		price      = ethutil.Big(tx["gasPrice"])
		value      = ethutil.Big(tx["value"])
		caddr      = FromHex(env["currentCoinbase"])
	)

	coinbase := statedb.GetOrNewStateObject(caddr)
	coinbase.SetGasPool(ethutil.Big(env["currentGasLimit"]))

	message := NewMessage(keyPair.Address(), to, data, value, gas, price)
	Log.DebugDetailf("message{ to: %x, from %x, value: %v, gas: %v, price: %v }\n", message.to[:4], message.from[:4], message.value, message.gas, message.price)
	st := core.NewStateTransition(coinbase, message, statedb, nil)
	vmenv := NewEnvFromMap(statedb, env, tx)
	vmenv.origin = keyPair.Address()
	st.Env = vmenv
	ret, err := st.TransitionState()
	statedb.Update(vmenv.Gas)

	return ret, vmenv.logs, vmenv.Gas, err
}

type Message struct {
	from, to          []byte
	value, gas, price *big.Int
	data              []byte
}

func NewMessage(from, to, data []byte, value, gas, price *big.Int) Message {
	return Message{from, to, value, gas, price, data}
}

func (self Message) Hash() []byte       { return nil }
func (self Message) From() []byte       { return self.from }
func (self Message) To() []byte         { return self.to }
func (self Message) GasPrice() *big.Int { return self.price }
func (self Message) Gas() *big.Int      { return self.gas }
func (self Message) Value() *big.Int    { return self.value }
func (self Message) Nonce() uint64      { return 0 }
func (self Message) Data() []byte       { return self.data }

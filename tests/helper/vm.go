package helper

import (
	"math/big"

	"github.com/ethereum/go-ethereum/chain"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/vm"
)

type Env struct {
	depth        int
	state        *state.State
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

func NewEnv(state *state.State) *Env {
	return &Env{
		state: state,
	}
}

func NewEnvFromMap(state *state.State, envValues map[string]string, exeValues map[string]string) *Env {
	env := NewEnv(state)

	env.origin = ethutil.Hex2Bytes(exeValues["caller"])
	env.parent = ethutil.Hex2Bytes(envValues["previousHash"])
	env.coinbase = ethutil.Hex2Bytes(envValues["currentCoinbase"])
	env.number = ethutil.Big(envValues["currentNumber"])
	env.time = ethutil.Big(envValues["currentTimestamp"]).Int64()
	env.difficulty = ethutil.Big(envValues["currentDifficulty"])
	env.gasLimit = ethutil.Big(envValues["currentGasLimit"])

	return env
}

func (self *Env) Origin() []byte        { return self.origin }
func (self *Env) BlockNumber() *big.Int { return self.number }
func (self *Env) PrevHash() []byte      { return self.parent }
func (self *Env) Coinbase() []byte      { return self.coinbase }
func (self *Env) Time() int64           { return self.time }
func (self *Env) Difficulty() *big.Int  { return self.difficulty }
func (self *Env) BlockHash() []byte     { return nil }
func (self *Env) State() *state.State   { return self.state }
func (self *Env) GasLimit() *big.Int    { return self.gasLimit }
func (self *Env) AddLog(log *state.Log) {
	self.logs = append(self.logs, log)
}
func (self *Env) Depth() int     { return self.depth }
func (self *Env) SetDepth(i int) { self.depth = i }
func (self *Env) Transfer(from, to vm.Account, amount *big.Int) error {
	return vm.Transfer(from, to, amount)
}

func (self *Env) vm(addr, data []byte, gas, price, value *big.Int) *chain.Execution {
	evm := vm.New(self, vm.DebugVmTy)
	exec := chain.NewExecution(evm, addr, data, gas, price, value)
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

func RunVm(state *state.State, env, exec map[string]string) ([]byte, state.Logs, *big.Int, error) {
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

func RunState(state *state.State, env, tx map[string]string) ([]byte, state.Logs, *big.Int, error) {
	var (
		keyPair, _ = crypto.NewKeyPairFromSec([]byte(ethutil.Hex2Bytes(tx["secretKey"])))
		to         = FromHex(tx["to"])
		data       = FromHex(tx["data"])
		gas        = ethutil.Big(tx["gasLimit"])
		price      = ethutil.Big(tx["gasPrice"])
		value      = ethutil.Big(tx["value"])
	)

	caller := state.GetOrNewStateObject(keyPair.Address())

	vmenv := NewEnvFromMap(state, env, tx)
	vmenv.origin = caller.Address()
	ret, err := vmenv.Call(caller, to, data, gas, price, value)

	return ret, vmenv.logs, vmenv.Gas, err
}

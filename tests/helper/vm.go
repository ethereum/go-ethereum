package helper

import (
	"math/big"

	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/vm"
)

type Env struct {
	state *ethstate.State

	origin   []byte
	parent   []byte
	coinbase []byte

	number     *big.Int
	time       int64
	difficulty *big.Int
	gasLimit   *big.Int
}

func NewEnv(state *ethstate.State) *Env {
	return &Env{
		state: state,
	}
}

func NewEnvFromMap(state *ethstate.State, envValues map[string]string, exeValues map[string]string) *Env {
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

func (self *Env) Origin() []byte         { return self.origin }
func (self *Env) BlockNumber() *big.Int  { return self.number }
func (self *Env) PrevHash() []byte       { return self.parent }
func (self *Env) Coinbase() []byte       { return self.coinbase }
func (self *Env) Time() int64            { return self.time }
func (self *Env) Difficulty() *big.Int   { return self.difficulty }
func (self *Env) BlockHash() []byte      { return nil }
func (self *Env) State() *ethstate.State { return self.state }
func (self *Env) GasLimit() *big.Int     { return self.gasLimit }

func RunVm(state *ethstate.State, env, exec map[string]string) ([]byte, *big.Int, error) {
	address := FromHex(exec["address"])
	caller := state.GetOrNewStateObject(FromHex(exec["caller"]))
	caller.Balance = ethutil.Big(exec["value"])

	evm := vm.New(NewEnvFromMap(state, env, exec), vm.DebugVmTy)

	execution := vm.NewExecution(evm, address, FromHex(exec["data"]), ethutil.Big(exec["gas"]), ethutil.Big(exec["gasPrice"]), ethutil.Big(exec["value"]))
	ret, err := execution.Exec(address, caller)

	return ret, execution.Gas, err
}

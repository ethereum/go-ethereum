package helper

import (
	"fmt"
	"math/big"

	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethtrie"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethvm"
)

type Env struct {
	state *ethstate.State

	origin   []byte
	parent   []byte
	coinbase []byte

	number     *big.Int
	time       int64
	difficulty *big.Int
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
	env.time = ethutil.Big(envValues["currentTime"]).Int64()

	return env
}

func (self *Env) Origin() []byte        { return self.origin }
func (self *Env) BlockNumber() *big.Int { return self.number }
func (self *Env) PrevHash() []byte      { return self.parent }
func (self *Env) Coinbase() []byte      { return self.coinbase }
func (self *Env) Time() int64           { return self.time }
func (self *Env) Difficulty() *big.Int  { return self.difficulty }
func (self *Env) BlockHash() []byte     { return nil }

// This is likely to fail if anything ever gets looked up in the state trie :-)
func (self *Env) State() *ethstate.State { return ethstate.New(ethtrie.New(nil, "")) }

func RunVm(state *ethstate.State, env, exec map[string]string) ([]byte, *big.Int) {
	caller := state.NewStateObject(ethutil.Hex2Bytes(exec["caller"]))
	callee := state.GetStateObject(ethutil.Hex2Bytes(exec["address"]))
	closure := ethvm.NewClosure(nil, caller, callee, callee.Code, ethutil.Big(exec["gas"]), ethutil.Big(exec["gasPrice"]))

	vm := ethvm.New(NewEnvFromMap(state, env, exec), ethvm.DebugVmTy)
	ret, _, e := closure.Call(vm, nil)
	if e != nil {
		fmt.Println(e)
	}

	return ret, closure.Gas
}

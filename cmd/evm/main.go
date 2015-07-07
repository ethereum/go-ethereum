// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
)

var (
	code     = flag.String("code", "", "evm code")
	loglevel = flag.Int("log", 4, "log level")
	gas      = flag.String("gas", "1000000000", "gas amount")
	price    = flag.String("price", "0", "gas price")
	value    = flag.String("value", "0", "tx value")
	dump     = flag.Bool("dump", false, "dump state after run")
	data     = flag.String("data", "", "data")
)

func perr(v ...interface{}) {
	fmt.Println(v...)
	//os.Exit(1)
}

func main() {
	flag.Parse()

	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.LogLevel(*loglevel)))

	vm.Debug = true
	db, _ := ethdb.NewMemDatabase()
	statedb := state.New(common.Hash{}, db)
	sender := statedb.CreateAccount(common.StringToAddress("sender"))
	receiver := statedb.CreateAccount(common.StringToAddress("receiver"))
	receiver.SetCode(common.Hex2Bytes(*code))

	vmenv := NewEnv(statedb, common.StringToAddress("evmuser"), common.Big(*value))

	tstart := time.Now()

	ret, e := vmenv.Call(sender, receiver.Address(), common.Hex2Bytes(*data), common.Big(*gas), common.Big(*price), common.Big(*value))

	logger.Flush()
	if e != nil {
		perr(e)
	}

	if *dump {
		fmt.Println(string(statedb.Dump()))
	}

	vm.StdErrFormat(vmenv.StructLogs())

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Printf("vm took %v\n", time.Since(tstart))
	fmt.Printf(`alloc:      %d
tot alloc:  %d
no. malloc: %d
heap alloc: %d
heap objs:  %d
num gc:     %d
`, mem.Alloc, mem.TotalAlloc, mem.Mallocs, mem.HeapAlloc, mem.HeapObjects, mem.NumGC)

	fmt.Printf("%x\n", ret)
}

type VMEnv struct {
	state *state.StateDB
	block *types.Block

	transactor *common.Address
	value      *big.Int

	depth int
	Gas   *big.Int
	time  uint64
	logs  []vm.StructLog
}

func NewEnv(state *state.StateDB, transactor common.Address, value *big.Int) *VMEnv {
	return &VMEnv{
		state:      state,
		transactor: &transactor,
		value:      value,
		time:       uint64(time.Now().Unix()),
	}
}

func (self *VMEnv) State() *state.StateDB    { return self.state }
func (self *VMEnv) Origin() common.Address   { return *self.transactor }
func (self *VMEnv) BlockNumber() *big.Int    { return common.Big0 }
func (self *VMEnv) Coinbase() common.Address { return *self.transactor }
func (self *VMEnv) Time() uint64             { return self.time }
func (self *VMEnv) Difficulty() *big.Int     { return common.Big1 }
func (self *VMEnv) BlockHash() []byte        { return make([]byte, 32) }
func (self *VMEnv) Value() *big.Int          { return self.value }
func (self *VMEnv) GasLimit() *big.Int       { return big.NewInt(1000000000) }
func (self *VMEnv) VmType() vm.Type          { return vm.StdVmTy }
func (self *VMEnv) Depth() int               { return 0 }
func (self *VMEnv) SetDepth(i int)           { self.depth = i }
func (self *VMEnv) GetHash(n uint64) common.Hash {
	if self.block.Number().Cmp(big.NewInt(int64(n))) == 0 {
		return self.block.Hash()
	}
	return common.Hash{}
}
func (self *VMEnv) AddStructLog(log vm.StructLog) {
	self.logs = append(self.logs, log)
}
func (self *VMEnv) StructLogs() []vm.StructLog {
	return self.logs
}
func (self *VMEnv) AddLog(log *state.Log) {
	self.state.AddLog(log)
}
func (self *VMEnv) Transfer(from, to vm.Account, amount *big.Int) error {
	return vm.Transfer(from, to, amount)
}

func (self *VMEnv) vm(addr *common.Address, data []byte, gas, price, value *big.Int) *core.Execution {
	return core.NewExecution(self, addr, data, gas, price, value)
}

func (self *VMEnv) Call(caller vm.ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	exe := self.vm(&addr, data, gas, price, value)
	ret, err := exe.Call(addr, caller)
	self.Gas = exe.Gas

	return ret, err
}
func (self *VMEnv) CallCode(caller vm.ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	a := caller.Address()
	exe := self.vm(&a, data, gas, price, value)
	return exe.Call(addr, caller)
}

func (self *VMEnv) Create(caller vm.ContextRef, data []byte, gas, price, value *big.Int) ([]byte, error, vm.ContextRef) {
	exe := self.vm(nil, data, gas, price, value)
	return exe.Create(caller)
}

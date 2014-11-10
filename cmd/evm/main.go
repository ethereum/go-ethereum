/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Jeffrey Wilcke <i@jev.io>
 * @date 2014
 *
 */

package main

import (
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/vm"
)

var (
	code     = flag.String("code", "", "evm code")
	loglevel = flag.Int("log", 4, "log level")
	gas      = flag.String("gas", "1000000", "gas amount")
	price    = flag.String("price", "0", "gas price")
	dump     = flag.Bool("dump", false, "dump state after run")
)

func perr(v ...interface{}) {
	fmt.Println(v...)
	//os.Exit(1)
}

func main() {
	flag.Parse()

	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.LogLevel(*loglevel)))

	ethutil.ReadConfig("/tm/evmtest", "/tmp/evm", "")

	stateObject := state.NewStateObject([]byte("evmuser"))
	closure := vm.NewClosure(nil, stateObject, stateObject, ethutil.Hex2Bytes(*code), ethutil.Big(*gas), ethutil.Big(*price))

	tstart := time.Now()

	env := NewVmEnv()
	ret, _, e := closure.Call(vm.New(env, vm.DebugVmTy), nil)

	logger.Flush()
	if e != nil {
		perr(e)
	}

	if *dump {
		fmt.Println(string(env.state.Dump()))
	}

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

type VmEnv struct {
	state *state.State
}

func NewVmEnv() *VmEnv {
	db, _ := ethdb.NewMemDatabase()
	return &VmEnv{state.New(trie.New(db, ""))}
}

func (VmEnv) Origin() []byte            { return nil }
func (VmEnv) BlockNumber() *big.Int     { return nil }
func (VmEnv) BlockHash() []byte         { return nil }
func (VmEnv) PrevHash() []byte          { return nil }
func (VmEnv) Coinbase() []byte          { return nil }
func (VmEnv) Time() int64               { return 0 }
func (VmEnv) GasLimit() *big.Int        { return nil }
func (VmEnv) Difficulty() *big.Int      { return nil }
func (VmEnv) Value() *big.Int           { return nil }
func (self *VmEnv) State() *state.State { return self.state }
func (VmEnv) AddLog(state.Log)          {}
func (VmEnv) Transfer(from, to vm.Account, amount *big.Int) error {
	return nil
}

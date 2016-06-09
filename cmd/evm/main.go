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
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// evm executes EVM code snippets.
package main

import (
	"fmt"
	"math/big"
	"os"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger/glog"
	"gopkg.in/urfave/cli.v1"
)

var (
	app       *cli.App
	DebugFlag = cli.BoolFlag{
		Name:  "debug",
		Usage: "output full trace logs",
	}
	ForceJitFlag = cli.BoolFlag{
		Name:  "forcejit",
		Usage: "forces jit compilation",
	}
	DisableJitFlag = cli.BoolFlag{
		Name:  "nojit",
		Usage: "disabled jit compilation",
	}
	CodeFlag = cli.StringFlag{
		Name:  "code",
		Usage: "EVM code",
	}
	GasFlag = cli.StringFlag{
		Name:  "gas",
		Usage: "gas limit for the evm",
		Value: "10000000000",
	}
	PriceFlag = cli.StringFlag{
		Name:  "price",
		Usage: "price set for the evm",
		Value: "0",
	}
	ValueFlag = cli.StringFlag{
		Name:  "value",
		Usage: "value set for the evm",
		Value: "0",
	}
	DumpFlag = cli.BoolFlag{
		Name:  "dump",
		Usage: "dumps the state after the run",
	}
	InputFlag = cli.StringFlag{
		Name:  "input",
		Usage: "input for the EVM",
	}
	SysStatFlag = cli.BoolFlag{
		Name:  "sysstat",
		Usage: "display system stats",
	}
	VerbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "sets the verbosity level",
	}
)

func init() {
	app = utils.NewApp("0.2", "the evm command line interface")
	app.Flags = []cli.Flag{
		DebugFlag,
		VerbosityFlag,
		ForceJitFlag,
		DisableJitFlag,
		SysStatFlag,
		CodeFlag,
		GasFlag,
		PriceFlag,
		ValueFlag,
		DumpFlag,
		InputFlag,
	}
	app.Action = run
}

func run(ctx *cli.Context) {
	glog.SetToStderr(true)
	glog.SetV(ctx.GlobalInt(VerbosityFlag.Name))

	db, _ := ethdb.NewMemDatabase()
	statedb, _ := state.New(common.Hash{}, db)
	sender := statedb.CreateAccount(common.StringToAddress("sender"))
	receiver := statedb.CreateAccount(common.StringToAddress("receiver"))
	receiver.SetCode(common.Hex2Bytes(ctx.GlobalString(CodeFlag.Name)))

	vmenv := NewEnv(statedb, common.StringToAddress("evmuser"), common.Big(ctx.GlobalString(ValueFlag.Name)), vm.Config{
		Debug:     ctx.GlobalBool(DebugFlag.Name),
		ForceJit:  ctx.GlobalBool(ForceJitFlag.Name),
		EnableJit: !ctx.GlobalBool(DisableJitFlag.Name),
	})

	tstart := time.Now()
	ret, e := vmenv.Call(
		sender,
		receiver.Address(),
		common.Hex2Bytes(ctx.GlobalString(InputFlag.Name)),
		common.Big(ctx.GlobalString(GasFlag.Name)),
		common.Big(ctx.GlobalString(PriceFlag.Name)),
		common.Big(ctx.GlobalString(ValueFlag.Name)),
	)
	vmdone := time.Since(tstart)

	if ctx.GlobalBool(DumpFlag.Name) {
		fmt.Println(string(statedb.Dump()))
	}
	vm.StdErrFormat(vmenv.StructLogs())

	if ctx.GlobalBool(SysStatFlag.Name) {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		fmt.Printf("vm took %v\n", vmdone)
		fmt.Printf(`alloc:      %d
tot alloc:  %d
no. malloc: %d
heap alloc: %d
heap objs:  %d
num gc:     %d
`, mem.Alloc, mem.TotalAlloc, mem.Mallocs, mem.HeapAlloc, mem.HeapObjects, mem.NumGC)
	}

	fmt.Printf("OUT: 0x%x", ret)
	if e != nil {
		fmt.Printf(" error: %v", e)
	}
	fmt.Println()
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type VMEnv struct {
	state *state.StateDB
	block *types.Block

	transactor *common.Address
	value      *big.Int

	depth int
	Gas   *big.Int
	time  *big.Int
	logs  []vm.StructLog

	evm *vm.EVM
}

func NewEnv(state *state.StateDB, transactor common.Address, value *big.Int, cfg vm.Config) *VMEnv {
	env := &VMEnv{
		state:      state,
		transactor: &transactor,
		value:      value,
		time:       big.NewInt(time.Now().Unix()),
	}
	cfg.Logger.Collector = env

	env.evm = vm.New(env, cfg)
	return env
}

// ruleSet implements vm.RuleSet and will always default to the homestead rule set.
type ruleSet struct{}

func (ruleSet) IsHomestead(*big.Int) bool { return true }

func (self *VMEnv) RuleSet() vm.RuleSet        { return ruleSet{} }
func (self *VMEnv) Vm() vm.Vm                  { return self.evm }
func (self *VMEnv) Db() vm.Database            { return self.state }
func (self *VMEnv) MakeSnapshot() vm.Database  { return self.state.Copy() }
func (self *VMEnv) SetSnapshot(db vm.Database) { self.state.Set(db.(*state.StateDB)) }
func (self *VMEnv) Origin() common.Address     { return *self.transactor }
func (self *VMEnv) BlockNumber() *big.Int      { return common.Big0 }
func (self *VMEnv) Coinbase() common.Address   { return *self.transactor }
func (self *VMEnv) Time() *big.Int             { return self.time }
func (self *VMEnv) Difficulty() *big.Int       { return common.Big1 }
func (self *VMEnv) BlockHash() []byte          { return make([]byte, 32) }
func (self *VMEnv) Value() *big.Int            { return self.value }
func (self *VMEnv) GasLimit() *big.Int         { return big.NewInt(1000000000) }
func (self *VMEnv) VmType() vm.Type            { return vm.StdVmTy }
func (self *VMEnv) Depth() int                 { return 0 }
func (self *VMEnv) SetDepth(i int)             { self.depth = i }
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
func (self *VMEnv) AddLog(log *vm.Log) {
	self.state.AddLog(log)
}
func (self *VMEnv) CanTransfer(from common.Address, balance *big.Int) bool {
	return self.state.GetBalance(from).Cmp(balance) >= 0
}
func (self *VMEnv) Transfer(from, to vm.Account, amount *big.Int) {
	core.Transfer(from, to, amount)
}

func (self *VMEnv) Call(caller vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	self.Gas = gas
	return core.Call(self, caller, addr, data, gas, price, value)
}

func (self *VMEnv) CallCode(caller vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return core.CallCode(self, caller, addr, data, gas, price, value)
}

func (self *VMEnv) DelegateCall(caller vm.ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error) {
	return core.DelegateCall(self, caller, addr, data, gas, price)
}

func (self *VMEnv) Create(caller vm.ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	return core.Create(self, caller, data, gas, price, value)
}

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
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger/glog"
	"gopkg.in/urfave/cli.v1"
)

var gitCommit = "" // Git SHA1 commit hash of the release (set via linker flags)

var (
	app = utils.NewApp(gitCommit, "the evm command line interface")

	DebugFlag = cli.BoolFlag{
		Name:  "debug",
		Usage: "output full trace logs",
	}
	CodeFlag = cli.StringFlag{
		Name:  "code",
		Usage: "EVM code",
	}
	GasFlag = cli.StringFlag{
		Name:  "gas",
		Usage: "set the gas limit for the EVM execution",
		Value: "10000000000",
	}
	PriceFlag = cli.StringFlag{
		Name:  "price",
		Usage: "set the price for the EVM execution",
		Value: "0",
	}
	ValueFlag = cli.StringFlag{
		Name:  "value",
		Usage: "set the value for the EVM execution",
		Value: "0",
	}
	SenderFlag = cli.StringFlag{
		Name:  "origin",
		Usage: "set the origin for the EVM execution",
		Value: "0x",
	}
	CoinbaseFlag = cli.StringFlag{
		Name:  "coinbase",
		Usage: "coinbase set for the evm",
		Value: "0x",
	}
	BlockNumberFlag = cli.StringFlag{
		Name:  "blocknumber",
		Usage: "set the block number for the EVM execution",
		Value: "1",
	}
	BlockTimeFlag = cli.StringFlag{
		Name:  "blocktime",
		Usage: "set the block time for the EVM execution",
		Value: "1",
	}
	BlockDifficultyFlag = cli.StringFlag{
		Name:  "difficulty",
		Usage: "set the block difficulty for the EVM execution",
		Value: "1",
	}
	GasLimitFlag = cli.StringFlag{
		Name:  "gaslimit",
		Usage: "sets the gas limit for the EVM execution",
		Value: "1",
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
	CreateFlag = cli.BoolFlag{
		Name:  "create",
		Usage: "indicates the action should be create rather than call",
	}
)

func init() {
	app.Flags = []cli.Flag{
		CreateFlag,
		DebugFlag,
		VerbosityFlag,
		SysStatFlag,
		CodeFlag,
		GasFlag,
		PriceFlag,
		ValueFlag,
		DumpFlag,
		InputFlag,
		GasLimitFlag,
		CoinbaseFlag,
		SenderFlag,
		BlockNumberFlag,
		BlockTimeFlag,
	}
	app.Action = run
}

func run(ctx *cli.Context) error {
	glog.SetToStderr(true)
	glog.SetV(ctx.GlobalInt(VerbosityFlag.Name))

	logger := vm.NewStructLogger(nil)

	db, err := ethdb.NewMemDatabase()
	if err != nil {
		panic(err)
	}
	st, err := state.New(common.Hash{}, db)
	if err != nil {
		panic(err)
	}
	backend := &core.EVMBackend{
		GetHashFn: func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(big.NewInt(int64(n)).String())))
		},
		State: st,
	}
	context := vm.Context{
		CallContext: core.EVMCallContext{core.CanTransfer, core.Transfer},
		Origin:      common.StringToAddress(ctx.GlobalString(SenderFlag.Name)),
		Coinbase:    common.StringToAddress(ctx.GlobalString(CoinbaseFlag.Name)),
		BlockNumber: common.String2Big(ctx.GlobalString(BlockNumberFlag.Name)),
		Time:        common.String2Big(ctx.GlobalString(BlockTimeFlag.Name)),
		Difficulty:  common.String2Big(ctx.GlobalString(BlockDifficultyFlag.Name)),
		GasLimit:    common.String2Big(ctx.GlobalString(GasLimitFlag.Name)),
		GasPrice:    common.String2Big(ctx.GlobalString(PriceFlag.Name)),
	}

	vmenv := vm.NewEnvironment(
		context,
		backend,
		ruleSet{},
		vm.Config{
			Debug:  ctx.GlobalBool(DebugFlag.Name),
			Tracer: logger,
		},
	)

	var (
		ret    []byte
		tstart = time.Now()
		sender = st.CreateAccount(context.Origin)
	)

	if ctx.GlobalBool(CreateFlag.Name) {
		input := append(common.Hex2Bytes(ctx.GlobalString(CodeFlag.Name)), common.Hex2Bytes(ctx.GlobalString(InputFlag.Name))...)
		ret, _, err = vmenv.Create(
			sender,
			input,
			common.Big(ctx.GlobalString(GasFlag.Name)),
			common.Big(ctx.GlobalString(ValueFlag.Name)),
		)
	} else {
		receiver := st.CreateAccount(common.StringToAddress("receiver"))
		receiver.SetCode(common.Hex2Bytes(ctx.GlobalString(CodeFlag.Name)))
		ret, err = vmenv.Call(
			sender,
			receiver.Address(),
			common.Hex2Bytes(ctx.GlobalString(InputFlag.Name)),
			common.Big(ctx.GlobalString(GasFlag.Name)),
			common.Big(ctx.GlobalString(ValueFlag.Name)),
		)
	}
	vmdone := time.Since(tstart)

	if ctx.GlobalBool(DumpFlag.Name) {
		state.Commit(st)
		fmt.Println(string(st.Dump()))
	}
	vm.StdErrFormat(logger.StructLogs())

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
	if err != nil {
		fmt.Printf(" error: %v", err)
	}
	fmt.Println()
	return nil
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ruleSet implements vm.RuleSet and will always default to the homestead rule set.
type ruleSet struct{}

func (ruleSet) IsHomestead(*big.Int) bool { return true }

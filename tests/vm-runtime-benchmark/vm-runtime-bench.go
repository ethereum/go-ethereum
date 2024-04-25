// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"flag"
	"fmt"
	"math"
	"math/big"
	"os"
	"strings"
	"testing"

	_ "unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var bytecodeStore string = ""
var preserveState bool = false
var csv bool = false

// Initialize some constant calldata of 128KB, 2^17 bytes.
// This means, if we offset between 0th and 2^16th byte, we can fetch between 0 and 2^16 bytes (64KB)
// In consequence, we need args to memory-copying OPCODEs to be between 0 and 2^16, 2^16 fits in a PUSH3,
// which we'll be using to generate arguments for those OPCODEs.
var calldata = []byte(strings.Repeat("{", 1<<17))

// sets defaults on the config
func setDefaults(cfg *runtime.Config) {
	cfg.State, _ = state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)

	var (
		origin   = common.HexToAddress("origin")
		coinbase = common.HexToAddress("coinbase")
		contract = common.HexToAddress("contract")
	)
	cfg.Origin = origin
	cfg.State.CreateAccount(origin)
	cfg.Coinbase = coinbase
	cfg.State.CreateAccount(coinbase)
	cfg.State.CreateAccount(contract)

	if cfg.ChainConfig == nil {
		cfg.ChainConfig = &params.ChainConfig{
			ChainID:             big.NewInt(1),
			HomesteadBlock:      new(big.Int),
			DAOForkBlock:        new(big.Int),
			DAOForkSupport:      false,
			EIP150Block:         new(big.Int),
			EIP155Block:         new(big.Int),
			EIP158Block:         new(big.Int),
			ByzantiumBlock:      new(big.Int),
			ConstantinopleBlock: new(big.Int),
			PetersburgBlock:     new(big.Int),
			IstanbulBlock:       new(big.Int),
			MuirGlacierBlock:    new(big.Int),
			BerlinBlock:         new(big.Int),
			LondonBlock:         new(big.Int),
		}
	}

	if cfg.Difficulty == nil {
		cfg.Difficulty = new(big.Int)
	}
	if cfg.GasLimit == 0 {
		cfg.GasLimit = math.MaxUint64
	}
	if cfg.GasPrice == nil {
		cfg.GasPrice = new(big.Int)
	}
	if cfg.Value == nil {
		cfg.Value = new(big.Int)
	}
	if cfg.BlockNumber == nil {
		cfg.BlockNumber = new(big.Int)
	}
	if cfg.GetHashFn == nil {
		cfg.GetHashFn = func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
		}
	}
	if cfg.BaseFee == nil {
		cfg.BaseFee = big.NewInt(params.InitialBaseFee)
	}
	if cfg.BlobBaseFee == nil {
		cfg.BlobBaseFee = big.NewInt(params.BlobTxMinBlobGasprice)
	}
}

func BenchmarkBytecodeExecution(b *testing.B) {
	b.ReportAllocs()

	bytecode := common.Hex2Bytes(bytecodeStore)
	cfg := new(runtime.Config)
	setDefaults(cfg)

	var snapshotId int

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snapshotId = cfg.State.Snapshot()
		if _, _, err := runtime.Execute(bytecode, calldata, cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			b.Fail()
		}
		cfg.State.RevertToSnapshot(snapshotId)
	}
}

func BenchmarkBytecodeExecutionNonModyfing(b *testing.B) {
	b.ReportAllocs()

	bytecode := common.Hex2Bytes(bytecodeStore)
	cfg := new(runtime.Config)
	setDefaults(cfg)

	sender := vm.AccountRef(cfg.Origin)
	contract := common.HexToAddress("contract")

	vmenv := runtime.NewEnv(cfg)
	cfg.State.SetCode(contract, bytecode)
	value := uint256.MustFromBig(cfg.Value)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := vmenv.Call(sender, contract, calldata, cfg.GasLimit, value); err != nil {
			fmt.Fprintln(os.Stderr, err)
			b.Fail()
		}
	}
}

func runBenchmark(samples int) {
	if csv {
		fmt.Println("SampleId, ops, ns/op, mem allocs/op, mem bytes/op")
	} else {
		fmt.Println("Results of benchmarking EVM bytecode execution:")
	}

	if preserveState {
		for i := 0; i < samples; i++ {
			result := testing.Benchmark(BenchmarkBytecodeExecution)
			outputResults(i, result)
		}
	} else {
		for i := 0; i < samples; i++ {
			result := testing.Benchmark(BenchmarkBytecodeExecutionNonModyfing)
			outputResults(i, result)
		}
	}
}

func outputResults(sampleId int, r testing.BenchmarkResult) {
	if csv {
		fmt.Printf("%v,%v,%v,%v,%v\n", sampleId, r.N, r.NsPerOp(), r.AllocsPerOp(), r.AllocedBytesPerOp())
	} else {
		fmt.Printf("%v: %v %v\n", sampleId, r.String(), r.MemString())
	}
}

func main() {
	bytecodePtr := flag.String("bytecode", "", "EVM bytecode to execute and measure, e.g. 61FFFF600020 (mandatory)")
	calldataPtr := flag.String("calldata", "", "Calldata to pass to the EVM bytecode")
	samplesPtr := flag.Int("samples", 1, "Number of measured repetitions of execution")
	preserveStatePtr := flag.Bool("preserveState", false, "Preserve state between executions, in case of a state modifying bytecode, adds overhead for snapshotting")
	csvPtr := flag.Bool("csv", true, "Output results in CSV format (default: true)")

	flag.Parse()

	bytecodeStore = *bytecodePtr
	samples := *samplesPtr
	preserveState = *preserveStatePtr
	csv = *csvPtr

	if bytecodeStore == "" {
		fmt.Println("Please provide a bytecode to execute")
		os.Exit(1)
	}

	if *calldataPtr != "" {
		calldata = common.Hex2Bytes(*calldataPtr)
		if len(calldata) == 0 {
			fmt.Println("Invalid calldata provided")
			os.Exit(1)
		}
	}

	runBenchmark(samples)
}

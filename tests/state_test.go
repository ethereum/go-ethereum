// Copyright 2015 The go-ethereum Authors
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

package tests

import (
	"bufio"
	"bytes"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
)

func TestState(t *testing.T) {
	t.Parallel()

	st := new(testMatcher)
	// Long tests:
	st.slow(`^stAttackTest/ContractCreationSpam`)
	st.slow(`^stBadOpcode/badOpcodes`)
	st.slow(`^stPreCompiledContracts/modexp`)
	st.slow(`^stQuadraticComplexityTest/`)
	st.slow(`^stStaticCall/static_Call50000`)
	st.slow(`^stStaticCall/static_Return50000`)
	st.slow(`^stSystemOperationsTest/CallRecursiveBomb`)
	st.slow(`^stTransactionTest/Opcodes_TransactionInit`)

	// Very time consuming
	st.skipLoad(`^stTimeConsuming/`)
	st.skipLoad(`.*vmPerformance/loop.*`)

	// Uses 1GB RAM per tested fork
	st.skipLoad(`^stStaticCall/static_Call1MB`)

	// Broken tests:
	//
	// The stEOF tests are generated with EOF as part of Shanghai, which
	// is erroneous. Therefore, these tests are skipped.
	st.skipLoad(`^EIPTests/stEOF/`)
	// Expected failures:

	// For Istanbul, older tests were moved into LegacyTests
	for _, dir := range []string{
		stateTestDir,
		legacyStateTestDir,
		benchmarksDir,
	} {
		st.walk(t, dir, func(t *testing.T, name string, test *StateTest) {
			for _, subtest := range test.Subtests() {
				subtest := subtest
				key := fmt.Sprintf("%s/%d", subtest.Fork, subtest.Index)

				t.Run(key+"/trie", func(t *testing.T) {
					withTrace(t, test.gasLimit(subtest), func(vmconfig vm.Config) error {
						_, _, err := test.Run(subtest, vmconfig, false)
						return st.checkFailure(t, err)
					})
				})
				t.Run(key+"/snap", func(t *testing.T) {
					withTrace(t, test.gasLimit(subtest), func(vmconfig vm.Config) error {
						snaps, statedb, err := test.Run(subtest, vmconfig, true)
						if snaps != nil && statedb != nil {
							if _, err := snaps.Journal(statedb.IntermediateRoot(false)); err != nil {
								return err
							}
						}
						return st.checkFailure(t, err)
					})
				})
			}
		})
	}
}

// Transactions with gasLimit above this value will not get a VM trace on failure.
const traceErrorLimit = 400000

func withTrace(t *testing.T, gasLimit uint64, test func(vm.Config) error) {
	// Use config from command line arguments.
	config := vm.Config{}
	err := test(config)
	if err == nil {
		return
	}

	// Test failed, re-run with tracing enabled.
	t.Error(err)
	if gasLimit > traceErrorLimit {
		t.Log("gas limit too high for EVM trace")
		return
	}
	buf := new(bytes.Buffer)
	w := bufio.NewWriter(buf)
	tracer := logger.NewJSONLogger(&logger.Config{}, w)
	config.Debug, config.Tracer = true, tracer
	err2 := test(config)
	if !reflect.DeepEqual(err, err2) {
		t.Errorf("different error for second run: %v", err2)
	}
	w.Flush()
	if buf.Len() == 0 {
		t.Log("no EVM operation logs generated")
	} else {
		t.Log("EVM operation log:\n" + buf.String())
	}
	// t.Logf("EVM output: 0x%x", tracer.Output())
	// t.Logf("EVM error: %v", tracer.Error())
}

func BenchmarkEVM(b *testing.B) {
	// Walk the directory.
	dir := benchmarksDir
	dirinfo, err := os.Stat(dir)
	if os.IsNotExist(err) || !dirinfo.IsDir() {
		fmt.Fprintf(os.Stderr, "can't find test files in %s, did you clone the evm-benchmarks submodule?\n", dir)
		b.Skip("missing test files")
	}
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if ext := filepath.Ext(path); ext == ".json" {
			name := filepath.ToSlash(strings.TrimPrefix(strings.TrimSuffix(path, ext), dir+string(filepath.Separator)))
			b.Run(name, func(b *testing.B) { runBenchmarkFile(b, path) })
		}
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}
}

func runBenchmarkFile(b *testing.B, path string) {
	m := make(map[string]StateTest)
	if err := readJSONFile(path, &m); err != nil {
		b.Fatal(err)
		return
	}
	if len(m) != 1 {
		b.Fatal("expected single benchmark in a file")
		return
	}
	for _, t := range m {
		t := t
		runBenchmark(b, &t)
	}
}

func runBenchmark(b *testing.B, t *StateTest) {
	for _, subtest := range t.Subtests() {
		subtest := subtest
		key := fmt.Sprintf("%s/%d", subtest.Fork, subtest.Index)

		b.Run(key, func(b *testing.B) {
			vmconfig := vm.Config{}

			config, eips, err := GetChainConfig(subtest.Fork)
			if err != nil {
				b.Error(err)
				return
			}
			var rules = config.Rules(new(big.Int), false, 0)

			vmconfig.ExtraEips = eips
			block := t.genesis(config).ToBlock()
			_, statedb := MakePreState(rawdb.NewMemoryDatabase(), t.json.Pre, false)

			var baseFee *big.Int
			if rules.IsLondon {
				baseFee = t.json.Env.BaseFee
				if baseFee == nil {
					// Retesteth uses `0x10` for genesis baseFee. Therefore, it defaults to
					// parent - 2 : 0xa as the basefee for 'this' context.
					baseFee = big.NewInt(0x0a)
				}
			}
			post := t.json.Post[subtest.Fork][subtest.Index]
			msg, err := t.json.Tx.toMessage(post, baseFee)
			if err != nil {
				b.Error(err)
				return
			}

			// Try to recover tx with current signer
			if len(post.TxBytes) != 0 {
				var ttx types.Transaction
				err := ttx.UnmarshalBinary(post.TxBytes)
				if err != nil {
					b.Error(err)
					return
				}

				if _, err := types.Sender(types.LatestSigner(config), &ttx); err != nil {
					b.Error(err)
					return
				}
			}

			// Prepare the EVM.
			txContext := core.NewEVMTxContext(msg)
			context := core.NewEVMBlockContext(block.Header(), nil, &t.json.Env.Coinbase)
			context.GetHash = vmTestBlockHash
			context.BaseFee = baseFee
			evm := vm.NewEVM(context, txContext, statedb, config, vmconfig)

			// Create "contract" for sender to cache code analysis.
			sender := vm.NewContract(vm.AccountRef(msg.From()), vm.AccountRef(msg.From()),
				nil, 0)

			var (
				gasUsed uint64
				elapsed uint64
				refund  uint64
			)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				snapshot := statedb.Snapshot()
				statedb.Prepare(rules, msg.From(), context.Coinbase, msg.To(), vm.ActivePrecompiles(rules), msg.AccessList())
				b.StartTimer()
				start := time.Now()

				// Execute the message.
				_, leftOverGas, err := evm.Call(sender, *msg.To(), msg.Data(), msg.Gas(), msg.Value())
				if err != nil {
					b.Error(err)
					return
				}

				b.StopTimer()
				elapsed += uint64(time.Since(start))
				refund += statedb.GetRefund()
				gasUsed += msg.Gas() - leftOverGas

				statedb.RevertToSnapshot(snapshot)
			}
			if elapsed < 1 {
				elapsed = 1
			}
			// Keep it as uint64, multiply 100 to get two digit float later
			mgasps := (100 * 1000 * (gasUsed - refund)) / elapsed
			b.ReportMetric(float64(mgasps)/100, "mgas/s")
		})
	}
}

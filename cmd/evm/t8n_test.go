// Copyright 2021 The go-ethereum Authors
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

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/cmd/evm/internal/t8ntool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/internal/cmdtest"
	"github.com/ethereum/go-ethereum/internal/reexec"
	"github.com/ethereum/go-ethereum/tests"
)

func TestMain(m *testing.M) {
	// Run the app if we've been exec'd as "ethkey-test" in runEthkey.
	reexec.Register("evm-test", func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
	// check if we have been reexec'd
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

type testT8n struct {
	*cmdtest.TestCmd
}

type t8nInput struct {
	inAlloc  string
	inTxs    string
	inEnv    string
	stFork   string
	stReward string
}

func (args *t8nInput) get(base string) []string {
	var out []string
	if opt := args.inAlloc; opt != "" {
		out = append(out, "--input.alloc")
		out = append(out, fmt.Sprintf("%v/%v", base, opt))
	}
	if opt := args.inTxs; opt != "" {
		out = append(out, "--input.txs")
		out = append(out, fmt.Sprintf("%v/%v", base, opt))
	}
	if opt := args.inEnv; opt != "" {
		out = append(out, "--input.env")
		out = append(out, fmt.Sprintf("%v/%v", base, opt))
	}
	if opt := args.stFork; opt != "" {
		out = append(out, "--state.fork", opt)
	}
	if opt := args.stReward; opt != "" {
		out = append(out, "--state.reward", opt)
	}
	return out
}

type t8nOutput struct {
	alloc  bool
	result bool
	body   bool
}

func (args *t8nOutput) get() (out []string) {
	if args.body {
		out = append(out, "--output.body", "stdout")
	} else {
		out = append(out, "--output.body", "") // empty means ignore
	}
	if args.result {
		out = append(out, "--output.result", "stdout")
	} else {
		out = append(out, "--output.result", "")
	}
	if args.alloc {
		out = append(out, "--output.alloc", "stdout")
	} else {
		out = append(out, "--output.alloc", "")
	}
	return out
}

func TestT8n(t *testing.T) {
	t.Parallel()
	tt := new(testT8n)
	tt.TestCmd = cmdtest.NewTestCmd(t, tt)
	for i, tc := range []struct {
		base        string
		input       t8nInput
		output      t8nOutput
		expExitCode int
		expOut      string
	}{
		{ // Test exit (3) on bad config
			base: "./testdata/1",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "Frontier+1346", "",
			},
			output:      t8nOutput{alloc: true, result: true},
			expExitCode: 3,
		},
		{
			base: "./testdata/1",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "Byzantium", "",
			},
			output: t8nOutput{alloc: true, result: true},
			expOut: "exp.json",
		},
		{ // blockhash test
			base: "./testdata/3",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "Berlin", "",
			},
			output: t8nOutput{alloc: true, result: true},
			expOut: "exp.json",
		},
		{ // missing blockhash test
			base: "./testdata/4",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "Berlin", "",
			},
			output:      t8nOutput{alloc: true, result: true},
			expExitCode: 4,
		},
		{ // Uncle test
			base: "./testdata/5",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "Byzantium", "0x80",
			},
			output: t8nOutput{alloc: true, result: true},
			expOut: "exp.json",
		},
		{ // Sign json transactions
			base: "./testdata/13",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "London", "",
			},
			output: t8nOutput{body: true},
			expOut: "exp.json",
		},
		{ // Already signed transactions
			base: "./testdata/13",
			input: t8nInput{
				"alloc.json", "signed_txs.rlp", "env.json", "London", "",
			},
			output: t8nOutput{result: true},
			expOut: "exp2.json",
		},
		{ // Difficulty calculation - no uncles
			base: "./testdata/14",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "London", "",
			},
			output: t8nOutput{result: true},
			expOut: "exp.json",
		},
		{ // Difficulty calculation - with uncles
			base: "./testdata/14",
			input: t8nInput{
				"alloc.json", "txs.json", "env.uncles.json", "London", "",
			},
			output: t8nOutput{result: true},
			expOut: "exp2.json",
		},
		{ // Difficulty calculation - with ommers + Berlin
			base: "./testdata/14",
			input: t8nInput{
				"alloc.json", "txs.json", "env.uncles.json", "Berlin", "",
			},
			output: t8nOutput{result: true},
			expOut: "exp_berlin.json",
		},
		{ // Difficulty calculation on arrow glacier
			base: "./testdata/19",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "London", "",
			},
			output: t8nOutput{result: true},
			expOut: "exp_london.json",
		},
		{ // Difficulty calculation on arrow glacier
			base: "./testdata/19",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "ArrowGlacier", "",
			},
			output: t8nOutput{result: true},
			expOut: "exp_arrowglacier.json",
		},
		{ // Difficulty calculation on gray glacier
			base: "./testdata/19",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "GrayGlacier", "",
			},
			output: t8nOutput{result: true},
			expOut: "exp_grayglacier.json",
		},
		{ // Sign unprotected (pre-EIP155) transaction
			base: "./testdata/23",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "Berlin", "",
			},
			output: t8nOutput{result: true},
			expOut: "exp.json",
		},
		{ // Test post-merge transition
			base: "./testdata/24",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "Paris", "",
			},
			output: t8nOutput{alloc: true, result: true},
			expOut: "exp.json",
		},
		{ // Test post-merge transition where input is missing random
			base: "./testdata/24",
			input: t8nInput{
				"alloc.json", "txs.json", "env-missingrandom.json", "Paris", "",
			},
			output:      t8nOutput{alloc: false, result: false},
			expExitCode: 3,
		},
		{ // Test base fee calculation
			base: "./testdata/25",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "Paris", "",
			},
			output: t8nOutput{alloc: true, result: true},
			expOut: "exp.json",
		},
		{ // Test withdrawals transition
			base: "./testdata/26",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "Shanghai", "",
			},
			output: t8nOutput{alloc: true, result: true},
			expOut: "exp.json",
		},
		{ // Cancun tests
			base: "./testdata/28",
			input: t8nInput{
				"alloc.json", "txs.rlp", "env.json", "Cancun", "",
			},
			output: t8nOutput{alloc: true, result: true},
			expOut: "exp.json",
		},
		{ // More cancun tests
			base: "./testdata/29",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "Cancun", "",
			},
			output: t8nOutput{alloc: true, result: true},
			expOut: "exp.json",
		},
		{ // More cancun test, plus example of rlp-transaction that cannot be decoded properly
			base: "./testdata/30",
			input: t8nInput{
				"alloc.json", "txs_more.rlp", "env.json", "Cancun", "",
			},
			output: t8nOutput{alloc: true, result: true},
			expOut: "exp.json",
		},
		{ // Prague test, EIP-7702 transaction
			base: "./testdata/33",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "Prague", "",
			},
			output: t8nOutput{alloc: true, result: true},
			expOut: "exp.json",
		},
		{ // Osaka test, EIP-7918 blob gas with parent base fee
			base: "./testdata/34",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "Osaka", "",
			},
			output: t8nOutput{alloc: true, result: true},
			expOut: "exp.json",
		},
		{ // Test post-London where input is missing base fee information
			base: "./testdata/35",
			input: t8nInput{
				"alloc.json", "txs.json", "env.json", "Osaka", "",
			},
			output:      t8nOutput{alloc: false, result: false},
			expExitCode: 3,
		},
	} {
		args := []string{"t8n"}
		args = append(args, tc.output.get()...)
		args = append(args, tc.input.get(tc.base)...)
		var qArgs []string // quoted args for debugging purposes
		for _, arg := range args {
			if len(arg) == 0 {
				qArgs = append(qArgs, `""`)
			} else {
				qArgs = append(qArgs, arg)
			}
		}
		tt.Logf("args: %v\n", strings.Join(qArgs, " "))
		tt.Run("evm-test", args...)
		// Compare the expected output, if provided
		if tc.expOut != "" {
			file := fmt.Sprintf("%v/%v", tc.base, tc.expOut)
			want, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("test %d: could not read expected output: %v", i, err)
			}
			have := tt.Output()
			ok, err := cmpJson(have, want)
			switch {
			case err != nil:
				t.Fatalf("test %d, file %v: json parsing failed: %v", i, file, err)
			case !ok:
				t.Fatalf("test %d, file %v: output wrong, have \n%v\nwant\n%v\n", i, file, string(have), string(want))
			}
		}
		tt.WaitExit()
		if have, want := tt.ExitStatus(), tc.expExitCode; have != want {
			t.Fatalf("test %d: wrong exit code, have %d, want %d", i, have, want)
		}
	}
}

func lineIterator(path string) func() (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return func() (string, error) { return err.Error(), err }
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	return func() (string, error) {
		if scanner.Scan() {
			return scanner.Text(), nil
		}
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF // scanner gobbles io.EOF, but we want it
	}
}

type t9nInput struct {
	inTxs  string
	stFork string
}

func (args *t9nInput) get(base string) []string {
	var out []string
	if opt := args.inTxs; opt != "" {
		out = append(out, "--input.txs")
		out = append(out, fmt.Sprintf("%v/%v", base, opt))
	}
	if opt := args.stFork; opt != "" {
		out = append(out, "--state.fork", opt)
	}
	return out
}

func TestT9n(t *testing.T) {
	t.Parallel()
	tt := new(testT8n)
	tt.TestCmd = cmdtest.NewTestCmd(t, tt)
	for i, tc := range []struct {
		base        string
		input       t9nInput
		expExitCode int
		expOut      string
	}{
		{ // London txs on homestead
			base: "./testdata/15",
			input: t9nInput{
				inTxs:  "signed_txs.rlp",
				stFork: "Homestead",
			},
			expOut: "exp.json",
		},
		{ // London txs on London
			base: "./testdata/15",
			input: t9nInput{
				inTxs:  "signed_txs.rlp",
				stFork: "London",
			},
			expOut: "exp2.json",
		},
		{ // An RLP list (a blockheader really)
			base: "./testdata/15",
			input: t9nInput{
				inTxs:  "blockheader.rlp",
				stFork: "London",
			},
			expOut: "exp3.json",
		},
		{ // Transactions with too low gas
			base: "./testdata/16",
			input: t9nInput{
				inTxs:  "signed_txs.rlp",
				stFork: "London",
			},
			expOut: "exp.json",
		},
		{ // Transactions with value exceeding 256 bits
			base: "./testdata/17",
			input: t9nInput{
				inTxs:  "signed_txs.rlp",
				stFork: "London",
			},
			expOut: "exp.json",
		},
		{ // Invalid RLP
			base: "./testdata/18",
			input: t9nInput{
				inTxs:  "invalid.rlp",
				stFork: "London",
			},
			expExitCode: t8ntool.ErrorIO,
		},
	} {
		args := []string{"t9n"}
		args = append(args, tc.input.get(tc.base)...)

		tt.Run("evm-test", args...)
		tt.Logf("args:\n go run . %v\n", strings.Join(args, " "))
		// Compare the expected output, if provided
		if tc.expOut != "" {
			want, err := os.ReadFile(fmt.Sprintf("%v/%v", tc.base, tc.expOut))
			if err != nil {
				t.Fatalf("test %d: could not read expected output: %v", i, err)
			}
			have := tt.Output()
			ok, err := cmpJson(have, want)
			switch {
			case err != nil:
				t.Log(string(have))
				t.Fatalf("test %d, json parsing failed: %v", i, err)
			case !ok:
				t.Fatalf("test %d: output wrong, have \n%v\nwant\n%v\n", i, string(have), string(want))
			}
		}
		tt.WaitExit()
		if have, want := tt.ExitStatus(), tc.expExitCode; have != want {
			t.Fatalf("test %d: wrong exit code, have %d, want %d", i, have, want)
		}
	}
}

type b11rInput struct {
	inEnv         string
	inOmmersRlp   string
	inWithdrawals string
	inTxsRlp      string
	inClique      string
	ethash        bool
	ethashMode    string
	ethashDir     string
}

func (args *b11rInput) get(base string) []string {
	var out []string
	if opt := args.inEnv; opt != "" {
		out = append(out, "--input.header")
		out = append(out, fmt.Sprintf("%v/%v", base, opt))
	}
	if opt := args.inOmmersRlp; opt != "" {
		out = append(out, "--input.ommers")
		out = append(out, fmt.Sprintf("%v/%v", base, opt))
	}
	if opt := args.inWithdrawals; opt != "" {
		out = append(out, "--input.withdrawals")
		out = append(out, fmt.Sprintf("%v/%v", base, opt))
	}
	if opt := args.inTxsRlp; opt != "" {
		out = append(out, "--input.txs")
		out = append(out, fmt.Sprintf("%v/%v", base, opt))
	}
	if opt := args.inClique; opt != "" {
		out = append(out, "--seal.clique")
		out = append(out, fmt.Sprintf("%v/%v", base, opt))
	}
	if args.ethash {
		out = append(out, "--seal.ethash")
	}
	if opt := args.ethashMode; opt != "" {
		out = append(out, "--seal.ethash.mode")
		out = append(out, fmt.Sprintf("%v/%v", base, opt))
	}
	if opt := args.ethashDir; opt != "" {
		out = append(out, "--seal.ethash.dir")
		out = append(out, fmt.Sprintf("%v/%v", base, opt))
	}
	out = append(out, "--output.block")
	out = append(out, "stdout")
	return out
}

func TestB11r(t *testing.T) {
	t.Parallel()
	tt := new(testT8n)
	tt.TestCmd = cmdtest.NewTestCmd(t, tt)
	for i, tc := range []struct {
		base        string
		input       b11rInput
		expExitCode int
		expOut      string
	}{
		{ // unsealed block
			base: "./testdata/20",
			input: b11rInput{
				inEnv:       "header.json",
				inOmmersRlp: "ommers.json",
				inTxsRlp:    "txs.rlp",
			},
			expOut: "exp.json",
		},
		{ // ethash test seal
			base: "./testdata/21",
			input: b11rInput{
				inEnv:       "header.json",
				inOmmersRlp: "ommers.json",
				inTxsRlp:    "txs.rlp",
			},
			expOut: "exp.json",
		},
		{ // clique test seal
			base: "./testdata/21",
			input: b11rInput{
				inEnv:       "header.json",
				inOmmersRlp: "ommers.json",
				inTxsRlp:    "txs.rlp",
				inClique:    "clique.json",
			},
			expOut: "exp-clique.json",
		},
		{ // block with ommers
			base: "./testdata/22",
			input: b11rInput{
				inEnv:       "header.json",
				inOmmersRlp: "ommers.json",
				inTxsRlp:    "txs.rlp",
			},
			expOut: "exp.json",
		},
		{ // block with withdrawals
			base: "./testdata/27",
			input: b11rInput{
				inEnv:         "header.json",
				inOmmersRlp:   "ommers.json",
				inWithdrawals: "withdrawals.json",
				inTxsRlp:      "txs.rlp",
			},
			expOut: "exp.json",
		},
	} {
		args := []string{"b11r"}
		args = append(args, tc.input.get(tc.base)...)

		tt.Run("evm-test", args...)
		tt.Logf("args:\n go run . %v\n", strings.Join(args, " "))
		// Compare the expected output, if provided
		if tc.expOut != "" {
			want, err := os.ReadFile(fmt.Sprintf("%v/%v", tc.base, tc.expOut))
			if err != nil {
				t.Fatalf("test %d: could not read expected output: %v", i, err)
			}
			have := tt.Output()
			ok, err := cmpJson(have, want)
			switch {
			case err != nil:
				t.Log(string(have))
				t.Fatalf("test %d, json parsing failed: %v", i, err)
			case !ok:
				t.Fatalf("test %d: output wrong, have \n%v\nwant\n%v\n", i, string(have), string(want))
			}
		}
		tt.WaitExit()
		if have, want := tt.ExitStatus(), tc.expExitCode; have != want {
			t.Fatalf("test %d: wrong exit code, have %d, want %d", i, have, want)
		}
	}
}

func TestEvmRun(t *testing.T) {
	t.Parallel()
	tt := cmdtest.NewTestCmd(t, nil)
	for i, tc := range []struct {
		input      []string
		wantStdout string
		wantStderr string
	}{
		{ // json tracing
			input:      []string{"run", "--trace", "--trace.format=json", "6040"},
			wantStdout: "./testdata/evmrun/1.out.1.txt",
			wantStderr: "./testdata/evmrun/1.out.2.txt",
		},
		{ // Same as above, using the deprecated --json
			input:      []string{"run", "--json", "6040"},
			wantStdout: "./testdata/evmrun/1.out.1.txt",
			wantStderr: "./testdata/evmrun/1.out.2.txt",
		},
		{ // Struct tracing
			input:      []string{"run", "--trace", "--trace.format=struct", "0x6040"},
			wantStdout: "./testdata/evmrun/2.out.1.txt",
			wantStderr: "./testdata/evmrun/2.out.2.txt",
		},
		{ // struct-tracing, plus alloc-dump
			input:      []string{"run", "--trace", "--trace.format=struct", "--dump", "0x6040"},
			wantStdout: "./testdata/evmrun/3.out.1.txt",
			//wantStderr: "./testdata/evmrun/3.out.2.txt",
		},
		{ // json-tracing (default), plus alloc-dump
			input:      []string{"run", "--trace", "--dump", "0x6040"},
			wantStdout: "./testdata/evmrun/4.out.1.txt",
			//wantStderr: "./testdata/evmrun/4.out.2.txt",
		},
		{ // md-tracing
			input:      []string{"run", "--trace", "--trace.format=md", "0x6040"},
			wantStdout: "./testdata/evmrun/5.out.1.txt",
			wantStderr: "./testdata/evmrun/5.out.2.txt",
		},
		{ // statetest subcommand
			input:      []string{"statetest", "./testdata/statetest.json"},
			wantStdout: "./testdata/evmrun/6.out.1.txt",
			wantStderr: "./testdata/evmrun/6.out.2.txt",
		},
		{ // statetest subcommand with output
			input:      []string{"statetest", "--trace", "--trace.format=md", "./testdata/statetest.json"},
			wantStdout: "./testdata/evmrun/7.out.1.txt",
			wantStderr: "./testdata/evmrun/7.out.2.txt",
		},
		{ // statetest subcommand with output
			input:      []string{"statetest", "--trace", "--trace.format=json", "./testdata/statetest.json"},
			wantStdout: "./testdata/evmrun/8.out.1.txt",
			wantStderr: "./testdata/evmrun/8.out.2.txt",
		},
	} {
		tt.Logf("args: go run ./cmd/evm %v\n", strings.Join(tc.input, " "))
		tt.Run("evm-test", tc.input...)

		haveStdOut := tt.Output()
		tt.WaitExit()
		haveStdErr := tt.StderrText()

		if have, wantFile := haveStdOut, tc.wantStdout; wantFile != "" {
			want, err := os.ReadFile(wantFile)
			if err != nil {
				t.Fatalf("test %d: could not read expected output: %v", i, err)
			}
			if string(haveStdOut) != string(want) {
				t.Fatalf("test %d, output wrong, have \n%v\nwant\n%v\n", i, string(have), string(want))
			}
		}
		if have, wantFile := haveStdErr, tc.wantStderr; wantFile != "" {
			want, err := os.ReadFile(wantFile)
			if err != nil {
				t.Fatalf("test %d: could not read expected output: %v", i, err)
			}
			if have != string(want) {
				t.Fatalf("test %d, output wrong\nhave %q\nwant %q\n", i, have, string(want))
			}
		}
	}
}

// balBlockHash is the hash of the only block in testdata/blocktest_bal.json and
// testdata/enginetest_bal.json.
var balBlockHash = common.HexToHash("0xd56692af6b9fc93c12c1baa7dd03339397ca040c712bcba5c41e93dd12959d74")

// TestBlockAccessListExecution checks that blocktest and enginetest run a block
// carrying an EIP-7928 access list on the parallel processor, or on the
// sequential one under --bal-sequential or when blocktest withholds the list,
// and report the choice on stderr only under --bal-report.
func TestBlockAccessListExecution(t *testing.T) {
	t.Parallel()
	for i, tc := range []struct {
		input      []string
		wantPath   string // empty when no event line is expected
		wantReason string
	}{
		{[]string{"blocktest", "--bal-report", "./testdata/blocktest_bal.json"}, "parallel", ""},
		{[]string{"blocktest", "--bal-report", "--bal-sequential", "./testdata/blocktest_bal.json"}, "sequential", "disabled"},
		{[]string{"blocktest", "--bal-report", "--bal-withhold", "./testdata/blocktest_bal.json"}, "sequential", "withheld"},
		{[]string{"blocktest", "--bal-report", "--bal-withhold", "--bal-sequential", "./testdata/blocktest_bal.json"}, "sequential", "withheld"},
		{[]string{"blocktest", "./testdata/blocktest_bal.json"}, "", ""},
		{[]string{"enginetest", "--bal-report", "./testdata/enginetest_bal.json"}, "parallel", ""},
		{[]string{"enginetest", "--bal-report", "--bal-sequential", "./testdata/enginetest_bal.json"}, "sequential", "disabled"},
		{[]string{"enginetest", "./testdata/enginetest_bal.json"}, "", ""},
	} {
		tt := cmdtest.NewTestCmd(t, nil)
		tt.Run("evm-test", tc.input...)
		stdout := tt.Output()
		tt.WaitExit()
		stderr := tt.StderrText()

		var results []testResult
		if err := json.Unmarshal(stdout, &results); err != nil {
			t.Fatalf("test %d: stdout is not a JSON result list: %v\n%s", i, err, stdout)
		}
		if len(results) != 1 || !results[0].Pass {
			t.Fatalf("test %d: unexpected results: %s", i, stdout)
		}
		events := balEvents(t, stderr)
		if tc.wantPath == "" {
			if len(events) != 0 {
				t.Fatalf("test %d: events %+v without --bal-report, want none", i, events)
			}
			continue
		}
		want := executionEvent{Event: "balExecution", Block: 1, Hash: balBlockHash, Path: tc.wantPath, Reason: tc.wantReason}
		if len(events) != 1 || events[0] != want {
			t.Fatalf("test %d: events %+v, want [%+v]", i, events, want)
		}
	}
}

// TestBlockAccessListDropped checks that blocktest drops a delivered access
// list that differs from the one the header commits to, in either fixture
// field, and imports the block with the list computed in execution, as the
// downloader does with a peer's list. The report names the dropped list in
// both modes.
func TestBlockAccessListDropped(t *testing.T) {
	t.Parallel()
	src, err := os.ReadFile("./testdata/blocktest_bal.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"blockAccessList", "rlp_decoded"} {
		var fixtures map[string]map[string]any
		if err := json.Unmarshal(src, &fixtures); err != nil {
			t.Fatal(err)
		}
		// Deliver the fixture's only block with one account missing from its
		// access list, leaving the block and its expected post state alone.
		for _, test := range fixtures {
			block := test["blocks"].([]any)[0].(map[string]any)
			list := block["blockAccessList"].([]any)
			tampered := list[:len(list)-1]
			if field == "rlp_decoded" {
				delete(block, "blockAccessList")
				block["rlp_decoded"] = map[string]any{"blockAccessList": tampered}
			} else {
				block["blockAccessList"] = tampered
			}
		}
		out, err := json.Marshal(fixtures)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "blocktest_bal_dropped.json")
		if err := os.WriteFile(path, out, 0644); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			args       []string
			wantPath   string
			wantReason string
		}{
			{[]string{"blocktest", "--bal-report", path}, "sequential", "bad-access-list"},
			{[]string{"blocktest", "--bal-report", "--bal-sequential", path}, "sequential", "bad-access-list"},
		} {
			tt := cmdtest.NewTestCmd(t, nil)
			tt.Run("evm-test", tc.args...)
			stdout := tt.Output()
			tt.WaitExit()
			stderr := tt.StderrText()

			var results []testResult
			if err := json.Unmarshal(stdout, &results); err != nil {
				t.Fatalf("%s %v: stdout is not a JSON result list: %v\n%s", field, tc.args, err, stdout)
			}
			if len(results) != 1 || !results[0].Pass {
				t.Fatalf("%s %v: block with a dropped access list not imported: %s", field, tc.args, stdout)
			}
			events := balEvents(t, stderr)
			want := executionEvent{Event: "balExecution", Block: 1, Hash: balBlockHash, Path: tc.wantPath, Reason: tc.wantReason}
			if len(events) != 1 || events[0] != want {
				t.Fatalf("%s %v: events %+v, want [%+v]", field, tc.args, events, want)
			}
		}
	}
}

// TestBlockAccessListWithheldWithoutList checks that --bal-withhold reports
// withheld only for a block delivered with a list: a block without one keeps
// the reason core gives it, as it does without the flag.
func TestBlockAccessListWithheldWithoutList(t *testing.T) {
	t.Parallel()
	src, err := os.ReadFile("./testdata/blocktest_bal.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]map[string]any
	if err := json.Unmarshal(src, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, test := range fixtures {
		delete(test["blocks"].([]any)[0].(map[string]any), "blockAccessList")
	}
	out, err := json.Marshal(fixtures)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "blocktest_bal_without_list.json")
	if err := os.WriteFile(path, out, 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"blocktest", "--bal-report", path},
		{"blocktest", "--bal-report", "--bal-withhold", path},
	} {
		tt := cmdtest.NewTestCmd(t, nil)
		tt.Run("evm-test", args...)
		stdout := tt.Output()
		tt.WaitExit()
		stderr := tt.StderrText()

		var results []testResult
		if err := json.Unmarshal(stdout, &results); err != nil {
			t.Fatalf("%v: stdout is not a JSON result list: %v\n%s", args, err, stdout)
		}
		if len(results) != 1 || !results[0].Pass {
			t.Fatalf("%v: unexpected results: %s", args, stdout)
		}
		events := balEvents(t, stderr)
		want := executionEvent{Event: "balExecution", Block: 1, Hash: balBlockHash, Path: "sequential", Reason: "no-access-list"}
		if len(events) != 1 || events[0] != want {
			t.Fatalf("%v: events %+v, want [%+v]", args, events, want)
		}
	}
}

// TestBlockAccessListDroppedTwins checks that the report ties a dropped list to
// the import that dropped it. Fixtures run in parallel import the same block,
// some delivering its own list and some a tampered one, so every clean import
// must report the parallel path and every tampered one bad-access-list.
func TestBlockAccessListDroppedTwins(t *testing.T) {
	t.Parallel()
	src, err := os.ReadFile("./testdata/blocktest_bal.json")
	if err != nil {
		t.Fatal(err)
	}
	const pairs = 4
	dir := t.TempDir()
	for i := 0; i < 2*pairs; i++ {
		var fixtures map[string]map[string]any
		if err := json.Unmarshal(src, &fixtures); err != nil {
			t.Fatal(err)
		}
		renamed := make(map[string]map[string]any)
		for name, test := range fixtures {
			if i%2 == 1 {
				block := test["blocks"].([]any)[0].(map[string]any)
				list := block["blockAccessList"].([]any)
				block["blockAccessList"] = list[:len(list)-1]
			}
			renamed[fmt.Sprintf("%s-%d", name, i)] = test
		}
		out, err := json.Marshal(renamed)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("twin%d.json", i)), out, 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		args      []string
		wantClean executionEvent
	}{
		{[]string{"blocktest", "--bal-report", "--workers", "8", dir}, executionEvent{Path: "parallel"}},
		{[]string{"blocktest", "--bal-report", "--bal-sequential", "--workers", "8", dir}, executionEvent{Path: "sequential", Reason: "disabled"}},
	} {
		tt := cmdtest.NewTestCmd(t, nil)
		tt.Run("evm-test", tc.args...)
		stdout := tt.Output()
		tt.WaitExit()
		stderr := tt.StderrText()

		var results []testResult
		if err := json.Unmarshal(stdout, &results); err != nil {
			t.Fatalf("%v: stdout is not a JSON result list: %v\n%s", tc.args, err, stdout)
		}
		if len(results) != 2*pairs {
			t.Fatalf("%v: %d results, want %d", tc.args, len(results), 2*pairs)
		}
		for _, r := range results {
			if !r.Pass {
				t.Fatalf("%v: %s failed: %s", tc.args, r.Name, r.Error)
			}
		}
		counts := make(map[executionEvent]int)
		for _, event := range balEvents(t, stderr) {
			counts[executionEvent{Path: event.Path, Reason: event.Reason}]++
		}
		want := map[executionEvent]int{
			tc.wantClean: pairs,
			{Path: "sequential", Reason: "bad-access-list"}: pairs,
		}
		if !reflect.DeepEqual(counts, want) {
			t.Fatalf("%v: decision lines %v, want %v", tc.args, counts, want)
		}
	}
}

// balEvents returns the execution events a runner printed on stderr under
// --bal-report.
func balEvents(t *testing.T, stderr string) []executionEvent {
	t.Helper()
	var events []executionEvent
	for _, line := range strings.Split(stderr, "\n") {
		if !strings.Contains(line, `"event"`) {
			continue
		}
		var event executionEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("bad event line %q: %v", line, err)
		}
		events = append(events, event)
	}
	return events
}

// TestRunnersTakeSeveralPaths checks that the test runners run every path they
// are given, not only the first.
func TestRunnersTakeSeveralPaths(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		runner string
		files  []string
	}{
		{"blocktest", []string{"./testdata/blocktest_bal.json", "./testdata/blocktest_exception.json"}},
		{"enginetest", []string{"./testdata/enginetest_bal.json", "./testdata/enginetest_exception.json"}},
	} {
		count := func(args ...string) int {
			tt := cmdtest.NewTestCmd(t, nil)
			tt.Run("evm-test", append([]string{tc.runner}, args...)...)
			stdout := tt.Output()
			tt.WaitExit()
			var results []testResult
			if err := json.Unmarshal(stdout, &results); err != nil {
				t.Fatalf("%s %v: stdout is not a JSON result list: %v\n%s", tc.runner, args, err, stdout)
			}
			return len(results)
		}
		want := count(tc.files[0]) + count(tc.files[1])
		if have := count(tc.files...); have != want {
			t.Errorf("%s with %d paths: have %d results, want %d", tc.runner, len(tc.files), have, want)
		}
	}
}

// TestRunnersRejectBadPaths checks that a run that cannot run all of its input
// exits 2 and names the bad path on stderr: a path that does not exist stops
// the run before any test runs, and a file that is not valid JSON is reported
// while the other files still run and print their results.
func TestRunnersRejectBadPaths(t *testing.T) {
	t.Parallel()
	corrupt := filepath.Join(t.TempDir(), "corrupt.json")
	if err := os.WriteFile(corrupt, []byte(`{"truncated": `), 0644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		runner string
		file   string
	}{
		{"blocktest", "./testdata/blocktest_bal.json"},
		{"enginetest", "./testdata/enginetest_bal.json"},
	} {
		for _, bad := range []struct {
			path    string
			results int // results printed for the good file
		}{
			{"./testdata/does-not-exist.json", 0},
			{corrupt, 1},
		} {
			tt := cmdtest.NewTestCmd(t, nil)
			tt.Run("evm-test", tc.runner, tc.file, bad.path)
			stdout := tt.Output()
			tt.WaitExit()
			if status := tt.ExitStatus(); status != 2 {
				t.Errorf("%s %s: exit status %d, want 2", tc.runner, bad.path, status)
			}
			if bad.results == 0 && len(stdout) != 0 {
				t.Errorf("%s %s: results printed despite a missing path:\n%s", tc.runner, bad.path, stdout)
			}
			if bad.results > 0 {
				var results []testResult
				if err := json.Unmarshal(stdout, &results); err != nil || len(results) != bad.results {
					t.Errorf("%s %s: want %d results for the good file, err %v:\n%s", tc.runner, bad.path, bad.results, err, stdout)
				}
			}
			if stderr := tt.StderrText(); !strings.Contains(stderr, filepath.Base(bad.path)) {
				t.Errorf("%s %s: stderr does not name the bad path:\n%s", tc.runner, bad.path, stderr)
			}
		}
	}
}

// TestRunnersSkipMetaDirectory checks that a fixture directory's .meta folder,
// which holds files that are not fixtures, is not run.
func TestRunnersSkipMetaDirectory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		runner string
		file   string
	}{
		{"blocktest", "./testdata/blocktest_bal.json"},
		{"enginetest", "./testdata/enginetest_bal.json"},
	} {
		dir := t.TempDir()
		src, err := os.ReadFile(tc.file)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(tc.file)), src, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(dir, ".meta"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".meta", "index.json"), []byte(`{"test_cases": []}`), 0644); err != nil {
			t.Fatal(err)
		}
		tt := cmdtest.NewTestCmd(t, nil)
		tt.Run("evm-test", tc.runner, dir)
		stdout := tt.Output()
		tt.WaitExit()
		var results []testResult
		if err := json.Unmarshal(stdout, &results); err != nil {
			t.Fatalf("%s: stdout is not a JSON result list: %v\n%s\n%s", tc.runner, err, stdout, tt.StderrText())
		}
		if len(results) == 0 {
			t.Errorf("%s: no results for the fixture beside .meta", tc.runner)
		}
	}
}

// TestRunnersDisablePrecompileCache checks that blocktest and enginetest give
// the chain the shared precompile result cache unless --cache.noprecompile
// turns it off.
func TestRunnersDisablePrecompileCache(t *testing.T) {
	t.Parallel()
	load := func(path string, into any) {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(src, into); err != nil {
			t.Fatal(err)
		}
	}
	for _, noPrecompileCache := range []bool{false, true} {
		// One entry per run whose chain was built, true if it had the cache.
		var cached []bool
		record := func(_ error, chain *core.BlockChain) {
			cached = append(cached, chain.PrecompileCache() != nil)
		}
		var blockTests map[string]*tests.BlockTest
		load("./testdata/blocktest_bal.json", &blockTests)
		for name, test := range blockTests {
			if err := test.Run(false, rawdb.PathScheme, false, nil, tests.RunConfig{NoPrecompileCache: noPrecompileCache}, record); err != nil {
				t.Fatalf("blocktest %s: %v", name, err)
			}
		}
		var engineTests map[string]*tests.EngineTest
		load("./testdata/enginetest_bal.json", &engineTests)
		for name, test := range engineTests {
			if err := test.Run(rawdb.PathScheme, nil, tests.RunConfig{NoPrecompileCache: noPrecompileCache}, attachEngineAPI, record); err != nil {
				t.Fatalf("enginetest %s: %v", name, err)
			}
		}
		want := []bool{!noPrecompileCache, !noPrecompileCache}
		if !reflect.DeepEqual(cached, want) {
			t.Errorf("noPrecompileCache %v: cache present in blocktest, enginetest: have %v, want %v", noPrecompileCache, cached, want)
		}
	}
}

// TestRunnersReportRejections checks that blocktest and enginetest report
// every rejected block with geth's own error and its index in the fixture, and
// that the error does not decide whether the test passes: the expected
// exceptions here name a different reason, and the tests still pass. A block
// whose delivered access list matches its header is run with that list even
// when it is then rejected.
func TestRunnersReportRejections(t *testing.T) {
	t.Parallel()
	const unrelated = "TransactionException.INSUFFICIENT_ACCOUNT_FUNDS"
	rejectedHash := common.HexToHash("0x1b8e8678c0b5d3ec4ad6873b4739a7bd7da73d2af166ea4672140d83b38cf244")
	const bad = "access list hash mismatch, local: 79fa24e990772eea49e38869e0148edec3eac2cb729d30899d869c3b8037f293, remote: bad19914b7e4665d1d7361029751c983b19b6090835c77d8138ae209e16d1c03"
	for _, tc := range []struct {
		runner, list string
		// addRejected changes the fixture's invalid entry to expect an
		// unrelated exception and appends a second one the client rejects.
		addRejected func(entries []any) []any
		want        []tests.Rejection
	}{
		{
			runner: "blocktest",
			list:   "blocks",
			addRejected: func(blocks []any) []any {
				blocks[0].(map[string]any)["expectException"] = unrelated
				return append(blocks, map[string]any{"rlp": "0xc0", "expectException": unrelated})
			},
			want: []tests.Rejection{
				{Index: 0, Hash: &rejectedHash, Error: bad},
				{Index: 1, Error: "rlp: too few elements for types.extblock"},
			},
		},
		{
			runner: "enginetest",
			list:   "engineNewPayloads",
			addRejected: func(payloads []any) []any {
				first := payloads[0].(map[string]any)
				first["validationError"] = unrelated
				// The same payload through engine_newPayloadV4, whose params do
				// not fit Amsterdam, fails with a JSON-RPC error whose data
				// names the cause.
				second := maps.Clone(first)
				delete(second, "validationError")
				second["newPayloadVersion"] = "4"
				second["errorCode"] = "-32602"
				return append(payloads, second)
			},
			want: []tests.Rejection{
				{Index: 0, Error: bad},
				{Index: 1, Error: `-32602: Invalid parameters: {"err":"slotNumber not supported pre-amsterdam"}`},
			},
		},
	} {
		run := func(path string) (testResult, []executionEvent) {
			tt := cmdtest.NewTestCmd(t, nil)
			tt.Run("evm-test", tc.runner, "--bal-report", path)
			stdout := tt.Output()
			tt.WaitExit()
			var results []testResult
			if err := json.Unmarshal(stdout, &results); err != nil || len(results) != 1 {
				t.Fatalf("%s: want one JSON result, err %v:\n%s", tc.runner, err, stdout)
			}
			if !strings.Contains(string(stdout), `"rejections": [`) {
				t.Fatalf("%s: result has no rejections list:\n%s", tc.runner, stdout)
			}
			return results[0], balEvents(t, tt.StderrText())
		}
		if res, _ := run(fmt.Sprintf("./testdata/%s_bal.json", tc.runner)); !res.Pass || len(res.Rejections) != 0 {
			t.Errorf("%s clean fixture: pass %v, rejections %+v", tc.runner, res.Pass, res.Rejections)
		}

		src, err := os.ReadFile(fmt.Sprintf("./testdata/%s_exception.json", tc.runner))
		if err != nil {
			t.Fatal(err)
		}
		var fixtures map[string]map[string]any
		if err := json.Unmarshal(src, &fixtures); err != nil {
			t.Fatal(err)
		}
		for _, fixture := range fixtures {
			fixture[tc.list] = tc.addRejected(fixture[tc.list].([]any))
		}
		out, err := json.Marshal(fixtures)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), tc.runner+".json")
		if err := os.WriteFile(path, out, 0644); err != nil {
			t.Fatal(err)
		}
		res, events := run(path)
		if !res.Pass {
			t.Errorf("%s: an expected rejection for another reason failed the test: %s", tc.runner, res.Error)
		}
		if !reflect.DeepEqual(res.Rejections, tc.want) {
			t.Errorf("%s: wrong rejections\nhave %+v\nwant %+v", tc.runner, res.Rejections, tc.want)
		}
		// The first rejected block's delivered list is the one its header
		// commits to, so it is attached and the parallel processor runs, and
		// rejects, the block.
		want := executionEvent{Event: "balExecution", Block: 1, Hash: rejectedHash, Path: "parallel"}
		if len(events) != 1 || events[0] != want {
			t.Errorf("%s: events %+v, want [%+v]", tc.runner, events, want)
		}
	}
}

// TestRunnersExitStatus checks that blocktest and enginetest exit 0 when every
// fixture passes and 1 when one fails, and that stdout still holds only the
// JSON results.
func TestRunnersExitStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		runner, list string
		// fail makes the fixture expect its only, valid, entry to be rejected.
		fail func(entry map[string]any)
	}{
		{
			runner: "blocktest",
			list:   "blocks",
			fail: func(block map[string]any) {
				delete(block, "blockHeader")
				block["expectException"] = "BlockException.INVALID_BLOCK_ACCESS_LIST"
			},
		},
		{
			runner: "enginetest",
			list:   "engineNewPayloads",
			fail: func(payload map[string]any) {
				payload["validationError"] = "BlockException.INVALID_BLOCK_ACCESS_LIST"
			},
		},
	} {
		run := func(path string) (int, []testResult) {
			tt := cmdtest.NewTestCmd(t, nil)
			tt.Run("evm-test", tc.runner, path)
			stdout := tt.Output()
			tt.WaitExit()
			var results []testResult
			if err := json.Unmarshal(stdout, &results); err != nil {
				t.Fatalf("%s: stdout is not a JSON result list: %v\n%s", tc.runner, err, stdout)
			}
			return tt.ExitStatus(), results
		}
		clean := fmt.Sprintf("./testdata/%s_bal.json", tc.runner)
		if status, results := run(clean); status != 0 || len(results) != 1 || !results[0].Pass {
			t.Errorf("%s clean fixture: exit status %d, results %+v", tc.runner, status, results)
		}

		src, err := os.ReadFile(clean)
		if err != nil {
			t.Fatal(err)
		}
		var fixtures map[string]map[string]any
		if err := json.Unmarshal(src, &fixtures); err != nil {
			t.Fatal(err)
		}
		for _, fixture := range fixtures {
			tc.fail(fixture[tc.list].([]any)[0].(map[string]any))
		}
		out, err := json.Marshal(fixtures)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), tc.runner+".json")
		if err := os.WriteFile(path, out, 0644); err != nil {
			t.Fatal(err)
		}
		if status, results := run(path); status != 1 || len(results) != 1 || results[0].Pass {
			t.Errorf("%s failing fixture: exit status %d, results %+v", tc.runner, status, results)
		}
	}
}

// TestEvmVersion checks that evm --version prints one line and exits 0.
func TestEvmVersion(t *testing.T) {
	t.Parallel()
	tt := cmdtest.NewTestCmd(t, nil)
	tt.Run("evm-test", "--version")
	stdout := tt.Output()
	tt.WaitExit()
	if status := tt.ExitStatus(); status != 0 {
		t.Errorf("exit status %d", status)
	}
	if lines := strings.Split(strings.TrimSpace(string(stdout)), "\n"); len(lines) != 1 || !strings.Contains(lines[0], " version ") {
		t.Errorf("want one version line, have:\n%s", stdout)
	}
}

func TestEvmRunRegEx(t *testing.T) {
	t.Parallel()
	tt := cmdtest.NewTestCmd(t, nil)
	for i, tc := range []struct {
		input      []string
		wantStdout string
		wantStderr string
	}{
		{ // json tracing
			input:      []string{"run", "--bench", "6040"},
			wantStdout: "./testdata/evmrun/9.out.1.txt",
			wantStderr: "./testdata/evmrun/9.out.2.txt",
		},
		{ // statetest subcommand
			input:      []string{"statetest", "--bench", "./testdata/statetest.json"},
			wantStdout: "./testdata/evmrun/10.out.1.txt",
			wantStderr: "./testdata/evmrun/10.out.2.txt",
		},
	} {
		tt.Logf("args: go run ./cmd/evm %v\n", strings.Join(tc.input, " "))
		tt.Run("evm-test", tc.input...)

		haveStdOut := tt.Output()
		tt.WaitExit()
		haveStdErr := tt.StderrText()

		if have, wantFile := haveStdOut, tc.wantStdout; wantFile != "" {
			want, err := os.ReadFile(wantFile)
			if err != nil {
				t.Fatalf("test %d: could not read expected output: %v", i, err)
			}
			re, err := regexp.Compile(string(want))
			if err != nil {
				t.Fatalf("test %d: could not compile regular expression: %v", i, err)
			}
			if !re.Match(have) {
				t.Fatalf("test %d, output wrong, have \n%v\nwant\n%v\n", i, string(have), re)
			}
		}
		if have, wantFile := haveStdErr, tc.wantStderr; wantFile != "" {
			want, err := os.ReadFile(wantFile)
			if err != nil {
				t.Fatalf("test %d: could not read expected output: %v", i, err)
			}
			re, err := regexp.Compile(string(want))
			if err != nil {
				t.Fatalf("test %d: could not compile regular expression: %v", i, err)
			}
			if !re.MatchString(have) {
				t.Fatalf("test %d, output wrong, have \n%v\nwant\n%v\n", i, have, re)
			}
		}
	}
}

// cmpJson compares the JSON in two byte slices.
func cmpJson(a, b []byte) (bool, error) {
	var j, j2 interface{}
	if err := json.Unmarshal(a, &j); err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, &j2); err != nil {
		return false, err
	}
	return reflect.DeepEqual(j2, j), nil
}

// TestEVMTracing is a test that checks the tracing-output from evm.
func TestEVMTracing(t *testing.T) {
	t.Parallel()
	tt := cmdtest.NewTestCmd(t, nil)
	for i, tc := range []struct {
		base           string
		input          []string
		expectedTraces []string
	}{
		{
			base: "./testdata/31",
			input: []string{"t8n",
				"--input.alloc=./testdata/31/alloc.json", "--input.txs=./testdata/31/txs.json",
				"--input.env=./testdata/31/env.json", "--state.fork=Cancun",
				"--trace",
			},
			//expectedTraces: []string{"trace-0-0x88f5fbd1524731a81e49f637aa847543268a5aaf2a6b32a69d2c6d978c45dcfb.jsonl"},
			expectedTraces: []string{"trace-0-0x88f5fbd1524731a81e49f637aa847543268a5aaf2a6b32a69d2c6d978c45dcfb.jsonl",
				"trace-1-0x03a7b0a91e61a170d64ea94b8263641ef5a8bbdb10ac69f466083a6789c77fb8.jsonl",
				"trace-2-0xd96e0ce6418ee3360e11d3c7b6886f5a9a08f7ef183da72c23bb3b2374530128.jsonl"},
		},
		{
			base: "./testdata/31",
			input: []string{"t8n",
				"--input.alloc=./testdata/31/alloc.json", "--input.txs=./testdata/31/txs.json",
				"--input.env=./testdata/31/env.json", "--state.fork=Cancun",
				"--trace.tracer", `
{   count: 0,
	result: function(){
		this.count = this.count + 1;
		return "hello world " + this.count
	},
	fault: function(){}
}`,
			},
			expectedTraces: []string{"trace-0-0x88f5fbd1524731a81e49f637aa847543268a5aaf2a6b32a69d2c6d978c45dcfb.json",
				"trace-1-0x03a7b0a91e61a170d64ea94b8263641ef5a8bbdb10ac69f466083a6789c77fb8.json",
				"trace-2-0xd96e0ce6418ee3360e11d3c7b6886f5a9a08f7ef183da72c23bb3b2374530128.json"},
		},
		{
			base: "./testdata/32",
			input: []string{"t8n",
				"--input.alloc=./testdata/32/alloc.json", "--input.txs=./testdata/32/txs.json",
				"--input.env=./testdata/32/env.json", "--state.fork=Paris",
				"--trace", "--trace.callframes",
			},
			expectedTraces: []string{"trace-0-0x47806361c0fa084be3caa18afe8c48156747c01dbdfc1ee11b5aecdbe4fcf23e.jsonl"},
		},
		// TODO, make it possible to run tracers on statetests, e.g:
		//{
		//			base: "./testdata/31",
		//			input: []string{"statetest", "--trace", "--trace.tracer", `{
		//	result: function(){
		//		return "hello world"
		//	},
		//	fault: function(){}
		//}`, "./testdata/statetest.json"},
		//			expectedTraces: []string{"trace-0-0x88f5fbd1524731a81e49f637aa847543268a5aaf2a6b32a69d2c6d978c45dcfb.json"},
		//		},
	} {
		// Place the output somewhere we can find it
		outdir := t.TempDir()
		args := append(tc.input, "--output.basedir", outdir)

		tt.Run("evm-test", args...)
		tt.Logf("args: go run ./cmd/evm %v\n", args)
		tt.WaitExit()
		//t.Log(string(tt.Output()))

		// Compare the expected traces
		for _, traceFile := range tc.expectedTraces {
			haveFn := lineIterator(filepath.Join(outdir, traceFile))
			wantFn := lineIterator(filepath.Join(tc.base, traceFile))

			for line := 0; ; line++ {
				want, wErr := wantFn()
				have, hErr := haveFn()
				if want != have {
					t.Fatalf("test %d, trace %v, line %d\nwant: %v\nhave: %v\n",
						i, traceFile, line, want, have)
				}
				if wErr != nil && hErr != nil {
					break
				}
				if wErr != nil {
					t.Fatal(wErr)
				}
				if hErr != nil {
					t.Fatal(hErr)
				}
				//t.Logf("%v\n", want)
			}
		}
	}
}

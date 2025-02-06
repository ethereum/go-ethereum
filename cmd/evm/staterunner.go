// Copyright 2017 The go-ethereum Authors
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
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/tests"
	"github.com/urfave/cli/v2"
)

var stateTestCommand = &cli.Command{
	Action:    stateTestCmd,
	Name:      "statetest",
	Usage:     "executes the given state tests",
	ArgsUsage: "<file>",
}

type StatetestResult struct {
	Name  string      `json:"name"`
	Pass  bool        `json:"pass"`
	Fork  string      `json:"fork"`
	Error string      `json:"error,omitempty"`
	State *state.Dump `json:"state,omitempty"`
}

func stateTestCmd(ctx *cli.Context) error {
	if len(ctx.Args().First()) == 0 {
		return errors.New("path-to-test argument required")
	}

	// Configure the EVM logger
	config := &vm.LogConfig{
		EnableMemory:     !ctx.Bool(DisableMemoryFlag.Name),
		DisableStack:     ctx.Bool(DisableStackFlag.Name),
		DisableStorage:   ctx.Bool(DisableStorageFlag.Name),
		EnableReturnData: !ctx.Bool(DisableReturnDataFlag.Name),
	}

	var (
		tracer   vm.EVMLogger
		debugger *vm.StructLogger
	)
	switch {
	case ctx.Bool(MachineFlag.Name):
		tracer = vm.NewJSONLogger(config, os.Stderr)

	case ctx.Bool(DebugFlag.Name):
		debugger = vm.NewStructLogger(config)
		tracer = debugger

	default:
		debugger = vm.NewStructLogger(config)
	}
	// Load the test content from the input file
	src, err := os.ReadFile(ctx.Args().First())
	if err != nil {
		return err
	}
	var tests map[string]tests.StateTest
	if err = json.Unmarshal(src, &tests); err != nil {
		return err
	}
	// Iterate over all the tests, run them and aggregate the results
	cfg := vm.Config{
		Tracer: tracer,
	}
	results := make([]StatetestResult, 0, len(tests))
	for key, test := range tests {
		for _, st := range test.Subtests() {
			// Run the test and aggregate the result
			result := &StatetestResult{Name: key, Fork: st.Fork, Pass: true}
			state, err := test.Run(st, cfg)
			if err != nil {
				// Test failed, mark as so and dump any state to aid debugging
				result.Pass, result.Error = false, err.Error()
				if ctx.Bool(DumpFlag.Name) && state != nil {
					dump := state.RawDump()
					result.State = &dump
				}
			}
			// print state root for evmlab tracing (already committed above, so no need to delete objects again
			if ctx.Bool(MachineFlag.Name) && state != nil {
				fmt.Fprintf(os.Stderr, "{\"stateRoot\": \"%x\"}\n", state.IntermediateRoot(false))
			}

			results = append(results, *result)

			// Print any structured logs collected
			if ctx.Bool(DebugFlag.Name) {
				if debugger != nil {
					fmt.Fprintln(os.Stderr, "#### TRACE ####")
					vm.WriteTrace(os.Stderr, debugger.StructLogs())
				}
			}
		}
	}
	out, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(out))
	return nil
}

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
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/tests"

	cli "gopkg.in/urfave/cli.v1"
)

var stateTestCommand = cli.Command{
	Action:    stateTestCmd,
	Name:      "statetest",
	Usage:     "executes the given state tests",
	ArgsUsage: "<file>",
}

type StatetestResult struct {
	Name  string `json:"name"`
	Pass  bool   `json:"pass"`
	Fork  string `json:"fork"`
	Error string `json:"error,omitempty"`
}

func stateTestCmd(ctx *cli.Context) error {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(ctx.GlobalInt(VerbosityFlag.Name)))
	log.Root().SetHandler(glogger)
	logconfig := &vm.LogConfig{
		DisableMemory: ctx.GlobalBool(DisableMemoryFlag.Name),
		DisableStack:  ctx.GlobalBool(DisableStackFlag.Name),
	}
	var (
		tracer      vm.Tracer
		debugLogger *vm.StructLogger
		//		statedb     *state.StateDB
		//		chainConfig *params.ChainConfig
	)
	if ctx.GlobalBool(MachineFlag.Name) {
		tracer = NewJSONLogger(logconfig, os.Stderr)
	} else if ctx.GlobalBool(DebugFlag.Name) {
		debugLogger = vm.NewStructLogger(logconfig)
		tracer = debugLogger
	} else {
		debugLogger = vm.NewStructLogger(logconfig)
	}

	if len(ctx.Args().First()) == 0 {
		return errors.New("filename required")
	}

	fn := ctx.Args().First()
	src, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}

	var tests map[string]tests.StateTest

	if err = json.Unmarshal(src, &tests); err != nil {
		return err
	}

	var results = make([]StatetestResult, 0, len(tests))

	cfg := vm.Config{
		Tracer: tracer,
		Debug:  ctx.GlobalBool(DebugFlag.Name) || ctx.GlobalBool(MachineFlag.Name),
	}

	for key, test := range tests {
		for _, st := range test.Subtests() {
			result := &StatetestResult{
				Name: key,
				Fork: st.Fork,
				Pass: true,
			}

			if err = test.Run(st, cfg); err != nil {
				result.Error = err.Error()
				result.Pass = false
			}
			results = append(results, *result)

			if ctx.GlobalBool(DebugFlag.Name) {
				if debugLogger != nil {
					fmt.Fprintln(os.Stderr, "#### TRACE ####")
					vm.WriteTrace(os.Stderr, debugLogger.StructLogs())
				}
			}
		}
	}
	json.NewEncoder(os.Stdout).Encode(results)

	if err != nil {
		return err
	}
	return nil
}

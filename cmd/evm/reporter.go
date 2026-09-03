// Copyright 2024 The go-ethereum Authors
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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/urfave/cli/v2"
)

const (
	PASS = "\033[32mPASS\033[0m"
	FAIL = "\033[31mFAIL\033[0m"
)

// testResult contains the execution status after running a state test, any
// error that might have occurred and a dump of the final state if requested.
type testResult struct {
	Name  string       `json:"name"`
	Pass  bool         `json:"pass"`
	Root  *common.Hash `json:"stateRoot,omitempty"`
	Fork  string       `json:"fork"`
	Error string       `json:"error,omitempty"`
	State *state.Dump  `json:"state,omitempty"`
	Stats *execStats   `json:"benchStats,omitempty"`
}

func (r testResult) String() string {
	var status string
	if r.Pass {
		status = fmt.Sprintf("[%s]", PASS)
	} else {
		status = fmt.Sprintf("[%s]", FAIL)
	}
	info := r.Name
	m := parseTestMetadata(r.Name)
	if m != nil {
		info = fmt.Sprintf("%s %s, param=%s", m.module, m.function, m.parameters)
	}
	var extra string
	if !r.Pass {
		extra = fmt.Sprintf(", err=%v, fork=%s", r.Error, r.Fork)
	}
	out := fmt.Sprintf("%s %s%s", status, info, extra)
	if r.State != nil {
		state, _ := json.MarshalIndent(r.State, "", "  ")
		out += "\n" + string(state)
	}
	return out
}

// report prints the after-test summary.
func report(ctx *cli.Context, results []testResult) {
	if ctx.Bool(HumanReadableFlag.Name) {
		pass := 0
		for _, r := range results {
			if r.Pass {
				pass++
			}
		}
		for _, r := range results {
			fmt.Println(r)
		}
		fmt.Println("--")
		fmt.Printf("%d tests passed, %d tests failed.\n", pass, len(results)-pass)
		return
	}
	out, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(out))
}

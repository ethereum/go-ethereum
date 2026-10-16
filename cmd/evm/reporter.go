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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/tests"
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

	// Rejections is set by blocktest and enginetest, empty when nothing was
	// rejected. The runner reports each error as is; checking it against
	// the fixture's expected exception is left to the consumer.
	Rejections []tests.Rejection `json:"rejections,omitzero"`
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

// exitIncomplete is the exit status of a blocktest or enginetest run that could
// not run all of its input: a path that does not exist or cannot be read, or a
// fixture file that cannot be parsed. A run that ran everything exits 0 when
// every test passed and 1 when any failed.
const exitIncomplete = 2

// failures returns an error counting the results that did not pass, if any,
// so the run exits 1.
func failures(results []testResult) error {
	var failed int
	for _, r := range results {
		if !r.Pass {
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d tests failed", failed, len(results))
	}
	return nil
}

// executionEvent reports which processor executed a block: the EIP-7928
// parallel one or the sequential one, and for the latter the first condition
// that ruled parallel out.
type executionEvent struct {
	Event  string      `json:"event"`
	Block  uint64      `json:"block"`
	Hash   common.Hash `json:"hash"`
	Path   string      `json:"path"`
	Reason string      `json:"reason"`
}

// executionReporter is a log handler that turns core's per-block "Executing
// block" debug record into a single-line JSON executionEvent on its writer, and
// passes every other record on to the wrapped handler. The runners install it
// only under --bal-report.
//
// Core only sees that a block has no access list, so the reporter asks
// blocktest whether the import running the block dropped its delivered list,
// and reports that as bad-access-list whatever reason core gave, or as
// withheld under --bal-withhold. A block delivered without a list keeps
// core's reason.
type executionReporter struct {
	inner    slog.Handler
	out      io.Writer
	lock     *sync.Mutex
	withheld bool
}

// reportExecution installs an executionReporter in front of the given log
// handler, writing the events to stderr.
func reportExecution(inner slog.Handler, withheld bool) {
	log.SetDefault(log.NewLogger(&executionReporter{inner: inner, out: os.Stderr, lock: new(sync.Mutex), withheld: withheld}))
}

// discardLogs silences logging for --fuzz, keeping the execution events when
// --bal-report is set.
func discardLogs(ctx *cli.Context) {
	if ctx.Bool(BALReportFlag.Name) {
		reportExecution(log.DiscardHandler(), ctx.Bool(BALWithholdFlag.Name))
		return
	}
	log.SetDefault(log.NewLogger(log.DiscardHandler()))
}

func (h *executionReporter) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= log.LevelDebug || h.inner.Enabled(ctx, level)
}

func (h *executionReporter) Handle(ctx context.Context, r slog.Record) error {
	if r.Message != "Executing block" {
		if !h.inner.Enabled(ctx, r.Level) {
			return nil
		}
		return h.inner.Handle(ctx, r)
	}
	event := executionEvent{Event: "balExecution"}
	r.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "number":
			event.Block, _ = a.Value.Any().(uint64)
		case "hash":
			event.Hash, _ = a.Value.Any().(common.Hash)
		case "path":
			event.Path = a.Value.String()
		case "reason":
			event.Reason = a.Value.String()
		}
		return true
	})
	if tests.AccessListDropped(event.Hash) {
		event.Reason = "bad-access-list"
		if h.withheld {
			event.Reason = "withheld"
		}
	}
	out, err := json.Marshal(event)
	if err != nil {
		return err
	}
	h.lock.Lock()
	defer h.lock.Unlock()
	_, err = fmt.Fprintln(h.out, string(out))
	return err
}

func (h *executionReporter) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &executionReporter{inner: h.inner.WithAttrs(attrs), out: h.out, lock: h.lock, withheld: h.withheld}
}

func (h *executionReporter) WithGroup(name string) slog.Handler {
	return &executionReporter{inner: h.inner.WithGroup(name), out: h.out, lock: h.lock, withheld: h.withheld}
}

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

package t8ntool

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
)

// fileWritingTracer wraps either a tracer or a logger. On tx start,
// it instantiates a tracer/logger, creates a new file to direct output to,
// and on tx end it closes the file.
type fileWritingTracer struct {
	txIndex     int            // transaction counter
	inner       *tracing.Hooks // inner hooks
	destination io.WriteCloser // the currently open file (if any)
	baseDir     string         // baseDir to write output-files to
	suffix      string         // suffix is the suffix to use when creating files

	// for custom tracing
	getResult func() (json.RawMessage, error)
}

func (l *fileWritingTracer) Write(p []byte) (n int, err error) {
	if l.destination != nil {
		return l.destination.Write(p)
	}
	log.Warn("Tracer wrote to non-existing output")
	// It is tempting to return an error here, however, the json encoder
	// will no retry writing to an io.Writer once it has returned an error once.
	// Therefore, we must squash the error.
	return n, nil
}

// newFileWriter creates a set of hooks which wraps inner hooks (typically a logger),
// and writes the output to a file, one file per transaction.
func newFileWriter(baseDir string, innerFn func(out io.Writer) *tracing.Hooks) *tracing.Hooks {
	t := &fileWritingTracer{
		baseDir: baseDir,
		suffix:  "jsonl",
	}
	t.inner = innerFn(t) // instantiate the inner tracer
	return t.hooks()
}

// newResultWriter creates a set of hooks wraps and invokes an underlying tracer,
// and writes the result (getResult-output) to file, one per transaction.
func newResultWriter(baseDir string, tracer *tracers.Tracer) *tracing.Hooks {
	t := &fileWritingTracer{
		baseDir:   baseDir,
		getResult: tracer.GetResult,
		inner:     tracer.Hooks,
		suffix:    "json",
	}
	return t.hooks()
}

// OnTxStart creates a new output-file specific for this transaction, and invokes
// the inner OnTxStart handler.
func (l *fileWritingTracer) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	// Open a new file, or print a warning log if it's failed
	fname := filepath.Join(l.baseDir, fmt.Sprintf("trace-%d-%v.%v", l.txIndex, tx.Hash().String(), l.suffix))
	traceFile, err := os.Create(fname)
	if err != nil {
		log.Warn("Failed creating trace-file", "err", err)
	} else {
		log.Info("Created tracing-file", "path", fname)
		l.destination = traceFile
	}
	if l.inner != nil && l.inner.OnTxStart != nil {
		l.inner.OnTxStart(env, tx, from)
	}
}

// OnTxEnd writes result (if getResult exist), closes any currently open output-file,
// and invokes the inner OnTxEnd handler.
func (l *fileWritingTracer) OnTxEnd(receipt *types.Receipt, err error) {
	if l.inner != nil && l.inner.OnTxEnd != nil {
		l.inner.OnTxEnd(receipt, err)
	}
	if l.getResult != nil && l.destination != nil {
		if result, err := l.getResult(); result != nil {
			json.NewEncoder(l.destination).Encode(result)
		} else {
			log.Warn("Error obtaining tracer result", "err", err)
		}
		l.destination.Close()
		l.destination = nil
	}
	l.txIndex++
}

func (l *fileWritingTracer) hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnTxStart: l.OnTxStart,
		OnTxEnd:   l.OnTxEnd,
		OnEnter: func(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
			if l.inner != nil && l.inner.OnEnter != nil {
				l.inner.OnEnter(depth, typ, from, to, input, gas, value)
			}
		},
		OnExit: func(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
			if l.inner != nil && l.inner.OnExit != nil {
				l.inner.OnExit(depth, output, gasUsed, err, reverted)
			}
		},
		OnOpcode: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
			if l.inner != nil && l.inner.OnOpcode != nil {
				l.inner.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
			}
		},
		OnFault: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
			if l.inner != nil && l.inner.OnFault != nil {
				l.inner.OnFault(pc, op, gas, cost, scope, depth, err)
			}
		},
		OnSystemCallStart: func() {
			if l.inner != nil && l.inner.OnSystemCallStart != nil {
				l.inner.OnSystemCallStart()
			}
		},
		OnSystemCallEnd: func() {
			if l.inner != nil && l.inner.OnSystemCallEnd != nil {
				l.inner.OnSystemCallEnd()
			}
		},
	}
}

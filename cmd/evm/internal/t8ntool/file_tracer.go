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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
)

// fileWritingTracer wraps either a tracer or a logger. On tx start,
// it instantiates a tracer/logger, creates a new file to direct output to,
// and on tx end it closes the file.
type fileWritingTracer struct {
	txIndex     int
	inner       *tracing.Hooks
	destination io.WriteCloser
	baseDir     string

	// for json-tracing
	logConfig *logger.Config

	// for custom tracing
	tracerName  string
	tracerConf  json.RawMessage
	chainConfig *params.ChainConfig
	getResult   func() (json.RawMessage, error)
}

// jsonToFile creates hooks which uses an underlying jsonlogger, and writes the
// jsonl-delimited output to a file, one per tx.
func jsonToFile(baseDir string, logConfig *logger.Config, callFrames bool) *tracing.Hooks {
	t := &fileWritingTracer{
		baseDir:   baseDir,
		logConfig: logConfig,
	}
	hooks := t.hooks()
	if !callFrames {
		hooks.OnEnter = nil
	}
	return hooks
}

// tracerToFile creates hooks which uses an underlying tracer, and writes the
// json-result to file, one per tx.
func tracerToFile(baseDir, tracerName string, traceConfig json.RawMessage, chainConfig *params.ChainConfig) *tracing.Hooks {
	t := &fileWritingTracer{
		baseDir:     baseDir,
		tracerName:  tracerName,
		chainConfig: chainConfig,
	}
	return t.hooks()
}

func (l *fileWritingTracer) hooks() *tracing.Hooks {
	hooks := &tracing.Hooks{
		OnTxStart: l.OnTxStartJSONL,
		OnTxEnd:   l.OnTxEnd,
		// intentional no-op: we instantiate the l.inner on tx start, which has
		// not yet happened at this point
		//OnSystemCallStart: func() {},
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
			if l.inner.OnOpcode != nil {
				l.inner.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
			}
		},
		OnFault: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
			if l.inner != nil && l.inner.OnFault != nil {
				l.inner.OnFault(pc, op, gas, cost, scope, depth, err)
			}
		},
	}
	if len(l.tracerName) > 0 { // a custom tracer
		hooks.OnTxStart = l.OnTxStartJSON
	}
	return hooks
}

// OnTxStartJSONL is the OnTxStart-handler for jsonl logger.
func (l *fileWritingTracer) OnTxStartJSONL(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	// Open a new file,
	fname := filepath.Join(l.baseDir, fmt.Sprintf("trace-%d-%v.jsonl", l.txIndex, tx.Hash().String()))
	traceFile, err := os.Create(fname)
	if err != nil {
		log.Warn("Failed creating trace-file", "err", err)
	}
	log.Debug("Created tracing-file", "path", fname)
	l.destination = traceFile
	l.inner = logger.NewJSONLoggerWithCallFrames(l.logConfig, traceFile)
	if l.inner.OnTxStart != nil {
		l.inner.OnTxStart(env, tx, from)
	}
}

// OnTxStartJSONL is the OnTxStart-handler for custom tracer.
func (l *fileWritingTracer) OnTxStartJSON(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	// Open a new file,
	fname := filepath.Join(l.baseDir, fmt.Sprintf("trace-%d-%v.json", l.txIndex, tx.Hash().String()))
	traceFile, err := os.Create(fname)
	if err != nil {
		log.Warn("Failed creating trace-file", "err", err)
	}
	fmt.Printf("Created tracing-file %v\n", fname)
	log.Info("Created tracing-file", "path", fname)
	l.destination = traceFile
	inner, err := tracers.DefaultDirectory.New(l.tracerName, nil, l.tracerConf, l.chainConfig)
	if err != nil {
		log.Warn("Failed instantiating tracer", "err", err)
		return
	}
	l.getResult = inner.GetResult
	l.inner = inner.Hooks
	if l.inner.OnTxStart != nil {
		l.inner.OnTxStart(env, tx, from)
	}
}

func (l *fileWritingTracer) OnTxEnd(receipt *types.Receipt, err error) {
	if l.inner.OnTxEnd != nil {
		l.inner.OnTxEnd(receipt, err)
	}
	if l.getResult != nil {
		if result, err := l.getResult(); result != nil {
			json.NewEncoder(l.destination).Encode(result)
		} else {
			log.Warn("Error obtaining tracer result", "err", err)
		}
	}
	if l.destination != nil { // Close old file
		l.destination.Close()
		l.destination = nil
	}
	l.txIndex++
}

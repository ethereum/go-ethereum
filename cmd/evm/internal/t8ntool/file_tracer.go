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

// fileWritingTracer is a tracer which wraps either a different tracer,
// or a logger. On tx start, it creates a new file to direct output to,
// and on tx end it closes the file.
type fileWritingTracer struct {
	txIndex     int
	inner       *tracing.Hooks
	destination io.WriteCloser
	baseDir     string

	// for json-tracing
	logConfig  *logger.Config
	callFrames bool

	// for custom tracing
	tracerName  string
	tracerConf  json.RawMessage
	chainConfig *params.ChainConfig
	getResult   func() (json.RawMessage, error)
}

func newFileWritingTracer(baseDir string, logConfig *logger.Config, callFrames bool) *fileWritingTracer {
	return &fileWritingTracer{
		baseDir:    baseDir,
		logConfig:  logConfig,
		callFrames: callFrames,
	}
}

func newFileWritingCustomTracer(baseDir, tracerName string, traceConfig json.RawMessage, chainConfig *params.ChainConfig) *fileWritingTracer {
	return &fileWritingTracer{
		baseDir:     baseDir,
		tracerName:  tracerName,
		chainConfig: chainConfig,
	}
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
	if !l.callFrames {
		l.inner = logger.NewJSONLogger(l.logConfig, traceFile)
	} else {
		l.inner = logger.NewJSONLoggerWithCallFrames(l.logConfig, traceFile)
	}
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

func (l *fileWritingTracer) Tracer() *tracers.Tracer {
	hooks := &tracing.Hooks{
		OnTxStart: l.OnTxStartJSONL,
		OnTxEnd:   l.OnTxEnd,
		OnSystemCallStart: func() {
			if l.inner.OnSystemCallStart != nil {
				l.inner.OnSystemCallStart()
			}
		},
		OnEnter: func(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
			if l.inner.OnEnter != nil {
				l.inner.OnEnter(depth, typ, from, to, input, gas, value)
			}
		},
		OnExit: func(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
			if l.inner.OnExit != nil {
				l.inner.OnExit(depth, output, gasUsed, err, reverted)
			}
		},
		OnOpcode: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
			if l.inner.OnOpcode != nil {
				l.inner.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
			}
		},
		OnFault: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
			if l.inner.OnFault != nil {
				l.inner.OnFault(pc, op, gas, cost, scope, depth, err)
			}
		},
	}
	if len(l.tracerName) > 0 { // a custom tracer
		hooks.OnTxStart = l.OnTxStartJSON
	}
	return &tracers.Tracer{
		Hooks:     hooks,
		GetResult: func() (json.RawMessage, error) { return nil, nil },
		Stop:      func(err error) {},
	}
}

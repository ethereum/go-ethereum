// Copyright 2022 The go-ethereum Authors
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

package native

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

func init() {
	tracers.DefaultDirectory.Register("muxTracer", newMuxTracer, false)
}

// muxTracer is a go implementation of the Tracer interface which
// runs multiple tracers in one go.
type muxTracer struct {
	names   []string
	tracers []*tracers.Tracer
}

// newMuxTracer returns a new mux tracer.
func newMuxTracer(ctx *tracers.Context, cfg json.RawMessage) (*tracers.Tracer, error) {
	var config map[string]json.RawMessage
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, err
		}
	}
	objects := make([]*tracers.Tracer, 0, len(config))
	names := make([]string, 0, len(config))
	for k, v := range config {
		t, err := tracers.DefaultDirectory.New(k, ctx, v)
		if err != nil {
			return nil, err
		}
		objects = append(objects, t)
		names = append(names, k)
	}

	t := &muxTracer{names: names, tracers: objects}
	return &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnTxStart:       t.OnTxStart,
			OnTxEnd:         t.OnTxEnd,
			OnEnter:         t.OnEnter,
			OnExit:          t.OnExit,
			OnOpcode:        t.OnOpcode,
			OnFault:         t.OnFault,
			OnGasChange:     t.OnGasChange,
			OnBalanceChange: t.OnBalanceChange,
			OnNonceChange:   t.OnNonceChange,
			OnCodeChange:    t.OnCodeChange,
			OnStorageChange: t.OnStorageChange,
			OnLog:           t.OnLog,
		},
		GetResult: t.GetResult,
		Stop:      t.Stop,
	}, nil
}

func (t *muxTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	for _, t := range t.tracers {
		if t.OnOpcode != nil {
			t.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
		}
	}
}

func (t *muxTracer) OnFault(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
	for _, t := range t.tracers {
		if t.OnFault != nil {
			t.OnFault(pc, op, gas, cost, scope, depth, err)
		}
	}
}

func (t *muxTracer) OnGasChange(old, new uint64, reason tracing.GasChangeReason) {
	for _, t := range t.tracers {
		if t.OnGasChange != nil {
			t.OnGasChange(old, new, reason)
		}
	}
}

func (t *muxTracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	for _, t := range t.tracers {
		if t.OnEnter != nil {
			t.OnEnter(depth, typ, from, to, input, gas, value)
		}
	}
}

func (t *muxTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	for _, t := range t.tracers {
		if t.OnExit != nil {
			t.OnExit(depth, output, gasUsed, err, reverted)
		}
	}
}

func (t *muxTracer) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	for _, t := range t.tracers {
		if t.OnTxStart != nil {
			t.OnTxStart(env, tx, from)
		}
	}
}

func (t *muxTracer) OnTxEnd(receipt *types.Receipt, err error) {
	for _, t := range t.tracers {
		if t.OnTxEnd != nil {
			t.OnTxEnd(receipt, err)
		}
	}
}

func (t *muxTracer) OnBalanceChange(a common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
	for _, t := range t.tracers {
		if t.OnBalanceChange != nil {
			t.OnBalanceChange(a, prev, new, reason)
		}
	}
}

func (t *muxTracer) OnNonceChange(a common.Address, prev, new uint64) {
	for _, t := range t.tracers {
		if t.OnNonceChange != nil {
			t.OnNonceChange(a, prev, new)
		}
	}
}

func (t *muxTracer) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
	for _, t := range t.tracers {
		if t.OnCodeChange != nil {
			t.OnCodeChange(a, prevCodeHash, prev, codeHash, code)
		}
	}
}

func (t *muxTracer) OnStorageChange(a common.Address, k, prev, new common.Hash) {
	for _, t := range t.tracers {
		if t.OnStorageChange != nil {
			t.OnStorageChange(a, k, prev, new)
		}
	}
}

func (t *muxTracer) OnLog(log *types.Log) {
	for _, t := range t.tracers {
		if t.OnLog != nil {
			t.OnLog(log)
		}
	}
}

// GetResult returns an empty json object.
func (t *muxTracer) GetResult() (json.RawMessage, error) {
	resObject := make(map[string]json.RawMessage)
	for i, tt := range t.tracers {
		r, err := tt.GetResult()
		if err != nil {
			return nil, err
		}
		resObject[t.names[i]] = r
	}
	res, err := json.Marshal(resObject)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *muxTracer) Stop(err error) {
	for _, t := range t.tracers {
		t.Stop(err)
	}
}

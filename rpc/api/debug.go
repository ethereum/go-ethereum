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

package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/rcrowley/go-metrics"
)

const (
	DebugApiVersion = "1.0"
)

var (
	// mapping between methods and handlers
	DebugMapping = map[string]debughandler{
		"debug_dumpBlock":    (*debugApi).DumpBlock,
		"debug_getBlockRlp":  (*debugApi).GetBlockRlp,
		"debug_printBlock":   (*debugApi).PrintBlock,
		"debug_processBlock": (*debugApi).ProcessBlock,
		"debug_seedHash":     (*debugApi).SeedHash,
		"debug_setHead":      (*debugApi).SetHead,
		"debug_metrics":      (*debugApi).Metrics,
	}
)

// debug callback handler
type debughandler func(*debugApi, *shared.Request) (interface{}, error)

// admin api provider
type debugApi struct {
	xeth     *xeth.XEth
	ethereum *eth.Ethereum
	methods  map[string]debughandler
	codec    codec.ApiCoder
}

// create a new debug api instance
func NewDebugApi(xeth *xeth.XEth, ethereum *eth.Ethereum, coder codec.Codec) *debugApi {
	return &debugApi{
		xeth:     xeth,
		ethereum: ethereum,
		methods:  DebugMapping,
		codec:    coder.New(nil),
	}
}

// collection with supported methods
func (self *debugApi) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

// Execute given request
func (self *debugApi) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := self.methods[req.Method]; ok {
		return callback(self, req)
	}

	return nil, &shared.NotImplementedError{req.Method}
}

func (self *debugApi) Name() string {
	return shared.DebugApiName
}

func (self *debugApi) ApiVersion() string {
	return DebugApiVersion
}

func (self *debugApi) PrintBlock(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	return fmt.Sprintf("%s", block), nil
}

func (self *debugApi) DumpBlock(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", args.BlockNumber)
	}

	stateDb := state.New(block.Root(), self.ethereum.StateDb())
	if stateDb == nil {
		return nil, nil
	}

	return stateDb.RawDump(), nil
}

func (self *debugApi) GetBlockRlp(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", args.BlockNumber)
	}
	encoded, err := rlp.EncodeToBytes(block)
	return fmt.Sprintf("%x", encoded), err
}

func (self *debugApi) SetHead(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", args.BlockNumber)
	}

	self.ethereum.ChainManager().SetHead(block)

	return nil, nil
}

func (self *debugApi) ProcessBlock(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", args.BlockNumber)
	}

	old := vm.Debug
	defer func() { vm.Debug = old }()
	vm.Debug = true

	_, err := self.ethereum.BlockProcessor().RetryProcess(block)
	if err == nil {
		return true, nil
	}
	return false, err
}

func (self *debugApi) SeedHash(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	if hash, err := ethash.GetSeedHash(uint64(args.BlockNumber)); err == nil {
		return fmt.Sprintf("0x%x", hash), nil
	} else {
		return nil, err
	}
}

func (self *debugApi) Metrics(req *shared.Request) (interface{}, error) {
	args := new(MetricsArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	// Create a rate formatter
	units := []string{"", "K", "M", "G", "T", "E", "P"}
	round := func(value float64, prec int) string {
		unit := 0
		for value >= 1000 {
			unit, value, prec = unit+1, value/1000, 2
		}
		return fmt.Sprintf(fmt.Sprintf("%%.%df%s", prec, units[unit]), value)
	}
	format := func(total float64, rate float64) string {
		return fmt.Sprintf("%s (%s/s)", round(total, 0), round(rate, 2))
	}
	// Iterate over all the metrics, and just dump for now
	counters := make(map[string]interface{})
	metrics.DefaultRegistry.Each(func(name string, metric interface{}) {
		// Create or retrieve the counter hierarchy for this metric
		root, parts := counters, strings.Split(name, "/")
		for _, part := range parts[:len(parts)-1] {
			if _, ok := root[part]; !ok {
				root[part] = make(map[string]interface{})
			}
			root = root[part].(map[string]interface{})
		}
		name = parts[len(parts)-1]

		// Fill the counter with the metric details, formatting if requested
		if args.Raw {
			switch metric := metric.(type) {
			case metrics.Meter:
				root[name] = map[string]interface{}{
					"AvgRate01Min": metric.Rate1(),
					"AvgRate05Min": metric.Rate5(),
					"AvgRate15Min": metric.Rate15(),
					"MeanRate":     metric.RateMean(),
					"Overall":      float64(metric.Count()),
				}

			case metrics.Timer:
				root[name] = map[string]interface{}{
					"AvgRate01Min": metric.Rate1(),
					"AvgRate05Min": metric.Rate5(),
					"AvgRate15Min": metric.Rate15(),
					"MeanRate":     metric.RateMean(),
					"Overall":      float64(metric.Count()),
					"Percentiles": map[string]interface{}{
						"5":  metric.Percentile(0.05),
						"20": metric.Percentile(0.2),
						"50": metric.Percentile(0.5),
						"80": metric.Percentile(0.8),
						"95": metric.Percentile(0.95),
					},
				}

			default:
				root[name] = "Unknown metric type"
			}
		} else {
			switch metric := metric.(type) {
			case metrics.Meter:
				root[name] = map[string]interface{}{
					"Avg01Min": format(metric.Rate1()*60, metric.Rate1()),
					"Avg05Min": format(metric.Rate5()*300, metric.Rate5()),
					"Avg15Min": format(metric.Rate15()*900, metric.Rate15()),
					"Overall":  format(float64(metric.Count()), metric.RateMean()),
				}

			case metrics.Timer:
				root[name] = map[string]interface{}{
					"Avg01Min": format(metric.Rate1()*60, metric.Rate1()),
					"Avg05Min": format(metric.Rate5()*300, metric.Rate5()),
					"Avg15Min": format(metric.Rate15()*900, metric.Rate15()),
					"Overall":  format(float64(metric.Count()), metric.RateMean()),
					"Maximum":  time.Duration(metric.Max()).String(),
					"Minimum":  time.Duration(metric.Min()).String(),
					"Percentiles": map[string]interface{}{
						"5":  time.Duration(metric.Percentile(0.05)).String(),
						"20": time.Duration(metric.Percentile(0.2)).String(),
						"50": time.Duration(metric.Percentile(0.5)).String(),
						"80": time.Duration(metric.Percentile(0.8)).String(),
						"95": time.Duration(metric.Percentile(0.95)).String(),
					},
				}

			default:
				root[name] = "Unknown metric type"
			}
		}
	})
	return counters, nil
}

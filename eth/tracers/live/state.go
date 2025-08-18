// Copyright 2024 The go-ethereum Authors
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

package live

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

func init() {
	tracers.LiveDirectory.Register("state", newStateTracer)
}

type stateTracer struct {
	logger *lumberjack.Logger
}

type stateTracerConfig struct {
	Path    string `json:"path"`    // Path to the directory where the tracer logs will be stored
	MaxSize int    `json:"maxSize"` // MaxSize is the maximum size in megabytes of the tracer log file before it gets rotated. It defaults to 100 megabytes.
}

func newStateTracer(cfg json.RawMessage) (*tracing.Hooks, error) {
	var config stateTracerConfig
	if err := json.Unmarshal(cfg, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}
	if config.Path == "" {
		return nil, errors.New("state tracer output path is required")
	}

	// Store traces in a rotating file
	logger := &lumberjack.Logger{
		Filename: filepath.Join(config.Path, "state.jsonl"),
	}
	if config.MaxSize > 0 {
		logger.MaxSize = config.MaxSize
	}

	t := &stateTracer{
		logger: logger,
	}
	return &tracing.Hooks{
		OnGenesisBlock: t.onGenesisBlock,
		OnStateCommit:  t.onStateCommit,
		OnClose:        t.onClose,
	}, nil
}

func (s *stateTracer) onGenesisBlock(b *types.Block, alloc types.GenesisAlloc) {
	var (
		accountSize, storageSize, codeSize int
		accounts, storages, codes          int
	)
	for _, account := range alloc {
		accounts++
		accountSize += common.HashLength
		storages += len(account.Storage)
		storageSize += len(account.Storage) * 2 * common.HashLength
		if len(account.Code) > 0 {
			codes++
			codeSize += len(account.Code) + common.HashLength
		}
	}
	update := &tracing.StateUpdate{
		Number:      b.NumberU64(),
		Hash:        b.Hash(),
		Time:        b.Time(),
		Accounts:    int64(accounts),
		AccountSize: int64(accountSize),
		Storages:    int64(storages),
		StorageSize: int64(storageSize),
		Codes:       int64(codes),
		CodeSize:    int64(codeSize),
	}
	s.write(update)
}

func (s *stateTracer) onStateCommit(update *tracing.StateUpdate) {
	s.write(update)
}

func (s *stateTracer) onClose() {
	if err := s.logger.Close(); err != nil {
		log.Warn("failed to close state tracer log file", "error", err)
	}
}

func (s *stateTracer) write(update *tracing.StateUpdate) {
	out, _ := json.Marshal(update)
	out = append(out, '\n')
	if _, err := s.logger.Write(out); err != nil {
		log.Warn("failed to write to state tracer log file", "error", err)
	}
}

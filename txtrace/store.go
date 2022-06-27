// Copyright 2021 The go-ethereum Authors
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

package txtrace

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/metrics"

	txtrace "github.com/DeBankDeFi/etherlib/pkg/txtracev2"
)

var (
	txTraceWriteSuccessCounter = metrics.NewRegisteredCounter("chain/txtraces/write/success", nil)
	txTraceWriteFailCounter    = metrics.NewRegisteredCounter("chain/txtraces/write/fail", nil)
)

var (
	once              sync.Once
	defaultTraceStore *traceStore
)

var _ txtrace.Store = (*traceStore)(nil)

type traceStore struct {
	db ethdb.Database
}

// NewTraceStore creates a new trace store.
func NewTraceStore(db ethdb.Database) txtrace.Store {
	if defaultTraceStore != nil {
		return defaultTraceStore
	}
	once.Do(func() {
		defaultTraceStore = &traceStore{db: db}
	})
	return defaultTraceStore
}

// GetTraceStore get singleton traceStore.
func GetTraceStore() *traceStore {
	return defaultTraceStore
}

func (t *traceStore) guard() error {
	if t.db == nil {
		return fmt.Errorf("txtrace mode not enabled")
	}
	return nil
}

// ReadTxTrace retrieves the result of tx by evm-tracing which stores in db.
func (t *traceStore) ReadTxTrace(ctx context.Context, txHash common.Hash) ([]byte, error) {
	if err := t.guard(); err != nil {
		return []byte{}, err
	}
	data := rawdb.ReadTxTrace(t.db, txHash)
	return data, nil
}

// WriteTxTrace write the result of tx tracing by evm-tracing to db.
func (t *traceStore) WriteTxTrace(ctx context.Context, txHash common.Hash, trace []byte) error {
	if err := t.guard(); err != nil {
		return err
	}
	err := rawdb.WriteTxTrace(t.db, txHash, trace)
	if err == nil {
		txTraceWriteSuccessCounter.Inc(1)
		return nil
	}
	txTraceWriteFailCounter.Inc(1)
	return err
}

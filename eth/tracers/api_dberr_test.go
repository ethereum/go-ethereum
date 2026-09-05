// Copyright 2026 The go-ethereum Authors
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

package tracers_test

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/triedb"
)

// errPoisonedRead is the sentinel returned by the fault-injecting state
// reader below.
var errPoisonedRead = errors.New("poisoned account read")

// faultReader wraps a state.Reader and fails every read of one account.
type faultReader struct {
	state.Reader
	poison common.Address
}

func (r *faultReader) Account(addr common.Address) (*types.StateAccount, error) {
	if addr == r.poison {
		return nil, errPoisonedRead
	}
	return r.Reader.Account(addr)
}

// faultStateDatabase wraps a state.Database so that readers derived from it
// fail on the poisoned account.
type faultStateDatabase struct {
	state.Database
	poison common.Address
}

func (db *faultStateDatabase) Reader(root common.Hash) (state.Reader, error) {
	r, err := db.Database.Reader(root)
	if err != nil {
		return nil, err
	}
	return &faultReader{Reader: r, poison: db.poison}, nil
}

// dbErrBackend is a minimal tracers.Backend serving a genesis-only chain.
// When poison is set, states handed out by StateAtBlock fail every read of
// that account.
type dbErrBackend struct {
	config  *params.ChainConfig
	engine  consensus.Engine
	chaindb ethdb.Database
	statedb state.Database
	genesis *types.Block
	poison  *common.Address
}

func newDBErrBackend(t *testing.T, alloc types.GenesisAlloc) *dbErrBackend {
	t.Helper()
	var (
		chaindb = rawdb.NewMemoryDatabase()
		tdb     = triedb.NewDatabase(chaindb, nil)
		gspec   = &core.Genesis{
			Config:  params.TestChainConfig,
			Alloc:   alloc,
			BaseFee: big.NewInt(params.InitialBaseFee),
		}
	)
	genesis := gspec.MustCommit(chaindb, tdb)
	return &dbErrBackend{
		config:  gspec.Config,
		engine:  ethash.NewFaker(),
		chaindb: chaindb,
		statedb: state.NewDatabase(tdb, nil),
		genesis: genesis,
	}
}

func (b *dbErrBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.genesis.Header(), nil
}

func (b *dbErrBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	return b.genesis.Header(), nil
}

func (b *dbErrBackend) CurrentHeader() *types.Header {
	return b.genesis.Header()
}

func (b *dbErrBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.genesis, nil
}

func (b *dbErrBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	return b.genesis, nil
}

func (b *dbErrBackend) GetCanonicalTransaction(txHash common.Hash) (bool, *types.Transaction, common.Hash, uint64, uint64) {
	return false, nil, common.Hash{}, 0, 0
}

func (b *dbErrBackend) TxIndexDone() bool {
	return true
}

func (b *dbErrBackend) RPCGasCap() uint64 {
	return 50_000_000
}

func (b *dbErrBackend) ChainConfig() *params.ChainConfig {
	return b.config
}

func (b *dbErrBackend) Engine() consensus.Engine {
	return b.engine
}

func (b *dbErrBackend) ChainDb() ethdb.Database {
	return b.chaindb
}

func (b *dbErrBackend) StateAtBlock(ctx context.Context, block *types.Block, base *state.StateDB, readOnly bool, preferDisk bool) (*state.StateDB, tracers.StateReleaseFunc, error) {
	db := b.statedb
	if b.poison != nil {
		db = &faultStateDatabase{Database: b.statedb, poison: *b.poison}
	}
	statedb, err := state.New(block.Root(), db)
	if err != nil {
		return nil, nil, err
	}
	return statedb, func() {}, nil
}

func (b *dbErrBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int) (*types.Transaction, vm.BlockContext, *state.StateDB, tracers.StateReleaseFunc, error) {
	return nil, vm.BlockContext{}, nil, nil, errors.New("not supported")
}

// TestTraceCallStateReadError verifies that a database read failure during
// tracing surfaces as an error. A failed read doesn't abort execution — the
// EVM keeps running on zero values — so without an explicit statedb error
// check the API returns a silently wrong trace.
func TestTraceCallStateReadError(t *testing.T) {
	t.Parallel()

	var (
		from    = common.HexToAddress("0xfacade0000000000000000000000000000000000")
		poison  = common.HexToAddress("0xbad0000000000000000000000000000000000bad")
		backend = newDBErrBackend(t, types.GenesisAlloc{
			from: {Balance: big.NewInt(params.Ether)},
		})
		api    = tracers.NewAPI(backend)
		latest = rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
		args   = ethapi.TransactionArgs{
			From:  &from,
			To:    &poison,
			Value: (*hexutil.Big)(big.NewInt(1)),
		}
	)
	// Control: tracing against a healthy state must succeed.
	if _, err := api.TraceCall(context.Background(), args, latest, nil); err != nil {
		t.Fatalf("control trace failed: %v", err)
	}
	// Poison the recipient: its account read fails mid-execution while the
	// call itself still completes.
	backend.poison = &poison
	res, err := api.TraceCall(context.Background(), args, latest, nil)
	if err == nil {
		t.Fatalf("expected state read error to surface, got trace: %v", res)
	}
	if !errors.Is(err, errPoisonedRead) {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestTraceCallStateReadErrorPrecedence verifies that a database read failure
// takes precedence over the execution error it causes. Poisoning the sender
// makes it look broke, so execution fails with "insufficient funds" while the
// actual cause is the failed read.
func TestTraceCallStateReadErrorPrecedence(t *testing.T) {
	t.Parallel()

	var (
		from    = common.HexToAddress("0xfacade0000000000000000000000000000000000")
		to      = common.HexToAddress("0x0000000000000000000000000000000000001234")
		backend = newDBErrBackend(t, types.GenesisAlloc{
			from: {Balance: big.NewInt(params.Ether)},
		})
		api    = tracers.NewAPI(backend)
		latest = rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
		args   = ethapi.TransactionArgs{
			From:  &from,
			To:    &to,
			Value: (*hexutil.Big)(big.NewInt(1)),
		}
	)
	backend.poison = &from
	res, err := api.TraceCall(context.Background(), args, latest, nil)
	if err == nil {
		t.Fatalf("expected state read error to surface, got trace: %v", res)
	}
	if !errors.Is(err, errPoisonedRead) {
		t.Fatalf("state read error masked by execution failure: %v", err)
	}
}

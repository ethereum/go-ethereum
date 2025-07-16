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

package tracers

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/internal/ethapi/override"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	// defaultTraceTimeout is the amount of time a single transaction can execute
	// by default before being forcefully aborted.
	defaultTraceTimeout = 5 * time.Second

	// defaultTraceReexec is the number of blocks the tracer is willing to go back
	// and reexecute to produce missing historical state necessary to run a specific
	// trace.
	defaultTraceReexec = uint64(128)

	// defaultTracechainMemLimit is the size of the triedb, at which traceChain
	// switches over and tries to use a disk-backed database instead of building
	// on top of memory.
	// For non-archive nodes, this limit _will_ be overblown, as disk-backed tries
	// will only be found every ~15K blocks or so.
	defaultTracechainMemLimit = common.StorageSize(500 * 1024 * 1024)

	// maximumPendingTraceStates is the maximum number of states allowed waiting
	// for tracing. The creation of trace state will be paused if the unused
	// trace states exceed this limit.
	maximumPendingTraceStates = 128
)

var errTxNotFound = errors.New("transaction not found")

// StateReleaseFunc is used to deallocate resources held by constructing a
// historical state for tracing purposes.
type StateReleaseFunc func()

// Backend interface provides the common API services (that are provided by
// both full and light clients) with access to necessary functions.
type Backend interface {
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error)
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error)
	GetCanonicalTransaction(txHash common.Hash) (bool, *types.Transaction, common.Hash, uint64, uint64)
	TxIndexDone() bool
	RPCGasCap() uint64
	ChainConfig() *params.ChainConfig
	Engine() consensus.Engine
	ChainDb() ethdb.Database
	StateAtBlock(ctx context.Context, block *types.Block, reexec uint64, base *state.StateDB, readOnly bool, preferDisk bool) (*state.StateDB, StateReleaseFunc, error)
	StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (*types.Transaction, vm.BlockContext, *state.StateDB, StateReleaseFunc, error)
}

// API is the collection of tracing APIs exposed over the private debugging endpoint.
type API struct {
	backend Backend
}

// NewAPI creates a new API definition for the tracing methods of the Ethereum service.
func NewAPI(backend Backend) *API {
	return &API{backend: backend}
}

// chainContext constructs the context reader which is used by the evm for reading
// the necessary chain context.
func (api *API) chainContext(ctx context.Context) core.ChainContext {
	return ethapi.NewChainContext(ctx, api.backend)
}

// blockByNumber is the wrapper of the chain access function offered by the backend.
// It will return an error if the block is not found.
func (api *API) blockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	block, err := api.backend.BlockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", number)
	}
	return block, nil
}

// blockByHash is the wrapper of the chain access function offered by the backend.
// It will return an error if the block is not found.
func (api *API) blockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	block, err := api.backend.BlockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block %s not found", hash.Hex())
	}
	return block, nil
}

// blockByNumberAndHash is the wrapper of the chain access function offered by
// the backend. It will return an error if the block is not found.
//
// Note this function is friendly for the light client which can only retrieve the
// historical(before the CHT) header/block by number.
func (api *API) blockByNumberAndHash(ctx context.Context, number rpc.BlockNumber, hash common.Hash) (*types.Block, error) {
	block, err := api.blockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	if block.Hash() == hash {
		return block, nil
	}
	return api.blockByHash(ctx, hash)
}

// TraceConfig holds extra parameters to trace functions.
type TraceConfig struct {
	*logger.Config
	Tracer  *string
	Timeout *string
	Reexec  *uint64
	// Config specific to given tracer. Note struct logger
	// config are historically embedded in main object.
	TracerConfig json.RawMessage
}

// TraceCallConfig is the config for traceCall API. It holds one more
// field to override the state for tracing.
type TraceCallConfig struct {
	TraceConfig
	StateOverrides *override.StateOverride
	BlockOverrides *override.BlockOverrides
	TxIndex        *hexutil.Uint
}

// StdTraceConfig holds extra parameters to standard-json trace functions.
type StdTraceConfig struct {
	logger.Config
	Reexec *uint64
	TxHash common.Hash
}

// txTraceResult is the result of a single transaction trace.
type txTraceResult struct {
	TxHash common.Hash `json:"txHash"`           // transaction hash
	Result interface{} `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string      `json:"error,omitempty"`  // Trace failure produced by the tracer
}

// blockTraceTask represents a single block trace task when an entire chain is
// being traced.
type blockTraceTask struct {
	statedb *state.StateDB   // Intermediate state prepped for tracing
	block   *types.Block     // Block to trace the transactions from
	release StateReleaseFunc // The function to release the held resource for this task
	results []*txTraceResult // Trace results produced by the task
}

// blockTraceResult represents the results of tracing a single block when an entire
// chain is being traced.
type blockTraceResult struct {
	Block  hexutil.Uint64   `json:"block"`  // Block number corresponding to this trace
	Hash   common.Hash      `json:"hash"`   // Block hash corresponding to this trace
	Traces []*txTraceResult `json:"traces"` // Trace results produced by the task
}

// txTraceTask represents a single transaction trace task when an entire block
// is being traced.
type txTraceTask struct {
	statedb *state.StateDB // Intermediate state prepped for tracing
	index   int            // Transaction offset in the block
}

// TraceChain returns the structured logs created during the execution of EVM
// between two blocks (excluding start) and returns them as a JSON object.
func (api *API) TraceChain(ctx context.Context, start, end rpc.BlockNumber, config *TraceConfig) (*rpc.Subscription, error) { // Fetch the block interval that we want to trace
	from, err := api.blockByNumber(ctx, start)
	if err != nil {
		return nil, err
	}
	to, err := api.blockByNumber(ctx, end)
	if err != nil {
		return nil, err
	}
	if from.Number().Cmp(to.Number()) >= 0 {
		return nil, fmt.Errorf("end block (#%d) needs to come after start block (#%d)", end, start)
	}
	// Tracing a chain is a **long** operation, only do with subscriptions
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	sub := notifier.CreateSubscription()

	resCh := api.traceChain(from, to, config, sub.Err())
	go func() {
		for result := range resCh {
			notifier.Notify(sub.ID, result)
		}
	}()
	return sub, nil
}

// traceChain configures a new tracer according to the provided configuration, and
// executes all the transactions contained within. The tracing chain range includes
// the end block but excludes the start one. The return value will be one item per
// transaction, dependent on the requested tracer.
// The tracing procedure should be aborted in case the closed signal is received.
func (api *API) traceChain(start, end *types.Block, config *TraceConfig, closed <-chan error) chan *blockTraceResult {
	blocks := int(end.NumberU64() - start.NumberU64())
	threads := runtime.NumCPU()
	if threads > blocks {
		threads = blocks
	}
	var (
		pend    = new(sync.WaitGroup)
		ctx     = context.Background()
		taskCh  = make(chan *blockTraceTask, threads)
		resCh   = make(chan *blockTraceTask, threads)
		tracker = newStateTracker(maximumPendingTraceStates, start.NumberU64())
		err     error
	)
	opt, err := traceExecOpt(false, config)
	if err != nil {
		log.Warn("invalid trace configuration", "err", err)
		return nil
	}

	instantiateTracer := func(ctx *Context) (*Tracer, error) {
		return api.instantiateTracer(config, ctx)
	}

	for th := 0; th < threads; th++ {
		pend.Add(1)
		go func() {
			defer pend.Done()

			// Fetch and execute the block trace taskCh
			for task := range taskCh {
				task.results, err = api.traceBlockWithState(ctx, task.block, task.statedb, opt, api.backend.ChainConfig(), instantiateTracer)
				if err != nil {
					log.Warn("Tracing failed", "err", err)
					break
				}
				// Tracing state is used up, queue it for de-referencing. Note the
				// state is the parent state of trace block, use block.number-1 as
				// the state number.
				tracker.releaseState(task.block.NumberU64()-1, task.release)

				// Stream the result back to the result catcher or abort on teardown
				select {
				case resCh <- task:
				case <-closed:
					return
				}
			}
		}()
	}
	// Start a goroutine to feed all the blocks into the tracers
	go func() {
		var (
			logged  time.Time
			begin   = time.Now()
			number  uint64
			traced  uint64
			failed  error
			statedb *state.StateDB
			release StateReleaseFunc
		)
		// Ensure everything is properly cleaned up on any exit path
		defer func() {
			close(taskCh)
			pend.Wait()

			// Clean out any pending release functions of trace states.
			tracker.callReleases()

			// Log the chain result
			switch {
			case failed != nil:
				log.Warn("Chain tracing failed", "start", start.NumberU64(), "end", end.NumberU64(), "transactions", traced, "elapsed", time.Since(begin), "err", failed)
			case number < end.NumberU64():
				log.Warn("Chain tracing aborted", "start", start.NumberU64(), "end", end.NumberU64(), "abort", number, "transactions", traced, "elapsed", time.Since(begin))
			default:
				log.Info("Chain tracing finished", "start", start.NumberU64(), "end", end.NumberU64(), "transactions", traced, "elapsed", time.Since(begin))
			}
			close(resCh)
		}()
		// Feed all the blocks both into the tracer, as well as fast process concurrently
		for number = start.NumberU64() + 1; number <= end.NumberU64(); number++ {
			// Stop tracing if interruption was requested
			select {
			case <-closed:
				return
			default:
			}
			// Print progress logs if long enough time elapsed
			if time.Since(logged) > 8*time.Second {
				logged = time.Now()
				log.Info("Tracing chain segment", "start", start.NumberU64(), "end", end.NumberU64(), "current", number, "transactions", traced, "elapsed", time.Since(begin))
			}
			// Retrieve the parent block and target block for tracing.
			block, err := api.blockByNumber(ctx, rpc.BlockNumber(number))
			if err != nil {
				failed = err
				break
			}
			// Make sure the state creator doesn't go too far. Too many unprocessed
			// trace state may cause the oldest state to become stale(e.g. in
			// path-based scheme).
			if err = tracker.wait(number); err != nil {
				failed = err
				break
			}
			// Prepare the statedb for tracing. Don't use the live database for
			// tracing to avoid persisting state junks into the database. Switch
			// over to `preferDisk` mode only if the memory usage exceeds the
			// limit, the trie database will be reconstructed from scratch only
			// if the relevant state is available in disk.
			var preferDisk bool
			if statedb != nil {
				s1, s2, s3 := statedb.Database().TrieDB().Size()
				preferDisk = s1+s2+s3 > defaultTracechainMemLimit
			}
			statedb, release, err = api.retrieveBlockPrestate(ctx, block, opt, false, preferDisk)
			if err != nil {
				failed = err
				break
			}
			// Clean out any pending release functions of trace state. Note this
			// step must be done after constructing tracing state, because the
			// tracing state of block next depends on the parent state and construction
			// may fail if we release too early.
			tracker.callReleases()

			// Send the block over to the concurrent tracers (if not in the fast-forward phase)
			txs := block.Transactions()
			select {
			case taskCh <- &blockTraceTask{statedb: statedb.Copy(), block: block, release: release, results: make([]*txTraceResult, len(txs))}:
			case <-closed:
				tracker.releaseState(number, release)
				return
			}
			traced += uint64(len(txs))
		}
	}()

	// Keep reading the trace results and stream them to result channel.
	retCh := make(chan *blockTraceResult)
	go func() {
		defer close(retCh)
		var (
			next = start.NumberU64() + 1
			done = make(map[uint64]*blockTraceResult)
		)
		for res := range resCh {
			// Queue up next received result
			result := &blockTraceResult{
				Block:  hexutil.Uint64(res.block.NumberU64()),
				Hash:   res.block.Hash(),
				Traces: res.results,
			}
			done[uint64(result.Block)] = result

			// Stream completed traces to the result channel
			for result, ok := done[next]; ok; result, ok = done[next] {
				if len(result.Traces) > 0 || next == end.NumberU64() {
					// It will be blocked in case the channel consumer doesn't take the
					// tracing result in time(e.g. the websocket connect is not stable)
					// which will eventually block the entire chain tracer. It's the
					// expected behavior to not waste node resources for a non-active user.
					retCh <- result
				}
				delete(done, next)
				next++
			}
		}
	}()
	return retCh
}

// traceBlockCustomTracer traces a block using a js/native tracer
func (api *API) traceBlockCustomTracer(ctx context.Context, block *types.Block, config *TraceConfig) ([]*txTraceResult, error) {
	instantiateTracer := func(ctx *Context) (*Tracer, error) {
		return api.instantiateTracer(config, ctx)
	}
	opt, err := traceExecOpt(true, config)
	if err != nil {
		return nil, err
	}
	return api.traceBlock(ctx, block, opt, api.backend.ChainConfig(), instantiateTracer)
}

// TraceBlockByNumber returns the structured logs created during the execution of
// EVM and returns them as a JSON object.
func (api *API) TraceBlockByNumber(ctx context.Context, number rpc.BlockNumber, config *TraceConfig) ([]*txTraceResult, error) {
	block, err := api.blockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	return api.traceBlockCustomTracer(ctx, block, config)
}

// TraceBlockByHash returns the structured logs created during the execution of
// EVM and returns them as a JSON object.
func (api *API) TraceBlockByHash(ctx context.Context, hash common.Hash, config *TraceConfig) ([]*txTraceResult, error) {
	block, err := api.blockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return api.traceBlockCustomTracer(ctx, block, config)
}

// TraceBlock returns the structured logs created during the execution of EVM
// and returns them as a JSON object.
func (api *API) TraceBlock(ctx context.Context, blob hexutil.Bytes, config *TraceConfig) ([]*txTraceResult, error) {
	block := new(types.Block)
	if err := rlp.DecodeBytes(blob, block); err != nil {
		return nil, fmt.Errorf("could not decode block: %v", err)
	}
	return api.traceBlockCustomTracer(ctx, block, config)
}

// TraceBlockFromFile returns the structured logs created during the execution of
// EVM and returns them as a JSON object.
func (api *API) TraceBlockFromFile(ctx context.Context, file string, config *TraceConfig) ([]*txTraceResult, error) {
	blob, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read file: %v", err)
	}
	return api.TraceBlock(ctx, blob, config)
}

// TraceBadBlock returns the structured logs created during the execution of
// EVM against a block pulled from the pool of bad ones and returns them as a JSON
// object.
func (api *API) TraceBadBlock(ctx context.Context, hash common.Hash, config *TraceConfig) ([]*txTraceResult, error) {
	block := rawdb.ReadBadBlock(api.backend.ChainDb(), hash)
	if block == nil {
		return nil, fmt.Errorf("bad block %#x not found", hash)
	}
	instantiateTracer := func(ctx *Context) (*Tracer, error) {
		return api.instantiateTracer(config, ctx)
	}

	opt, err := traceExecOpt(true, config)
	if err != nil {
		return nil, err
	}
	return api.traceBlock(ctx, block, opt, api.backend.ChainConfig(), instantiateTracer)
}

// StandardTraceBlockToFile dumps the structured logs created during the
// execution of EVM to the local file system and returns a list of files
// to the caller.
func (api *API) StandardTraceBlockToFile(ctx context.Context, hash common.Hash, config *StdTraceConfig) ([]string, error) {
	block, err := api.blockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return api.standardTraceBlockToFile(ctx, block, config)
}

// IntermediateRoots executes a block (bad- or canon- or side-), and returns a list
// of intermediate roots: the stateroot after each transaction.
func (api *API) IntermediateRoots(ctx context.Context, hash common.Hash, config *TraceConfig) ([]common.Hash, error) {
	block, _ := api.blockByHash(ctx, hash)
	if block == nil {
		// Check in the bad blocks
		block = rawdb.ReadBadBlock(api.backend.ChainDb(), hash)
	}
	if block == nil {
		return nil, fmt.Errorf("block %#x not found", hash)
	}
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	parent, err := api.blockByNumberAndHash(ctx, rpc.BlockNumber(block.NumberU64()-1), block.ParentHash())
	if err != nil {
		return nil, err
	}
	reexec := defaultTraceReexec
	if config != nil && config.Reexec != nil {
		reexec = *config.Reexec
	}
	statedb, release, err := api.backend.StateAtBlock(ctx, parent, reexec, nil, true, false)
	if err != nil {
		return nil, err
	}
	defer release()
	var (
		roots              []common.Hash
		signer             = types.MakeSigner(api.backend.ChainConfig(), block.Number(), block.Time())
		chainConfig        = api.backend.ChainConfig()
		vmctx              = core.NewEVMBlockContext(block.Header(), api.chainContext(ctx), nil)
		deleteEmptyObjects = chainConfig.IsEIP158(block.Number())
	)
	evm := vm.NewEVM(vmctx, statedb, chainConfig, vm.Config{})
	if beaconRoot := block.BeaconRoot(); beaconRoot != nil {
		core.ProcessBeaconBlockRoot(*beaconRoot, evm)
	}
	if chainConfig.IsPrague(block.Number(), block.Time()) {
		core.ProcessParentBlockHash(block.ParentHash(), evm)
	}
	for i, tx := range block.Transactions() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		msg, _ := core.TransactionToMessage(tx, signer, block.BaseFee())
		statedb.SetTxContext(tx.Hash(), i)
		if _, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(msg.GasLimit)); err != nil {
			log.Warn("Tracing intermediate roots did not complete", "txindex", i, "txhash", tx.Hash(), "err", err)
			// We intentionally don't return the error here: if we do, then the RPC server will not
			// return the roots. Most likely, the caller already knows that a certain transaction fails to
			// be included, but still want the intermediate roots that led to that point.
			// It may happen the tx_N causes an erroneous state, which in turn causes tx_N+M to not be
			// executable.
			// N.B: This should never happen while tracing canon blocks, only when tracing bad blocks.
			return roots, nil
		}
		// calling IntermediateRoot will internally call Finalize on the state
		// so any modifications are written to the trie
		roots = append(roots, statedb.IntermediateRoot(deleteEmptyObjects))
	}
	return roots, nil
}

// StandardTraceBadBlockToFile dumps the structured logs created during the
// execution of EVM against a block pulled from the pool of bad ones to the
// local file system and returns a list of files to the caller.
func (api *API) StandardTraceBadBlockToFile(ctx context.Context, hash common.Hash, config *StdTraceConfig) ([]string, error) {
	block := rawdb.ReadBadBlock(api.backend.ChainDb(), hash)
	if block == nil {
		return nil, fmt.Errorf("bad block %#x not found", hash)
	}
	return api.standardTraceBlockToFile(ctx, block, config)
}

func (api *API) retrieveBlockPrestate(ctx context.Context, block *types.Block, execOpt *traceExecOptions, readOnly, preferDisk bool) (db *state.StateDB, stateRelease StateReleaseFunc, err error) {
	if block.NumberU64() == 0 {
		return nil, nil, fmt.Errorf("genesis is not traceable")
	}
	// Prepare base state
	parent, err := api.blockByNumberAndHash(ctx, rpc.BlockNumber(block.NumberU64()-1), block.ParentHash())
	if err != nil {
		return nil, nil, err
	}
	statedb, release, err := api.backend.StateAtBlock(ctx, parent, execOpt.reexec, nil, readOnly, preferDisk)
	if err != nil {
		return nil, nil, err
	}
	return statedb, release, nil
}

func (api *API) traceBlockWithState(ctx context.Context, block *types.Block, statedb *state.StateDB, execOpt *traceExecOptions, chainConfig *params.ChainConfig, instantiateTracer func(*Context) (*Tracer, error)) ([]*txTraceResult, error) {
	blockCtx := core.NewEVMBlockContext(block.Header(), api.chainContext(ctx), nil)
	evm := vm.NewEVM(blockCtx, statedb, chainConfig, vm.Config{})
	if beaconRoot := block.BeaconRoot(); beaconRoot != nil {
		core.ProcessBeaconBlockRoot(*beaconRoot, evm)
	}
	if api.backend.ChainConfig().IsPrague(block.Number(), block.Time()) {
		core.ProcessParentBlockHash(block.ParentHash(), evm)
	}

	if execOpt.parallel {
		return api.traceBlockParallel(ctx, block, statedb, execOpt.txTimeout, instantiateTracer)
	}

	// Native tracers have low overhead
	var (
		txs       = block.Transactions()
		blockHash = block.Hash()
		signer    = types.MakeSigner(api.backend.ChainConfig(), block.Number(), block.Time())
		results   = make([]*txTraceResult, len(txs))
	)
	for i, tx := range txs {
		// Generate the next state snapshot fast without tracing
		msg, _ := core.TransactionToMessage(tx, signer, block.BaseFee())
		txctx := &Context{
			BlockHash:   blockHash,
			BlockNumber: block.Number(),
			TxIndex:     i,
			TxHash:      tx.Hash(),
		}
		tracer, err := instantiateTracer(txctx)
		if err != nil {
			return nil, err
		}
		res, err := api.execTx(ctx, tx, msg, txctx, blockCtx, statedb, tracer, nil, execOpt.txTimeout, chainConfig)
		if err != nil {
			return nil, err
		}
		results[i] = &txTraceResult{TxHash: tx.Hash(), Result: res}
	}
	return results, nil
}

// traceBlock configures a new tracer according to the provided configuration, and
// executes all the transactions contained within. The return value will be one item
// per transaction, dependent on the requested tracer.
// instantiateTracer is called before execution of each transaction in the block.
func (api *API) traceBlock(ctx context.Context, block *types.Block, execOpt *traceExecOptions, chainConfig *params.ChainConfig, instantiateTracer func(*Context) (*Tracer, error)) ([]*txTraceResult, error) {
	statedb, release, err := api.retrieveBlockPrestate(ctx, block, execOpt, true, false)
	if err != nil {
		return nil, err
	}
	defer release()
	return api.traceBlockWithState(ctx, block, statedb, execOpt, chainConfig, instantiateTracer)
}

// traceBlockParallel is for tracers that have a high overhead (read JS tracers). One thread
// runs along and executes txes without tracing enabled to generate their prestate.
// Worker threads take the tasks and the prestate and trace them.
// instantiateTracer is called before executing each transaction.
func (api *API) traceBlockParallel(ctx context.Context, block *types.Block, statedb *state.StateDB, txTimeout time.Duration, instantiateTracer func(*Context) (*Tracer, error)) ([]*txTraceResult, error) {
	// Execute all the transaction contained within the block concurrently
	var (
		txs       = block.Transactions()
		blockHash = block.Hash()
		signer    = types.MakeSigner(api.backend.ChainConfig(), block.Number(), block.Time())
		results   = make([]*txTraceResult, len(txs))
		pend      sync.WaitGroup
	)
	threads := runtime.NumCPU()
	if threads > len(txs) {
		threads = len(txs)
	}

	jobs := make(chan *txTraceTask, threads)
	for th := 0; th < threads; th++ {
		pend.Add(1)
		go func() {
			defer pend.Done()
			// Fetch and execute the next transaction trace tasks
			for task := range jobs {
				msg, _ := core.TransactionToMessage(txs[task.index], signer, block.BaseFee())
				txctx := &Context{
					BlockHash:   blockHash,
					BlockNumber: block.Number(),
					TxIndex:     task.index,
					TxHash:      txs[task.index].Hash(),
				}
				tracer, err := instantiateTracer(txctx)
				if err != nil {
					results[task.index] = &txTraceResult{TxHash: txs[task.index].Hash(), Error: err.Error()}
					continue
				}
				// Reconstruct the block context for each transaction
				// as the GetHash function of BlockContext is not safe fs
				// concurrent use.
				// See: https://github.com/ethereum/go-ethereum/issues/29114
				blockCtx := core.NewEVMBlockContext(block.Header(), api.chainContext(ctx), nil)
				res, err := api.execTx(ctx, txs[task.index], msg, txctx, blockCtx, task.statedb, tracer, nil, txTimeout, api.backend.ChainConfig())
				if err != nil {
					results[task.index] = &txTraceResult{TxHash: txs[task.index].Hash(), Error: err.Error()}
					continue
				}
				results[task.index] = &txTraceResult{TxHash: txs[task.index].Hash(), Result: res}
			}
		}()
	}

	// Feed the transactions into the tracers and return
	var failed error
	blockCtx := core.NewEVMBlockContext(block.Header(), api.chainContext(ctx), nil)
txloop:
	for i, tx := range txs {
		// Send the trace task over for execution
		task := &txTraceTask{statedb: statedb.Copy(), index: i}
		select {
		case <-ctx.Done():
			failed = ctx.Err()
			break txloop
		case jobs <- task:
		}

		// Generate the next state snapshot fast without tracing
		msg, _ := core.TransactionToMessage(tx, signer, block.BaseFee())
		traceCtx := Context{
			block.Hash(),
			block.Number(),
			i,
			tx.Hash(),
		}
		if _, err := api.execTx(ctx, tx, msg, &traceCtx, blockCtx, statedb, nil, nil, defaultTraceTimeout, api.backend.ChainConfig()); err != nil {
			failed = err
			break txloop
		}
	}

	close(jobs)
	pend.Wait()

	// If execution failed in between, abort
	if failed != nil {
		return nil, failed
	}
	return results, nil
}

func createTraceDumpFile(blockHash, txHash common.Hash, txIdx int, canon bool) (*os.File, error) {
	// Generate a unique temporary file to dump it into
	prefix := fmt.Sprintf("block_%#x-%d-%#x-", blockHash.Bytes()[:4], txIdx, txHash.Bytes()[:4])
	if !canon {
		prefix = fmt.Sprintf("%valt-", prefix)
	}
	dump, err := os.CreateTemp(os.TempDir(), prefix)
	if err != nil {
		return nil, err
	}
	return dump, nil
}

func (api *API) standardTraceTxToFile(ctx context.Context, block *types.Block, tx *types.Transaction, txIdx int, config *logger.Config, signer types.Signer, canon bool) (string, error) {
	msg, _ := core.TransactionToMessage(tx, signer, block.BaseFee())
	txctx := &Context{
		BlockHash:   block.Hash(),
		BlockNumber: block.Number(),
		TxIndex:     txIdx,
		TxHash:      tx.Hash(),
	}
	_, blockCtx, statedb, release, err := api.backend.StateAtTransaction(ctx, block, txIdx, defaultTraceReexec)
	if err != nil {
		return "", err
	}
	defer release()

	var dump *os.File

	dump, err = createTraceDumpFile(block.Hash(), tx.Hash(), txIdx, canon)
	if err != nil {
		return "", err
	}

	// Swap out the noop logger to the standard traces
	writer := bufio.NewWriter(dump)

	var logConfig *logger.Config
	if config != nil {
		logConfig = config
	}

	logger := logger.NewJSONLogger(logConfig, writer)
	tracer := Tracer{
		Hooks:     logger,
		Stop:      func(err error) {},
		GetResult: func() (json.RawMessage, error) { return nil, nil },
	}
	traceTimeout := 1 * time.Hour
	_, err = api.execTx(ctx, tx, msg, txctx, blockCtx, statedb, &tracer, nil, traceTimeout, api.backend.ChainConfig())
	if err != nil {
		return "", err
	}
	if err = writer.Flush(); err != nil {
		return "", err
	}
	dump.Close()

	return dump.Name(), nil
}

func traceCallExecOpt(config *TraceCallConfig) (*traceExecOptions, error) {
	if config == nil {
		return traceExecOpt(false, nil)
	}
	return traceExecOpt(false, &config.TraceConfig)
}

type traceExecOptions struct {
	txTimeout time.Duration
	reexec    uint64
	parallel  bool
}

func traceExecOpt(allowParallelTx bool, config *TraceConfig) (*traceExecOptions, error) {
	opt := &traceExecOptions{
		defaultTraceTimeout,
		defaultTraceReexec,
		false,
	}
	if config != nil {
		if config.Reexec != nil {
			opt.reexec = *config.Reexec
		}
		if config.Timeout != nil {
			timeout, err := time.ParseDuration(*config.Timeout)
			if err != nil {
				return nil, err
			}
			opt.txTimeout = timeout
		}
		// JS tracers have high overhead. In this case run a parallel
		// process that generates states in one thread and traces txes
		// in separate worker threads.
		opt.parallel = allowParallelTx && config.Tracer != nil && *config.Tracer != "" && DefaultDirectory.IsJS(*config.Tracer)
	}
	return opt, nil
}

func standardTraceExecOpt(config *StdTraceConfig) *traceExecOptions {
	opt := traceExecOptions{
		defaultTraceTimeout,
		defaultTraceReexec,
		false,
	}
	if config != nil {
		if config.Reexec != nil {
			opt.reexec = *config.Reexec
		}
	}
	return &opt
}

// standardTraceBlockToFile configures a new tracer which uses standard JSON output,
// and traces either a full block or an individual transaction. The return value will
// be one filename per transaction traced.
func (api *API) standardTraceBlockToFile(ctx context.Context, block *types.Block, config *StdTraceConfig) ([]string, error) {
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}

	chainConfig := api.backend.ChainConfig()
	canon := true
	// Check if there are any overrides: the caller may wish to enable a future
	// fork when executing this block. Note, such overrides are only applicable to the
	// actual specified block, not any preceding blocks that we have to go through
	// in order to obtain the state.
	// Therefore, it's perfectly valid to specify `"futureForkBlock": 0`, to enable `futureFork`
	if config != nil && config.Overrides != nil {
		chainConfig, canon = overrideConfig(chainConfig, config.Overrides)
	}

	var signer = types.MakeSigner(api.backend.ChainConfig(), block.Number(), block.Time())
	if config != nil && config.TxHash != (common.Hash{}) {
		idx, ok := indexOf(block, config.TxHash)
		if !ok {
			return nil, fmt.Errorf("transaction %#x not found in block", config.TxHash)
		}
		dump, err := api.standardTraceTxToFile(ctx, block, block.Transactions()[idx], idx, &config.Config, signer, canon)
		if err != nil {
			return nil, err
		}
		return []string{dump}, nil
	}
	// Retrieve the tracing configurations, or use default values
	var logConfig logger.Config
	if config != nil {
		logConfig = config.Config
	}

	var (
		dumps  []string
		dump   *os.File
		writer *bufio.Writer
	)
	instantiateTracer := func(ctx *Context) (*Tracer, error) {
		// cleanup previous tx trace if one just ran
		if writer != nil {
			writer.Flush()
		}
		if dump != nil {
			dump.Close()
			log.Info("Wrote standard trace", "file", dump.Name())
		}

		// instantiate a new logger for this tx which will dump the trace to a temp file
		dump, err := createTraceDumpFile(ctx.BlockHash, ctx.TxHash, ctx.TxIndex, canon)
		if err != nil {
			return nil, err
		}
		dumps = append(dumps, dump.Name())

		// Swap out the noop logger to the standard tracer
		writer = bufio.NewWriter(dump)
		tracer := Tracer{
			logger.NewJSONLogger(&logConfig, writer),
			func() (json.RawMessage, error) {
				return nil, nil
			},
			func(err error) {},
		}
		return &tracer, nil
	}

	execOpt := standardTraceExecOpt(config)
	execOpt.txTimeout = 100 * time.Hour
	if _, err := api.traceBlock(ctx, block, execOpt, chainConfig, instantiateTracer); err != nil {
		return nil, err
	}

	// cleanup dumping of the last trace in the block
	if writer != nil {
		writer.Flush()
	}
	if dump != nil {
		dump.Close()
		log.Info("Wrote standard trace", "file", dump.Name())
	}
	return dumps, nil
}

// containsTx reports whether the transaction with a certain hash
// is contained within the specified block.
func indexOf(block *types.Block, hash common.Hash) (int, bool) {
	for i, tx := range block.Transactions() {
		if tx.Hash() == hash {
			return i, true
		}
	}
	return 0, false
}

// TraceTransaction returns the structured logs created during the execution of EVM
// and returns them as a JSON object.
func (api *API) TraceTransaction(ctx context.Context, hash common.Hash, config *TraceConfig) (interface{}, error) {
	found, _, blockHash, blockNumber, index := api.backend.GetCanonicalTransaction(hash)
	if !found {
		// Warn in case tx indexer is not done.
		if !api.backend.TxIndexDone() {
			return nil, ethapi.NewTxIndexingError()
		}
		// Only mined txes are supported
		return nil, errTxNotFound
	}
	// It shouldn't happen in practice.
	if blockNumber == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	opt, err := traceExecOpt(false, config)
	if err != nil {
		return nil, err
	}
	block, err := api.blockByNumberAndHash(ctx, rpc.BlockNumber(blockNumber), blockHash)
	if err != nil {
		return nil, err
	}
	tx, vmctx, statedb, release, err := api.backend.StateAtTransaction(ctx, block, int(index), opt.reexec)
	if err != nil {
		return nil, err
	}
	defer release()

	msg, err := core.TransactionToMessage(tx, types.MakeSigner(api.backend.ChainConfig(), block.Number(), block.Time()), block.BaseFee())
	if err != nil {
		return nil, err
	}
	txctx := &Context{
		BlockHash:   blockHash,
		BlockNumber: block.Number(),
		TxIndex:     int(index),
		TxHash:      hash,
	}
	tracer, err := api.instantiateTracer(config, txctx)
	if err != nil {
		return nil, err
	}
	return api.execTx(ctx, tx, msg, txctx, vmctx, statedb, tracer, nil, opt.txTimeout, api.backend.ChainConfig())
}

// TraceCall lets you trace a given eth_call. It collects the structured logs
// created during the execution of EVM if the given transaction was added on
// top of the provided block and returns them as a JSON object.
// If no transaction index is specified, the trace will be conducted on the state
// after executing the specified block. However, if a transaction index is provided,
// the trace will be conducted on the state after executing the specified transaction
// within the specified block.
func (api *API) TraceCall(ctx context.Context, args ethapi.TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, config *TraceCallConfig) (interface{}, error) {
	// Try to retrieve the specified block
	var (
		err         error
		block       *types.Block
		statedb     *state.StateDB
		release     StateReleaseFunc
		precompiles vm.PrecompiledContracts
	)
	if hash, ok := blockNrOrHash.Hash(); ok {
		block, err = api.blockByHash(ctx, hash)
	} else if number, ok := blockNrOrHash.Number(); ok {
		if number == rpc.PendingBlockNumber {
			// We don't have access to the miner here. For tracing 'future' transactions,
			// it can be done with block- and state-overrides instead, which offers
			// more flexibility and stability than trying to trace on 'pending', since
			// the contents of 'pending' is unstable and probably not a true representation
			// of what the next actual block is likely to contain.
			return nil, errors.New("tracing on top of pending is not supported")
		}
		block, err = api.blockByNumber(ctx, number)
	} else {
		return nil, errors.New("invalid arguments; neither block nor hash specified")
	}
	if err != nil {
		return nil, err
	}
	// try to recompute the state

	opt, err := traceCallExecOpt(config)
	if err != nil {
		return nil, err
	}

	if config != nil && config.TxIndex != nil {
		_, _, statedb, release, err = api.backend.StateAtTransaction(ctx, block, int(*config.TxIndex), opt.reexec)
	} else {
		statedb, release, err = api.backend.StateAtBlock(ctx, block, opt.reexec, nil, true, false)
	}
	if err != nil {
		return nil, err
	}
	defer release()

	vmctx := core.NewEVMBlockContext(block.Header(), api.chainContext(ctx), nil)
	var traceConfig TraceConfig
	// Apply the customization rules if required.
	if config != nil {
		if overrideErr := config.BlockOverrides.Apply(&vmctx); overrideErr != nil {
			return nil, overrideErr
		}
		rules := api.backend.ChainConfig().Rules(vmctx.BlockNumber, vmctx.Random != nil, vmctx.Time)

		precompiles = vm.ActivePrecompiledContracts(rules)
		if err := config.StateOverrides.Apply(statedb, precompiles); err != nil {
			return nil, err
		}
		traceConfig = config.TraceConfig
	}
	// Execute the trace
	if err := args.CallDefaults(api.backend.RPCGasCap(), vmctx.BaseFee, api.backend.ChainConfig().ChainID); err != nil {
		return nil, err
	}
	var (
		msg = args.ToMessage(vmctx.BaseFee, true, true)
		tx  = args.ToTransaction(types.LegacyTxType)
	)
	// Lower the basefee to 0 to avoid breaking EVM
	// invariants (basefee < feecap).
	if msg.GasPrice.Sign() == 0 {
		vmctx.BaseFee = new(big.Int)
	}
	if msg.BlobGasFeeCap != nil && msg.BlobGasFeeCap.BitLen() == 0 {
		vmctx.BlobBaseFee = new(big.Int)
	}
	tracer, err := api.instantiateTracer(&traceConfig, new(Context))
	if err != nil {
		return nil, err
	}
	return api.execTx(ctx, tx, msg, new(Context), vmctx, statedb, tracer, precompiles, opt.txTimeout, api.backend.ChainConfig())
}

func (api *API) instantiateTracer(config *TraceConfig, txctx *Context) (*Tracer, error) {
	var tracer *Tracer
	var err error
	if config == nil {
		// TODO: use this case to simplify the code below
		config = &TraceConfig{}
	}
	// Default tracer is the struct logger
	if config.Tracer == nil {
		logger := logger.NewStructLogger(config.Config)
		tracer = &Tracer{
			Hooks:     logger.Hooks(),
			GetResult: logger.GetResult,
			Stop:      logger.Stop,
		}
	} else {
		chainConfig := api.backend.ChainConfig()
		if config.Config != nil && config.Config.Overrides != nil {
			chainConfig = config.Config.Overrides
		}
		tracer, err = DefaultDirectory.New(*config.Tracer, txctx, config.TracerConfig, chainConfig)
	}
	return tracer, err
}

// execTx executes a transaction against the given statedb, optionally configuring the execution to use a tracer
// if tracer is non-nil.  The result is nil if a tracer is not configured or the value returned from the tracer if it is
// configured.
func (api *API) execTx(ctx context.Context, tx *types.Transaction, message *core.Message, txctx *Context, vmctx vm.BlockContext, statedb *state.StateDB, tracer *Tracer, precompiles vm.PrecompiledContracts, timeout time.Duration, chainConfig *params.ChainConfig) (res interface{}, err error) {
	var (
		usedGas uint64
		db      vm.StateDB = statedb
		evm     *vm.EVM
		vmConf  = vm.Config{NoBaseFee: true}
	)
	if tracer != nil {
		db = state.NewHookedState(statedb, tracer.Hooks)

		deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
		go func() {
			<-deadlineCtx.Done()
			if errors.Is(deadlineCtx.Err(), context.DeadlineExceeded) {
				tracer.Stop(errors.New("execution timeout"))
				// Stop evm execution. Note cancellation is not necessarily immediate.
				evm.Cancel()
			}
		}()
		defer cancel()
		vmConf.Tracer = tracer.Hooks
	}
	evm = vm.NewEVM(vmctx, db, api.backend.ChainConfig(), vmConf)
	if precompiles != nil {
		evm.SetPrecompiles(precompiles)
	}

	statedb.SetTxContext(txctx.TxHash, txctx.TxIndex)
	_, err = core.ApplyTransactionWithEVM(message, new(core.GasPool).AddGas(message.GasLimit), statedb, vmctx.BlockNumber, txctx.BlockHash, vmctx.Time, tx, &usedGas, evm)
	if err != nil {
		return nil, fmt.Errorf("tracing failed: %w", err)
	}
	if tracer != nil {
		res, err = tracer.GetResult()
	}
	return res, err
}

// APIs return the collection of RPC services the tracer package offers.
func APIs(backend Backend) []rpc.API {
	// Append all the local APIs and return
	return []rpc.API{
		{
			Namespace: "debug",
			Service:   NewAPI(backend),
		},
	}
}

// overrideConfig returns a copy of original with forks enabled by override enabled,
// along with a boolean that indicates whether the copy is canonical (equivalent to the original).
// Note: the Clique-part is _not_ deep copied
func overrideConfig(original *params.ChainConfig, override *params.ChainConfig) (*params.ChainConfig, bool) {
	copy := new(params.ChainConfig)
	*copy = *original
	canon := true

	// Apply forks (after Berlin) to the copy.
	if block := override.BerlinBlock; block != nil {
		copy.BerlinBlock = block
		canon = false
	}
	if block := override.LondonBlock; block != nil {
		copy.LondonBlock = block
		canon = false
	}
	if block := override.ArrowGlacierBlock; block != nil {
		copy.ArrowGlacierBlock = block
		canon = false
	}
	if block := override.GrayGlacierBlock; block != nil {
		copy.GrayGlacierBlock = block
		canon = false
	}
	if block := override.MergeNetsplitBlock; block != nil {
		copy.MergeNetsplitBlock = block
		canon = false
	}
	if timestamp := override.ShanghaiTime; timestamp != nil {
		copy.ShanghaiTime = timestamp
		canon = false
	}
	if timestamp := override.CancunTime; timestamp != nil {
		copy.CancunTime = timestamp
		canon = false
	}
	if timestamp := override.PragueTime; timestamp != nil {
		copy.PragueTime = timestamp
		canon = false
	}
	if timestamp := override.OsakaTime; timestamp != nil {
		copy.OsakaTime = timestamp
		canon = false
	}
	if timestamp := override.VerkleTime; timestamp != nil {
		copy.VerkleTime = timestamp
		canon = false
	}

	return copy, canon
}

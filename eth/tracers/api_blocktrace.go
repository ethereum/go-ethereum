package tracers

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/scroll-tech/go-ethereum/trie/zkproof"
)

type TraceBlock interface {
	GetBlockTraceByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (trace *types.BlockTrace, err error)
}

type traceEnv struct {
	config *TraceConfig

	coinbase common.Address
	feeVault common.Address

	// rMu lock is used to protect txs executed in parallel.
	signer   types.Signer
	state    *state.StateDB
	blockCtx vm.BlockContext

	// pMu lock is used to protect Proofs' read and write mutual exclusion,
	// since txs are executed in parallel, so this lock is required.
	pMu sync.Mutex
	// sMu is required because of txs are executed in parallel,
	// this lock is used to protect StorageTrace's read and write mutual exclusion.
	sMu sync.Mutex
	*types.StorageTrace
	executionResults []*types.ExecutionResult
}

// GetBlockTraceByNumberOrHash replays the block and returns the structured BlockTrace by hash or number.
func (api *API) GetBlockTraceByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (trace *types.BlockTrace, err error) {
	var block *types.Block
	if number, ok := blockNrOrHash.Number(); ok {
		block, err = api.blockByNumber(ctx, number)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		block, err = api.blockByHash(ctx, hash)
	}
	if err != nil {
		return nil, err
	}
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	if config == nil {
		config = &TraceConfig{
			LogConfig: &vm.LogConfig{
				EnableMemory:     false,
				EnableReturnData: true,
			},
		}
	} else if config.Tracer != nil {
		config.Tracer = nil
		log.Warn("Tracer params is unsupported")
	}

	// create current execution environment.
	env, err := api.createTraceEnv(ctx, config, block)
	if err != nil {
		return nil, err
	}

	return api.getBlockTrace(block, env)
}

// Make trace environment for current block.
func (api *API) createTraceEnv(ctx context.Context, config *TraceConfig, block *types.Block) (*traceEnv, error) {
	parent, err := api.blockByNumberAndHash(ctx, rpc.BlockNumber(block.NumberU64()-1), block.ParentHash())
	if err != nil {
		return nil, err
	}
	reexec := defaultTraceReexec
	if config != nil && config.Reexec != nil {
		reexec = *config.Reexec
	}
	statedb, err := api.backend.StateAtBlock(ctx, parent, reexec, nil, true, true)
	if err != nil {
		return nil, err
	}

	// get coinbase
	coinbase, err := api.backend.Engine().Author(block.Header())
	if err != nil {
		return nil, err
	}

	// get feeeVaultAddress
	var feeVault common.Address
	if api.backend.ChainConfig().FeeVaultAddress != nil {
		feeVault = *api.backend.ChainConfig().FeeVaultAddress
	} else {
		feeVault = coinbase
	}

	env := &traceEnv{
		config:   config,
		coinbase: coinbase,
		feeVault: feeVault,
		signer:   types.MakeSigner(api.backend.ChainConfig(), block.Number()),
		state:    statedb,
		blockCtx: core.NewEVMBlockContext(block.Header(), api.chainContext(ctx), nil),
		StorageTrace: &types.StorageTrace{
			RootBefore:    parent.Root(),
			RootAfter:     block.Root(),
			Proofs:        make(map[string][]hexutil.Bytes),
			StorageProofs: make(map[string]map[string][]hexutil.Bytes),
		},
		executionResults: make([]*types.ExecutionResult, block.Transactions().Len()),
	}

	key := coinbase.String()
	if _, exist := env.Proofs[key]; !exist {
		proof, err := env.state.GetProof(coinbase)
		if err != nil {
			log.Error("Proof for coinbase not available", "coinbase", coinbase, "error", err)
			// but we still mark the proofs map with nil array
		}
		wrappedProof := make([]hexutil.Bytes, len(proof))
		for i, bt := range proof {
			wrappedProof[i] = bt
		}
		env.Proofs[key] = wrappedProof
	}

	key = feeVault.String()
	if _, exist := env.Proofs[key]; !exist {
		proof, err := env.state.GetProof(feeVault)
		if err != nil {
			log.Error("Proof for feeVault not available", "feeVault", feeVault, "error", err)
			// but we still mark the proofs map with nil array
		}
		wrappedProof := make([]hexutil.Bytes, len(proof))
		for i, bt := range proof {
			wrappedProof[i] = bt
		}
		env.Proofs[key] = wrappedProof
	}

	return env, nil
}

func (api *API) getBlockTrace(block *types.Block, env *traceEnv) (*types.BlockTrace, error) {
	// Execute all the transaction contained within the block concurrently
	var (
		txs   = block.Transactions()
		pend  = new(sync.WaitGroup)
		jobs  = make(chan *txTraceTask, len(txs))
		errCh = make(chan error, 1)
	)
	threads := runtime.NumCPU()
	if threads > len(txs) {
		threads = len(txs)
	}
	for th := 0; th < threads; th++ {
		pend.Add(1)
		go func() {
			defer pend.Done()
			// Fetch and execute the next transaction trace tasks
			for task := range jobs {
				if err := api.getTxResult(env, task.statedb, task.index, block); err != nil {
					select {
					case errCh <- err:
					default:
					}
					log.Error("failed to trace tx", "txHash", txs[task.index].Hash().String())
				}
			}
		}()
	}

	// Feed the transactions into the tracers and return
	var failed error
	for i, tx := range txs {
		// Send the trace task over for execution
		jobs <- &txTraceTask{statedb: env.state.Copy(), index: i}

		// Generate the next state snapshot fast without tracing
		msg, _ := tx.AsMessage(env.signer, block.BaseFee())
		env.state.Prepare(tx.Hash(), i)
		vmenv := vm.NewEVM(env.blockCtx, core.NewEVMTxContext(msg), env.state, api.backend.ChainConfig(), vm.Config{})
		if _, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(msg.Gas())); err != nil {
			failed = err
			break
		}
		// Finalize the state so any modifications are written to the trie
		// Only delete empty objects if EIP158/161 (a.k.a Spurious Dragon) is in effect
		env.state.Finalise(vmenv.ChainConfig().IsEIP158(block.Number()))
	}
	close(jobs)
	pend.Wait()

	// If execution failed in between, abort
	select {
	case err := <-errCh:
		return nil, err
	default:
		if failed != nil {
			return nil, failed
		}
	}

	return api.fillBlockTrace(env, block)
}

func (api *API) getTxResult(env *traceEnv, state *state.StateDB, index int, block *types.Block) error {
	tx := block.Transactions()[index]
	msg, _ := tx.AsMessage(env.signer, block.BaseFee())
	from, _ := types.Sender(env.signer, tx)
	to := tx.To()

	txctx := &Context{
		BlockHash: block.TxHash(),
		TxIndex:   index,
		TxHash:    tx.Hash(),
	}

	sender := &types.AccountWrapper{
		Address:  from,
		Nonce:    state.GetNonce(from),
		Balance:  (*hexutil.Big)(state.GetBalance(from)),
		CodeHash: state.GetCodeHash(from),
	}
	var receiver *types.AccountWrapper
	if to != nil {
		receiver = &types.AccountWrapper{
			Address:  *to,
			Nonce:    state.GetNonce(*to),
			Balance:  (*hexutil.Big)(state.GetBalance(*to)),
			CodeHash: state.GetCodeHash(*to),
		}
	}

	tracer := vm.NewStructLogger(env.config.LogConfig)
	// Run the transaction with tracing enabled.
	vmenv := vm.NewEVM(env.blockCtx, core.NewEVMTxContext(msg), state, api.backend.ChainConfig(), vm.Config{Debug: true, Tracer: tracer, NoBaseFee: true})

	// Call Prepare to clear out the statedb access list
	state.Prepare(txctx.TxHash, txctx.TxIndex)

	// Computes the new state by applying the given message.
	result, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(msg.Gas()))
	if err != nil {
		return fmt.Errorf("tracing failed: %w", err)
	}
	// If the result contains a revert reason, return it.
	returnVal := result.Return()
	if len(result.Revert()) > 0 {
		returnVal = result.Revert()
	}

	createdAcc := tracer.CreatedAccount()
	var after []*types.AccountWrapper
	if to == nil {
		if createdAcc == nil {
			return errors.New("unexpected tx: address for created contract unavailable")
		}
		to = &createdAcc.Address
	}
	// collect affected account after tx being applied
	afterAccounts := []common.Address{from, *to, env.coinbase}
	if env.coinbase != env.feeVault {
		afterAccounts = append(afterAccounts, env.feeVault)
	}
	for _, acc := range afterAccounts {
		after = append(after, &types.AccountWrapper{
			Address:  acc,
			Nonce:    state.GetNonce(acc),
			Balance:  (*hexutil.Big)(state.GetBalance(acc)),
			CodeHash: state.GetCodeHash(acc),
		})
	}

	// merge required proof data
	proofAccounts := tracer.UpdatedAccounts()
	for addr := range proofAccounts {
		addrStr := addr.String()

		env.pMu.Lock()
		_, existed := env.Proofs[addrStr]
		env.pMu.Unlock()
		if existed {
			continue
		}
		proof, err := state.GetProof(addr)
		if err != nil {
			log.Error("Proof not available", "address", addrStr, "error", err)
			// but we still mark the proofs map with nil array
		}
		wrappedProof := make([]hexutil.Bytes, len(proof))
		for i, bt := range proof {
			wrappedProof[i] = bt
		}
		env.pMu.Lock()
		env.Proofs[addrStr] = wrappedProof
		env.pMu.Unlock()
	}

	proofStorages := tracer.UpdatedStorages()
	for addr, keys := range proofStorages {
		for key := range keys {
			addrStr := addr.String()
			keyStr := key.String()

			env.sMu.Lock()
			m, existed := env.StorageProofs[addrStr]
			if !existed {
				m = make(map[string][]hexutil.Bytes)
				env.StorageProofs[addrStr] = m
			} else if _, existed := m[keyStr]; existed {
				env.sMu.Unlock()
				continue
			}
			env.sMu.Unlock()

			proof, err := state.GetStorageTrieProof(addr, key)
			if err != nil {
				log.Error("Storage proof not available", "error", err, "address", addrStr, "key", keyStr)
				// but we still mark the proofs map with nil array
			}
			wrappedProof := make([]hexutil.Bytes, len(proof))
			for i, bt := range proof {
				wrappedProof[i] = bt
			}
			env.sMu.Lock()
			m[keyStr] = wrappedProof
			env.sMu.Unlock()
		}
	}

	env.executionResults[index] = &types.ExecutionResult{
		From:           sender,
		To:             receiver,
		AccountCreated: createdAcc,
		AccountsAfter:  after,
		Gas:            result.UsedGas,
		Failed:         result.Failed(),
		ReturnValue:    fmt.Sprintf("%x", returnVal),
		StructLogs:     vm.FormatLogs(tracer.StructLogs()),
	}

	return nil
}

// Fill blockTrace content after all the txs are finished running.
func (api *API) fillBlockTrace(env *traceEnv, block *types.Block) (*types.BlockTrace, error) {
	statedb := env.state

	txs := make([]*types.TransactionData, block.Transactions().Len())
	for i, tx := range block.Transactions() {
		txs[i] = types.NewTransactionData(tx, block.NumberU64(), api.backend.ChainConfig())
	}

	blockTrace := &types.BlockTrace{
		Coinbase: &types.AccountWrapper{
			Address:  env.coinbase,
			Nonce:    statedb.GetNonce(env.coinbase),
			Balance:  (*hexutil.Big)(statedb.GetBalance(env.coinbase)),
			CodeHash: statedb.GetCodeHash(env.coinbase),
		},
		FeeVault: &types.AccountWrapper{
			Address:  env.feeVault,
			Nonce:    statedb.GetNonce(env.feeVault),
			Balance:  (*hexutil.Big)(statedb.GetBalance(env.feeVault)),
			CodeHash: statedb.GetCodeHash(env.feeVault),
		},
		Header:           block.Header(),
		StorageTrace:     env.StorageTrace,
		ExecutionResults: env.executionResults,
		Transactions:     txs,
	}

	for i, tx := range block.Transactions() {
		evmTrace := env.executionResults[i]
		// probably a Contract Call
		if len(tx.Data()) != 0 && tx.To() != nil {
			evmTrace.ByteCode = hexutil.Encode(statedb.GetCode(*tx.To()))
			// Get tx.to address's code hash.
			codeHash := statedb.GetCodeHash(*tx.To())
			evmTrace.CodeHash = &codeHash
		} else if tx.To() == nil { // Contract is created.
			evmTrace.ByteCode = hexutil.Encode(tx.Data())
		}
	}

	// only zktrie model has the ability to get `mptwitness`.
	if api.backend.ChainConfig().Zktrie {
		if err := zkproof.FillBlockTraceForMPTWitness(zkproof.MPTWitnessType(api.backend.CacheConfig().MPTWitness), blockTrace); err != nil {
			log.Error("fill mpt witness fail", "error", err)
		}
	}

	return blockTrace, nil
}

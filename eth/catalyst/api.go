// Copyright 2020 The go-ethereum Authors
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

// Package catalyst implements the temporary eth1/eth2 RPC integration.
package catalyst

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	chainParams "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

// Register adds catalyst APIs to the full node.
func Register(stack *node.Node, backend *eth.Ethereum) error {
	log.Warn("Catalyst mode enabled", "protocol", "eth")
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace: "consensus",
			Version:   "1.0",
			Service:   NewConsensusAPI(backend, nil),
			Public:    true,
		},
	})
	return nil
}

// RegisterLight adds catalyst APIs to the light client.
func RegisterLight(stack *node.Node, backend *les.LightEthereum) error {
	log.Warn("Catalyst mode enabled", "protocol", "les")
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace: "consensus",
			Version:   "1.0",
			Service:   NewConsensusAPI(nil, backend),
			Public:    true,
		},
	})
	return nil
}

type ConsensusAPI struct {
	light  bool
	eth    *eth.Ethereum
	les    *les.LightEthereum
	engine consensus.Engine // engine is the post-merge consensus engine, only for block creation
	syncer *syncer          // syncer is responsible for triggering chain sync
}

func NewConsensusAPI(eth *eth.Ethereum, les *les.LightEthereum) *ConsensusAPI {
	var engine consensus.Engine
	if eth == nil {
		if b, ok := les.Engine().(*beacon.Beacon); ok {
			engine = beacon.New(b.InnerEngine(), true)
		} else {
			engine = beacon.New(les.Engine(), true)
		}
	} else {
		if b, ok := eth.Engine().(*beacon.Beacon); ok {
			engine = beacon.New(b.InnerEngine(), true)
		} else {
			engine = beacon.New(eth.Engine(), true)
		}
	}
	return &ConsensusAPI{
		light:  eth == nil,
		eth:    eth,
		les:    les,
		engine: engine,
		syncer: newSyncer(),
	}
}

// blockExecutionEnv gathers all the data required to execute
// a block, either when assembling it or when inserting it.
type blockExecutionEnv struct {
	chain   *core.BlockChain
	state   *state.StateDB
	tcount  int
	gasPool *core.GasPool

	header   *types.Header
	txs      []*types.Transaction
	receipts []*types.Receipt
}

func (env *blockExecutionEnv) commitTransaction(tx *types.Transaction, coinbase common.Address) error {
	vmconfig := *env.chain.GetVMConfig()
	snap := env.state.Snapshot()
	receipt, err := core.ApplyTransaction(env.chain.Config(), env.chain, &coinbase, env.gasPool, env.state, env.header, tx, &env.header.GasUsed, vmconfig)
	if err != nil {
		env.state.RevertToSnapshot(snap)
		return err
	}
	env.txs = append(env.txs, tx)
	env.receipts = append(env.receipts, receipt)
	return nil
}

func (api *ConsensusAPI) makeEnv(parent *types.Block, header *types.Header) (*blockExecutionEnv, error) {
	// The parent state might be missing. It can be the special scenario
	// that consensus layer tries to build a new block based on the very
	// old side chain block and the relevant state is already pruned. So
	// try to retrieve the live state from the chain, if it's not existent,
	// do the necessary recovery work.
	var (
		err   error
		state *state.StateDB
	)
	if api.eth.BlockChain().HasState(parent.Root()) {
		state, err = api.eth.BlockChain().StateAt(parent.Root())
	} else {
		// The maximum acceptable reorg depth can be limited by the
		// finalised block somehow. TODO(rjl493456442) fix the hard-
		// coded number here later.
		state, err = api.eth.StateAtBlock(parent, 1000, nil, false)
	}
	if err != nil {
		return nil, err
	}
	env := &blockExecutionEnv{
		chain:   api.eth.BlockChain(),
		state:   state,
		header:  header,
		gasPool: new(core.GasPool).AddGas(header.GasLimit),
	}
	return env, nil
}

// AssembleBlock creates a new block, inserts it into the chain, and returns the "execution
// data" required for eth2 clients to process the new block.
func (api *ConsensusAPI) AssembleBlock(params AssembleBlockParams) (*ExecutableData, error) {
	if api.light {
		return nil, errors.New("not supported")
	}
	log.Info("Producing block", "parentHash", params.ParentHash)

	bc := api.eth.BlockChain()
	parent := bc.GetBlockByHash(params.ParentHash)
	if parent == nil {
		log.Warn("Cannot assemble block with parent hash to unknown block", "parentHash", params.ParentHash)
		return nil, fmt.Errorf("cannot assemble block with unknown parent %s", params.ParentHash)
	}

	if parent.Time() >= params.Timestamp {
		return nil, fmt.Errorf("child timestamp lower than parent's: %d >= %d", parent.Time(), params.Timestamp)
	}
	if now := uint64(time.Now().Unix()); params.Timestamp > now+1 {
		wait := time.Duration(params.Timestamp-now) * time.Second
		log.Info("Producing block too far in the future", "wait", common.PrettyDuration(wait))
		time.Sleep(wait)
	}

	pool := api.eth.TxPool()
	pending, err := pool.Pending()
	if err != nil {
		return nil, err
	}

	coinbase, err := api.eth.Etherbase()
	if err != nil {
		return nil, err
	}
	num := parent.Number()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     num.Add(num, common.Big1),
		Coinbase:   coinbase,
		GasLimit:   parent.GasLimit(), // Keep the gas limit constant in this prototype
		Extra:      []byte{},
		Time:       params.Timestamp,
	}
	err = api.engine.Prepare(bc, header)
	if err != nil {
		return nil, err
	}

	env, err := api.makeEnv(parent, header)
	if err != nil {
		return nil, err
	}

	var (
		signer       = types.MakeSigner(bc.Config(), header.Number)
		txHeap       = types.NewTransactionsByPriceAndNonce(signer, pending)
		transactions []*types.Transaction
	)
	for {
		if env.gasPool.Gas() < chainParams.TxGas {
			log.Trace("Not enough gas for further transactions", "have", env.gasPool, "want", chainParams.TxGas)
			break
		}
		tx := txHeap.Peek()
		if tx == nil {
			break
		}

		// The sender is only for logging purposes, and it doesn't really matter if it's correct.
		from, _ := types.Sender(signer, tx)

		// Execute the transaction
		env.state.Prepare(tx.Hash(), common.Hash{}, env.tcount)
		err = env.commitTransaction(tx, coinbase)
		switch err {
		case core.ErrGasLimitReached:
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Trace("Gas limit exceeded for current block", "sender", from)
			txHeap.Pop()

		case core.ErrNonceTooLow:
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txHeap.Shift()

		case core.ErrNonceTooHigh:
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Trace("Skipping account with high nonce", "sender", from, "nonce", tx.Nonce())
			txHeap.Pop()

		case nil:
			// Everything ok, collect the logs and shift in the next transaction from the same account
			env.tcount++
			txHeap.Shift()
			transactions = append(transactions, tx)

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txHeap.Shift()
		}
	}

	// Create the block.
	block, err := api.engine.FinalizeAndAssemble(bc, header, env.state, transactions, nil /* uncles */, env.receipts)
	if err != nil {
		return nil, err
	}
	return &ExecutableData{
		BlockHash:    block.Hash(),
		ParentHash:   block.ParentHash(),
		Miner:        block.Coinbase(),
		StateRoot:    block.Root(),
		Number:       block.NumberU64(),
		GasLimit:     block.GasLimit(),
		GasUsed:      block.GasUsed(),
		Timestamp:    block.Time(),
		ReceiptRoot:  block.ReceiptHash(),
		LogsBloom:    block.Bloom().Bytes(),
		Transactions: encodeTransactions(block.Transactions()),
	}, nil
}

func encodeTransactions(txs []*types.Transaction) [][]byte {
	var enc = make([][]byte, len(txs))
	for i, tx := range txs {
		enc[i], _ = tx.MarshalBinary()
	}
	return enc
}

func decodeTransactions(enc [][]byte) ([]*types.Transaction, error) {
	var txs = make([]*types.Transaction, len(enc))
	for i, encTx := range enc {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(encTx); err != nil {
			return nil, fmt.Errorf("invalid transaction %d: %v", i, err)
		}
		txs[i] = &tx
	}
	return txs, nil
}

func InsertBlockParamsToBlock(params ExecutableData) (*types.Block, error) {
	txs, err := decodeTransactions(params.Transactions)
	if err != nil {
		return nil, err
	}
	number := big.NewInt(0)
	number.SetUint64(params.Number)
	header := &types.Header{
		ParentHash:  params.ParentHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    params.Miner,
		Root:        params.StateRoot,
		TxHash:      types.DeriveSha(types.Transactions(txs), trie.NewStackTrie(nil)),
		ReceiptHash: params.ReceiptRoot,
		Bloom:       types.BytesToBloom(params.LogsBloom),
		Difficulty:  big.NewInt(1),
		Number:      number,
		GasLimit:    params.GasLimit,
		GasUsed:     params.GasUsed,
		Time:        params.Timestamp,
	}
	block := types.NewBlockWithHeader(header).WithBody(txs, nil /* uncles */)
	return block, nil
}

// NewBlock creates an Eth1 block, inserts it in the chain, and either returns true,
// or false + an error. This is a bit redundant for go, but simplifies things on the
// eth2 side.
func (api *ConsensusAPI) NewBlock(params ExecutableData) (*NewBlockResponse, error) {
	block, err := InsertBlockParamsToBlock(params)
	if err != nil {
		return nil, err
	}
	if api.light {
		parent := api.les.BlockChain().GetHeaderByHash(block.ParentHash())
		if parent == nil {
			return &NewBlockResponse{false}, fmt.Errorf("could not find parent %d %x", block.NumberU64(), block.ParentHash())
		}
		err = api.les.BlockChain().InsertHeader(block.Header())
		return &NewBlockResponse{err == nil}, err
	}
	parent := api.eth.BlockChain().GetBlockByHash(block.ParentHash())
	if parent == nil {
		return &NewBlockResponse{false}, fmt.Errorf("could not find parent %d %x", block.NumberU64(), block.ParentHash())
	}
	err = api.eth.BlockChain().InsertBlock(block)
	return &NewBlockResponse{err == nil}, err
}

// Used in tests to add a the list of transactions from a block to the tx pool.
func (api *ConsensusAPI) addBlockTxs(block *types.Block) error {
	for _, tx := range block.Transactions() {
		api.eth.TxPool().AddLocal(tx)
	}
	return nil
}

// FinalizeBlock is called to mark a block as synchronized, so
// that data that is no longer needed can be removed.
func (api *ConsensusAPI) FinalizeBlock(blockHash common.Hash) (*GenericResponse, error) {
	// Finalize the transition if it's the first `FinalisedBlock` event.
	merger := api.merger()
	if !merger.EnteredPoS() {
		merger.EnterPoS()
	}
	return &GenericResponse{true}, nil
}

// SetHead is called to perform a force choice.
func (api *ConsensusAPI) SetHead(newHead common.Hash) (*GenericResponse, error) {
	// Trigger the transition if it's the first `NewHead` event.
	merger := api.merger()
	if !merger.LeftPoW() {
		merger.LeavePoW()
	}
	if api.light {
		headHeader := api.les.BlockChain().CurrentHeader()
		if headHeader.Hash() == newHead {
			return &GenericResponse{true}, nil
		}
		newHeadHeader := api.les.BlockChain().GetHeaderByHash(newHead)
		if newHeadHeader == nil {
			return &GenericResponse{false}, nil
		}
		if err := api.les.BlockChain().SetChainHead(newHeadHeader); err != nil {
			return &GenericResponse{false}, nil
		}
		return &GenericResponse{true}, nil
	}
	headBlock := api.eth.BlockChain().CurrentBlock()
	if headBlock.Hash() == newHead {
		return &GenericResponse{true}, nil
	}
	newHeadBlock := api.eth.BlockChain().GetBlockByHash(newHead)
	if newHeadBlock == nil {
		return &GenericResponse{false}, nil
	}
	if err := api.eth.BlockChain().SetChainHead(newHeadBlock); err != nil {
		return &GenericResponse{false}, nil
	}
	api.eth.SetSynced()
	return &GenericResponse{true}, nil
}

// Helper function, return the merger instance.
func (api *ConsensusAPI) merger() *core.Merger {
	if api.light {
		return api.les.Merger()
	}
	return api.eth.Merger()
}

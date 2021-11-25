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
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/misc"
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

var (
	VALID          = GenericStringResponse{"VALID"}
	INVALID        = GenericStringResponse{"INVALID"}
	SYNCING        = GenericStringResponse{"SYNCING"}
	UnknownHeader  = rpc.CustomError{Code: -32000, Message: "unknown header"}
	UnknownPayload = rpc.CustomError{Code: -32001, Message: "unknown payload"}
)

// Register adds catalyst APIs to the full node.
func Register(stack *node.Node, backend *eth.Ethereum) error {
	log.Warn("Catalyst mode enabled", "protocol", "eth")
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace: "engine",
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
			Namespace: "engine",
			Version:   "1.0",
			Service:   NewConsensusAPI(nil, backend),
			Public:    true,
		},
	})
	return nil
}

type ConsensusAPI struct {
	light          bool
	eth            *eth.Ethereum
	les            *les.LightEthereum
	engine         consensus.Engine // engine is the post-merge consensus engine, only for block creation
	preparedBlocks map[int]*ExecutableData
}

func NewConsensusAPI(eth *eth.Ethereum, les *les.LightEthereum) *ConsensusAPI {
	var engine consensus.Engine
	if eth == nil {
		if les.BlockChain().Config().TerminalTotalDifficulty == nil {
			panic("Catalyst started without valid total difficulty")
		}
		if b, ok := les.Engine().(*beacon.Beacon); ok {
			engine = beacon.New(b.InnerEngine())
		} else {
			engine = beacon.New(les.Engine())
		}
	} else {
		if eth.BlockChain().Config().TerminalTotalDifficulty == nil {
			panic("Catalyst started without valid total difficulty")
		}
		if b, ok := eth.Engine().(*beacon.Beacon); ok {
			engine = beacon.New(b.InnerEngine())
		} else {
			engine = beacon.New(eth.Engine())
		}
	}
	return &ConsensusAPI{
		light:          eth == nil,
		eth:            eth,
		les:            les,
		engine:         engine,
		preparedBlocks: make(map[int]*ExecutableData),
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
		state, err = api.eth.StateAtBlock(parent, 1000, nil, false, false)
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

func (api *ConsensusAPI) PreparePayload(params AssembleBlockParams) (*PayloadResponse, error) {
	data, err := api.assembleBlock(params)
	if err != nil {
		return nil, err
	}
	id := len(api.preparedBlocks)
	api.preparedBlocks[id] = data
	return &PayloadResponse{PayloadID: uint64(id)}, nil
}

func (api *ConsensusAPI) GetPayload(PayloadID hexutil.Uint64) (*ExecutableData, error) {
	data, ok := api.preparedBlocks[int(PayloadID)]
	if !ok {
		return nil, &UnknownPayload
	}
	return data, nil
}

// ConsensusValidated is called to mark a block as valid, so
// that data that is no longer needed can be removed.
func (api *ConsensusAPI) ConsensusValidated(params ConsensusValidatedParams) error {
	switch params.Status {
	case VALID.Status:
		return nil
	case INVALID.Status:
		// TODO (MariusVanDerWijden) delete the block from the bc
		return nil
	default:
		return errors.New("invalid params.status")
	}
}

func (api *ConsensusAPI) ForkchoiceUpdated(params ForkChoiceParams) error {
	var emptyHash = common.Hash{}
	if !bytes.Equal(params.HeadBlockHash[:], emptyHash[:]) {
		if err := api.checkTerminalTotalDifficulty(params.HeadBlockHash); err != nil {
			return err
		}
		return api.setHead(params.HeadBlockHash)
	}
	return nil
}

// ExecutePayload creates an Eth1 block, inserts it in the chain, and returns the status of the chain.
func (api *ConsensusAPI) ExecutePayload(params ExecutableData) (GenericStringResponse, error) {
	block, err := ExecutableDataToBlock(params)
	if err != nil {
		return INVALID, err
	}
	if api.light {
		parent := api.les.BlockChain().GetHeaderByHash(params.ParentHash)
		if parent == nil {
			return INVALID, fmt.Errorf("could not find parent %x", params.ParentHash)
		}
		if err = api.les.BlockChain().InsertHeader(block.Header()); err != nil {
			return INVALID, err
		}
		return VALID, nil
	}
	if !api.eth.BlockChain().HasBlock(block.ParentHash(), block.NumberU64()-1) {
		/*
			TODO (MariusVanDerWijden) reenable once sync is merged
			if err := api.eth.Downloader().BeaconSync(api.eth.SyncMode(), block.Header()); err != nil {
				return SYNCING, err
			}
		*/
		return SYNCING, nil
	}
	parent := api.eth.BlockChain().GetBlockByHash(params.ParentHash)
	td := api.eth.BlockChain().GetTd(parent.Hash(), block.NumberU64()-1)
	ttd := api.eth.BlockChain().Config().TerminalTotalDifficulty
	if td.Cmp(ttd) < 0 {
		return INVALID, fmt.Errorf("can not execute payload on top of block with low td got: %v threshold %v", td, ttd)
	}
	if err := api.eth.BlockChain().InsertBlockWithoutSetHead(block); err != nil {
		return INVALID, err
	}
	merger := api.merger()
	if !merger.TDDReached() {
		merger.ReachTTD()
	}
	return VALID, nil
}

// AssembleBlock creates a new block, inserts it into the chain, and returns the "execution
// data" required for eth2 clients to process the new block.
func (api *ConsensusAPI) assembleBlock(params AssembleBlockParams) (*ExecutableData, error) {
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

	if params.Timestamp < parent.Time() {
		return nil, fmt.Errorf("child timestamp lower than parent's: %d < %d", params.Timestamp, parent.Time())
	}
	if now := uint64(time.Now().Unix()); params.Timestamp > now+1 {
		diff := time.Duration(params.Timestamp-now) * time.Second
		log.Warn("Producing block too far in the future", "diff", common.PrettyDuration(diff))
	}
	pending := api.eth.TxPool().Pending(true)
	coinbase := params.FeeRecipient
	num := parent.Number()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     num.Add(num, common.Big1),
		Coinbase:   coinbase,
		GasLimit:   parent.GasLimit(), // Keep the gas limit constant in this prototype
		Extra:      []byte{},          // TODO (MariusVanDerWijden) properly set extra data
		Time:       params.Timestamp,
	}
	if config := api.eth.BlockChain().Config(); config.IsLondon(header.Number) {
		header.BaseFee = misc.CalcBaseFee(config, parent.Header())
	}
	if err := api.engine.Prepare(bc, header); err != nil {
		return nil, err
	}
	env, err := api.makeEnv(parent, header)
	if err != nil {
		return nil, err
	}
	var (
		signer       = types.MakeSigner(bc.Config(), header.Number)
		txHeap       = types.NewTransactionsByPriceAndNonce(signer, pending, nil)
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
		env.state.Prepare(tx.Hash(), env.tcount)
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
	return BlockToExecutableData(block, params.Random), nil
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

func ExecutableDataToBlock(params ExecutableData) (*types.Block, error) {
	txs, err := decodeTransactions(params.Transactions)
	if err != nil {
		return nil, err
	}
	if len(params.ExtraData) > 32 {
		return nil, fmt.Errorf("invalid extradata length: %v", len(params.ExtraData))
	}
	number := big.NewInt(0)
	number.SetUint64(params.Number)
	header := &types.Header{
		ParentHash:  params.ParentHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    params.Coinbase,
		Root:        params.StateRoot,
		TxHash:      types.DeriveSha(types.Transactions(txs), trie.NewStackTrie(nil)),
		ReceiptHash: params.ReceiptRoot,
		Bloom:       types.BytesToBloom(params.LogsBloom),
		Difficulty:  common.Big0,
		Number:      number,
		GasLimit:    params.GasLimit,
		GasUsed:     params.GasUsed,
		Time:        params.Timestamp,
		BaseFee:     params.BaseFeePerGas,
		Extra:       params.ExtraData,
		// TODO (MariusVanDerWijden) add params.Random to header once required
	}
	block := types.NewBlockWithHeader(header).WithBody(txs, nil /* uncles */)
	if block.Hash() != params.BlockHash {
		return nil, fmt.Errorf("blockhash mismatch, want %x, got %x", params.BlockHash, block.Hash())
	}
	return block, nil
}

func BlockToExecutableData(block *types.Block, random common.Hash) *ExecutableData {
	return &ExecutableData{
		BlockHash:     block.Hash(),
		ParentHash:    block.ParentHash(),
		Coinbase:      block.Coinbase(),
		StateRoot:     block.Root(),
		Number:        block.NumberU64(),
		GasLimit:      block.GasLimit(),
		GasUsed:       block.GasUsed(),
		BaseFeePerGas: block.BaseFee(),
		Timestamp:     block.Time(),
		ReceiptRoot:   block.ReceiptHash(),
		LogsBloom:     block.Bloom().Bytes(),
		Transactions:  encodeTransactions(block.Transactions()),
		Random:        random,
		ExtraData:     block.Extra(),
	}
}

// Used in tests to add a the list of transactions from a block to the tx pool.
func (api *ConsensusAPI) insertTransactions(txs types.Transactions) error {
	for _, tx := range txs {
		api.eth.TxPool().AddLocal(tx)
	}
	return nil
}

func (api *ConsensusAPI) checkTerminalTotalDifficulty(head common.Hash) error {
	// shortcut if we entered PoS already
	if api.merger().PoSFinalized() {
		return nil
	}
	// make sure the parent has enough terminal total difficulty
	newHeadBlock := api.eth.BlockChain().GetBlockByHash(head)
	if newHeadBlock == nil {
		return &UnknownHeader
	}
	parent := api.eth.BlockChain().GetBlockByHash(newHeadBlock.ParentHash())
	if parent == nil {
		return fmt.Errorf("parent unavailable: %v", newHeadBlock.ParentHash())
	}
	td := api.eth.BlockChain().GetTd(parent.Hash(), parent.NumberU64())
	if td != nil && td.Cmp(api.eth.BlockChain().Config().TerminalTotalDifficulty) < 0 {
		return errors.New("total difficulty not reached yet")
	}
	return nil
}

// setHead is called to perform a force choice.
func (api *ConsensusAPI) setHead(newHead common.Hash) error {
	// Trigger the transition if it's the first `NewHead` event.
	merger := api.merger()
	if !merger.PoSFinalized() {
		merger.FinalizePoS()
	}
	log.Info("Setting head", "head", newHead)
	if api.light {
		headHeader := api.les.BlockChain().CurrentHeader()
		if headHeader.Hash() == newHead {
			return nil
		}
		newHeadHeader := api.les.BlockChain().GetHeaderByHash(newHead)
		if newHeadHeader == nil {
			return &UnknownHeader
		}
		if err := api.les.BlockChain().SetChainHead(newHeadHeader); err != nil {
			return err
		}
		return nil
	}
	headBlock := api.eth.BlockChain().CurrentBlock()
	if headBlock.Hash() == newHead {
		return nil
	}
	newHeadBlock := api.eth.BlockChain().GetBlockByHash(newHead)
	if newHeadBlock == nil {
		return &UnknownHeader
	}
	if err := api.eth.BlockChain().SetChainHead(newHeadBlock); err != nil {
		return err
	}
	api.eth.SetSynced()
	return nil
}

// Helper function, return the merger instance.
func (api *ConsensusAPI) merger() *consensus.Merger {
	if api.light {
		return api.les.Merger()
	}
	return api.eth.Merger()
}

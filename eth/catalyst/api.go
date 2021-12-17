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
	"crypto/sha256"
	"encoding/binary"
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
	VALID              = GenericStringResponse{"VALID"}
	SUCCESS            = GenericStringResponse{"SUCCESS"}
	INVALID            = ForkChoiceResponse{Status: "INVALID", PayloadID: nil}
	SYNCING            = ForkChoiceResponse{Status: "SYNCING", PayloadID: nil}
	GenericServerError = rpc.CustomError{Code: -32000, ValidationError: "Server error"}
	UnknownPayload     = rpc.CustomError{Code: -32001, ValidationError: "Unknown payload"}
	InvalidTB          = rpc.CustomError{Code: -32002, ValidationError: "Invalid terminal block"}
	InvalidPayloadID   = rpc.CustomError{Code: 1, ValidationError: "invalid payload id"}
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
	preparedBlocks map[uint64]*ExecutableDataV1
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
		preparedBlocks: make(map[uint64]*ExecutableDataV1),
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

func (api *ConsensusAPI) GetPayloadV1(payloadID hexutil.Bytes) (*ExecutableDataV1, error) {
	hash := []byte(payloadID)
	if len(hash) < 8 {
		return nil, &InvalidPayloadID
	}
	id := binary.BigEndian.Uint64(hash[:8])
	data, ok := api.preparedBlocks[id]
	if !ok {
		return nil, &UnknownPayload
	}
	return data, nil
}

func (api *ConsensusAPI) ForkchoiceUpdatedV1(heads ForkchoiceStateV1, PayloadAttributes *PayloadAttributesV1) (ForkChoiceResponse, error) {
	if heads.HeadBlockHash == (common.Hash{}) {
		return ForkChoiceResponse{Status: SUCCESS.Status, PayloadID: nil}, nil
	}
	if err := api.checkTerminalTotalDifficulty(heads.HeadBlockHash); err != nil {
		if block := api.eth.BlockChain().GetBlockByHash(heads.HeadBlockHash); block == nil {
			// TODO (MariusVanDerWijden) trigger sync
			return SYNCING, nil
		}
		return INVALID, err
	}
	// If the finalized block is set, check if it is in our blockchain
	if heads.FinalizedBlockHash != (common.Hash{}) {
		if block := api.eth.BlockChain().GetBlockByHash(heads.FinalizedBlockHash); block == nil {
			// TODO (MariusVanDerWijden) trigger sync
			return SYNCING, nil
		}
	}
	// SetHead
	if err := api.setHead(heads.HeadBlockHash); err != nil {
		return INVALID, err
	}
	// Assemble block (if needed)
	if PayloadAttributes != nil {
		data, err := api.assembleBlock(heads.HeadBlockHash, PayloadAttributes)
		if err != nil {
			return INVALID, err
		}
		hash := computePayloadId(heads.HeadBlockHash, PayloadAttributes)
		id := binary.BigEndian.Uint64(hash)
		api.preparedBlocks[id] = data
		log.Info("Created payload", "payloadid", id)
		// TODO (MariusVanDerWijden) do something with the payloadID?
		hex := hexutil.Bytes(hash)
		return ForkChoiceResponse{Status: SUCCESS.Status, PayloadID: &hex}, nil
	}
	return ForkChoiceResponse{Status: SUCCESS.Status, PayloadID: nil}, nil
}

func computePayloadId(headBlockHash common.Hash, params *PayloadAttributesV1) []byte {
	// Hash
	hasher := sha256.New()
	hasher.Write(headBlockHash[:])
	binary.Write(hasher, binary.BigEndian, params.Timestamp)
	hasher.Write(params.Random[:])
	hasher.Write(params.SuggestedFeeRecipient[:])
	return hasher.Sum([]byte{})[:8]
}

func (api *ConsensusAPI) invalid() ExecutePayloadResponse {
	if api.light {
		return ExecutePayloadResponse{Status: INVALID.Status, LatestValidHash: api.les.BlockChain().CurrentHeader().Hash()}
	}
	return ExecutePayloadResponse{Status: INVALID.Status, LatestValidHash: api.eth.BlockChain().CurrentHeader().Hash()}
}

// ExecutePayload creates an Eth1 block, inserts it in the chain, and returns the status of the chain.
func (api *ConsensusAPI) ExecutePayloadV1(params ExecutableDataV1) (ExecutePayloadResponse, error) {
	block, err := ExecutableDataToBlock(params)
	if err != nil {
		return api.invalid(), err
	}
	if api.light {
		parent := api.les.BlockChain().GetHeaderByHash(params.ParentHash)
		if parent == nil {
			return api.invalid(), fmt.Errorf("could not find parent %x", params.ParentHash)
		}
		if err = api.les.BlockChain().InsertHeader(block.Header()); err != nil {
			return api.invalid(), err
		}
		return ExecutePayloadResponse{Status: VALID.Status, LatestValidHash: block.Hash()}, nil
	}
	if !api.eth.BlockChain().HasBlock(block.ParentHash(), block.NumberU64()-1) {
		/*
			TODO (MariusVanDerWijden) reenable once sync is merged
			if err := api.eth.Downloader().BeaconSync(api.eth.SyncMode(), block.Header()); err != nil {
				return SYNCING, err
			}
		*/
		// TODO (MariusVanDerWijden) we should return nil here not empty hash
		return ExecutePayloadResponse{Status: SYNCING.Status, LatestValidHash: common.Hash{}}, nil
	}
	parent := api.eth.BlockChain().GetBlockByHash(params.ParentHash)
	td := api.eth.BlockChain().GetTd(parent.Hash(), block.NumberU64()-1)
	ttd := api.eth.BlockChain().Config().TerminalTotalDifficulty
	if td.Cmp(ttd) < 0 {
		return api.invalid(), fmt.Errorf("can not execute payload on top of block with low td got: %v threshold %v", td, ttd)
	}
	if err := api.eth.BlockChain().InsertBlockWithoutSetHead(block); err != nil {
		return api.invalid(), err
	}

	if merger := api.merger(); !merger.TDDReached() {
		merger.ReachTTD()
	}
	return ExecutePayloadResponse{Status: VALID.Status, LatestValidHash: block.Hash()}, nil
}

// AssembleBlock creates a new block, inserts it into the chain, and returns the "execution
// data" required for eth2 clients to process the new block.
func (api *ConsensusAPI) assembleBlock(parentHash common.Hash, params *PayloadAttributesV1) (*ExecutableDataV1, error) {
	if api.light {
		return nil, errors.New("not supported")
	}
	log.Info("Producing block", "parentHash", parentHash)

	bc := api.eth.BlockChain()
	parent := bc.GetBlockByHash(parentHash)
	if parent == nil {
		log.Warn("Cannot assemble block with parent hash to unknown block", "parentHash", parentHash)
		return nil, fmt.Errorf("cannot assemble block with unknown parent %s", parentHash)
	}

	if params.Timestamp < parent.Time() {
		return nil, fmt.Errorf("child timestamp lower than parent's: %d < %d", params.Timestamp, parent.Time())
	}
	if now := uint64(time.Now().Unix()); params.Timestamp > now+1 {
		diff := time.Duration(params.Timestamp-now) * time.Second
		log.Warn("Producing block too far in the future", "diff", common.PrettyDuration(diff))
	}
	pending := api.eth.TxPool().Pending(true)
	coinbase := params.SuggestedFeeRecipient
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

func ExecutableDataToBlock(params ExecutableDataV1) (*types.Block, error) {
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
		Coinbase:    params.FeeRecipient,
		Root:        params.StateRoot,
		TxHash:      types.DeriveSha(types.Transactions(txs), trie.NewStackTrie(nil)),
		ReceiptHash: params.ReceiptsRoot,
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

func BlockToExecutableData(block *types.Block, random common.Hash) *ExecutableDataV1 {
	return &ExecutableDataV1{
		BlockHash:     block.Hash(),
		ParentHash:    block.ParentHash(),
		FeeRecipient:  block.Coinbase(),
		StateRoot:     block.Root(),
		Number:        block.NumberU64(),
		GasLimit:      block.GasLimit(),
		GasUsed:       block.GasUsed(),
		BaseFeePerGas: block.BaseFee(),
		Timestamp:     block.Time(),
		ReceiptsRoot:  block.ReceiptHash(),
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
		return &GenericServerError
	}
	td := api.eth.BlockChain().GetTd(newHeadBlock.Hash(), newHeadBlock.NumberU64())
	if td != nil && td.Cmp(api.eth.BlockChain().Config().TerminalTotalDifficulty) < 0 {
		return &InvalidTB
	}
	return nil
}

// setHead is called to perform a force choice.
func (api *ConsensusAPI) setHead(newHead common.Hash) error {
	log.Info("Setting head", "head", newHead)
	if api.light {
		headHeader := api.les.BlockChain().CurrentHeader()
		if headHeader.Hash() == newHead {
			return nil
		}
		newHeadHeader := api.les.BlockChain().GetHeaderByHash(newHead)
		if newHeadHeader == nil {
			return &GenericServerError
		}
		if err := api.les.BlockChain().SetChainHead(newHeadHeader); err != nil {
			return err
		}
		// Trigger the transition if it's the first `NewHead` event.
		merger := api.merger()
		if !merger.PoSFinalized() {
			merger.FinalizePoS()
		}
		return nil
	}
	headBlock := api.eth.BlockChain().CurrentBlock()
	if headBlock.Hash() == newHead {
		return nil
	}
	newHeadBlock := api.eth.BlockChain().GetBlockByHash(newHead)
	if newHeadBlock == nil {
		return &GenericServerError
	}
	if err := api.eth.BlockChain().SetChainHead(newHeadBlock); err != nil {
		return err
	}
	// Trigger the transition if it's the first `NewHead` event.
	if merger := api.merger(); !merger.PoSFinalized() {
		merger.FinalizePoS()
	}
	// TODO (MariusVanDerWijden) are we really synced now?
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

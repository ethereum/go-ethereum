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

package ethapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"strings"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/hexutil"
	math "github.com/XinFinOrg/XDPoSChain/common/math"
	xdc_sort "github.com/XinFinOrg/XDPoSChain/common/sort"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/consensus/misc/eip1559"
	contractValidator "github.com/XinFinOrg/XDPoSChain/contracts/validator/contract"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/tracing"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/eth/gasestimator"
	"github.com/XinFinOrg/XDPoSChain/eth/tracers/logger"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/p2p"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/XinFinOrg/XDPoSChain/rpc"
)

const (
	defaultGasPrice = 50 * params.Shannon

	// statuses of candidates
	statusMasternode = "MASTERNODE"
	statusSlashed    = "SLASHED"
	statusProposed   = "PROPOSED"
	fieldStatus      = "status"
	fieldCapacity    = "capacity"
	fieldCandidates  = "candidates"
	fieldSuccess     = "success"
	fieldEpoch       = "epoch"
)

// maxGetStorageSlots is the maximum total number of storage slots that can
// be requested in a single eth_getStorageValues call.
const maxGetStorageSlots = 1024

var errEmptyHeader = errors.New("empty header")

// EthereumAPI provides an API to access Ethereum related information.
// It offers only methods that operate on public data that is freely available to anyone.
type EthereumAPI struct {
	b Backend
}

// NewEthereumAPI creates a new Ethereum protocol API.
func NewEthereumAPI(b Backend) *EthereumAPI {
	return &EthereumAPI{b}
}

// GasPrice returns a suggestion for a gas price for legacy transactions.
func (s *EthereumAPI) GasPrice(ctx context.Context) (*hexutil.Big, error) {
	tipcap, err := s.b.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}
	if head := s.b.CurrentHeader(); head.BaseFee != nil {
		tipcap.Add(tipcap, head.BaseFee)
	}
	return (*hexutil.Big)(tipcap), err
}

// MaxPriorityFeePerGas returns a suggestion for a gas tip cap for dynamic transactions.
func (s *EthereumAPI) MaxPriorityFeePerGas(ctx context.Context) (*hexutil.Big, error) {
	tipcap, err := s.b.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}
	return (*hexutil.Big)(tipcap), err
}

type feeHistoryResult struct {
	OldestBlock  *hexutil.Big     `json:"oldestBlock"`
	Reward       [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee      []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio []float64        `json:"gasUsedRatio"`
}

// FeeHistory returns the fee market history.
func (s *EthereumAPI) FeeHistory(ctx context.Context, blockCount math.HexOrDecimal64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*feeHistoryResult, error) {
	oldest, reward, baseFee, gasUsed, err := s.b.FeeHistory(ctx, uint64(blockCount), lastBlock, rewardPercentiles)
	if err != nil {
		return nil, err
	}
	results := &feeHistoryResult{
		OldestBlock:  (*hexutil.Big)(oldest),
		GasUsedRatio: gasUsed,
	}
	if reward != nil {
		results.Reward = make([][]*hexutil.Big, len(reward))
		for i, w := range reward {
			results.Reward[i] = make([]*hexutil.Big, len(w))
			for j, v := range w {
				results.Reward[i][j] = (*hexutil.Big)(v)
			}
		}
	}
	if baseFee != nil {
		results.BaseFee = make([]*hexutil.Big, len(baseFee))
		for i, v := range baseFee {
			results.BaseFee[i] = (*hexutil.Big)(v)
		}
	}
	return results, nil
}

// BlobBaseFee returns the base fee for blob gas at the current head.
func (s *EthereumAPI) BlobBaseFee(ctx context.Context) *hexutil.Big {
	return (*hexutil.Big)(new(big.Int))
}

// ProtocolVersion returns the current Ethereum protocol version this node supports
func (s *EthereumAPI) ProtocolVersion() hexutil.Uint {
	return hexutil.Uint(s.b.ProtocolVersion())
}

// Syncing returns false in case the node is currently not syncing with the network. It can be up to date or has not
// yet received the latest block headers from its pears. In case it is synchronizing:
// - startingBlock: block number this node started to synchronise from
// - currentBlock:  block number this node is currently importing
// - highestBlock:  block number of the highest block header this node has received from peers
// - pulledStates:  number of state entries processed until now
// - knownStates:   number of known state entries that still need to be pulled
func (s *EthereumAPI) Syncing() (interface{}, error) {
	progress := s.b.Downloader().Progress()

	// Return not syncing if the synchronisation already completed
	if progress.CurrentBlock >= progress.HighestBlock {
		return false, nil
	}
	// Otherwise gather the block sync stats
	return map[string]interface{}{
		"startingBlock": hexutil.Uint64(progress.StartingBlock),
		"currentBlock":  hexutil.Uint64(progress.CurrentBlock),
		"highestBlock":  hexutil.Uint64(progress.HighestBlock),
		"pulledStates":  hexutil.Uint64(progress.PulledStates),
		"knownStates":   hexutil.Uint64(progress.KnownStates),
	}, nil
}

// TxPoolAPI offers and API for the transaction pool. It only operates on data that is non confidential.
type TxPoolAPI struct {
	b Backend
}

// NewTxPoolAPI creates a new tx pool service that gives information about the transaction pool.
func NewTxPoolAPI(b Backend) *TxPoolAPI {
	return &TxPoolAPI{b}
}

// flattenTxs builds the RPC transaction map keyed by nonce for a set of pool txs.
func flattenTxs(txs types.Transactions, header *types.Header, cfg *params.ChainConfig) map[string]*RPCTransaction {
	dump := make(map[string]*RPCTransaction, len(txs))
	for _, tx := range txs {
		dump[fmt.Sprintf("%d", tx.Nonce())] = newRPCPendingTransaction(tx, header, cfg)
	}
	return dump
}

// Content returns the transactions contained within the transaction pool.
func (s *TxPoolAPI) Content() map[string]map[string]map[string]*RPCTransaction {
	pending, queue := s.b.TxPoolContent()
	content := map[string]map[string]map[string]*RPCTransaction{
		"pending": make(map[string]map[string]*RPCTransaction, len(pending)),
		"queued":  make(map[string]map[string]*RPCTransaction, len(queue)),
	}
	curHeader := s.b.CurrentHeader()
	// Flatten the pending transactions
	for account, txs := range pending {
		content["pending"][account.Hex()] = flattenTxs(txs, curHeader, s.b.ChainConfig())
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		content["queued"][account.Hex()] = flattenTxs(txs, curHeader, s.b.ChainConfig())
	}
	return content
}

// ContentFrom returns the transactions contained within the transaction pool.
func (s *TxPoolAPI) ContentFrom(addr common.Address) map[string]map[string]*RPCTransaction {
	content := make(map[string]map[string]*RPCTransaction, 2)
	pending, queue := s.b.TxPoolContentFrom(addr)
	curHeader := s.b.CurrentHeader()

	// Build the pending transactions
	content["pending"] = flattenTxs(pending, curHeader, s.b.ChainConfig())

	// Build the queued transactions
	content["queued"] = flattenTxs(queue, curHeader, s.b.ChainConfig())

	return content
}

// Status returns the number of pending and queued transaction in the pool.
func (s *TxPoolAPI) Status() map[string]hexutil.Uint {
	pending, queue := s.b.Stats()
	return map[string]hexutil.Uint{
		"pending": hexutil.Uint(pending),
		"queued":  hexutil.Uint(queue),
	}
}

// Inspect retrieves the content of the transaction pool and flattens it into an
// easily inspectable list.
func (s *TxPoolAPI) Inspect() map[string]map[string]map[string]string {
	pending, queue := s.b.TxPoolContent()
	content := map[string]map[string]map[string]string{
		"pending": make(map[string]map[string]string, len(pending)),
		"queued":  make(map[string]map[string]string, len(queue)),
	}

	// Define a formatter to flatten a transaction into a string
	var format = func(tx *types.Transaction) string {
		if to := tx.To(); to != nil {
			return fmt.Sprintf("%s: %v wei + %v gas × %v wei", to, tx.Value(), tx.Gas(), tx.GasPrice())
		}
		return fmt.Sprintf("contract creation: %v wei + %v gas × %v wei", tx.Value(), tx.Gas(), tx.GasPrice())
	}
	// Flatten the pending transactions
	for account, txs := range pending {
		dump := make(map[string]string, len(txs))
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = format(tx)
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		dump := make(map[string]string, len(txs))
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = format(tx)
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// EthereumAccountAPI provides an API to access accounts managed by this node.
// It offers only methods that can retrieve accounts.
type EthereumAccountAPI struct {
	am *accounts.Manager
}

// NewEthereumAccountAPI creates a new EthereumAccountAPI.
func NewEthereumAccountAPI(am *accounts.Manager) *EthereumAccountAPI {
	return &EthereumAccountAPI{am: am}
}

// Accounts returns the collection of accounts this node manages
func (s *EthereumAccountAPI) Accounts() []common.Address {
	return s.am.Accounts()
}

// BlockChainAPI provides an API to access Ethereum blockchain data.
type BlockChainAPI struct {
	b           Backend
	chainReader consensus.ChainReader
}

// NewBlockChainAPI creates a new Ethereum blockchain API.
func NewBlockChainAPI(b Backend, chainReader consensus.ChainReader) *BlockChainAPI {
	return &BlockChainAPI{
		b,
		chainReader,
	}
}

// ChainId returns the chainID value for transaction replay protection.
func (api *BlockChainAPI) ChainId() *hexutil.Big {
	return (*hexutil.Big)(api.b.ChainConfig().ChainID)
}

// BlockNumber returns the block number of the chain head.
func (api *BlockChainAPI) BlockNumber() hexutil.Uint64 {
	header, _ := api.b.HeaderByNumber(context.Background(), rpc.LatestBlockNumber) // latest header should always be available
	return hexutil.Uint64(header.Number.Uint64())
}

// BlockNumber returns the block number of the chain head.
func (api *BlockChainAPI) GetRewardByHash(hash common.Hash) map[string]map[string]map[string]*big.Int {
	return api.b.GetRewardByHash(hash)
}

// GetBalance returns the amount of wei for the given address in the state of the
// given block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta
// block numbers are also allowed.
func (api *BlockChainAPI) GetBalance(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	state, _, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	return (*hexutil.Big)(state.GetBalance(address)), state.Error()
}

// GetTransactionAndReceiptProof returns the Trie transaction and receipt proof of the given transaction hash.
func (api *BlockChainAPI) GetTransactionAndReceiptProof(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	tx, blockHash, _, index := rawdb.ReadTransaction(api.b.ChainDb(), hash)
	if tx == nil {
		return nil, nil
	}
	block, err := api.b.GetBlock(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	tx_tr := deriveTrie(block.Transactions())

	keybuf := new(bytes.Buffer)
	if err := rlp.Encode(keybuf, uint(index)); err != nil {
		return nil, err
	}
	var tx_proof proofPairList
	if err := tx_tr.Prove(keybuf.Bytes(), &tx_proof); err != nil {
		return nil, err
	}
	receipts, err := api.b.GetReceipts(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	if len(receipts) <= int(index) {
		return nil, nil
	}
	receipt_tr := deriveTrie(receipts)
	var receipt_proof proofPairList
	if err := receipt_tr.Prove(keybuf.Bytes(), &receipt_proof); err != nil {
		return nil, err
	}
	fields := map[string]interface{}{
		"blockHash":          blockHash,
		"txRoot":             tx_tr.Hash(),
		"receiptRoot":        receipt_tr.Hash(),
		"key":                hexutil.Encode(keybuf.Bytes()),
		"txProofKeys":        tx_proof.keys,
		"txProofValues":      tx_proof.values,
		"receiptProofKeys":   receipt_proof.keys,
		"receiptProofValues": receipt_proof.values,
	}
	return fields, nil
}

// GetHeaderByNumber returns the requested canonical block header.
//   - When blockNr is -1 the chain pending header is returned.
//   - When blockNr is -2 the chain latest header is returned.
//   - When blockNr is -3 the chain finalized header is returned.
//   - When blockNr is -4 the chain safe header is returned.
func (api *BlockChainAPI) GetHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (map[string]interface{}, error) {
	header, err := api.b.HeaderByNumber(ctx, number)
	if header != nil && err == nil {
		response := RPCMarshalHeader(header)
		if number == rpc.PendingBlockNumber {
			// Pending header need to nil out a few fields
			for _, field := range []string{"hash", "nonce", "miner"} {
				response[field] = nil
			}
		}
		return response, err
	}
	return nil, err
}

// GetHeaderByHash returns the requested header by hash.
func (api *BlockChainAPI) GetHeaderByHash(ctx context.Context, hash common.Hash) map[string]interface{} {
	header, _ := api.b.HeaderByHash(ctx, hash)
	if header != nil {
		return RPCMarshalHeader(header)
	}
	return nil
}

// GetBlockByNumber returns the requested block. When blockNr is -1 the chain head is returned. When fullTx is true all
// transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (api *BlockChainAPI) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	block, err := api.b.BlockByNumber(ctx, number)
	if block != nil {
		response, err := api.rpcMarshalBlock(ctx, block, true, fullTx)
		if err == nil && number == rpc.PendingBlockNumber {
			// Pending blocks need to nil out a few fields
			for _, field := range []string{"hash", "nonce", "miner", "number"} {
				response[field] = nil
			}
		}
		return response, err
	}
	return nil, err
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (api *BlockChainAPI) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block, err := api.b.GetBlock(ctx, hash)
	if block != nil {
		return api.rpcMarshalBlock(ctx, block, true, fullTx)
	}
	return nil, err
}

// GetUncleByBlockNumberAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (api *BlockChainAPI) GetUncleByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := api.b.BlockByNumber(ctx, blockNr)
	if block != nil {
		uncles := block.Uncles()
		if index >= hexutil.Uint(len(uncles)) {
			log.Debug("Requested uncle not found", "number", blockNr, "hash", block.Hash(), "index", index)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index])
		return api.rpcMarshalBlock(ctx, block, false, false)
	}
	return nil, err
}

// GetUncleByBlockHashAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
// DEPRECATED SINCE 1.0
func (api *BlockChainAPI) GetUncleByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := api.b.GetBlock(ctx, blockHash)
	if block != nil {
		uncles := block.Uncles()
		if index >= hexutil.Uint(len(uncles)) {
			log.Debug("Requested uncle not found", "number", block.Number(), "hash", blockHash, "index", index)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index])
		return api.rpcMarshalBlock(ctx, block, false, false)
	}
	return nil, err
}

// GetUncleCountByBlockNumber returns number of uncles in the block for the given block number
// DEPRECATED SINCE 1.0
func (api *BlockChainAPI) GetUncleCountByBlockNumber(ctx context.Context, blockNr rpc.BlockNumber) *hexutil.Uint {
	if block, _ := api.b.BlockByNumber(ctx, blockNr); block != nil {
		n := hexutil.Uint(len(block.Uncles()))
		return &n
	}
	return nil
}

// GetUncleCountByBlockHash returns number of uncles in the block for the given block hash
// DEPRECATED SINCE 1.0
func (api *BlockChainAPI) GetUncleCountByBlockHash(ctx context.Context, blockHash common.Hash) *hexutil.Uint {
	if block, _ := api.b.GetBlock(ctx, blockHash); block != nil {
		n := hexutil.Uint(len(block.Uncles()))
		return &n
	}
	return nil
}

// GetCode returns the code stored at the given address in the state for the given block number.
func (api *BlockChainAPI) GetCode(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	state, _, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	code := state.GetCode(address)
	return code, state.Error()
}

// GetAccountInfo returns the information at the given address in the state for the given block number.
func (api *BlockChainAPI) GetAccountInfo(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (map[string]interface{}, error) {
	state, _, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	info := state.GetAccountInfo(address)
	result := map[string]interface{}{
		"address":     address,
		"balance":     (*hexutil.Big)(info.Balance),
		"codeSize":    info.CodeSize,
		"codeHash":    info.CodeHash,
		"nonce":       info.Nonce,
		"storageHash": info.StorageHash,
	}
	return result, state.Error()
}

// GetStorageAt returns the storage from the state at the given address, key and
// block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta block
// numbers are also allowed.
func (api *BlockChainAPI) GetStorageAt(ctx context.Context, address common.Address, key string, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	state, _, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	res := state.GetState(address, common.HexToHash(key))
	return res[:], state.Error()
}

// GetStorageValues returns multiple storage slot values for multiple accounts
// at the given block.
func (api *BlockChainAPI) GetStorageValues(ctx context.Context, requests map[common.Address][]common.Hash, blockNrOrHash rpc.BlockNumberOrHash) (map[common.Address][]hexutil.Bytes, error) {
	// Count total slots requested.
	var totalSlots int
	for _, keys := range requests {
		totalSlots += len(keys)
		if totalSlots > maxGetStorageSlots {
			return nil, &clientLimitExceededError{message: fmt.Sprintf("too many slots (max %d)", maxGetStorageSlots)}
		}
	}
	if totalSlots == 0 {
		return nil, &invalidParamsError{message: "empty request"}
	}

	state, _, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}

	result := make(map[common.Address][]hexutil.Bytes, len(requests))
	for addr, keys := range requests {
		vals := make([]hexutil.Bytes, len(keys))
		for i, key := range keys {
			v := state.GetState(addr, key)
			vals[i] = v[:]
		}
		if err := state.Error(); err != nil {
			return nil, err
		}
		result[addr] = vals
	}
	return result, nil
}

// GetBlockReceipts returns the block receipts for the given block hash or number or tag.
func (api *BlockChainAPI) GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]map[string]interface{}, error) {
	block, err := api.b.BlockByNumberOrHash(ctx, blockNrOrHash)
	if err != nil {
		return nil, err
	}
	if block == nil {
		// When the block doesn't exist, the RPC method should return JSON null
		// as per specification.
		return nil, nil
	}
	receipts, err := api.b.GetReceipts(ctx, block.Hash())
	if err != nil {
		return nil, err
	}
	txs := block.Transactions()
	if len(txs) != len(receipts) {
		return nil, fmt.Errorf("receipts length mismatch: %d vs %d", len(txs), len(receipts))
	}

	// Derive the sender.
	signer := types.MakeSigner(api.b.ChainConfig(), block.Number())

	result := make([]map[string]interface{}, len(receipts))
	for i, receipt := range receipts {
		result[i] = marshalReceipt(receipt, block.Hash(), block.NumberU64(), signer, txs[i], i)
	}

	return result, nil
}

// OverrideAccount indicates the overriding fields of account during the execution
// of a message call.
// Note, state and stateDiff can't be specified at the same time. If state is
// set, message execution will only use the data in the given state. Otherwise
// if statDiff is set, all diff will be applied first and then execute the call
// message.
type OverrideAccount struct {
	Nonce            *hexutil.Uint64             `json:"nonce"`
	Code             *hexutil.Bytes              `json:"code"`
	Balance          *hexutil.Big                `json:"balance"`
	State            map[common.Hash]common.Hash `json:"state"`
	StateDiff        map[common.Hash]common.Hash `json:"stateDiff"`
	MovePrecompileTo *common.Address             `json:"movePrecompileToAddress"`
}

// StateOverride is the collection of overridden accounts.
type StateOverride map[common.Address]OverrideAccount

func (diff *StateOverride) has(address common.Address) bool {
	_, ok := (*diff)[address]
	return ok
}

// Apply overrides the fields of specified accounts into the given state.
func (diff *StateOverride) Apply(statedb *state.StateDB, precompiles vm.PrecompiledContracts) error {
	if diff == nil {
		return nil
	}
	// Tracks destinations of precompiles that were moved.
	dirtyAddrs := make(map[common.Address]struct{})
	for addr, account := range *diff {
		// If a precompile was moved to this address already, it can't be overridden.
		if _, ok := dirtyAddrs[addr]; ok {
			return fmt.Errorf("account %s has already been overridden by a precompile", addr.Hex())
		}
		p, isPrecompile := precompiles[addr]
		// The MoveTo feature makes it possible to move a precompile
		// code to another address. If the target address is another precompile
		// the code for the latter is lost for this session.
		// Note the destination account is not cleared upon move.
		if account.MovePrecompileTo != nil {
			if !isPrecompile {
				return fmt.Errorf("account %s is not a precompile", addr.Hex())
			}
			// Refuse to move a precompile to an address that has been
			// or will be overridden.
			if diff.has(*account.MovePrecompileTo) {
				return fmt.Errorf("account %s is already overridden", account.MovePrecompileTo.Hex())
			}
			precompiles[*account.MovePrecompileTo] = p
			dirtyAddrs[*account.MovePrecompileTo] = struct{}{}
		}
		if isPrecompile {
			delete(precompiles, addr)
		}
		// Override account nonce.
		if account.Nonce != nil {
			statedb.SetNonce(addr, uint64(*account.Nonce))
		}
		// Override account(contract) code.
		if account.Code != nil {
			statedb.SetCode(addr, *account.Code)
		}
		// Override account balance.
		if account.Balance != nil {
			statedb.SetBalance(addr, (*big.Int)(account.Balance), tracing.BalanceChangeUnspecified)
		}
		if account.State != nil && account.StateDiff != nil {
			return fmt.Errorf("account %s has both 'state' and 'stateDiff'", addr.Hex())
		}
		// Replace entire state if caller requires.
		if account.State != nil {
			statedb.SetStorage(addr, account.State)
		}
		// Apply state diff into specified accounts.
		if account.StateDiff != nil {
			for key, value := range account.StateDiff {
				statedb.SetState(addr, key, value)
			}
		}
	}
	// Now finalize the changes. Finalize is normally performed between transactions.
	// By using finalize, the overrides are semantically behaving as
	// if they were created in a transaction just before the tracing occur.
	statedb.Finalise(false)
	return nil
}

// BlockOverrides is a set of header fields to override.
type BlockOverrides struct {
	Number        *hexutil.Big
	Difficulty    *hexutil.Big
	Time          *hexutil.Uint64
	GasLimit      *hexutil.Uint64
	FeeRecipient  *common.Address
	PrevRandao    *common.Hash
	BaseFeePerGas *hexutil.Big
}

// Apply overrides the given header fields into the given block context.
func (o *BlockOverrides) Apply(blockCtx *vm.BlockContext) {
	if o == nil {
		return
	}
	if o.Number != nil {
		blockCtx.BlockNumber = o.Number.ToInt()
	}
	if o.Difficulty != nil {
		blockCtx.Difficulty = o.Difficulty.ToInt()
	}
	if o.Time != nil {
		blockCtx.Time = uint64(*o.Time)
	}
	if o.GasLimit != nil {
		blockCtx.GasLimit = uint64(*o.GasLimit)
	}
	if o.FeeRecipient != nil {
		blockCtx.Coinbase = *o.FeeRecipient
	}
	if o.PrevRandao != nil {
		blockCtx.Random = o.PrevRandao
	}
	if o.BaseFeePerGas != nil {
		blockCtx.BaseFee = o.BaseFeePerGas.ToInt()
	}
}

// MakeHeader returns a new header object with the overridden
// fields.
// Note: MakeHeader ignores BlobBaseFee if set. That's because
// header has no such field.
func (o *BlockOverrides) MakeHeader(header *types.Header) *types.Header {
	if o == nil {
		return header
	}
	h := types.CopyHeader(header)
	if o.Number != nil {
		h.Number = o.Number.ToInt()
	}
	if o.Difficulty != nil {
		h.Difficulty = o.Difficulty.ToInt()
	}
	if o.Time != nil {
		h.Time = uint64(*o.Time)
	}
	if o.GasLimit != nil {
		h.GasLimit = uint64(*o.GasLimit)
	}
	if o.FeeRecipient != nil {
		h.Coinbase = *o.FeeRecipient
	}
	if o.PrevRandao != nil {
		h.MixDigest = *o.PrevRandao
	}
	if o.BaseFeePerGas != nil {
		h.BaseFee = o.BaseFeePerGas.ToInt()
	}
	return h
}

func (api *BlockChainAPI) GetBlockSignersByHash(ctx context.Context, blockHash common.Hash) ([]common.Address, error) {
	block, err := api.b.GetBlock(ctx, blockHash)
	if err != nil || block == nil {
		return []common.Address{}, err
	}
	masternodes, err := api.GetMasternodes(ctx, block)
	if err != nil || len(masternodes) == 0 {
		log.Error("Failed to get masternodes", "err", err, "len(masternodes)", len(masternodes))
		return []common.Address{}, err
	}
	return api.rpcOutputBlockSigners(block, ctx, masternodes)
}

func (api *BlockChainAPI) GetBlockSignersByNumber(ctx context.Context, blockNumber rpc.BlockNumber) ([]common.Address, error) {
	block, err := api.b.BlockByNumber(ctx, blockNumber)
	if err != nil || block == nil {
		return []common.Address{}, err
	}
	masternodes, err := api.GetMasternodes(ctx, block)
	if err != nil || len(masternodes) == 0 {
		log.Error("Failed to get masternodes", "err", err, "len(masternodes)", len(masternodes))
		return []common.Address{}, err
	}
	return api.rpcOutputBlockSigners(block, ctx, masternodes)
}

func (api *BlockChainAPI) GetBlockFinalityByHash(ctx context.Context, blockHash common.Hash) (uint, error) {
	block, err := api.b.GetBlock(ctx, blockHash)
	if err != nil || block == nil {
		return uint(0), err
	}
	masternodes, err := api.GetMasternodes(ctx, block)
	if err != nil || len(masternodes) == 0 {
		log.Error("Failed to get masternodes", "err", err, "len(masternodes)", len(masternodes))
		return uint(0), err
	}
	return api.findFinalityOfBlock(ctx, block, masternodes)
}

func (api *BlockChainAPI) GetBlockFinalityByNumber(ctx context.Context, blockNumber rpc.BlockNumber) (uint, error) {
	block, err := api.b.BlockByNumber(ctx, blockNumber)
	if err != nil || block == nil {
		return uint(0), err
	}
	masternodes, err := api.GetMasternodes(ctx, block)
	if err != nil || len(masternodes) == 0 {
		log.Error("Failed to get masternodes", "err", err, "len(masternodes)", len(masternodes))
		return uint(0), err
	}
	return api.findFinalityOfBlock(ctx, block, masternodes)
}

// GetMasternodes returns masternodes set at the starting block of epoch of the given block
func (api *BlockChainAPI) GetMasternodes(ctx context.Context, b *types.Block) ([]common.Address, error) {
	var masternodes []common.Address
	if b.Number().Sign() >= 0 {
		curBlockNumber := b.Number().Uint64()
		prevBlockNumber := curBlockNumber + (common.MergeSignRange - (curBlockNumber % common.MergeSignRange))
		latestBlockNumber := api.b.CurrentBlock().Number.Uint64()
		if prevBlockNumber >= latestBlockNumber || !api.b.ChainConfig().IsTIP2019(b.Number()) {
			prevBlockNumber = curBlockNumber
		}
		if engine, ok := api.b.Engine().(*XDPoS.XDPoS); ok {
			// Get block epoc latest.
			return engine.GetMasternodesByNumber(api.chainReader, prevBlockNumber), nil
		} else {
			log.Error("Undefined XDPoS consensus engine")
		}
	}
	return masternodes, nil
}

// GetCandidateStatus returns status of the given candidate at a specified epochNumber
func (api *BlockChainAPI) GetCandidateStatus(ctx context.Context, coinbaseAddress common.Address, epoch rpc.EpochNumber) (map[string]interface{}, error) {
	var (
		block                    *types.Block
		header                   *types.Header
		checkpointNumber         rpc.BlockNumber
		epochNumber              rpc.EpochNumber // if epoch == "latest", print the latest epoch number to epochNumber
		masternodes, penaltyList []common.Address
		candidates               []utils.Masternode
		penalties                []byte
		err                      error
	)

	result := map[string]interface{}{
		fieldStatus:   "",
		fieldCapacity: 0,
		fieldSuccess:  true,
	}

	epochConfig := api.b.ChainConfig().XDPoS.Epoch

	// checkpoint block
	checkpointNumber, epochNumber = api.GetCheckpointFromEpoch(ctx, epoch)
	result[fieldEpoch] = epochNumber.Int64()

	block, err = api.b.BlockByNumber(ctx, checkpointNumber)
	if err != nil || block == nil { // || checkpointNumber == 0 {
		result[fieldSuccess] = false
		return result, err
	}

	header = block.Header()
	if header == nil {
		log.Error("Empty header at checkpoint ", "num", checkpointNumber)
		return result, errEmptyHeader
	}

	// list of candidates (masternode, slash, propose) at block checkpoint
	if epoch == rpc.LatestEpochNumber {
		candidates, err = api.getCandidatesFromSmartContract()
	} else {
		statedb, _, err := api.b.StateAndHeaderByNumber(ctx, checkpointNumber)
		if err != nil {
			result[fieldSuccess] = false
			return result, err
		}
		if statedb == nil {
			result[fieldSuccess] = false
			return result, errors.New("nil statedb in GetCandidateStatus")
		}
		candidatesAddresses := statedb.GetCandidates()
		candidates = make([]utils.Masternode, 0, len(candidatesAddresses))
		for _, address := range candidatesAddresses {
			v := statedb.GetCandidateCap(address)
			candidates = append(candidates, utils.Masternode{Address: address, Stake: v})
		}
	}
	if err != nil || len(candidates) == 0 {
		log.Debug("Candidates list cannot be found", "len(candidates)", len(candidates), "err", err)
		result[fieldSuccess] = false
		return result, err
	}

	var maxMasternodes int
	if header.Number.Cmp(api.b.ChainConfig().XDPoS.V2.SwitchBlock) == 1 {
		if engine, ok := api.b.Engine().(*XDPoS.XDPoS); ok {
			round, err := engine.EngineV2.GetRoundNumber(header)
			if err != nil {
				return result, err
			}
			maxMasternodes = api.b.ChainConfig().XDPoS.V2.Config(uint64(round)).MaxMasternodes
		} else {
			return result, errors.New("undefined XDPoS consensus engine")
		}
	} else if api.b.ChainConfig().IsTIPIncreaseMasternodes(block.Number()) {
		maxMasternodes = common.MaxMasternodesV2
	} else {
		maxMasternodes = common.MaxMasternodes
	}

	// check penalties from checkpoint headers and modify status of a node to SLASHED if it's in top maxMasternodes candidates.
	// if it's SLASHED but it's out of top maxMasternodes, the status should be still PROPOSED.
	isCandidate := false
	for i := 0; i < len(candidates); i++ {
		if coinbaseAddress == candidates[i].Address {
			isCandidate = true
			result[fieldStatus] = statusProposed
			result[fieldCapacity] = candidates[i].Stake
			break
		}
	}

	// Get masternode list
	if engine, ok := api.b.Engine().(*XDPoS.XDPoS); ok {
		masternodes = engine.GetMasternodesFromCheckpointHeader(header)
		if len(masternodes) == 0 {
			log.Error("Failed to get masternodes", "err", err, "len(masternodes)", len(masternodes), "blockNum", header.Number.Uint64())
			result[fieldSuccess] = false
			return result, err
		}
	} else {
		log.Error("Undefined XDPoS consensus engine")
	}

	// Set to statusMasternode if it is masternode
	for _, masternode := range masternodes {
		if coinbaseAddress == masternode {
			result[fieldStatus] = statusMasternode
			if !isCandidate {
				result[fieldCapacity] = -1
				log.Warn("Find non-candidate masternode", "masternode", masternode, "checkpointNumber", checkpointNumber, "epoch", epoch, "epochNumber", epochNumber)
			}
			return result, nil
		}
	}

	if !isCandidate || len(masternodes) >= maxMasternodes {
		return result, nil
	}

	if len(candidates) > maxMasternodes {
		xdc_sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Stake.Cmp(candidates[j].Stake) > 0
		})
	}

	// Get penalties list
	penalties = append(penalties, header.Penalties...)
	// check last 5 epochs to find penalize masternodes
	for i := 1; i <= common.LimitPenaltyEpoch; i++ {
		if header.Number.Uint64() < epochConfig*uint64(i) {
			break
		}
		blockNum := header.Number.Uint64() - epochConfig*uint64(i)
		checkpointHeader, err := api.b.HeaderByNumber(ctx, rpc.BlockNumber(blockNum))
		if checkpointHeader == nil || err != nil {
			log.Error("Failed to get header by number", "num", blockNum, "err", err)
			continue
		}
		penalties = append(penalties, checkpointHeader.Penalties...)
	}
	penaltyList = common.ExtractAddressFromBytes(penalties)

	// map slashing status
	total := len(masternodes)
	for _, candidate := range candidates {
		for _, pen := range penaltyList {
			if candidate.Address == pen {
				if coinbaseAddress == pen {
					result[fieldStatus] = statusSlashed
					return result, nil
				}
				total++
				if total >= maxMasternodes {
					return result, nil
				}
			}
		}
	}

	return result, nil
}

// GetCandidates returns status of all candidates at a specified epochNumber
func (api *BlockChainAPI) GetCandidates(ctx context.Context, epoch rpc.EpochNumber) (map[string]interface{}, error) {
	var (
		block            *types.Block
		header           *types.Header
		checkpointNumber rpc.BlockNumber
		epochNumber      rpc.EpochNumber
		masternodes      []common.Address
		penaltyList      []common.Address
		candidates       []utils.Masternode
		penalties        []byte
		err              error
	)
	result := map[string]interface{}{
		fieldSuccess: true,
	}
	epochConfig := api.b.ChainConfig().XDPoS.Epoch

	checkpointNumber, epochNumber = api.GetCheckpointFromEpoch(ctx, epoch)
	result[fieldEpoch] = epochNumber.Int64()

	block, err = api.b.BlockByNumber(ctx, checkpointNumber)
	if err != nil || block == nil { // || checkpointNumber == 0 {
		result[fieldSuccess] = false
		return result, err
	}

	header = block.Header()

	if header == nil {
		log.Error("Empty header at checkpoint", "num", checkpointNumber)
		return result, errEmptyHeader
	}
	// list of candidates (masternode, slash, propose) at block checkpoint
	if epoch == rpc.LatestEpochNumber {
		candidates, err = api.getCandidatesFromSmartContract()
	} else {
		statedb, _, err := api.b.StateAndHeaderByNumber(ctx, checkpointNumber)
		if err != nil {
			result[fieldSuccess] = false
			return result, err
		}
		if statedb == nil {
			result[fieldSuccess] = false
			return result, errors.New("nil statedb in GetCandidates")
		}
		candidatesAddresses := statedb.GetCandidates()
		candidates = make([]utils.Masternode, 0, len(candidatesAddresses))
		for _, address := range candidatesAddresses {
			v := statedb.GetCandidateCap(address)
			candidates = append(candidates, utils.Masternode{Address: address, Stake: v})
		}
	}

	if err != nil || len(candidates) == 0 {
		log.Debug("Candidates list cannot be found", "len(candidates)", len(candidates), "err", err)
		result[fieldSuccess] = false
		return result, err
	}

	// Find candidates that have masternode status
	if engine, ok := api.b.Engine().(*XDPoS.XDPoS); ok {
		masternodes = engine.GetMasternodesFromCheckpointHeader(header)
		if len(masternodes) == 0 {
			log.Error("Failed to get masternodes", "err", err, "len(masternodes)", len(masternodes), "blockNum", header.Number.Uint64())
			result[fieldSuccess] = false
			return result, err
		}
	} else {
		log.Error("Undefined XDPoS consensus engine")
	}

	// Set all candidate to statusProposed
	candidatesStatusMap := make(map[string]map[string]interface{}, len(candidates))
	for _, candidate := range candidates {
		candidatesStatusMap[candidate.Address.String()] = map[string]interface{}{
			fieldStatus:   statusProposed,
			fieldCapacity: candidate.Stake,
		}
	}

	// Set masternodes to statusMasternode
	for _, masternode := range masternodes {
		key := masternode.String()
		if candidatesStatusMap[key] != nil {
			candidatesStatusMap[key][fieldStatus] = statusMasternode
		} else {
			candidatesStatusMap[key] = map[string]interface{}{
				fieldStatus:   statusMasternode,
				fieldCapacity: -1,
			}
			log.Warn("Masternode is not candidate", "masternode", key, "checkpointNumber", checkpointNumber, "epoch", epoch, "epochNumber", epochNumber)
		}
	}

	var maxMasternodes int
	if header.Number.Cmp(api.b.ChainConfig().XDPoS.V2.SwitchBlock) == 1 {
		if engine, ok := api.b.Engine().(*XDPoS.XDPoS); ok {
			round, err := engine.EngineV2.GetRoundNumber(header)
			if err != nil {
				return result, err
			}
			maxMasternodes = api.b.ChainConfig().XDPoS.V2.Config(uint64(round)).MaxMasternodes
		} else {
			return result, errors.New("undefined XDPoS consensus engine")
		}
	} else if api.b.ChainConfig().IsTIPIncreaseMasternodes(block.Number()) {
		maxMasternodes = common.MaxMasternodesV2
	} else {
		maxMasternodes = common.MaxMasternodes
	}

	if len(masternodes) >= maxMasternodes {
		result[fieldCandidates] = candidatesStatusMap
		return result, nil
	}

	if len(candidates) > maxMasternodes {
		xdc_sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Stake.Cmp(candidates[j].Stake) > 0
		})
	}

	// Get penalties list
	penalties = append(penalties, header.Penalties...)
	// check last 5 epochs to find penalize masternodes
	for i := 1; i <= common.LimitPenaltyEpoch; i++ {
		if header.Number.Uint64() < epochConfig*uint64(i) {
			break
		}
		blockNum := header.Number.Uint64() - epochConfig*uint64(i)
		checkpointHeader, err := api.b.HeaderByNumber(ctx, rpc.BlockNumber(blockNum))
		if checkpointHeader == nil || err != nil {
			log.Error("Failed to get header by number", "num", blockNum, "err", err)
			continue
		}
		penalties = append(penalties, checkpointHeader.Penalties...)
	}
	// map slashing status
	if len(penalties) == 0 {
		result[fieldCandidates] = candidatesStatusMap
		return result, nil
	}
	penaltyList = common.ExtractAddressFromBytes(penalties)

	// check penalties from checkpoint headers and modify status of a node to SLASHED if it's in top maxMasternodes candidates.
	// if it's SLASHED but it's out of top maxMasternodes, the status should be still PROPOSED.
	total := len(masternodes)
	for _, candidate := range candidates {
		for _, pen := range penaltyList {
			if candidate.Address == pen {
				candidatesStatusMap[pen.String()][fieldStatus] = statusSlashed
				total++
				if total >= maxMasternodes {
					result[fieldCandidates] = candidatesStatusMap
					return result, nil
				}
			}
		}
	}

	// update result
	result[fieldCandidates] = candidatesStatusMap
	return result, nil
}

// GetCheckpointFromEpoch returns header of the previous checkpoint
func (api *BlockChainAPI) GetCheckpointFromEpoch(ctx context.Context, epochNum rpc.EpochNumber) (rpc.BlockNumber, rpc.EpochNumber) {
	var checkpointNumber uint64
	epoch := api.b.ChainConfig().XDPoS.Epoch

	if epochNum == rpc.LatestEpochNumber {
		blockNumer := api.b.CurrentBlock().Number
		if engine, ok := api.b.Engine().(*XDPoS.XDPoS); ok {
			var err error
			var currentEpoch uint64
			checkpointNumber, currentEpoch, err = engine.GetCurrentEpochSwitchBlock(api.chainReader, blockNumer)
			if err != nil {
				log.Error("[GetCheckpointFromEpoch] Fail to get GetCurrentEpochSwitchBlock for current checkpoint block", "block", blockNumer, "err", err)
				return 0, epochNum
			}

			epochNum = rpc.EpochNumber(currentEpoch)
		}
	} else if epochNum < 2 {
		checkpointNumber = 0
	} else {
		// TODO this checkpointNumber needs to be recalculated for v2 blocks
		checkpointNumber = epoch * (uint64(epochNum) - 1)
	}

	return rpc.BlockNumber(checkpointNumber), epochNum
}

// getCandidatesFromSmartContract returns all candidates with their capacities at the current time
func (api *BlockChainAPI) getCandidatesFromSmartContract() ([]utils.Masternode, error) {
	client, err := api.b.GetIPCClient()
	if err != nil {
		return []utils.Masternode{}, err
	}

	addr := common.MasternodeVotingSMCBinary
	validator, err := contractValidator.NewXDCValidator(addr, client)
	if err != nil {
		return []utils.Masternode{}, err
	}

	opts := new(bind.CallOpts)
	candidates, err := validator.GetCandidates(opts)
	if err != nil {
		return []utils.Masternode{}, err
	}

	candidatesWithStakeInfo := make([]utils.Masternode, 0, len(candidates))

	for _, candidate := range candidates {
		if !candidate.IsZero() {
			v, err := validator.GetCandidateCap(opts, candidate)
			if err != nil {
				return []utils.Masternode{}, err
			}

			candidatesWithStakeInfo = append(candidatesWithStakeInfo, utils.Masternode{Address: candidate, Stake: v})
		}
	}

	return candidatesWithStakeInfo, nil
}

// ChainContextBackend provides methods required to implement ChainContext.
type ChainContextBackend interface {
	Engine() consensus.Engine
	HeaderByNumber(context.Context, rpc.BlockNumber) (*types.Header, error)
}

// ChainContext is an implementation of core.ChainContext. It's main use-case
// is instantiating a vm.BlockContext without having access to the BlockChain object.
type ChainContext struct {
	b   ChainContextBackend
	ctx context.Context
}

// NewChainContext creates a new ChainContext object.
func NewChainContext(ctx context.Context, backend ChainContextBackend) *ChainContext {
	return &ChainContext{ctx: ctx, b: backend}
}

func (context *ChainContext) Engine() consensus.Engine {
	return context.b.Engine()
}

func (context *ChainContext) GetHeader(hash common.Hash, number uint64) *types.Header {
	// This method is called to get the hash for a block number when executing the BLOCKHASH
	// opcode. Hence no need to search for non-canonical blocks.
	header, err := context.b.HeaderByNumber(context.ctx, rpc.BlockNumber(number))
	if err != nil || header.Hash() != hash {
		return nil
	}
	return header
}

func DoCall(ctx context.Context, b Backend, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *StateOverride, blockOverrides *BlockOverrides, timeout time.Duration, globalGasCap uint64) (*core.ExecutionResult, error) {
	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	state, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	if header == nil {
		return nil, errors.New("nil header in DoCall")
	}
	block, err := b.BlockByNumberOrHash(ctx, blockNrOrHash)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("nil block in DoCall: number=%d, hash=%s", header.Number.Uint64(), header.Hash().Hex())
	}

	return doCall(ctx, b, args, state, block, overrides, blockOverrides, timeout, globalGasCap)
}

func doCall(ctx context.Context, b Backend, args TransactionArgs, state *state.StateDB, block *types.Block, overrides *StateOverride, blockOverrides *BlockOverrides, timeout time.Duration, globalGasCap uint64) (*core.ExecutionResult, error) {
	header := block.Header()
	blockCtx := core.NewEVMBlockContext(header, NewChainContext(ctx, b), nil)
	if blockOverrides != nil {
		blockOverrides.Apply(&blockCtx)
	}
	rules := b.ChainConfig().Rules(blockCtx.BlockNumber)
	precompiles := maps.Clone(vm.ActivePrecompiledContracts(rules))
	if err := overrides.Apply(state, precompiles); err != nil {
		return nil, err
	}

	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()
	return applyMessage(ctx, b, args, state, block, timeout, new(core.GasPool).AddGas(globalGasCap), &blockCtx, &vm.Config{NoBaseFee: true}, precompiles, true)
}

func applyMessage(ctx context.Context, b Backend, args TransactionArgs, state *state.StateDB, block *types.Block, timeout time.Duration, gp *core.GasPool, blockContext *vm.BlockContext, vmConfig *vm.Config, precompiles vm.PrecompiledContracts, skipChecks bool) (*core.ExecutionResult, error) {
	header := block.Header()
	author, err := b.Engine().Author(header)
	if err != nil {
		return nil, err
	}
	XDCxState, err := b.XDCxService().GetTradingState(block, author)
	if err != nil {
		return nil, err
	}

	// Get a new instance of the EVM.
	if err := args.CallDefaults(gp.Gas(), blockContext.BaseFee, b.ChainConfig().ChainID); err != nil {
		return nil, err
	}
	msg := args.ToMessage(b, header.BaseFee, skipChecks, skipChecks)
	msg.BalanceTokenFee = new(big.Int).SetUint64(msg.GasLimit)
	msg.BalanceTokenFee.Mul(msg.BalanceTokenFee, msg.GasPrice)
	// Lower the basefee to 0 to avoid breaking EVM
	// invariants (basefee < feecap).
	if msg.GasPrice.Sign() == 0 {
		blockContext.BaseFee = new(big.Int)
	}
	state.SetBalance(msg.From, math.MaxBig256, tracing.BalanceChangeUnspecified)
	evm, vmError, err := b.GetEVM(ctx, state, XDCxState, header, vmConfig, blockContext)
	if err != nil {
		return nil, err
	}
	if err := vmError(); err != nil {
		return nil, err
	}
	if precompiles != nil {
		evm.SetPrecompiles(precompiles)
	}
	evm.SetTxContext(core.NewEVMTxContext(msg))
	res, err := applyMessageWithEVM(ctx, evm, msg, timeout, gp)
	// If an internal state error occurred, let that have precedence. Otherwise,
	// a "trie root missing" type of error will masquerade as e.g. "insufficient gas"
	if err := state.Error(); err != nil {
		return nil, err
	}
	return res, err
}

func applyMessageWithEVM(ctx context.Context, evm *vm.EVM, msg *core.Message, timeout time.Duration, gp *core.GasPool) (*core.ExecutionResult, error) {
	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	// Execute the message.
	result, err := core.ApplyMessage(evm, msg, gp, common.Address{})

	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.GasLimit)
	}
	return result, err
}

// Call executes the given transaction on the state for the given block number.
// It doesn't make and changes in the state/blockchain and is useful to execute and retrieve values.
func (api *BlockChainAPI) Call(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, overrides *StateOverride, blockOverrides *BlockOverrides) (hexutil.Bytes, error) {
	if blockNrOrHash == nil {
		latest := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
		blockNrOrHash = &latest
	}
	timeout := api.b.RPCEVMTimeout()
	if args.To != nil && *args.To == common.MasternodeVotingSMCBinary {
		timeout = 0
	}
	result, err := DoCall(ctx, api.b, args, *blockNrOrHash, overrides, blockOverrides, timeout, api.b.RPCGasCap())
	if err != nil {
		return nil, err
	}
	// If the result contains a revert reason, try to unpack and return it.
	if len(result.Revert()) > 0 {
		return nil, newRevertError(result.Revert())
	}
	return result.Return(), result.Err
}

// SimulateV1 executes series of transactions on top of a base state.
// The transactions are packed into blocks. For each block, block header
// fields can be overridden. The state can also be overridden prior to
// execution of each block.
//
// Note, this function doesn't make any changes in the state/blockchain and is
// useful to execute and retrieve values.
func (api *BlockChainAPI) SimulateV1(ctx context.Context, opts simOpts, blockNrOrHash *rpc.BlockNumberOrHash) ([]map[string]interface{}, error) {
	if len(opts.BlockStateCalls) == 0 {
		return nil, &invalidParamsError{message: "empty input"}
	} else if len(opts.BlockStateCalls) > maxSimulateBlocks {
		return nil, &clientLimitExceededError{message: "too many blocks"}
	}
	if blockNrOrHash == nil {
		n := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
		blockNrOrHash = &n
	}
	state, base, err := api.b.StateAndHeaderByNumberOrHash(ctx, *blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	sim := &simulator{
		b:           api.b,
		state:       state,
		base:        base,
		chainConfig: api.b.ChainConfig(),
		// Each tx and all the series of txes shouldn't consume more gas than cap
		gp:             new(core.GasPool).AddGas(api.b.RPCGasCap()),
		traceTransfers: opts.TraceTransfers,
		validate:       opts.Validation,
		fullTx:         opts.ReturnFullTransactions,
	}
	return sim.execute(ctx, opts.BlockStateCalls)
}

// DoEstimateGas returns the lowest possible gas limit that allows the transaction to run
// successfully at block `blockNrOrHash`. It returns error if the transaction would revert, or if
// there are unexpected failures. The gas limit is capped by both `args.Gas` (if non-nil &
// non-zero) and `gasCap` (if non-zero).
func DoEstimateGas(ctx context.Context, b Backend, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *StateOverride, gasCap uint64) (hexutil.Uint64, error) {
	// Retrieve the base state and mutate it with any overrides
	state, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return 0, err
	}
	if err = overrides.Apply(state, nil); err != nil {
		return 0, err
	}
	// Construct the gas estimator option from the user input
	opts := &gasestimator.Options{
		Config: b.ChainConfig(),
		Chain:  NewChainContext(ctx, b),
		Header: header,
		State:  state,
	}
	// Set any required transaction default, but make sure the gas cap itself is not messed with
	// if it was not specified in the original argument list.
	if args.Gas == nil {
		args.Gas = new(hexutil.Uint64)
	}
	if err := args.CallDefaults(gasCap, header.BaseFee, b.ChainConfig().ChainID); err != nil {
		return 0, err
	}
	call := args.ToMessage(b, header.BaseFee, true, true)

	// Run the gas estimation andwrap any revertals into a custom return
	estimate, revert, err := gasestimator.Estimate(ctx, call, opts, gasCap)
	if err != nil {
		if len(revert) > 0 {
			return 0, newRevertError(revert)
		}
		return 0, err
	}
	return hexutil.Uint64(estimate), nil
}

// EstimateGas returns an estimate of the amount of gas needed to execute the
// given transaction against the current pending block.
func (api *BlockChainAPI) EstimateGas(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, overrides *StateOverride) (hexutil.Uint64, error) {
	bNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}
	return DoEstimateGas(ctx, api.b, args, bNrOrHash, overrides, api.b.RPCGasCap())
}

// RPCMarshalHeader converts the given header to the RPC output .
func RPCMarshalHeader(head *types.Header) map[string]interface{} {
	result := map[string]interface{}{
		"number":           (*hexutil.Big)(head.Number),
		"hash":             head.Hash(),
		"parentHash":       head.ParentHash,
		"nonce":            head.Nonce,
		"mixHash":          head.MixDigest,
		"sha3Uncles":       head.UncleHash,
		"logsBloom":        head.Bloom,
		"stateRoot":        head.Root,
		"miner":            head.Coinbase,
		"difficulty":       (*hexutil.Big)(head.Difficulty),
		"extraData":        hexutil.Bytes(head.Extra),
		"size":             hexutil.Uint64(head.Size()),
		"gasLimit":         hexutil.Uint64(head.GasLimit),
		"gasUsed":          hexutil.Uint64(head.GasUsed),
		"timestamp":        hexutil.Uint64(head.Time),
		"transactionsRoot": head.TxHash,
		"receiptsRoot":     head.ReceiptHash,
		"validators":       hexutil.Bytes(head.Validators),
		"validator":        hexutil.Bytes(head.Validator),
		"penalties":        hexutil.Bytes(head.Penalties),
	}

	if head.BaseFee != nil {
		result["baseFeePerGas"] = (*hexutil.Big)(head.BaseFee)
	}

	return result
}

// RPCMarshalBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func RPCMarshalBlock(block *types.Block, inclTx bool, fullTx bool, config *params.ChainConfig) map[string]interface{} {
	fields := RPCMarshalHeader(block.Header())
	fields["size"] = hexutil.Uint64(block.Size())

	if inclTx {
		formatTx := func(idx int, tx *types.Transaction) interface{} {
			return tx.Hash()
		}
		if fullTx {
			formatTx = func(idx int, tx *types.Transaction) interface{} {
				return newRPCTransactionFromBlockIndex(block, uint64(idx), config)
			}
		}
		txs := block.Transactions()
		transactions := make([]interface{}, len(txs))
		for i, tx := range txs {
			transactions[i] = formatTx(i, tx)
		}
		fields["transactions"] = transactions
	}
	uncles := block.Uncles()
	uncleHashes := make([]common.Hash, len(uncles))
	for i, uncle := range uncles {
		uncleHashes[i] = uncle.Hash()
	}
	fields["uncles"] = uncleHashes
	return fields
}

// rpcMarshalBlock uses the generalized output filler, then adds the total difficulty field, which requires
// a `BlockChainAPI`.
func (api *BlockChainAPI) rpcMarshalBlock(ctx context.Context, b *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
	fields := RPCMarshalBlock(b, inclTx, fullTx, api.b.ChainConfig())
	if inclTx {
		fields["totalDifficulty"] = (*hexutil.Big)(api.b.GetTd(ctx, b.Hash()))
	}
	return fields, nil
}

// findNearestSignedBlock finds the nearest checkpoint from input block
func (api *BlockChainAPI) findNearestSignedBlock(ctx context.Context, b *types.Block) *types.Block {
	if b.Number().Sign() <= 0 {
		return nil
	}

	blockNumber := b.Number().Uint64()
	signedBlockNumber := blockNumber + (common.MergeSignRange - (blockNumber % common.MergeSignRange))
	latestBlockNumber := api.b.CurrentBlock().Number

	if signedBlockNumber >= latestBlockNumber.Uint64() || !api.b.ChainConfig().IsTIPSigning(b.Number()) {
		signedBlockNumber = blockNumber
	}

	// Get block epoc latest
	checkpointNumber, _, err := api.b.Engine().(*XDPoS.XDPoS).GetCurrentEpochSwitchBlock(api.chainReader, big.NewInt(int64(signedBlockNumber)))
	if err != nil {
		log.Error("[findNearestSignedBlock] Error while trying to get current Epoch switch block", "Number", signedBlockNumber)
	}

	checkpointBlock, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(checkpointNumber))

	if checkpointBlock != nil {
		signedBlock, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(signedBlockNumber))
		return signedBlock
	}

	return nil
}

/*
findFinalityOfBlock return finality of a block
Use blocksHashCache for to keep track - refer core/blockchain.go for more detail
*/
func (api *BlockChainAPI) findFinalityOfBlock(ctx context.Context, b *types.Block, masternodes []common.Address) (uint, error) {
	engine, _ := api.b.Engine().(*XDPoS.XDPoS)
	signedBlock := api.findNearestSignedBlock(ctx, b)

	if signedBlock == nil {
		return 0, nil
	}

	signedBlocksHash := api.b.GetBlocksHashCache(signedBlock.Number().Uint64())

	// there is no cache for this block's number
	// return the number(signers) / number(masternode) * 100 if this block is on canonical path
	// else return 0 for fork path
	if signedBlocksHash == nil {
		if !api.b.AreTwoBlockSamePath(signedBlock.Hash(), b.Hash()) {
			return 0, nil
		}

		blockSigners, err := api.getSigners(ctx, signedBlock, engine)
		if blockSigners == nil {
			return 0, err
		}

		return uint(100 * len(blockSigners) / len(masternodes)), nil
	}

	/*
		With Hashes cache - we can track all chain's path
		back to current's block number by parent's Hash
		If found the current block so the finality = signedBlock's finality
		else return 0
	*/

	var signedBlockSamePath common.Hash

	for count := 0; count < len(signedBlocksHash); count++ {
		blockHash := signedBlocksHash[count]
		if api.b.AreTwoBlockSamePath(blockHash, b.Hash()) {
			signedBlockSamePath = blockHash
			break
		}
	}

	// return 0 if not same path with any signed block
	if len(signedBlockSamePath) == 0 {
		return 0, nil
	}

	// get signers and return finality
	samePathSignedBlock, err := api.b.GetBlock(ctx, signedBlockSamePath)
	if samePathSignedBlock == nil {
		return 0, err
	}

	blockSigners, err := api.getSigners(ctx, samePathSignedBlock, engine)
	if blockSigners == nil {
		return 0, err
	}

	return uint(100 * len(blockSigners) / len(masternodes)), nil
}

/*
Extract signers from block
*/
func (api *BlockChainAPI) getSigners(ctx context.Context, block *types.Block, engine *XDPoS.XDPoS) ([]common.Address, error) {
	var err error
	var filterSigners []common.Address
	var signers []common.Address

	masternodes := engine.GetMasternodes(api.chainReader, block.Header())

	signers, err = GetSignersFromBlocks(api.b, block.NumberU64(), block.Hash(), masternodes)
	if err != nil {
		log.Error("Fail to get signers from block signer SC.", "error", err)
		return nil, err
	}
	validator, _ := engine.RecoverValidator(block.Header())
	creator, _ := engine.RecoverSigner(block.Header())
	signers = append(signers, validator)
	signers = append(signers, creator)

	for _, masternode := range masternodes {
		for _, signer := range signers {
			if signer == masternode {
				filterSigners = append(filterSigners, masternode)
				break
			}
		}
	}
	return filterSigners, nil
}

func (api *BlockChainAPI) rpcOutputBlockSigners(b *types.Block, ctx context.Context, masternodes []common.Address) ([]common.Address, error) {
	_, err := api.b.GetIPCClient()
	if err != nil {
		log.Error("Fail to connect IPC client for block status", "error", err)
		return []common.Address{}, err
	}

	engine, ok := api.b.Engine().(*XDPoS.XDPoS)
	if !ok {
		log.Error("Undefined XDPoS consensus engine")
		return []common.Address{}, nil
	}

	signedBlock := api.findNearestSignedBlock(ctx, b)
	if signedBlock == nil {
		return []common.Address{}, nil
	}

	return api.getSigners(ctx, signedBlock, engine)
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash         *common.Hash                 `json:"blockHash"`
	BlockNumber       *hexutil.Big                 `json:"blockNumber"`
	From              common.Address               `json:"from"`
	Gas               hexutil.Uint64               `json:"gas"`
	GasPrice          *hexutil.Big                 `json:"gasPrice"`
	GasFeeCap         *hexutil.Big                 `json:"maxFeePerGas,omitempty"`
	GasTipCap         *hexutil.Big                 `json:"maxPriorityFeePerGas,omitempty"`
	Hash              common.Hash                  `json:"hash"`
	Input             hexutil.Bytes                `json:"input"`
	Nonce             hexutil.Uint64               `json:"nonce"`
	To                *common.Address              `json:"to"`
	TransactionIndex  *hexutil.Uint64              `json:"transactionIndex"`
	Value             *hexutil.Big                 `json:"value"`
	Type              hexutil.Uint64               `json:"type"`
	Accesses          *types.AccessList            `json:"accessList,omitempty"`
	ChainID           *hexutil.Big                 `json:"chainId,omitempty"`
	AuthorizationList []types.SetCodeAuthorization `json:"authorizationList,omitempty"`
	V                 *hexutil.Big                 `json:"v"`
	R                 *hexutil.Big                 `json:"r"`
	S                 *hexutil.Big                 `json:"s"`
	YParity           *hexutil.Uint64              `json:"yParity,omitempty"`
}

// newRPCTransaction returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func newRPCTransaction(tx *types.Transaction, blockHash common.Hash, blockNumber uint64, index uint64, baseFee *big.Int, config *params.ChainConfig) *RPCTransaction {
	signer := types.MakeSigner(config, new(big.Int).SetUint64(blockNumber))
	from, _ := types.Sender(signer, tx)
	v, r, s := tx.RawSignatureValues()
	result := &RPCTransaction{
		Type:     hexutil.Uint64(tx.Type()),
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}
	if blockHash != (common.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = (*hexutil.Uint64)(&index)
	}

	switch tx.Type() {
	case types.LegacyTxType:
		// if a legacy transaction has an EIP-155 chain id, include it explicitly
		if id := tx.ChainId(); id.Sign() > 0 {
			result.ChainID = (*hexutil.Big)(id)
		}
	case types.AccessListTxType:
		al := tx.AccessList()
		yparity := hexutil.Uint64(v.Sign())
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.YParity = &yparity

	case types.DynamicFeeTxType:
		al := tx.AccessList()
		yparity := hexutil.Uint64(v.Sign())
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.YParity = &yparity
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// if the transaction has been mined, compute the effective gas price
		if baseFee != nil && blockHash != (common.Hash{}) {
			// price = min(tip, gasFeeCap - baseFee) + baseFee
			result.GasPrice = (*hexutil.Big)(effectiveGasPrice(tx, baseFee))
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}

	case types.SetCodeTxType:
		al := tx.AccessList()
		yparity := hexutil.Uint64(v.Sign())
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.YParity = &yparity
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// if the transaction has been mined, compute the effective gas price
		if baseFee != nil && blockHash != (common.Hash{}) {
			result.GasPrice = (*hexutil.Big)(effectiveGasPrice(tx, baseFee))
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}
		result.AuthorizationList = tx.SetCodeAuthorizations()
	}
	return result
}

// effectiveGasPrice computes the transaction gas fee, based on the given basefee value.
//
//	price = min(gasTipCap + baseFee, gasFeeCap)
func effectiveGasPrice(tx *types.Transaction, baseFee *big.Int) *big.Int {
	fee := tx.GasTipCap()
	fee = fee.Add(fee, baseFee)
	if tx.GasFeeCapIntCmp(fee) < 0 {
		return tx.GasFeeCap()
	}
	return fee
}

// newRPCPendingTransaction returns a pending transaction that will serialize to the RPC representation
func newRPCPendingTransaction(tx *types.Transaction, current *types.Header, config *params.ChainConfig) *RPCTransaction {
	var (
		baseFee     *big.Int
		blockNumber = uint64(0)
	)
	if current != nil {
		baseFee = eip1559.CalcBaseFee(config, current)
		blockNumber = current.Number.Uint64()
	}
	return newRPCTransaction(tx, common.Hash{}, blockNumber, 0, baseFee, config)
}

// newRPCTransactionFromBlockIndex returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockIndex(b *types.Block, index uint64, config *params.ChainConfig) *RPCTransaction {
	txs := b.Transactions()
	if index >= uint64(len(txs)) {
		return nil
	}
	return newRPCTransaction(txs[index], b.Hash(), b.NumberU64(), index, b.BaseFee(), config)
}

// newRPCRawTransactionFromBlockIndex returns the bytes of a transaction given a block and a transaction index.
func newRPCRawTransactionFromBlockIndex(b *types.Block, index uint64) hexutil.Bytes {
	txs := b.Transactions()
	if index >= uint64(len(txs)) {
		return nil
	}
	blob, _ := txs[index].MarshalBinary()
	return blob
}

// accessListResult returns an optional accesslist
// Its the result of the `debug_createAccessList` RPC call.
// It contains an error if the transaction itself failed.
type accessListResult struct {
	Accesslist *types.AccessList `json:"accessList"`
	Error      string            `json:"error,omitempty"`
	GasUsed    hexutil.Uint64    `json:"gasUsed"`
}

// CreateAccessList creates a EIP-2930 type AccessList for the given transaction.
// Reexec and BlockNrOrHash can be specified to create the accessList on top of a certain state.
func (api *BlockChainAPI) CreateAccessList(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash) (*accessListResult, error) {
	bNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}
	acl, gasUsed, vmerr, err := AccessList(ctx, api.b, bNrOrHash, args)
	if err != nil {
		return nil, err
	}
	result := &accessListResult{Accesslist: &acl, GasUsed: hexutil.Uint64(gasUsed)}
	if vmerr != nil {
		result.Error = vmerr.Error()
	}
	return result, nil
}

// AccessList creates an access list for the given transaction.
// If the accesslist creation fails an error is returned.
// If the transaction itself fails, an vmErr is returned.
func AccessList(ctx context.Context, b Backend, blockNrOrHash rpc.BlockNumberOrHash, args TransactionArgs) (acl types.AccessList, gasUsed uint64, vmErr error, err error) {
	// Retrieve the execution context
	db, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if db == nil || err != nil {
		return nil, 0, nil, err
	}
	block, err := b.BlockByHash(ctx, header.Hash())
	if err != nil {
		return nil, 0, nil, err
	}
	if block == nil {
		return nil, 0, nil, fmt.Errorf("nil block in AccessList: number=%d, hash=%s", header.Number.Uint64(), header.Hash().Hex())
	}
	author, err := b.Engine().Author(block.Header())
	if err != nil {
		return nil, 0, nil, err
	}
	XDCxState, err := b.XDCxService().GetTradingState(block, author)
	if err != nil {
		return nil, 0, nil, err
	}

	// Ensure any missing fields are filled, extract the recipient and input data
	if err := args.setDefaults(ctx, b, true); err != nil {
		return nil, 0, nil, err
	}
	if args.Nonce == nil {
		nonce := hexutil.Uint64(db.GetNonce(args.from()))
		args.Nonce = &nonce
	}
	blockCtx := core.NewEVMBlockContext(header, NewChainContext(ctx, b), nil)
	if err = args.CallDefaults(b.RPCGasCap(), blockCtx.BaseFee, b.ChainConfig().ChainID); err != nil {
		return nil, 0, nil, err
	}

	var to common.Address
	if args.To != nil {
		to = *args.To
	} else {
		to = crypto.CreateAddress(args.from(), uint64(*args.Nonce))
	}
	// Retrieve the precompiles since they don't need to be added to the access list
	precompiles := vm.ActivePrecompiles(b.ChainConfig().Rules(header.Number))

	// Create an initial tracer
	prevTracer := logger.NewAccessListTracer(nil, args.from(), to, precompiles)
	if args.AccessList != nil {
		prevTracer = logger.NewAccessListTracer(*args.AccessList, args.from(), to, precompiles)
	}
	for {
		// Retrieve the current access list to expand
		accessList := prevTracer.AccessList()
		log.Trace("Creating access list", "input", accessList)

		// Copy the original db so we don't modify it
		statedb := db.Copy()
		// Set the accesslist to the last al
		args.AccessList = &accessList
		msg := args.ToMessage(b, header.BaseFee, true, true)

		feeCapacity := statedb.GetTRC21FeeCapacityFromState()
		var balanceTokenFee *big.Int
		if value, ok := feeCapacity[to]; ok {
			balanceTokenFee = value
		}
		msg.BalanceTokenFee = balanceTokenFee

		// Apply the transaction with the access list tracer
		tracer := logger.NewAccessListTracer(accessList, args.from(), to, precompiles)
		config := vm.Config{Tracer: tracer.Hooks(), NoBaseFee: true}
		statedb.SetBalance(msg.From, math.MaxBig256, tracing.BalanceChangeUnspecified)
		evm, _, err := b.GetEVM(ctx, statedb, XDCxState, header, &config, nil)
		if err != nil {
			return nil, 0, nil, err
		}
		// Lower the basefee to 0 to avoid breaking EVM
		// invariants (basefee < feecap).
		if msg.GasPrice.Sign() == 0 {
			evm.Context.BaseFee = new(big.Int)
		}
		evm.SetTxContext(core.NewEVMTxContext(msg))
		res, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(msg.GasLimit), common.Address{})
		if err != nil {
			return nil, 0, nil, fmt.Errorf("failed to apply transaction: %v err: %v", args.ToTransaction(types.LegacyTxType).Hash(), err)
		}
		if tracer.Equal(prevTracer) {
			return accessList, res.UsedGas, res.Err, nil
		}
		prevTracer = tracer
	}
}

// TransactionAPI exposes methods for reading and creating transaction data.
type TransactionAPI struct {
	b         Backend
	nonceLock *AddrLocker
	signer    types.Signer
}

// NewTransactionAPI creates a new RPC service with methods specific for the transaction pool.
func NewTransactionAPI(b Backend, nonceLock *AddrLocker) *TransactionAPI {
	// The signer used by the API should always be the 'latest' known one because we expect
	// signers to be backwards-compatible with old transactions.
	signer := types.LatestSigner(b.ChainConfig())
	return &TransactionAPI{b, nonceLock, signer}
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
func (s *TransactionAPI) GetBlockTransactionCountByNumber(ctx context.Context, blockNr rpc.BlockNumber) *hexutil.Uint {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		n := hexutil.Uint(len(block.Transactions()))
		return &n
	}
	return nil
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func (s *TransactionAPI) GetBlockTransactionCountByHash(ctx context.Context, blockHash common.Hash) *hexutil.Uint {
	if block, _ := s.b.GetBlock(ctx, blockHash); block != nil {
		n := hexutil.Uint(len(block.Transactions()))
		return &n
	}
	return nil
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (s *TransactionAPI) GetTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) *RPCTransaction {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		return newRPCTransactionFromBlockIndex(block, uint64(index), s.b.ChainConfig())
	}
	return nil
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *TransactionAPI) GetTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) *RPCTransaction {
	if block, _ := s.b.GetBlock(ctx, blockHash); block != nil {
		return newRPCTransactionFromBlockIndex(block, uint64(index), s.b.ChainConfig())
	}
	return nil
}

// GetRawTransactionByBlockNumberAndIndex returns the bytes of the transaction for the given block number and index.
func (s *TransactionAPI) GetRawTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) hexutil.Bytes {
	if block, _ := s.b.BlockByNumber(ctx, blockNr); block != nil {
		return newRPCRawTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetRawTransactionByBlockHashAndIndex returns the bytes of the transaction for the given block hash and index.
func (s *TransactionAPI) GetRawTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) hexutil.Bytes {
	if block, _ := s.b.GetBlock(ctx, blockHash); block != nil {
		return newRPCRawTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (s *TransactionAPI) GetTransactionCount(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Uint64, error) {
	// Ask transaction pool for the nonce which includes pending transactions
	if blockNr, ok := blockNrOrHash.Number(); ok && blockNr == rpc.PendingBlockNumber {
		nonce, err := s.b.GetPoolNonce(ctx, address)
		if err != nil {
			return nil, err
		}
		return (*hexutil.Uint64)(&nonce), nil
	}
	// Resolve block number and use its state to ask for the nonce
	state, _, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	nonce := state.GetNonce(address)
	return (*hexutil.Uint64)(&nonce), state.Error()
}

// GetTransactionByHash returns the transaction for the given hash
func (s *TransactionAPI) GetTransactionByHash(ctx context.Context, hash common.Hash) (*RPCTransaction, error) {
	// Try to return an already finalized transaction
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(s.b.ChainDb(), hash)
	if tx != nil {
		header, err := s.b.HeaderByHash(ctx, blockHash)
		if err != nil {
			return nil, err
		}
		return newRPCTransaction(tx, blockHash, blockNumber, index, header.BaseFee, s.b.ChainConfig()), nil
	}
	// No finalized transaction, try to retrieve it from the pool
	if tx := s.b.GetPoolTransaction(hash); tx != nil {
		return newRPCPendingTransaction(tx, s.b.CurrentHeader(), s.b.ChainConfig()), nil
	}

	// Transaction unknown, return as such
	return nil, nil
}

// GetRawTransactionByHash returns the bytes of the transaction for the given hash.
func (s *TransactionAPI) GetRawTransactionByHash(ctx context.Context, hash common.Hash) (hexutil.Bytes, error) {
	// Retrieve a finalized transaction, or a pooled otherwise
	tx, _, _, _ := rawdb.ReadTransaction(s.b.ChainDb(), hash)
	if tx == nil {
		if tx = s.b.GetPoolTransaction(hash); tx == nil {
			// Transaction not found anywhere, abort
			return nil, nil
		}
	}
	// Serialize to RLP and return
	return tx.MarshalBinary()
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *TransactionAPI) GetTransactionReceipt(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(s.b.ChainDb(), hash)
	if tx == nil {
		// When the transaction doesn't exist, the RPC method should return JSON null
		// as per specification.
		return nil, nil
	}
	receipts, err := s.b.GetReceipts(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	if uint64(len(receipts)) <= index {
		return nil, nil
	}
	receipt := receipts[index]

	// Derive the sender.
	bigblock := new(big.Int).SetUint64(blockNumber)
	signer := types.MakeSigner(s.b.ChainConfig(), bigblock)
	return marshalReceipt(receipt, blockHash, blockNumber, signer, tx, int(index)), nil
}

// marshalReceipt marshals a transaction receipt into a JSON object.
func marshalReceipt(receipt *types.Receipt, blockHash common.Hash, blockNumber uint64, signer types.Signer, tx *types.Transaction, txIndex int) map[string]interface{} {
	from, _ := types.Sender(signer, tx)

	fields := map[string]interface{}{
		"blockHash":         blockHash,
		"blockNumber":       hexutil.Uint64(blockNumber),
		"transactionHash":   tx.Hash(),
		"transactionIndex":  hexutil.Uint64(txIndex),
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
		"logsBloom":         receipt.Bloom,
		"type":              hexutil.Uint(tx.Type()),
		"effectiveGasPrice": (*hexutil.Big)(receipt.EffectiveGasPrice),
	}

	// Assign receipt status or post state.
	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	} else {
		fields["status"] = hexutil.Uint(receipt.Status)
	}
	if receipt.Logs == nil {
		fields["logs"] = []*types.Log{}
	}

	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if receipt.ContractAddress != (common.Address{}) {
		fields["contractAddress"] = receipt.ContractAddress
	}
	return fields
}

// sign is a helper function that signs a transaction with the private key of the given address.
func (s *TransactionAPI) sign(addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Request the wallet to sign the transaction
	var chainID *big.Int
	if config := s.b.ChainConfig(); config.IsEIP155(s.b.CurrentBlock().Number) {
		chainID = config.ChainID
	}
	return wallet.SignTx(account, tx, chainID)
}

// SubmitTransaction is a helper function that submits tx to txPool and logs a message.
func SubmitTransaction(ctx context.Context, b Backend, tx *types.Transaction) (common.Hash, error) {
	if tx.IsSpecialTransaction() {
		return common.Hash{}, errors.New("don't allow transaction sent to BlockSigners & RandomizeSMC smart contract via API")
	}

	// If the transaction fee cap is already specified, ensure the
	// fee of the given transaction is _reasonable_.
	if err := checkTxFee(tx.GasPrice(), tx.Gas(), b.RPCTxFeeCap()); err != nil {
		return common.Hash{}, err
	}
	if !b.UnprotectedAllowed() && !tx.Protected() {
		// Ensure only eip155 signed transactions are submitted if EIP155Required is set.
		return common.Hash{}, errors.New("only replay-protected (EIP-155) transactions allowed over RPC")
	}
	if err := b.SendTx(ctx, tx); err != nil {
		return common.Hash{}, err
	}

	// Print a log with full tx details for manual investigations and interventions
	signer := types.MakeSigner(b.ChainConfig(), b.CurrentBlock().Number)
	from, err := types.Sender(signer, tx)
	if err != nil {
		return common.Hash{}, err
	}

	if tx.To() == nil {
		addr := crypto.CreateAddress(from, tx.Nonce())
		log.Info("Submitted contract creation", "hash", tx.Hash().Hex(), "from", from, "nonce", tx.Nonce(), "contract", addr.Hex(), "value", tx.Value())
	} else {
		log.Info("Submitted transaction", "hash", tx.Hash().Hex(), "from", from, "nonce", tx.Nonce(), "recipient", tx.To(), "value", tx.Value())
	}
	return tx.Hash(), nil
}

// SendTransaction creates a transaction for the given argument, sign it and submit it to the
// transaction pool.
func (s *TransactionAPI) SendTransaction(ctx context.Context, args TransactionArgs) (common.Hash, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: args.from()}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return common.Hash{}, err
	}

	if args.Nonce == nil {
		// Hold the addresse's mutex around signing to prevent concurrent assignment of
		// the same nonce to multiple accounts.
		s.nonceLock.LockAddr(args.from())
		defer s.nonceLock.UnlockAddr(args.from())
	}

	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, s.b, false); err != nil {
		return common.Hash{}, err
	}
	// Assemble the transaction and sign with the wallet
	tx := args.ToTransaction(types.LegacyTxType)

	var chainID *big.Int
	if config := s.b.ChainConfig(); config.IsEIP155(s.b.CurrentBlock().Number) {
		chainID = config.ChainID
	}
	signed, err := wallet.SignTx(account, tx, chainID)
	if err != nil {
		return common.Hash{}, err
	}
	return SubmitTransaction(ctx, s.b, signed)
}

// FillTransaction fills the defaults (nonce, gas, gasPrice or 1559 fields)
// on a given unsigned transaction, and returns it to the caller for further
// processing (signing + broadcast).
func (s *TransactionAPI) FillTransaction(ctx context.Context, args TransactionArgs) (*SignTransactionResult, error) {
	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, s.b, false); err != nil {
		return nil, err
	}
	// Assemble the transaction and obtain rlp
	tx := args.ToTransaction(types.LegacyTxType)
	data, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &SignTransactionResult{data, tx}, nil
}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *TransactionAPI) SendRawTransaction(ctx context.Context, input hexutil.Bytes) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(input); err != nil {
		return common.Hash{}, err
	}
	return SubmitTransaction(ctx, s.b, tx)
}

// OrderMsg struct
type OrderMsg struct {
	AccountNonce    hexutil.Uint64 `json:"nonce"    gencodec:"required"`
	Quantity        hexutil.Big    `json:"quantity,omitempty"`
	Price           hexutil.Big    `json:"price,omitempty"`
	ExchangeAddress common.Address `json:"exchangeAddress,omitempty"`
	UserAddress     common.Address `json:"userAddress,omitempty"`
	BaseToken       common.Address `json:"baseToken,omitempty"`
	QuoteToken      common.Address `json:"quoteToken,omitempty"`
	Status          string         `json:"status,omitempty"`
	Side            string         `json:"side,omitempty"`
	Type            string         `json:"type,omitempty"`
	OrderID         hexutil.Uint64 `json:"orderid,omitempty"`
	// Signature values
	V hexutil.Big `json:"v" gencodec:"required"`
	R hexutil.Big `json:"r" gencodec:"required"`
	S hexutil.Big `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash common.Hash `json:"hash" rlp:"-"`
}

// LendingMsg api message for lending
type LendingMsg struct {
	AccountNonce    hexutil.Uint64 `json:"nonce"    gencodec:"required"`
	Quantity        hexutil.Big    `json:"quantity,omitempty"`
	RelayerAddress  common.Address `json:"relayerAddress,omitempty"`
	UserAddress     common.Address `json:"userAddress,omitempty"`
	CollateralToken common.Address `json:"collateralToken,omitempty"`
	AutoTopUp       bool           `json:"autoTopUp,omitempty"`
	LendingToken    common.Address `json:"lendingToken,omitempty"`
	Term            hexutil.Uint64 `json:"term,omitempty"`
	Interest        hexutil.Uint64 `json:"interest,omitempty"`
	Status          string         `json:"status,omitempty"`
	Side            string         `json:"side,omitempty"`
	Type            string         `json:"type,omitempty"`
	LendingId       hexutil.Uint64 `json:"lendingId,omitempty"`
	LendingTradeId  hexutil.Uint64 `json:"tradeId,omitempty"`
	ExtraData       string         `json:"extraData,omitempty"`

	// Signature values
	V hexutil.Big `json:"v" gencodec:"required"`
	R hexutil.Big `json:"r" gencodec:"required"`
	S hexutil.Big `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash common.Hash `json:"hash" rlp:"-"`
}

type PriceVolume struct {
	Price  *big.Int `json:"price,omitempty"`
	Volume *big.Int `json:"volume,omitempty"`
}

type InterestVolume struct {
	Interest *big.Int `json:"interest,omitempty"`
	Volume   *big.Int `json:"volume,omitempty"`
}

// Sign calculates an ECDSA signature for:
// keccack256("\x19Ethereum Signed Message:\n" + len(message) + message).
//
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons.
//
// The account associated with addr must be unlocked.
//
// https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_sign
func (s *TransactionAPI) Sign(addr common.Address, data hexutil.Bytes) (hexutil.Bytes, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := s.b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Sign the requested hash with the wallet
	signature, err := wallet.SignText(account, data)
	if err == nil {
		signature[crypto.RecoveryIDOffset] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	}
	return signature, err
}

// SignTransactionResult represents a RLP encoded signed transaction.
type SignTransactionResult struct {
	Raw hexutil.Bytes      `json:"raw"`
	Tx  *types.Transaction `json:"tx"`
}

// SignTransaction will sign the given transaction with the from account.
// The node needs to have the private key of the account corresponding with
// the given from address and it needs to be unlocked.
func (s *TransactionAPI) SignTransaction(ctx context.Context, args TransactionArgs) (*SignTransactionResult, error) {
	if args.Gas == nil {
		return nil, errors.New("not specify Gas")
	}
	if args.GasPrice == nil && (args.MaxPriorityFeePerGas == nil || args.MaxFeePerGas == nil) {
		return nil, errors.New("missing gasPrice or maxFeePerGas/maxPriorityFeePerGas")
	}
	if args.Nonce == nil {
		return nil, errors.New("not specify Nonce")
	}
	if err := args.setDefaults(ctx, s.b, false); err != nil {
		return nil, err
	}
	// Before actually sign the transaction, ensure the transaction fee is reasonable.
	tx := args.ToTransaction(types.LegacyTxType)
	if err := checkTxFee(tx.GasPrice(), tx.Gas(), s.b.RPCTxFeeCap()); err != nil {
		return nil, err
	}
	signed, err := s.sign(args.from(), tx)
	if err != nil {
		return nil, err
	}
	data, err := signed.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &SignTransactionResult{data, tx}, nil
}

// PendingTransactions returns the transactions that are in the transaction pool
// and have a from address that is one of the accounts this node manages.
func (s *TransactionAPI) PendingTransactions() ([]*RPCTransaction, error) {
	pending, err := s.b.GetPoolTransactions()
	if err != nil {
		return nil, err
	}
	accounts := make(map[common.Address]struct{})
	for _, wallet := range s.b.AccountManager().Wallets() {
		for _, account := range wallet.Accounts() {
			accounts[account.Address] = struct{}{}
		}
	}
	curHeader := s.b.CurrentHeader()
	transactions := make([]*RPCTransaction, 0, len(pending))
	for _, tx := range pending {
		from, _ := types.Sender(s.signer, tx)
		if _, exists := accounts[from]; exists {
			transactions = append(transactions, newRPCPendingTransaction(tx, curHeader, s.b.ChainConfig()))
		}
	}
	return transactions, nil
}

// Resend accepts an existing transaction and a new gas price and limit. It will remove
// the given transaction from the pool and reinsert it with the new gas price and limit.
func (s *TransactionAPI) Resend(ctx context.Context, sendArgs TransactionArgs, gasPrice *hexutil.Big, gasLimit *hexutil.Uint64) (common.Hash, error) {
	if sendArgs.Nonce == nil {
		return common.Hash{}, errors.New("missing transaction nonce in transaction spec")
	}
	if err := sendArgs.setDefaults(ctx, s.b, false); err != nil {
		return common.Hash{}, err
	}
	matchTx := sendArgs.ToTransaction(types.LegacyTxType)

	// Before replacing the old transaction, ensure the _new_ transaction fee is reasonable.
	var price = matchTx.GasPrice()
	if gasPrice != nil {
		price = gasPrice.ToInt()
	}
	var gas = matchTx.Gas()
	if gasLimit != nil {
		gas = uint64(*gasLimit)
	}
	if err := checkTxFee(price, gas, s.b.RPCTxFeeCap()); err != nil {
		return common.Hash{}, err
	}

	// Iterate the pending list for replacement
	pending, err := s.b.GetPoolTransactions()
	if err != nil {
		return common.Hash{}, err
	}
	for _, p := range pending {
		wantSigHash := s.signer.Hash(matchTx)
		pFrom, err := types.Sender(s.signer, p)
		if err == nil && pFrom == sendArgs.from() && s.signer.Hash(p) == wantSigHash {
			// Match. Re-sign and send the transaction.
			if gasPrice != nil && (*big.Int)(gasPrice).Sign() != 0 {
				sendArgs.GasPrice = gasPrice
			}
			if gasLimit != nil && *gasLimit != 0 {
				sendArgs.Gas = gasLimit
			}
			signedTx, err := s.sign(sendArgs.from(), sendArgs.ToTransaction(types.LegacyTxType))
			if err != nil {
				return common.Hash{}, err
			}
			if err = s.b.SendTx(ctx, signedTx); err != nil {
				return common.Hash{}, err
			}
			return signedTx.Hash(), nil
		}
	}
	return common.Hash{}, fmt.Errorf("transaction %#x not found", matchTx.Hash())
}

// DebugAPI is the collection of Ethereum APIs exposed over the debugging
// namespace.
type DebugAPI struct {
	b Backend
}

// NewDebugAPI creates a new instance of DebugAPI.
func NewDebugAPI(b Backend) *DebugAPI {
	return &DebugAPI{b: b}
}

// GetBlockRlp retrieves the RLP encoded for of a single block.
func (api *DebugAPI) GetBlockRlp(ctx context.Context, number uint64) (string, error) {
	block, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(number))
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	encoded, err := rlp.EncodeToBytes(block)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", encoded), nil
}

// PrintBlock retrieves a block and returns its pretty printed form.
func (api *DebugAPI) PrintBlock(ctx context.Context, number uint64) (string, error) {
	block, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(number))
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	return block.String(), nil
}

// ChaindbProperty returns leveldb properties of the key-value database.
func (api *DebugAPI) ChaindbProperty(property string) (string, error) {
	if property == "" {
		property = "leveldb.stats"
	} else if !strings.HasPrefix(property, "leveldb.") {
		property = "leveldb." + property
	}
	return api.b.ChainDb().Stat(property)
}

// ChaindbCompact flattens the entire key-value database into a single level,
// removing all unused slots and merging all keys.
func (api *DebugAPI) ChaindbCompact() error {
	cstart := time.Now()
	for b := 0; b <= 255; b++ {
		var (
			start = []byte{byte(b)}
			end   = []byte{byte(b + 1)}
		)
		if b == 255 {
			end = nil
		}
		log.Info("Compacting database", "range", fmt.Sprintf("%#X-%#X", start, end), "elapsed", common.PrettyDuration(time.Since(cstart)))
		if err := api.b.ChainDb().Compact(start, end); err != nil {
			log.Error("Database compaction failed", "err", err)
			return err
		}
	}
	return nil
}

// SetHead rewinds the head of the blockchain to a previous block.
func (api *DebugAPI) SetHead(number hexutil.Uint64) error {
	header := api.b.CurrentHeader()
	if header == nil {
		return errors.New("current header is not available")
	}
	if header.Number.Uint64() <= uint64(number) {
		return errors.New("not allowed to rewind to a future block")
	}
	api.b.SetHead(uint64(number))
	return nil
}

// DbGet returns the raw value of a key stored in the database.
func (api *DebugAPI) DbGet(key string) (hexutil.Bytes, error) {
	blob, err := common.ParseHexOrString(key)
	if err != nil {
		return nil, err
	}
	return api.b.ChainDb().Get(blob)
}

// NetAPI offers network related RPC methods
type NetAPI struct {
	net            *p2p.Server
	networkVersion uint64
}

// NewNetAPI creates a new net API instance.
func NewNetAPI(net *p2p.Server, networkVersion uint64) *NetAPI {
	return &NetAPI{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (s *NetAPI) Listening() bool {
	return true // always listening
}

// PeerCount returns the number of connected peers
func (s *NetAPI) PeerCount() hexutil.Uint {
	return hexutil.Uint(s.net.PeerCount())
}

// Version returns the current ethereum protocol version.
func (s *NetAPI) Version() string {
	return fmt.Sprintf("%d", s.networkVersion)
}

// checkTxFee is an internal function used to check whether the fee of
// the given transaction is _reasonable_(under the cap).
func checkTxFee(gasPrice *big.Int, gas uint64, cap float64) error {
	// Short circuit if there is no cap for transaction fee at all.
	if cap == 0 {
		return nil
	}
	feeEth := new(big.Float).Quo(new(big.Float).SetInt(new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gas))), new(big.Float).SetInt(big.NewInt(params.Ether)))
	feeFloat, _ := feeEth.Float64()
	if feeFloat > cap {
		return fmt.Errorf("tx fee (%.2f ether) exceeds the configured cap (%.2f ether)", feeFloat, cap)
	}
	return nil
}

func GetSignersFromBlocks(b Backend, blockNumber uint64, blockHash common.Hash, masternodes []common.Address) ([]common.Address, error) {
	var addrs []common.Address
	mapMN := map[common.Address]bool{}
	for _, node := range masternodes {
		mapMN[node] = true
	}
	signer := types.MakeSigner(b.ChainConfig(), new(big.Int).SetUint64(blockNumber))
	if engine, ok := b.Engine().(*XDPoS.XDPoS); ok {
		limitNumber := blockNumber + common.LimitTimeFinality
		currentNumber := b.CurrentBlock().Number.Uint64()
		if limitNumber > currentNumber {
			limitNumber = currentNumber
		}
		for i := blockNumber + 1; i <= limitNumber; i++ {
			header, err := b.HeaderByNumber(context.TODO(), rpc.BlockNumber(i))
			if err != nil {
				return addrs, err
			}
			if header == nil {
				return addrs, errors.New("nil header in GetSignersFromBlocks")
			}
			blockData, err := b.BlockByNumber(context.TODO(), rpc.BlockNumber(i))
			if err != nil {
				return addrs, err
			}
			if blockData == nil {
				return addrs, errors.New("nil blockData in GetSignersFromBlocks")
			}
			signTxs := engine.CacheSigningTxs(header.Hash(), blockData.Transactions())
			for _, signtx := range signTxs {
				blkHash := common.BytesToHash(signtx.Data()[len(signtx.Data())-32:])
				from, _ := types.Sender(signer, signtx)
				if blkHash == blockHash && mapMN[from] {
					addrs = append(addrs, from)
					delete(mapMN, from)
				}
			}
			if len(mapMN) == 0 {
				break
			}
		}
	}
	return addrs, nil
}

// GetStakerROI Estimate ROI for stakers using the last epoc reward
// then multiple by epoch per year, if the address is not masternode of last epoch - return 0
// Formular:
//
//	ROI = average_latest_epoch_reward_for_voters*number_of_epoch_per_year/latest_total_cap*100
func (api *BlockChainAPI) GetStakerROI() float64 {
	blockNumber := api.b.CurrentBlock().Number.Uint64()
	lastCheckpointNumber := blockNumber - (blockNumber % api.b.ChainConfig().XDPoS.Epoch) - api.b.ChainConfig().XDPoS.Epoch // calculate for 2 epochs ago
	totalCap := new(big.Int).SetUint64(0)

	mastersCap := api.b.GetMasternodesCap(lastCheckpointNumber)
	if mastersCap == nil {
		return 0
	}

	masternodeReward := new(big.Int).Mul(new(big.Int).SetUint64(api.b.ChainConfig().XDPoS.Reward), new(big.Int).SetUint64(params.Ether))

	for _, cap := range mastersCap {
		totalCap.Add(totalCap, cap)
	}

	holderReward := new(big.Int).Rsh(masternodeReward, 1)
	EpochPerYear := 365 * 86400 / api.b.GetEpochDuration().Uint64()
	voterRewardAYear := new(big.Int).Mul(holderReward, new(big.Int).SetUint64(EpochPerYear))
	return 100.0 / float64(totalCap.Div(totalCap, voterRewardAYear).Uint64())
}

// GetStakerROIMasternode Estimate ROI for stakers of a specific masternode using the last epoc reward
// then multiple by epoch per year, if the address is not masternode of last epoch - return 0
// Formular:
//
//	ROI = latest_epoch_reward_for_voters*number_of_epoch_per_year/latest_total_cap*100
func (api *BlockChainAPI) GetStakerROIMasternode(masternode common.Address) float64 {
	votersReward := api.b.GetVotersRewards(masternode)
	if votersReward == nil {
		return 0
	}

	masternodeReward := new(big.Int).SetUint64(0) // this includes all reward for this masternode
	voters := []common.Address{}
	for voter, reward := range votersReward {
		voters = append(voters, voter)
		masternodeReward.Add(masternodeReward, reward)
	}

	blockNumber := api.b.CurrentBlock().Number.Uint64()
	lastCheckpointNumber := blockNumber - blockNumber%api.b.ChainConfig().XDPoS.Epoch
	totalCap := new(big.Int).SetUint64(0)
	votersCap := api.b.GetVotersCap(new(big.Int).SetUint64(lastCheckpointNumber), masternode, voters)

	for _, cap := range votersCap {
		totalCap.Add(totalCap, cap)
	}

	// holder reward = 50% total reward of a masternode
	holderReward := new(big.Int).Rsh(masternodeReward, 1)
	EpochPerYear := 365 * 86400 / api.b.GetEpochDuration().Uint64()
	voterRewardAYear := new(big.Int).Mul(holderReward, new(big.Int).SetUint64(EpochPerYear))

	return 100.0 / float64(totalCap.Div(totalCap, voterRewardAYear).Uint64())
}

type supplyV1 struct {
	Minted *hexutil.Big `json:"minted"`
}

type supplyV2 struct {
	Minted *hexutil.Big `json:"minted"`
	Burned *hexutil.Big `json:"burned"`
}

type tokenSupply struct {
	V1              *supplyV1    `json:"v1"`
	V2              *supplyV2    `json:"v2"`
	Minted          *hexutil.Big `json:"minted"`
	UpgradeEpochNum *hexutil.Big `json:"upgradeEpochNum"`
	EpochNum        *hexutil.Big `json:"epochNum"`
	BlockHash       common.Hash  `json:"blockHash"`
	BlockNumber     *hexutil.Big `json:"blockNumber"`
}

func (api *BlockChainAPI) GetTokenStats(ctx context.Context, epochNr rpc.EpochNumber) (*tokenSupply, error) {
	engine, ok := api.b.Engine().(*XDPoS.XDPoS)
	if !ok {
		return nil, errors.New("undefined XDPoS consensus engine")
	}
	statedb, header, _ := api.b.StateAndHeaderByNumber(ctx, rpc.LatestBlockNumber)
	nonce := statedb.GetNonce(common.MintedRecordAddressBinary)
	if nonce == 0 {
		return nil, errors.New("mintedRecordAddress is not initialized due to Reward Upgrade is not applied")
	}
	currentRound, err := engine.EngineV2.GetRoundNumber(header)
	currentEpoch := api.b.ChainConfig().XDPoS.V2.SwitchEpoch + uint64(currentRound)/api.b.ChainConfig().XDPoS.Epoch
	if err != nil {
		return nil, err
	}
	onsetEpoch := statedb.GetMintedRecordOnsetEpoch().Big().Uint64()
	if epochNr >= 0 {
		if uint64(epochNr) < onsetEpoch {
			return nil, errors.New("epoch number is before reward upgrade")
		}
		if uint64(epochNr) > currentEpoch {
			return nil, errors.New("epoch number is after current epoch")
		}
	}
	epochNum := uint64(epochNr)
	if epochNr == rpc.LatestEpochNumber {
		epochNum = currentEpoch
	}
	postMinted := statedb.GetPostMinted(epochNum).Big()
	number := statedb.GetPostRewardBlock(epochNum).Big()
	targetHeader, err := api.b.HeaderByNumber(ctx, rpc.BlockNumber(number.Int64()))
	if err != nil {
		return nil, err
	}
	config := api.b.ChainConfig().XDPoS
	if config == nil {
		return nil, errors.New("xdpos config is nil")
	}
	preEpochMinted := new(big.Int).Mul(new(big.Int).SetUint64(config.Reward), new(big.Int).SetUint64(params.Ether))
	onsetEpochMinus := onsetEpoch
	if onsetEpochMinus > 0 {
		onsetEpochMinus--
	} else {
		log.Warn("OnsetEpoch is 0 which could not happen", epochNum)
	}
	preMinted := new(big.Int).Mul(preEpochMinted, new(big.Int).SetUint64(onsetEpochMinus))
	postBurned := statedb.GetPostBurned(epochNum).Big()
	result := &tokenSupply{
		V1: &supplyV1{
			Minted: (*hexutil.Big)(preMinted),
		},
		V2: &supplyV2{
			Minted: (*hexutil.Big)(postMinted),
			Burned: (*hexutil.Big)(postBurned),
		},
		Minted:          (*hexutil.Big)(new(big.Int).Add(postMinted, preMinted)),
		UpgradeEpochNum: (*hexutil.Big)(new(big.Int).SetUint64(onsetEpoch)),
		EpochNum:        (*hexutil.Big)(new(big.Int).SetUint64(epochNum)),
		BlockHash:       targetHeader.Hash(),
		BlockNumber:     (*hexutil.Big)(number),
	}
	return result, nil
}

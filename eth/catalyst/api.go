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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	chainParams "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

// Register adds catalyst APIs to the node.
func Register(stack *node.Node, backend *eth.Ethereum) error {
	if backend.BlockChain().Config().CatalystBlock == nil {
		return errors.New("can't enable catalyst service without catalyst fork block in chain config")
	}

	log.Warn("Catalyst mode enabled")
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace: "consensus",
			Version:   "1.0",
			Service:   newConsensusAPI(backend),
			Public:    true,
		},
	})
	return nil
}

type consensusAPI struct {
	eth  *eth.Ethereum
	env  *blockExecutionEnv
	head common.Hash
}

func newConsensusAPI(eth *eth.Ethereum) *consensusAPI {
	return &consensusAPI{eth: eth}
}

// blockExecutionEnv gathers all the data required to execute
// a block, either when assembling it or when inserting it.
type blockExecutionEnv struct {
	state   *state.StateDB
	tcount  int
	gasPool *core.GasPool

	header   *types.Header
	txs      []*types.Transaction
	receipts []*types.Receipt
}

func (api *consensusAPI) commitTransaction(tx *types.Transaction, coinbase common.Address) error {
	//snap := eth2rpc.current.state.Snapshot()

	chain := api.eth.BlockChain()
	receipt, err := core.ApplyTransaction(chain.Config(), chain, &coinbase, api.env.gasPool, api.env.state, api.env.header, tx, &api.env.header.GasUsed, *chain.GetVMConfig())
	if err != nil {
		//w.current.state.RevertToSnapshot(snap)
		return err
	}
	api.env.txs = append(api.env.txs, tx)
	api.env.receipts = append(api.env.receipts, receipt)

	return nil
}

func (api *consensusAPI) makeEnv(parent *types.Block, header *types.Header) error {
	state, err := api.eth.BlockChain().StateAt(parent.Root())
	if err != nil {
		return err
	}
	api.env = &blockExecutionEnv{
		state:   state,
		header:  header,
		gasPool: new(core.GasPool).AddGas(header.GasLimit),
	}
	return nil
}

// AssembleBlock creates a new block, inserts it into the chain, and returns the "execution
// data" required for eth2 clients to process the new block.
func (api *consensusAPI) AssembleBlock(params assembleBlockParams) (*executableData, error) {
	log.Info("Produce block", "parentHash", params.ParentHash)

	bc := api.eth.BlockChain()
	parent := bc.GetBlockByHash(params.ParentHash)
	pool := api.eth.TxPool()

	if parent.Time() >= params.Timestamp {
		return nil, fmt.Errorf("child timestamp lower than parent's: %d >= %d", parent.Time(), params.Timestamp)
	}
	// this will ensure we're not going off too far in the future
	if now := uint64(time.Now().Unix()); params.Timestamp > now+1 {
		wait := time.Duration(params.Timestamp-now) * time.Second
		log.Info("Producing block too far in the future", "wait", common.PrettyDuration(wait))
		time.Sleep(wait)
	}

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
	err = api.eth.Engine().Prepare(bc, header)
	if err != nil {
		return nil, err
	}

	err = api.makeEnv(parent, header)
	if err != nil {
		return nil, err
	}
	signer := types.LatestSigner(bc.Config())
	txs := types.NewTransactionsByPriceAndNonce(signer, pending)

	var transactions []*types.Transaction

	for {
		if api.env.gasPool.Gas() < chainParams.TxGas {
			log.Trace("Not enough gas for further transactions", "have", api.env.gasPool, "want", chainParams.TxGas)
			break
		}

		tx := txs.Peek()
		if tx == nil {
			fmt.Println("no tx")
			break
		}

		from, _ := types.Sender(signer, tx)
		// XXX replay protection check is missing

		// Execute the transaction
		api.env.state.Prepare(tx.Hash(), common.Hash{}, api.env.tcount)
		err := api.commitTransaction(tx, coinbase)
		switch err {
		case core.ErrGasLimitReached:
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Trace("Gas limit exceeded for current block", "sender", from)
			txs.Pop()

		case core.ErrNonceTooLow:
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case core.ErrNonceTooHigh:
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Trace("Skipping account with high nonce", "sender", from, "nonce", tx.Nonce())
			txs.Pop()

		case nil:
			// Everything ok, collect the logs and shift in the next transaction from the same account
			api.env.tcount++
			txs.Shift()
			transactions = append(transactions, tx)

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txs.Shift()
		}
	}

	block, err := api.eth.Engine().FinalizeAndAssemble(bc, header, api.env.state, transactions, nil /* uncles */, api.env.receipts)
	if err != nil {
		return nil, err
	}

	var logs []*types.Log
	var receipts = make(types.Receipts, len(api.env.receipts))
	hash := block.Hash()
	for i, receipt := range api.env.receipts {
		// add block location fields
		receipt.BlockHash = hash
		receipt.BlockNumber = block.Number()
		receipt.TransactionIndex = uint(i)

		receipts[i] = new(types.Receipt)
		*receipts[i] = *receipt
		// Update the block hash in all logs since it is now available and not when the
		// receipt/log of individual transactions were created.
		for _, log := range receipt.Logs {
			log.BlockHash = hash
		}
		logs = append(logs, receipt.Logs...)
	}

	block.Header().ReceiptHash = types.DeriveSha(receipts, new(trie.Trie))

	return &executableData{
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

func insertBlockParamsToBlock(params executableData) (*types.Block, error) {
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
func (api *consensusAPI) NewBlock(params executableData) (*newBlockResponse, error) {
	// compute block number as parent.number + 1
	parent := api.eth.BlockChain().GetBlockByHash(params.ParentHash)
	if parent == nil {
		return &newBlockResponse{false}, fmt.Errorf("could not find parent %x", params.ParentHash)
	}
	block, err := insertBlockParamsToBlock(params)
	if err != nil {
		return nil, err
	}

	_, err = api.eth.BlockChain().InsertChainWithoutSealVerification(block)
	return &newBlockResponse{err == nil}, err
}

// Used in tests to add a the list of transactions from a block to the tx pool.
func (api *consensusAPI) addBlockTxs(block *types.Block) error {
	for _, tx := range block.Transactions() {
		api.eth.TxPool().AddLocal(tx)
	}
	return nil
}

// FinalizeBlock is called to mark a block as synchronized, so
// that data that is no longer needed can be removed.
func (api *consensusAPI) FinalizeBlock(blockHash common.Hash) (*genericResponse, error) {
	return &genericResponse{true}, nil
}

// SetHead is called to perform a force choice.
func (api *consensusAPI) SetHead(newHead common.Hash) (*genericResponse, error) {
	return &genericResponse{true}, nil
}

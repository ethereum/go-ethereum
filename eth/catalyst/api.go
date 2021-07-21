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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
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
	chainconfig := backend.BlockChain().Config()
	if chainconfig.CatalystBlock == nil {
		return errors.New("catalystBlock is not set in genesis config")
	} else if chainconfig.CatalystBlock.Sign() != 0 {
		return errors.New("catalystBlock of genesis config must be zero")
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
	eth *eth.Ethereum
}

func newConsensusAPI(eth *eth.Ethereum) *consensusAPI {
	return &consensusAPI{eth: eth}
}

// AssembleBlock creates a new block, inserts it into the chain, and returns the "execution
// data" required for eth2 clients to process the new block.
func (api *consensusAPI) AssembleBlock(params assembleBlockParams) (*executableData, error) {
	log.Info("Producing block", "parentHash", params.ParentHash)
	block, err := api.eth.Miner().GetSealingBlock(params.ParentHash, params.Timestamp)
	if err != nil {
		return nil, err
	}
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

func insertBlockParamsToBlock(config *chainParams.ChainConfig, parent *types.Header, params executableData) (*types.Block, error) {
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
	if config.IsLondon(number) {
		header.BaseFee = misc.CalcBaseFee(config, parent)
	}
	block := types.NewBlockWithHeader(header).WithBody(txs, nil /* uncles */)
	return block, nil
}

// NewBlock creates an Eth1 block, inserts it in the chain, and either returns true,
// or false + an error. This is a bit redundant for go, but simplifies things on the
// eth2 side.
func (api *consensusAPI) NewBlock(params executableData) (*newBlockResponse, error) {
	parent := api.eth.BlockChain().GetBlockByHash(params.ParentHash)
	if parent == nil {
		return &newBlockResponse{false}, fmt.Errorf("could not find parent %x", params.ParentHash)
	}
	block, err := insertBlockParamsToBlock(api.eth.BlockChain().Config(), parent.Header(), params)
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

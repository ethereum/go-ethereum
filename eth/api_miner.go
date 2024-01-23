// Copyright 2023 The go-ethereum Authors
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

package eth

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
)

// MinerAPI provides an API to control the miner.
type MinerAPI struct {
	e *Ethereum
}

// NewMinerAPI create a new MinerAPI instance.
func NewMinerAPI(e *Ethereum) *MinerAPI {
	return &MinerAPI{e}
}

// Start starts the miner with the given number of threads. If threads is nil,
// the number of workers started is equal to the number of logical CPUs that are
// usable by this process. If mining is already running, this method adjust the
// number of threads allowed to use and updates the minimum price required by the
// transaction pool.
func (api *MinerAPI) Start() error {
	return api.e.StartMining()
}

// Stop terminates the miner, both at the consensus engine level as well as at
// the block creation level.
func (api *MinerAPI) Stop() {
	api.e.StopMining()
}

// SetExtra sets the extra data string that is included when this miner mines a block.
func (api *MinerAPI) SetExtra(extra string) (bool, error) {
	if err := api.e.Miner().SetExtra([]byte(extra)); err != nil {
		return false, err
	}
	return true, nil
}

// SetGasPrice sets the minimum accepted gas price for the miner.
func (api *MinerAPI) SetGasPrice(gasPrice hexutil.Big) bool {
	api.e.lock.Lock()
	api.e.gasPrice = (*big.Int)(&gasPrice)
	api.e.lock.Unlock()

	api.e.txPool.SetGasTip((*big.Int)(&gasPrice))
	return true
}

// SetGasLimit sets the gaslimit to target towards during mining.
func (api *MinerAPI) SetGasLimit(gasLimit hexutil.Uint64) bool {
	api.e.Miner().SetGasCeil(uint64(gasLimit))
	return true
}

// SetEtherbase sets the etherbase of the miner.
func (api *MinerAPI) SetEtherbase(etherbase common.Address) bool {
	api.e.SetEtherbase(etherbase)
	return true
}

// SetRecommitInterval updates the interval for miner sealing work recommitting.
func (api *MinerAPI) SetRecommitInterval(interval int) {
	api.e.Miner().SetRecommitInterval(time.Duration(interval) * time.Millisecond)
}

// Init initializes the miner without starting mining tasks
func (api *MinerAPI) Init() (common.Address, error) {
	return api.e.InitMiner()
}

type SealBlockRequest struct {
	Parent       common.Hash     `json:"parent"    gencodec:"required"`
	Random       common.Hash     `json:"random"        gencodec:"required"`
	Timestamp    hexutil.Uint64  `json:"timestamp"     gencodec:"required"`
	Transactions []hexutil.Bytes `json:"transactions"  gencodec:"optional"`
}

func decodeTransactions(enc []hexutil.Bytes) ([]*types.Transaction, error) {
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

// SealBlock mines and seals a block without changing the canonical chain
// If `args.Transactions` is not nil then produces a block with only those transactions. If nil, then it consumes from the transaction pool.
// Returns the block if successful.
func (api *MinerAPI) SealBlock(args SealBlockRequest) (map[string]interface{}, error) {
	var transactions []*types.Transaction

	if args.Transactions != nil {
		txs, err := decodeTransactions(args.Transactions)
		if err != nil {
			return nil, err
		}
		transactions = txs
	}

	block, err := api.e.Miner().SealBlockWith(args.Parent, args.Random, uint64(args.Timestamp), transactions)

	if err != nil {
		return nil, err
	}

	return ethapi.RPCMarshalBlock(block, true, true, api.e.APIBackend.ChainConfig()), nil
}

// SetHead updates the canonical chain and announces the block on the p2p layer
func (api *MinerAPI) SetHead(hash common.Hash) (bool, error) {
	block := api.e.BlockChain().GetBlockByHash(hash)

	if block == nil {
		return false, fmt.Errorf("block %s not found", hash.Hex())
	}

	if _, err := api.e.BlockChain().SetCanonical(block); err != nil {
		return false, err
	}

	// Broadcast the block and announce chain insertion event
	api.e.Miner().AnnounceBlock(block)

	return true, nil
}

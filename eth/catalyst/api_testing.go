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

package catalyst

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/rpc"
)

// testingAPI implements the testing_ namespace.
// It's an engine-API adjacent namespace for testing purposes.
type testingAPI struct {
	eth *eth.Ethereum
}

func newTestingAPI(backend *eth.Ethereum) rpc.API {
	return rpc.API{
		Namespace:     "testing",
		Service:       &testingAPI{backend},
		Version:       "1.0",
		Authenticated: false,
	}
}

func (api *testingAPI) BuildBlockV1(parentHash common.Hash, payloadAttributes engine.PayloadAttributes, transactions *[]hexutil.Bytes, extraData *hexutil.Bytes) (*engine.ExecutionPayloadEnvelope, error) {
	if api.eth.BlockChain().CurrentBlock().Hash() != parentHash {
		return nil, errors.New("parentHash is not current head")
	}
	// If transactions is empty but not nil, build an empty block
	// If the transactions is nil, build a block with the current transactions from the txpool
	// If the transactions is not nil and not empty, build a block with the transactions
	buildEmpty := transactions != nil && len(*transactions) == 0
	var txs []*types.Transaction
	if transactions != nil {
		dec := make([][]byte, 0, len(*transactions))
		for _, tx := range *transactions {
			dec = append(dec, tx)
		}
		var err error
		txs, err = engine.DecodeTransactions(dec)
		if err != nil {
			return nil, err
		}
	}
	extra := make([]byte, 0)
	if extraData != nil {
		extra = *extraData
	}
	args := &miner.BuildPayloadArgs{
		Parent:       parentHash,
		Timestamp:    payloadAttributes.Timestamp,
		FeeRecipient: payloadAttributes.SuggestedFeeRecipient,
		Random:       payloadAttributes.Random,
		Withdrawals:  payloadAttributes.Withdrawals,
		BeaconRoot:   payloadAttributes.BeaconRoot,
		SlotNum:      payloadAttributes.SlotNumber,
	}
	return api.eth.Miner().BuildTestingPayload(args, txs, buildEmpty, extra)
}

// CommitBlockV1 builds a block from the supplied attributes and transactions, inserts
// it into the chain, and sets it as the new canonical head. It is the equivalent of
// BuildBlockV1 followed by engine_newPayload + engine_forkchoiceUpdated, but skips the
// serialize/deserialize round-trip through ExecutableData. The block is built on top of
// the current head.
func (api *testingAPI) CommitBlockV1(ctx context.Context, payloadAttributes engine.PayloadAttributes, transactions *[]hexutil.Bytes, extraData *hexutil.Bytes) (common.Hash, error) {
	parentHash := api.eth.BlockChain().CurrentBlock().Hash()
	// If transactions is empty but not nil, build an empty block
	// If the transactions is nil, build a block with the current transactions from the txpool
	// If the transactions is not nil and not empty, build a block with the transactions
	buildEmpty := transactions != nil && len(*transactions) == 0
	var txs []*types.Transaction
	if transactions != nil {
		dec := make([][]byte, 0, len(*transactions))
		for _, tx := range *transactions {
			dec = append(dec, tx)
		}
		var err error
		txs, err = engine.DecodeTransactions(dec)
		if err != nil {
			return common.Hash{}, err
		}
	}
	extra := make([]byte, 0)
	if extraData != nil {
		extra = *extraData
	}
	args := &miner.BuildPayloadArgs{
		Parent:       parentHash,
		Timestamp:    payloadAttributes.Timestamp,
		FeeRecipient: payloadAttributes.SuggestedFeeRecipient,
		Random:       payloadAttributes.Random,
		Withdrawals:  payloadAttributes.Withdrawals,
		BeaconRoot:   payloadAttributes.BeaconRoot,
		SlotNum:      payloadAttributes.SlotNumber,
	}
	block, err := api.eth.Miner().CommitTestingBlock(args, txs, buildEmpty, extra)
	if err != nil {
		return common.Hash{}, err
	}
	if _, err := api.eth.BlockChain().InsertBlockWithoutSetHead(ctx, block, false); err != nil {
		return common.Hash{}, err
	}
	if _, err := api.eth.BlockChain().SetCanonical(block); err != nil {
		return common.Hash{}, err
	}
	return block.Hash(), nil
}

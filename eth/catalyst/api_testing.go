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
	"errors"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

// TestingAPI implements the testing_ namespace.
// It's an engine-API adjacent namespace for testing purposes.
type TestingAPI struct {
	eth *eth.Ethereum
}

func NewTestingAPI(eth *eth.Ethereum) *TestingAPI {
	return &TestingAPI{
		eth: eth,
	}
}

func RegisterTestingAPI(stack *node.Node, backend *eth.Ethereum) error {
	stack.RegisterAPIs([]rpc.API{{
		Namespace:     "testing",
		Service:       NewTestingAPI(backend),
		Authenticated: false,
	},
	})
	return nil
}

func (api *TestingAPI) BuildBlockV1(parentHash common.Hash, payloadAttributes engine.PayloadAttributes, transactions *[]hexutil.Bytes, extraData *hexutil.Bytes) (*engine.ExecutionPayloadEnvelope, error) {
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
	}
	return api.eth.Miner().BuildTestingPayload(args, txs, buildEmpty, extra)
}

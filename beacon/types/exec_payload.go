// Copyright 2024 The go-ethereum Authors
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

package types

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	zrntcommon "github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/beacon/deneb"
)

type payloadType interface {
	*capella.ExecutionPayload | *deneb.ExecutionPayload
}

// convertPayload converts a beacon chain execution payload to types.Block.
func convertPayload[T payloadType](payload T, parentRoot *zrntcommon.Root) (*types.Block, error) {
	var (
		header       types.Header
		transactions []*types.Transaction
		withdrawals  []*types.Withdrawal
		expectedHash [32]byte
		err          error
	)
	switch p := any(payload).(type) {
	case *capella.ExecutionPayload:
		convertCapellaHeader(p, &header)
		transactions, err = convertTransactions(p.Transactions, &header)
		if err != nil {
			return nil, err
		}
		withdrawals = convertWithdrawals(p.Withdrawals, &header)
		expectedHash = p.BlockHash
	case *deneb.ExecutionPayload:
		convertDenebHeader(p, common.Hash(*parentRoot), &header)
		transactions, err = convertTransactions(p.Transactions, &header)
		if err != nil {
			return nil, err
		}
		withdrawals = convertWithdrawals(p.Withdrawals, &header)
		expectedHash = p.BlockHash
	default:
		panic("unsupported block type")
	}

	block := types.NewBlockWithHeader(&header).WithBody(types.Body{Transactions: transactions, Withdrawals: withdrawals})
	if hash := block.Hash(); hash != expectedHash {
		return nil, fmt.Errorf("sanity check failed, payload hash does not match (expected %x, got %x)", expectedHash, hash)
	}
	return block, nil
}

func convertCapellaHeader(payload *capella.ExecutionPayload, h *types.Header) {
	// note: h.TxHash is set in convertTransactions
	h.ParentHash = common.Hash(payload.ParentHash)
	h.UncleHash = types.EmptyUncleHash
	h.Coinbase = common.Address(payload.FeeRecipient)
	h.Root = common.Hash(payload.StateRoot)
	h.ReceiptHash = common.Hash(payload.ReceiptsRoot)
	h.Bloom = types.Bloom(payload.LogsBloom)
	h.Difficulty = common.Big0
	h.Number = new(big.Int).SetUint64(uint64(payload.BlockNumber))
	h.GasLimit = uint64(payload.GasLimit)
	h.GasUsed = uint64(payload.GasUsed)
	h.Time = uint64(payload.Timestamp)
	h.Extra = []byte(payload.ExtraData)
	h.MixDigest = common.Hash(payload.PrevRandao)
	h.Nonce = types.BlockNonce{}
	h.BaseFee = (*uint256.Int)(&payload.BaseFeePerGas).ToBig()
}

func convertDenebHeader(payload *deneb.ExecutionPayload, parentRoot common.Hash, h *types.Header) {
	// note: h.TxHash is set in convertTransactions
	h.ParentHash = common.Hash(payload.ParentHash)
	h.UncleHash = types.EmptyUncleHash
	h.Coinbase = common.Address(payload.FeeRecipient)
	h.Root = common.Hash(payload.StateRoot)
	h.ReceiptHash = common.Hash(payload.ReceiptsRoot)
	h.Bloom = types.Bloom(payload.LogsBloom)
	h.Difficulty = common.Big0
	h.Number = new(big.Int).SetUint64(uint64(payload.BlockNumber))
	h.GasLimit = uint64(payload.GasLimit)
	h.GasUsed = uint64(payload.GasUsed)
	h.Time = uint64(payload.Timestamp)
	h.Extra = []byte(payload.ExtraData)
	h.MixDigest = common.Hash(payload.PrevRandao)
	h.Nonce = types.BlockNonce{}
	h.BaseFee = (*uint256.Int)(&payload.BaseFeePerGas).ToBig()
	// new in deneb
	h.BlobGasUsed = (*uint64)(&payload.BlobGasUsed)
	h.ExcessBlobGas = (*uint64)(&payload.ExcessBlobGas)
	h.ParentBeaconRoot = &parentRoot
}

func convertTransactions(list zrntcommon.PayloadTransactions, execHeader *types.Header) ([]*types.Transaction, error) {
	txs := make([]*types.Transaction, len(list))
	for i, opaqueTx := range list {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(opaqueTx); err != nil {
			return nil, fmt.Errorf("failed to parse tx %d: %v", i, err)
		}
		txs[i] = &tx
	}
	execHeader.TxHash = types.DeriveSha(types.Transactions(txs), trie.NewStackTrie(nil))
	return txs, nil
}

func convertWithdrawals(list zrntcommon.Withdrawals, execHeader *types.Header) []*types.Withdrawal {
	withdrawals := make([]*types.Withdrawal, len(list))
	for i, w := range list {
		withdrawals[i] = &types.Withdrawal{
			Index:     uint64(w.Index),
			Validator: uint64(w.ValidatorIndex),
			Address:   common.Address(w.Address),
			Amount:    uint64(w.Amount),
		}
	}
	wroot := types.DeriveSha(types.Withdrawals(withdrawals), trie.NewStackTrie(nil))
	execHeader.WithdrawalsHash = &wroot
	return withdrawals
}

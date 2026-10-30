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

// Package eip7807 maps a block to the EIP-7807 ExecutionPayload, the SSZ
// container whose hash tree root the EIP defines as the block hash. The
// container is core/types.ExecutionPayload; this package decides where each
// field comes from: the header, the block body, the receipts and EIP-7685
// requests a block does not carry, and the blob schedule of the chain
// configuration.
package eip7807

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

// PayloadFromBlock fills the ExecutionPayload of a block from the block, its
// receipts and its EIP-7685 requests. Every field of the container must
// exist and fit: a block from before the forks that added one is not an
// EIP-7807 block and has no root, and extra data past MAX_EXTRA_DATA_BYTES
// has no place in the container. The receipts, the requests and the block
// access list are checked against the header's commitments, as are the
// transaction and withdrawal lists, so the payload describes the data the
// block hash commits to.
//
// Two fields have no source in the EIP: excess_gas.regular, which EIP-7706
// would have defined and which is zero here, and the ExecutionRequests
// container behind requests_hash, see RequestsRoot.
func PayloadFromBlock(config *params.ChainConfig, block *types.Block, receipts types.Receipts, requests [][]byte) (*types.ExecutionPayload, error) {
	header := block.Header()
	switch {
	case !header.Number.IsUint64():
		return nil, errors.New("block number does not fit uint64")
	case header.BaseFee == nil:
		return nil, errors.New("header has no base fee")
	case header.WithdrawalsHash == nil:
		return nil, errors.New("header has no withdrawals hash")
	case header.BlobGasUsed == nil || header.ExcessBlobGas == nil:
		return nil, errors.New("header has no blob gas fields")
	case header.ParentBeaconRoot == nil:
		return nil, errors.New("header has no parent beacon block root")
	case header.RequestsHash == nil:
		return nil, errors.New("header has no requests hash")
	case header.BlockAccessListHash == nil:
		return nil, errors.New("header has no block access list hash")
	case header.SlotNumber == nil:
		return nil, errors.New("header has no slot number")
	case uint64(len(header.Extra)) > params.MaximumExtraDataSize:
		return nil, fmt.Errorf("extra data of %d bytes exceeds MAX_EXTRA_DATA_BYTES", len(header.Extra))
	}
	txs, withdrawals := block.Transactions(), block.Withdrawals()
	if len(receipts) != len(txs) {
		return nil, fmt.Errorf("%d receipts for %d transactions", len(receipts), len(txs))
	}
	hasher := trie.NewStackTrie(nil)
	if h := types.DeriveSha(txs, hasher); h != header.TxHash {
		return nil, fmt.Errorf("transactions root %x, header %x", h, header.TxHash)
	}
	if h := types.DeriveSha(receipts, hasher); h != header.ReceiptHash {
		return nil, fmt.Errorf("receipts root %x, header %x", h, header.ReceiptHash)
	}
	if h := types.DeriveSha(withdrawals, hasher); h != *header.WithdrawalsHash {
		return nil, fmt.Errorf("withdrawals root %x, header %x", h, *header.WithdrawalsHash)
	}
	if h := types.CalcRequestsHash(requests); h != *header.RequestsHash {
		return nil, fmt.Errorf("requests hash %x, header %x", h, *header.RequestsHash)
	}
	if block.AccessList() == nil {
		return nil, errors.New("block carries no access list")
	}
	accessList, err := rlp.EncodeToBytes(block.AccessList())
	if err != nil {
		return nil, err
	}
	if h := crypto.Keccak256Hash(accessList); h != *header.BlockAccessListHash {
		return nil, fmt.Errorf("block access list hash %x, header %x", h, *header.BlockAccessListHash)
	}

	// The blob gas limit and the blob base fee come from the blob schedule in
	// force at the block's time; the header carries neither.
	blobGasLimit := eip4844.MaxBlobGasPerBlock(config, header.Time)
	if blobGasLimit == 0 {
		return nil, errors.New("no blob schedule at the block's time")
	}
	blobFee, overflow := uint256.FromBig(eip4844.CalcBlobFee(config, header))
	if overflow {
		return nil, errors.New("blob base fee does not fit uint256")
	}
	baseFee, overflow := uint256.FromBig(header.BaseFee)
	if overflow {
		return nil, errors.New("base fee does not fit uint256")
	}

	// The list elements keep their existing encodings.
	p := &types.ExecutionPayload{
		ParentHash:            header.ParentHash,
		Miner:                 header.Coinbase,
		StateRoot:             header.Root,
		Transactions:          make([][]byte, len(txs)),
		Number:                header.Number.Uint64(),
		GasLimits:             types.GasAmounts{Regular: header.GasLimit, Blob: blobGasLimit},
		GasUsed:               types.GasAmounts{Regular: header.GasUsed, Blob: *header.BlobGasUsed},
		Timestamp:             header.Time,
		ExtraData:             header.Extra,
		MixHash:               header.MixDigest,
		BaseFeesPerGas:        types.BlobFeesPerGas{Regular: *baseFee, Blob: *blobFee},
		Withdrawals:           make([][]byte, len(withdrawals)),
		ExcessGas:             types.GasAmounts{Regular: 0, Blob: *header.ExcessBlobGas},
		ParentBeaconBlockRoot: *header.ParentBeaconRoot,
		BlockAccessList:       accessList,
		SlotNumber:            *header.SlotNumber,
	}
	for i, tx := range txs {
		if p.Transactions[i], err = tx.MarshalBinary(); err != nil {
			return nil, fmt.Errorf("transaction %d: %w", i, err)
		}
	}
	for i, w := range withdrawals {
		if p.Withdrawals[i], err = rlp.EncodeToBytes(w); err != nil {
			return nil, fmt.Errorf("withdrawal %d: %w", i, err)
		}
	}
	receiptBytes := make([][]byte, len(receipts))
	for i, r := range receipts {
		if receiptBytes[i], err = r.MarshalBinary(); err != nil {
			return nil, fmt.Errorf("receipt %d: %w", i, err)
		}
	}
	p.ReceiptsRoot = types.ByteListsRoot(receiptBytes)
	if p.RequestsHash, err = RequestsRoot(requests); err != nil {
		return nil, err
	}
	return p, nil
}

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

package ethapi

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

func (r *SignTransactionResult) MarshalJSON() ([]byte, error) {
	if r.frame != nil {
		return json.Marshal(struct {
			Tx *filledFrameTransaction `json:"tx"`
		}{r.frame})
	}
	type result SignTransactionResult
	return json.Marshal((*result)(r))
}

type filledFrameTransaction struct {
	Type                 hexutil.Uint64       `json:"type"`
	ChainID              *hexutil.Big         `json:"chainId"`
	Nonce                hexutil.Uint64       `json:"nonce"`
	From                 common.Address       `json:"from"`
	Frames               []FrameArgs          `json:"frames"`
	Signatures           []FrameSignatureArgs `json:"signatures,omitempty"`
	MaxFeePerGas         *hexutil.Big         `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big         `json:"maxPriorityFeePerGas"`
	MaxFeePerBlobGas     *hexutil.Big         `json:"maxFeePerBlobGas"`
	BlobVersionedHashes  []common.Hash        `json:"blobVersionedHashes,omitempty"`
}

func (api *TransactionAPI) fillFrameTransaction(ctx context.Context, args TransactionArgs) (*SignTransactionResult, error) {
	var cancel context.CancelFunc
	if api.b.RPCEVMTimeout() > 0 {
		ctx, cancel = context.WithTimeout(ctx, api.b.RPCEVMTimeout())
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()
	db, header, err := api.b.StateAndHeaderByNumberOrHash(ctx, rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber))
	if err != nil {
		return nil, err
	}
	// Zero blob fees are valid when the frame transaction carries no blobs.
	blobFee := args.BlobFeeCap
	if len(args.BlobHashes) == 0 {
		args.BlobFeeCap = nil
	}
	if err := args.setFeeDefaults(ctx, api.b, header); err != nil {
		return nil, err
	}
	if len(args.BlobHashes) == 0 {
		args.BlobFeeCap = blobFee
	}
	if args.Nonce == nil {
		nonce, err := api.b.GetPoolNonce(ctx, args.from())
		if err != nil {
			return nil, err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	blockContext := core.NewEVMBlockContext(header, NewChainContext(ctx, api.b), nil)
	if err := args.prepareFrames(ctx, api.b, db, header, &blockContext, nil, api.b.RPCGasCap()); err != nil {
		return nil, err
	}
	return &SignTransactionResult{
		Tx:    args.frameTransaction(),
		frame: &filledFrameTransaction{Type: types.FrameTxType, ChainID: args.ChainID, Nonce: *args.Nonce, From: args.from(), Frames: args.Frames, Signatures: args.Signatures, MaxFeePerGas: args.MaxFeePerGas, MaxPriorityFeePerGas: args.MaxPriorityFeePerGas, MaxFeePerBlobGas: args.BlobFeeCap, BlobVersionedHashes: args.BlobHashes},
	}, nil
}

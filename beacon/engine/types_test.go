// Copyright 2025 The go-ethereum Authors
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

package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestBlobs(t *testing.T) {
	var (
		emptyBlob          = new(kzg4844.Blob)
		emptyBlobCommit, _ = kzg4844.BlobToCommitment(emptyBlob)
		emptyBlobProof, _  = kzg4844.ComputeBlobProof(emptyBlob, emptyBlobCommit)
		emptyCellProof, _  = kzg4844.ComputeCellProofs(emptyBlob)
	)
	header := types.Header{}
	block := types.NewBlock(&header, &types.Body{}, nil, nil)

	sidecarWithoutCellProofs := types.NewBlobTxSidecar(types.BlobSidecarVersion0, []kzg4844.Blob{*emptyBlob}, []kzg4844.Commitment{emptyBlobCommit}, []kzg4844.Proof{emptyBlobProof})
	env := BlockToExecutableData(block, common.Big0, []*types.BlobTxSidecar{sidecarWithoutCellProofs}, nil)
	if len(env.BlobsBundle.Proofs) != 1 {
		t.Fatalf("Expect 1 proof in blobs bundle, got %v", len(env.BlobsBundle.Proofs))
	}

	sidecarWithCellProofs := types.NewBlobTxSidecar(types.BlobSidecarVersion0, []kzg4844.Blob{*emptyBlob}, []kzg4844.Commitment{emptyBlobCommit}, emptyCellProof)
	env = BlockToExecutableData(block, common.Big0, []*types.BlobTxSidecar{sidecarWithCellProofs}, nil)
	if len(env.BlobsBundle.Proofs) != 128 {
		t.Fatalf("Expect 128 proofs in blobs bundle, got %v", len(env.BlobsBundle.Proofs))
	}
}

func TestAttachAccessList(t *testing.T) {
	header := types.Header{GasLimit: 2 * params.BALItemCost}
	block := types.NewBlock(&header, &types.Body{}, nil, nil)

	// 1. Nil access list: returns block unchanged
	res, err := attachAccessList(block, ExecutableData{})
	if err != nil {
		t.Fatalf("unexpected error for nil access list: %v", err)
	}
	if res.AccessList() != nil {
		t.Fatalf("expected nil access list")
	}

	// 2. Empty byte slice: rejected like any other undecodable payload
	_, err = attachAccessList(block, ExecutableData{BlockAccessList: []byte{}})
	if err == nil {
		t.Fatalf("expected error for empty byte slice")
	}

	// 3. Oversized raw byte slice exceeding MaxBlockSize: rejected early
	oversizedBytes := make([]byte, params.MaxBlockSize+1)
	_, err = attachAccessList(block, ExecutableData{BlockAccessList: oversizedBytes})
	if err == nil || !strings.Contains(err.Error(), "exceeds max block size") {
		t.Fatalf("expected max block size error, got: %v", err)
	}

	// Helper to encode BAL
	encodeBAL := func(list bal.BlockAccessList) []byte {
		var buf bytes.Buffer
		if err := rlp.Encode(&buf, list); err != nil {
			t.Fatalf("failed to encode test BAL: %v", err)
		}
		return buf.Bytes()
	}

	// 4. Valid access list within budget (2 accounts = 2 items <= 2 * BALItemCost / BALItemCost)
	balWithinBudget := bal.BlockAccessList{
		{Address: common.HexToAddress("0x1111111111111111111111111111111111111111")},
		{Address: common.HexToAddress("0x2222222222222222222222222222222222222222")},
	}
	res, err = attachAccessList(block, ExecutableData{BlockAccessList: encodeBAL(balWithinBudget)})
	if err != nil {
		t.Fatalf("unexpected error for valid BAL within budget: %v", err)
	}
	if res.AccessList() == nil || len(*res.AccessList()) != 2 {
		t.Fatalf("expected attached access list with 2 accounts")
	}

	// 5. Access list exceeding gas limit item budget (3 accounts = 3 items > 2)
	balExceedingBudget := bal.BlockAccessList{
		{Address: common.HexToAddress("0x1111111111111111111111111111111111111111")},
		{Address: common.HexToAddress("0x2222222222222222222222222222222222222222")},
		{Address: common.HexToAddress("0x3333333333333333333333333333333333333333")},
	}
	_, err = attachAccessList(block, ExecutableData{BlockAccessList: encodeBAL(balExceedingBudget)})
	if err == nil || !strings.Contains(err.Error(), "exceeds item limit") {
		t.Fatalf("expected item limit error, got: %v", err)
	}
}

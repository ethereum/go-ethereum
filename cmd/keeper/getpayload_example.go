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

//go:build example

package main

import (
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

func getChainConfig() *params.ChainConfig {
	return params.MainnetChainConfig
}

func getInput() []byte {
	header := &types.Header{
		ParentHash:  common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Root:        common.Hash{}, // Will be computed by stateless execution
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
		Bloom:       types.Bloom{},
		Difficulty:  big.NewInt(0),
		Number:      big.NewInt(20000000), // Post-merge block number
		GasLimit:    30000000,
		GasUsed:     0,
		Time:        1700000000, // Recent timestamp
		Extra:       []byte("Example block for platform builders"),
		MixDigest:   common.Hash{},
		Nonce:       types.BlockNonce{},
		BaseFee:     big.NewInt(1000000000), // 1 gwei base fee
	}

	// For this example, create an empty block (no transactions)
	// A real implementation would include properly funded accounts
	body := &types.Body{
		Transactions: []*types.Transaction{},
	}
	block := types.NewBlock(header, body, nil, trie.NewStackTrie(nil))

	// Create a parent header (required by witness)
	parentHeader := &types.Header{
		ParentHash:  common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.HexToAddress("0x0000000000000000000000000000000000000000"),
		Root:        common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
		Bloom:       types.Bloom{},
		Difficulty:  big.NewInt(0),
		Number:      big.NewInt(19999999),
		GasLimit:    30000000,
		GasUsed:     0,
		Time:        1699999999,
		Extra:       []byte("Parent block"),
		MixDigest:   common.Hash{},
		Nonce:       types.BlockNonce{},
		BaseFee:     big.NewInt(1000000000),
	}

	// The witness needs state nodes for any accounts accessed
	// For an empty block, minimal state is needed
	witness := &stateless.Witness{
		Headers: []*types.Header{parentHeader},
		Codes: map[string]struct{}{},
		State: map[string]struct{}{},
	}

	payload := Payload{
		Block:   block,
		Witness: witness,
	}

	encoded, err := rlp.EncodeToBytes(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode payload: %v\n", err)
		os.Exit(20)
	}
	return encoded
}

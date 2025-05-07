// Copyright 2025 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/urfave/cli/v2"
)

var (
	historyGenerateCommand = &cli.Command{
		Name:      "historygen",
		Usage:     "Generates history retrieval tests",
		ArgsUsage: "<RPC endpoint URL>",
		Action:    generateHistoryTests,
		Flags: []cli.Flag{
			historyTestFileFlag,
			historyTestEarliestFlag,
		},
	}

	historyTestFileFlag = &cli.StringFlag{
		Name:     "history-tests",
		Usage:    "JSON file containing filter test queries",
		Value:    "history_tests.json",
		Category: flags.TestingCategory,
	}
	historyTestEarliestFlag = &cli.IntFlag{
		Name:     "earliest",
		Usage:    "JSON file containing filter test queries",
		Value:    0,
		Category: flags.TestingCategory,
	}
)

const historyTestBlockCount = 2000

func generateHistoryTests(clictx *cli.Context) error {
	var (
		client     = makeClient(clictx)
		earliest   = uint64(clictx.Int(historyTestEarliestFlag.Name))
		outputFile = clictx.String(historyTestFileFlag.Name)
		ctx        = context.Background()
	)

	test := new(historyTest)

	// Create the block numbers. Here we choose 1k blocks between earliest and head.
	latest, err := client.Eth.BlockNumber(ctx)
	if err != nil {
		exit(err)
	}
	if latest < historyTestBlockCount {
		exit(fmt.Errorf("node seems not synced, latest block is %d", latest))
	}
	test.BlockNumbers = make([]uint64, 0, historyTestBlockCount)
	stride := (latest - earliest) / historyTestBlockCount
	for b := earliest; b < latest; b += stride {
		test.BlockNumbers = append(test.BlockNumbers, b)
	}

	// Get blocks and assign block info into the test
	fmt.Println("Fetching blocks")
	blocks := make([]*types.Block, len(test.BlockNumbers))
	for i, blocknum := range test.BlockNumbers {
		b, err := client.Eth.BlockByNumber(ctx, new(big.Int).SetUint64(blocknum))
		if err != nil {
			exit(fmt.Errorf("error fetching block %d: %v", blocknum, err))
		}
		blocks[i] = b
	}
	test.BlockHashes = make([]common.Hash, len(blocks))
	test.TxCounts = make([]int, len(blocks))
	for i, block := range blocks {
		test.BlockHashes[i] = block.Hash()
		test.TxCounts[i] = len(block.Transactions())
	}

	// Fill tx index.
	test.TxHashIndex = make([]int, len(blocks))
	test.TxHashes = make([]*common.Hash, len(blocks))
	for i, block := range blocks {
		txs := block.Transactions()
		if len(txs) == 0 {
			continue
		}
		index := len(txs) / 2
		txhash := txs[index].Hash()
		test.TxHashIndex[i] = index
		test.TxHashes[i] = &txhash
	}

	// Get receipts.
	fmt.Println("Fetching receipts")
	test.ReceiptsHashes = make([]common.Hash, len(blocks))
	for i, blockHash := range test.BlockHashes {
		receipts, err := client.getBlockReceipts(ctx, blockHash)
		if err != nil {
			exit(fmt.Errorf("error fetching block %v receipts: %v", blockHash, err))
		}
		test.ReceiptsHashes[i] = calcReceiptsHash(receipts)
	}

	// Write output file.
	writeJSON(outputFile, test)
	return nil
}

func calcReceiptsHash(rcpt []*types.Receipt) common.Hash {
	h := crypto.NewKeccakState()
	rlp.Encode(h, rcpt)
	return common.Hash(h.Sum(nil))
}

func writeJSON(fileName string, value any) {
	file, err := os.Create(fileName)
	if err != nil {
		exit(fmt.Errorf("Error creating %s: %v", fileName, err))
		return
	}
	defer file.Close()
	json.NewEncoder(file).Encode(value)
}

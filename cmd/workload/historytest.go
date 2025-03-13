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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/utesting"
)

// historyTest is the content of a history test.
type historyTest struct {
	BlockNumbers   []uint64       `json:"blockNumbers"`
	BlockHashes    []common.Hash  `json:"blockHashes"`
	TxCounts       []int          `json:"txCounts"`
	TxHashIndex    []int          `json:"txHashIndex"`
	TxHashes       []*common.Hash `json:"txHashes"`
	ReceiptsHashes []common.Hash  `json:"blockReceiptsHashes"`
}

type historyTestSuite struct {
	cfg   testConfig
	tests historyTest
}

func newHistoryTestSuite(cfg testConfig) *historyTestSuite {
	s := &historyTestSuite{cfg: cfg}
	if err := s.loadTests(); err != nil {
		exit(err)
	}
	return s
}

func (s *historyTestSuite) loadTests() error {
	file, err := s.cfg.fsys.Open(s.cfg.historyTestFile)
	if err != nil {
		return fmt.Errorf("can't open historyTestFile: %v", err)
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(&s.tests); err != nil {
		return fmt.Errorf("invalid JSON in %s: %v", s.cfg.historyTestFile, err)
	}
	if len(s.tests.BlockNumbers) == 0 {
		return fmt.Errorf("historyTestFile %s has no test data", s.cfg.historyTestFile)
	}
	return nil
}

func (s *historyTestSuite) allTests() []utesting.Test {
	return []utesting.Test{
		{
			Name: "History/getBlockByHash",
			Fn:   s.testGetBlockByHash,
		},
		{
			Name: "History/getBlockByNumber",
			Fn:   s.testGetBlockByNumber,
		},
		{
			Name: "History/getBlockReceiptsByHash",
			Fn:   s.testGetBlockReceiptsByHash,
		},
		{
			Name: "History/getBlockReceiptsByNumber",
			Fn:   s.testGetBlockReceiptsByNumber,
		},
		{
			Name: "History/getBlockTransactionCountByHash",
			Fn:   s.testGetBlockTransactionCountByHash,
		},
		{
			Name: "History/getBlockTransactionCountByNumber",
			Fn:   s.testGetBlockTransactionCountByNumber,
		},
		{
			Name: "History/getTransactionByBlockHashAndIndex",
			Fn:   s.testGetTransactionByBlockHashAndIndex,
		},
		{
			Name: "History/getTransactionByBlockNumberAndIndex",
			Fn:   s.testGetTransactionByBlockNumberAndIndex,
		},
	}
}

func (s *historyTestSuite) testGetBlockByHash(t *utesting.T) {
	ctx := context.Background()

	for i, num := range s.tests.BlockNumbers {
		bhash := s.tests.BlockHashes[i]
		b, err := s.cfg.client.getBlockByHash(ctx, bhash, false)
		if err = validateHistoryPruneErr(err, num, s.cfg.historyPruneBlock); err == errPrunedHistory {
			continue
		} else if err != nil {
			t.Errorf("block %d (hash %v): error %v", num, bhash, err)
			continue
		}
		if b == nil {
			t.Errorf("block %d (hash %v): not found", num, bhash)
			continue
		}
		if b.Hash != bhash || uint64(b.Number) != num {
			t.Errorf("block %d (hash %v): invalid number/hash", num, bhash)
		}
	}
}

func (s *historyTestSuite) testGetBlockByNumber(t *utesting.T) {
	ctx := context.Background()

	for i, num := range s.tests.BlockNumbers {
		bhash := s.tests.BlockHashes[i]
		b, err := s.cfg.client.getBlockByNumber(ctx, num, false)
		if err = validateHistoryPruneErr(err, num, s.cfg.historyPruneBlock); err == errPrunedHistory {
			continue
		} else if err != nil {
			t.Errorf("block %d (hash %v): error %v", num, bhash, err)
			continue
		}
		if b == nil {
			t.Errorf("block %d (hash %v): not found", num, bhash)
			continue
		}
		if b.Hash != bhash || uint64(b.Number) != num {
			t.Errorf("block %d (hash %v): invalid number/hash", num, bhash)
		}
	}
}

func (s *historyTestSuite) testGetBlockTransactionCountByHash(t *utesting.T) {
	ctx := context.Background()

	for i, num := range s.tests.BlockNumbers {
		bhash := s.tests.BlockHashes[i]
		count, err := s.cfg.client.getBlockTransactionCountByHash(ctx, bhash)
		if err = validateHistoryPruneErr(err, num, s.cfg.historyPruneBlock); err == errPrunedHistory {
			continue
		} else if err != nil {
			t.Errorf("block %d (hash %v): error %v", num, bhash, err)
			continue
		}
		expectedCount := uint64(s.tests.TxCounts[i])
		if count != expectedCount {
			t.Errorf("block %d (hash %v): wrong txcount %d, want %d", count, expectedCount)
		}
	}
}

func (s *historyTestSuite) testGetBlockTransactionCountByNumber(t *utesting.T) {
	ctx := context.Background()

	for i, num := range s.tests.BlockNumbers {
		bhash := s.tests.BlockHashes[i]
		count, err := s.cfg.client.getBlockTransactionCountByNumber(ctx, num)
		if err = validateHistoryPruneErr(err, num, s.cfg.historyPruneBlock); err == errPrunedHistory {
			continue
		} else if err != nil {
			t.Errorf("block %d (hash %v): error %v", num, bhash, err)
			continue
		}
		expectedCount := uint64(s.tests.TxCounts[i])
		if count != expectedCount {
			t.Errorf("block %d (hash %v): wrong txcount %d, want %d", count, expectedCount)
		}
	}
}

func (s *historyTestSuite) testGetBlockReceiptsByHash(t *utesting.T) {
	ctx := context.Background()

	for i, num := range s.tests.BlockNumbers {
		bhash := s.tests.BlockHashes[i]
		receipts, err := s.cfg.client.getBlockReceipts(ctx, bhash)
		if err = validateHistoryPruneErr(err, num, s.cfg.historyPruneBlock); err == errPrunedHistory {
			continue
		} else if err != nil {
			t.Errorf("block %d (hash %v): error %v", num, bhash, err)
			continue
		}
		hash := calcReceiptsHash(receipts)
		expectedHash := s.tests.ReceiptsHashes[i]
		if hash != expectedHash {
			t.Errorf("block %d (hash %v): wrong receipts hash %v, want %v", num, bhash, hash, expectedHash)
		}
	}
}

func (s *historyTestSuite) testGetBlockReceiptsByNumber(t *utesting.T) {
	ctx := context.Background()

	for i, num := range s.tests.BlockNumbers {
		bhash := s.tests.BlockHashes[i]
		receipts, err := s.cfg.client.getBlockReceipts(ctx, hexutil.Uint64(num))
		if err = validateHistoryPruneErr(err, num, s.cfg.historyPruneBlock); err == errPrunedHistory {
			continue
		} else if err != nil {
			t.Errorf("block %d (hash %v): error %v", num, bhash, err)
			continue
		}
		hash := calcReceiptsHash(receipts)
		expectedHash := s.tests.ReceiptsHashes[i]
		if hash != expectedHash {
			t.Errorf("block %d (hash %v): wrong receipts hash %v, want %v", num, bhash, hash, expectedHash)
		}
	}
}

func (s *historyTestSuite) testGetTransactionByBlockHashAndIndex(t *utesting.T) {
	ctx := context.Background()

	for i, num := range s.tests.BlockNumbers {
		bhash := s.tests.BlockHashes[i]
		txIndex := s.tests.TxHashIndex[i]
		expectedHash := s.tests.TxHashes[i]
		if expectedHash == nil {
			continue // no txs in block
		}

		tx, err := s.cfg.client.getTransactionByBlockHashAndIndex(ctx, bhash, uint64(txIndex))
		if err = validateHistoryPruneErr(err, num, s.cfg.historyPruneBlock); err == errPrunedHistory {
			continue
		} else if err != nil {
			t.Errorf("block %d (hash %v): error %v", num, bhash, err)
			continue
		}
		if tx == nil {
			t.Errorf("block %d (hash %v): txIndex %d not found", num, bhash, txIndex)
			continue
		}
		if tx.Hash != *expectedHash || uint64(tx.TransactionIndex) != uint64(txIndex) {
			t.Errorf("block %d (hash %v): txIndex %d has wrong txHash/Index", num, bhash, txIndex)
		}
	}
}

func (s *historyTestSuite) testGetTransactionByBlockNumberAndIndex(t *utesting.T) {
	ctx := context.Background()

	for i, num := range s.tests.BlockNumbers {
		bhash := s.tests.BlockHashes[i]
		txIndex := s.tests.TxHashIndex[i]
		expectedHash := s.tests.TxHashes[i]
		if expectedHash == nil {
			continue // no txs in block
		}

		tx, err := s.cfg.client.getTransactionByBlockNumberAndIndex(ctx, num, uint64(txIndex))
		if err = validateHistoryPruneErr(err, num, s.cfg.historyPruneBlock); err == errPrunedHistory {
			continue
		} else if err != nil {
			t.Errorf("block %d (hash %v): error %v", num, bhash, err)
			continue
		}
		if tx == nil {
			t.Errorf("block %d (hash %v): txIndex %d not found", num, bhash, txIndex)
			continue
		}
		if tx.Hash != *expectedHash || uint64(tx.TransactionIndex) != uint64(txIndex) {
			t.Errorf("block %d (hash %v): txIndex %d has wrong txHash/Index", num, bhash, txIndex)
		}
	}
}

type simpleBlock struct {
	Number hexutil.Uint64 `json:"number"`
	Hash   common.Hash    `json:"hash"`
}

type simpleTransaction struct {
	Hash             common.Hash    `json:"hash"`
	TransactionIndex hexutil.Uint64 `json:"transactionIndex"`
}

func (c *client) getBlockByHash(ctx context.Context, arg common.Hash, fullTx bool) (*simpleBlock, error) {
	var r *simpleBlock
	err := c.RPC.CallContext(ctx, &r, "eth_getBlockByHash", arg, fullTx)
	return r, err
}

func (c *client) getBlockByNumber(ctx context.Context, arg uint64, fullTx bool) (*simpleBlock, error) {
	var r *simpleBlock
	err := c.RPC.CallContext(ctx, &r, "eth_getBlockByNumber", hexutil.Uint64(arg), fullTx)
	return r, err
}

func (c *client) getTransactionByBlockHashAndIndex(ctx context.Context, block common.Hash, index uint64) (*simpleTransaction, error) {
	var r *simpleTransaction
	err := c.RPC.CallContext(ctx, &r, "eth_getTransactionByBlockHashAndIndex", block, hexutil.Uint64(index))
	return r, err
}

func (c *client) getTransactionByBlockNumberAndIndex(ctx context.Context, block uint64, index uint64) (*simpleTransaction, error) {
	var r *simpleTransaction
	err := c.RPC.CallContext(ctx, &r, "eth_getTransactionByBlockNumberAndIndex", hexutil.Uint64(block), hexutil.Uint64(index))
	return r, err
}

func (c *client) getBlockTransactionCountByHash(ctx context.Context, block common.Hash) (uint64, error) {
	var r hexutil.Uint64
	err := c.RPC.CallContext(ctx, &r, "eth_getBlockTransactionCountByHash", block)
	return uint64(r), err
}

func (c *client) getBlockTransactionCountByNumber(ctx context.Context, block uint64) (uint64, error) {
	var r hexutil.Uint64
	err := c.RPC.CallContext(ctx, &r, "eth_getBlockTransactionCountByNumber", hexutil.Uint64(block))
	return uint64(r), err
}

func (c *client) getBlockReceipts(ctx context.Context, arg any) ([]*types.Receipt, error) {
	var result []*types.Receipt
	err := c.RPC.CallContext(ctx, &result, "eth_getBlockReceipts", arg)
	return result, err
}

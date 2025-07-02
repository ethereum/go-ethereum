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
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
		// If not found in embedded FS, try to load it from disk
		if !os.IsNotExist(err) {
			return err
		}
		file, err = os.OpenFile(s.cfg.historyTestFile, os.O_RDONLY, 0666)
		if err != nil {
			return fmt.Errorf("can't open historyTestFile: %v", err)
		}
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

func (s *historyTestSuite) allTests() []workloadTest {
	return []workloadTest{
		newWorkLoadTest("History/getBlockByHash", s.testGetBlockByHash),
		newWorkLoadTest("History/getBlockByNumber", s.testGetBlockByNumber),
		newWorkLoadTest("History/getBlockReceiptsByHash", s.testGetBlockReceiptsByHash),
		newWorkLoadTest("History/getBlockReceiptsByNumber", s.testGetBlockReceiptsByNumber),
		newWorkLoadTest("History/getBlockTransactionCountByHash", s.testGetBlockTransactionCountByHash),
		newWorkLoadTest("History/getBlockTransactionCountByNumber", s.testGetBlockTransactionCountByNumber),
		newWorkLoadTest("History/getTransactionByBlockHashAndIndex", s.testGetTransactionByBlockHashAndIndex),
		newWorkLoadTest("History/getTransactionByBlockNumberAndIndex", s.testGetTransactionByBlockNumberAndIndex),
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

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
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/urfave/cli/v2"
)

// proofTest is the content of a state-proof test.
type proofTest struct {
	BlockNumbers []uint64           `json:"blockNumbers"`
	Addresses    [][]common.Address `json:"addresses"`
	StorageKeys  [][][]string       `json:"storageKeys"`
	Results      [][]common.Hash    `json:"results"`
}

type proofTestSuite struct {
	cfg        testConfig
	tests      proofTest
	invalidDir string
}

func newProofTestSuite(cfg testConfig, ctx *cli.Context) *proofTestSuite {
	s := &proofTestSuite{
		cfg:        cfg,
		invalidDir: ctx.String(proofTestInvalidOutputFlag.Name),
	}
	if err := s.loadTests(); err != nil {
		exit(err)
	}
	return s
}

func (s *proofTestSuite) loadTests() error {
	file, err := s.cfg.fsys.Open(s.cfg.proofTestFile)
	if err != nil {
		// If not found in embedded FS, try to load it from disk
		if !os.IsNotExist(err) {
			return err
		}
		file, err = os.OpenFile(s.cfg.proofTestFile, os.O_RDONLY, 0666)
		if err != nil {
			return fmt.Errorf("can't open proofTestFile: %v", err)
		}
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(&s.tests); err != nil {
		return fmt.Errorf("invalid JSON in %s: %v", s.cfg.proofTestFile, err)
	}
	if len(s.tests.BlockNumbers) == 0 {
		return fmt.Errorf("proofTestFile %s has no test data", s.cfg.proofTestFile)
	}
	return nil
}

func (s *proofTestSuite) allTests() []workloadTest {
	return []workloadTest{
		newArchiveWorkloadTest("Proof/GetProof", s.getProof),
	}
}

func (s *proofTestSuite) getProof(t *utesting.T) {
	ctx := context.Background()
	for i, blockNumber := range s.tests.BlockNumbers {
		for j := 0; j < len(s.tests.Addresses[i]); j++ {
			res, err := s.cfg.client.Geth.GetProof(ctx, s.tests.Addresses[i][j], s.tests.StorageKeys[i][j], big.NewInt(int64(blockNumber)))
			if err != nil {
				t.Errorf("State proving fails, blockNumber: %d, address: %x, keys: %v, err: %v\n", blockNumber, s.tests.Addresses[i][j], strings.Join(s.tests.StorageKeys[i][j], " "), err)
				continue
			}
			blob, err := json.Marshal(res)
			if err != nil {
				t.Fatalf("State proving fails: error %v", err)
				continue
			}
			if crypto.Keccak256Hash(blob) != s.tests.Results[i][j] {
				t.Errorf("State proof mismatch, %d, number: %d, address: %x, keys: %v: invalid result", i, blockNumber, s.tests.Addresses[i][j], strings.Join(s.tests.StorageKeys[i][j], " "))
			}
		}
	}
}

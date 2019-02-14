// Copyright 2015 The go-ethereum Authors
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

package core

import (
	"runtime"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
)

const height = 8

// Tests that simple header verification works, for both good and bad blocks.
func TestHeaderVerification(t *testing.T) {
	db, blocks, _, _ := setupDatabaseAndBlocks(t)

	// Run the header checker for blocks one-by-one, checking for both valid and invalid nonces.
	chain, _ := NewBlockChain(db, nil, params.TestChainConfig, ethash.NewFaker(), vm.Config{}, nil)
	defer chain.Stop()

	for i, block := range blocks {
		for j, valid := range []bool{true, false} {
			header := block.Header()
			var results <-chan error

			var engine *ethash.Ethash
			if valid {
				engine = ethash.NewFaker()
			} else {
				engine = ethash.NewFakeFailer(header.Number.Uint64())
			}
			_, results = engine.VerifyHeaders(chain, []*types.Header{header}, []bool{true})

			// Wait for the verification result
			select {
			case result := <-results:
				if (result == nil) != valid {
					t.Errorf("test %d.%d: validity mismatch: have %v, want %v", i, j, result, valid)
				}
			case <-time.After(time.Second):
				t.Fatalf("test %d.%d: verification timeout", i, j)
			}
			// Make sure no more data is returned
			select {
			case result := <-results:
				t.Fatalf("test %d.%d: unexpected result returned: %v", i, j, result)
			case <-time.After(25 * time.Millisecond):
			}
		}
		chain.InsertChain([]*types.Block{block})
	}
}

func TestConcurrentHeaderVerification(t *testing.T) {
	tests := []struct {
		name    string
		threads int
		valid   bool
	}{
		{
			"2ThreadsSucceeds",
			2,
			true,
		},
		{
			"2ThreadsFails",
			2,
			false,
		},
		{
			"8ThreadsSucceeds",
			8,
			true,
		},
		{
			"8ThreadsFails",
			8,
			false,
		},
		{
			"32ThreadsSucceeds",
			32,
			true,
		},
		{
			"32ThreadsFails",
			32,
			false,
		},
	}

	for _, tc := range tests {
		// Tests cannot be run in parallel due to modifying runtime.GOMAXPROCS.
		t.Run(tc.name, func(t *testing.T) {
			db, _, headers, seals := setupDatabaseAndBlocks(t)

			// Set the number of threads to verify on
			old := runtime.GOMAXPROCS(tc.threads)
			defer runtime.GOMAXPROCS(old)

			// Run the header checker for the entire block chain at once both for a valid and
			// also an invalid chain (enough if one arbitrary block is invalid).
			var results <-chan error

			var chain *BlockChain
			var err error
			if tc.valid {
				chain, err = NewBlockChain(db, nil, params.TestChainConfig, ethash.NewFaker(), vm.Config{}, nil)
			} else {
				chain, err = NewBlockChain(db, nil, params.TestChainConfig, ethash.NewFakeFailer(uint64(height-1)), vm.Config{}, nil)
			}
			if err != nil {
				t.Fatalf("Error creating blockchain: %v", err)
			}
			_, results = chain.engine.VerifyHeaders(chain, headers, seals)
			chain.Stop()

			// Wait for all the verification results
			checks := make(map[int]error)
			for i := range headers {
				select {
				case result := <-results:
					checks[i] = result

				case <-time.After(time.Second):
					t.Fatalf("Verification timed out after receiving %d results", i)
				}
			}
			// Check nonce check validity
			for i := range headers {
				want := tc.valid || (i < len(headers)-2) // We chose the last-but-one nonce in the chain to fail
				if (checks[i] == nil) != want {
					t.Errorf("Validity mismatch for result %d: got %v, want %v", i, checks[i], want)
				}
				if !want {
					// A few blocks after the first error may pass verification due to concurrent
					// workers. We don't care about those in this test, just that the correct block
					// errors out.
					break
				}
			}
			// Make sure no more data is returned
			select {
			case result := <-results:
				t.Fatalf("Received unexpected result: %v", result)
			case <-time.After(25 * time.Millisecond):
			}
		})
	}
}

// Tests that aborting a header validation indeed prevents further checks from being
// run, as well as checks that no left-over goroutines are leaked.
func TestConcurrentHeaderVerificationAbortion(t *testing.T) {
	tests := []struct {
		name    string
		threads int
	}{
		{
			"2Threads",
			2,
		},
		{
			"8Threads",
			8,
		},
		{
			"32Threads",
			32,
		},
	}

	for _, tc := range tests {
		// Tests cannot be run in parallel due to modifying runtime.GOMAXPROCS.
		t.Run(tc.name, func(t *testing.T) {
			db, _, headers, seals := setupDatabaseAndBlocks(t)

			// Set the number of threads to verify on
			old := runtime.GOMAXPROCS(tc.threads)
			defer runtime.GOMAXPROCS(old)

			// Start the verifications and immediately abort
			chain, _ := NewBlockChain(db, nil, params.TestChainConfig, ethash.NewFakeDelayer(time.Millisecond), vm.Config{}, nil)
			defer chain.Stop()

			abort, results := chain.engine.VerifyHeaders(chain, headers, seals)
			close(abort)

			// Deplete the results channel
			verified := 0
			for depleted := false; !depleted; {
				select {
				case result := <-results:
					if result != nil {
						t.Errorf("Header %d validation failed: %v", verified, result)
					}
					verified++
				case <-time.After(50 * time.Millisecond):
					depleted = true
				}
			}
			// Check that abortion was honored by not processing too many POWs
			if verified > 2*tc.threads {
				t.Errorf("Verification count too large: got %d, want below %d", verified, 2*tc.threads)
			}
		})
	}
}

func setupDatabaseAndBlocks(t *testing.T) (ethdb.Database, []*types.Block, []*types.Header, []bool) {
	t.Helper()

	db := ethdb.NewMemDatabase()
	gspec := &Genesis{Config: params.TestChainConfig}
	genesis := gspec.MustCommit(db)
	blocks, _ := GenerateChain(params.TestChainConfig, genesis, ethash.NewFaker(), db, height, nil)
	headers := make([]*types.Header, 0)
	seals := make([]bool, 0)
	for _, block := range blocks {
		headers = append(headers, block.Header())
		seals = append(seals, true)
	}
	return db, blocks, headers, seals
}

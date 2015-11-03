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
	"math/big"
	"runtime"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/pow"
)

// failPow is a non-validating proof of work implementation, that returns true
// from Verify for all but one block.
type failPow struct {
	failing uint64
}

func (pow failPow) Search(pow.Block, <-chan struct{}, int) (uint64, []byte) {
	return 0, nil
}
func (pow failPow) Verify(block pow.Block) bool { return block.NumberU64() != pow.failing }
func (pow failPow) GetHashrate() int64          { return 0 }
func (pow failPow) Turbo(bool)                  {}

// delayedPow is a non-validating proof of work implementation, that returns true
// from Verify for all blocks, but delays them the configured amount of time.
type delayedPow struct {
	delay time.Duration
}

func (pow delayedPow) Search(pow.Block, <-chan struct{}, int) (uint64, []byte) {
	return 0, nil
}
func (pow delayedPow) Verify(block pow.Block) bool { time.Sleep(pow.delay); return true }
func (pow delayedPow) GetHashrate() int64          { return 0 }
func (pow delayedPow) Turbo(bool)                  {}

// Tests that simple POW verification works, for both good and bad blocks.
func TestPowVerification(t *testing.T) {
	// Create a simple chain to verify
	var (
		testdb, _ = ethdb.NewMemDatabase()
		genesis   = GenesisBlockForTesting(testdb, common.Address{}, new(big.Int))
		blocks, _ = GenerateChain(genesis, testdb, 8, nil)
	)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	// Run the POW checker for blocks one-by-one, checking for both valid and invalid nonces
	for i := 0; i < len(blocks); i++ {
		for j, full := range []bool{true, false} {
			for k, valid := range []bool{true, false} {
				var results <-chan nonceCheckResult

				switch {
				case full && valid:
					_, results = verifyNoncesFromBlocks(FakePow{}, []*types.Block{blocks[i]})
				case full && !valid:
					_, results = verifyNoncesFromBlocks(failPow{blocks[i].NumberU64()}, []*types.Block{blocks[i]})
				case !full && valid:
					_, results = verifyNoncesFromHeaders(FakePow{}, []*types.Header{headers[i]})
				case !full && !valid:
					_, results = verifyNoncesFromHeaders(failPow{headers[i].Number.Uint64()}, []*types.Header{headers[i]})
				}
				// Wait for the verification result
				select {
				case result := <-results:
					if result.index != 0 {
						t.Errorf("test %d.%d.%d: invalid index: have %d, want 0", i, j, k, result.index)
					}
					if result.valid != valid {
						t.Errorf("test %d.%d.%d: validity mismatch: have %v, want %v", i, j, k, result.valid, valid)
					}
				case <-time.After(time.Second):
					t.Fatalf("test %d.%d.%d: verification timeout", i, j, k)
				}
				// Make sure no more data is returned
				select {
				case result := <-results:
					t.Fatalf("test %d.%d.%d: unexpected result returned: %v", i, j, k, result)
				case <-time.After(25 * time.Millisecond):
				}
			}
		}
	}
}

// Tests that concurrent POW verification works, for both good and bad blocks.
func TestPowConcurrentVerification2(t *testing.T)  { testPowConcurrentVerification(t, 2) }
func TestPowConcurrentVerification8(t *testing.T)  { testPowConcurrentVerification(t, 8) }
func TestPowConcurrentVerification32(t *testing.T) { testPowConcurrentVerification(t, 32) }

func testPowConcurrentVerification(t *testing.T, threads int) {
	// Create a simple chain to verify
	var (
		testdb, _ = ethdb.NewMemDatabase()
		genesis   = GenesisBlockForTesting(testdb, common.Address{}, new(big.Int))
		blocks, _ = GenerateChain(genesis, testdb, 8, nil)
	)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	// Set the number of threads to verify on
	old := runtime.GOMAXPROCS(threads)
	defer runtime.GOMAXPROCS(old)

	// Run the POW checker for the entire block chain at once both for a valid and
	// also an invalid chain (enough if one is invalid, last but one (arbitrary)).
	for i, full := range []bool{true, false} {
		for j, valid := range []bool{true, false} {
			var results <-chan nonceCheckResult

			switch {
			case full && valid:
				_, results = verifyNoncesFromBlocks(FakePow{}, blocks)
			case full && !valid:
				_, results = verifyNoncesFromBlocks(failPow{uint64(len(blocks) - 1)}, blocks)
			case !full && valid:
				_, results = verifyNoncesFromHeaders(FakePow{}, headers)
			case !full && !valid:
				_, results = verifyNoncesFromHeaders(failPow{uint64(len(headers) - 1)}, headers)
			}
			// Wait for all the verification results
			checks := make(map[int]bool)
			for k := 0; k < len(blocks); k++ {
				select {
				case result := <-results:
					if _, ok := checks[result.index]; ok {
						t.Fatalf("test %d.%d.%d: duplicate results for %d", i, j, k, result.index)
					}
					if result.index < 0 || result.index >= len(blocks) {
						t.Fatalf("test %d.%d.%d: result %d out of bounds [%d, %d]", i, j, k, result.index, 0, len(blocks)-1)
					}
					checks[result.index] = result.valid

				case <-time.After(time.Second):
					t.Fatalf("test %d.%d.%d: verification timeout", i, j, k)
				}
			}
			// Check nonce check validity
			for k := 0; k < len(blocks); k++ {
				want := valid || (k != len(blocks)-2) // We chose the last but one nonce in the chain to fail
				if checks[k] != want {
					t.Errorf("test %d.%d.%d: validity mismatch: have %v, want %v", i, j, k, checks[k], want)
				}
			}
			// Make sure no more data is returned
			select {
			case result := <-results:
				t.Fatalf("test %d.%d: unexpected result returned: %v", i, j, result)
			case <-time.After(25 * time.Millisecond):
			}
		}
	}
}

// Tests that aborting a POW validation indeed prevents further checks from being
// run, as well as checks that no left-over goroutines are leaked.
func TestPowConcurrentAbortion2(t *testing.T)  { testPowConcurrentAbortion(t, 2) }
func TestPowConcurrentAbortion8(t *testing.T)  { testPowConcurrentAbortion(t, 8) }
func TestPowConcurrentAbortion32(t *testing.T) { testPowConcurrentAbortion(t, 32) }

func testPowConcurrentAbortion(t *testing.T, threads int) {
	// Create a simple chain to verify
	var (
		testdb, _ = ethdb.NewMemDatabase()
		genesis   = GenesisBlockForTesting(testdb, common.Address{}, new(big.Int))
		blocks, _ = GenerateChain(genesis, testdb, 1024, nil)
	)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	// Set the number of threads to verify on
	old := runtime.GOMAXPROCS(threads)
	defer runtime.GOMAXPROCS(old)

	// Run the POW checker for the entire block chain at once
	for i, full := range []bool{true, false} {
		var abort chan<- struct{}
		var results <-chan nonceCheckResult

		// Start the verifications and immediately abort
		if full {
			abort, results = verifyNoncesFromBlocks(delayedPow{time.Millisecond}, blocks)
		} else {
			abort, results = verifyNoncesFromHeaders(delayedPow{time.Millisecond}, headers)
		}
		close(abort)

		// Deplete the results channel
		verified := make(map[int]struct{})
		for depleted := false; !depleted; {
			select {
			case result := <-results:
				verified[result.index] = struct{}{}
			case <-time.After(50 * time.Millisecond):
				depleted = true
			}
		}
		// Check that abortion was honored by not processing too many POWs
		if len(verified) > 2*threads {
			t.Errorf("test %d: verification count too large: have %d, want below %d", i, len(verified), 2*threads)
		}
		// Check that there are no gaps in the results
		for j := 0; j < len(verified); j++ {
			if _, ok := verified[j]; !ok {
				t.Errorf("test %d.%d: gap found in verification results", i, j)
			}
		}
	}
}

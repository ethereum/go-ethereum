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

	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// Tests that simple header verification works, for both good and bad blocks.
func TestHeaderVerification(t *testing.T) {
	// Create a simple chain to verify
	var (
		testdb    = rawdb.NewMemoryDatabase()
		gspec     = &Genesis{Config: params.TestChainConfig}
		genesis   = gspec.MustCommit(testdb)
		blocks, _ = GenerateChain(params.TestChainConfig, genesis, ethash.NewFaker(), testdb, 8, nil)
	)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	// Run the header checker for blocks one-by-one, checking for both valid and invalid nonces
	chain, _ := NewBlockChain(testdb, nil, params.TestChainConfig, ethash.NewFaker(), vm.Config{}, nil, nil)
	defer chain.Stop()

	for i := 0; i < len(blocks); i++ {
		for j, valid := range []bool{true, false} {
			var results <-chan error

			if valid {
				engine := ethash.NewFaker()
				_, results = engine.VerifyHeaders(chain, []*types.Header{headers[i]}, []bool{true})
			} else {
				engine := ethash.NewFakeFailer(headers[i].Number.Uint64())
				_, results = engine.VerifyHeaders(chain, []*types.Header{headers[i]}, []bool{true})
			}
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
		chain.InsertChain(blocks[i : i+1])
	}
}

func TestHeaderVerificationEIP1559(t *testing.T) {
	// Create a simple chain to verify
	var (
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		key3, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = crypto.PubkeyToAddress(key2.PublicKey)
		addr3   = crypto.PubkeyToAddress(key3.PublicKey)
		testdb  = rawdb.NewMemoryDatabase()
		gspec   = &Genesis{
			Config:  params.EIP1559ChainConfig,
			Alloc:   GenesisAlloc{addr1: {Balance: big.NewInt(1000000)}, addr2: {Balance: new(big.Int).SetUint64((params.EIP1559InitialBaseFee * params.TxGas) + 1000)}},
			BaseFee: new(big.Int).SetUint64(params.EIP1559InitialBaseFee)}
		genesis   = gspec.MustCommit(testdb)
		signer    = types.HomesteadSigner{}
		blocks, _ = GenerateChain(params.EIP1559ChainConfig, genesis, ethash.NewFaker(), testdb, 5, func(i int, gen *BlockGen) {
			switch i {
			case 0:
				// In block 1, addr1 sends addr2 some ether.
				tx, _ := types.SignTx(types.NewTransaction(gen.TxNonce(addr1), addr2, big.NewInt(10000), params.TxGas, new(big.Int), nil, nil, nil), signer, key1)
				gen.AddTx(tx)
			case 1:
				// In block 2, addr1 sends some more ether to addr2.
				// addr2 attempts to pass it on to addr3 using a EIP1559 transaction
				tx1, _ := types.SignTx(types.NewTransaction(gen.TxNonce(addr1), addr2, big.NewInt(1000), params.TxGas, new(big.Int), nil, nil, nil), signer, key1)
				tx2, _ := types.SignTx(types.NewTransaction(gen.TxNonce(addr2), addr3, big.NewInt(1000), params.TxGas, nil, nil, new(big.Int), new(big.Int).SetUint64(params.EIP1559InitialBaseFee)), signer, key2)
				gen.AddTx(tx1)
				gen.AddTx(tx2)
			case 2:
				// Block 3 is empty but was mined by addr3.
				gen.SetCoinbase(addr3)
				gen.SetExtra([]byte("yeehaw"))
			case 3:
				// Block 4 includes blocks 2 and 3 as uncle headers (with modified extra data).
				b2 := gen.PrevBlock(1).Header()
				b2.Extra = []byte("foo")
				gen.AddUncle(b2)
				b3 := gen.PrevBlock(2).Header()
				b3.Extra = []byte("foo")
				gen.AddUncle(b3)
			}
		})
	)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	// Run the header checker for blocks one-by-one, checking for both valid and invalid nonces
	chain, _ := NewBlockChain(testdb, nil, params.EIP1559ChainConfig, ethash.NewFaker(), vm.Config{}, nil, nil)
	defer chain.Stop()

	for i := 0; i < len(blocks); i++ {
		for j, valid := range []bool{true, false} {
			var results <-chan error

			if valid {
				engine := ethash.NewFaker()
				_, results = engine.VerifyHeaders(chain, []*types.Header{headers[i]}, []bool{true})
			} else {
				engine := ethash.NewFakeFailer(headers[i].Number.Uint64())
				_, results = engine.VerifyHeaders(chain, []*types.Header{headers[i]}, []bool{true})
			}
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
		chain.InsertChain(blocks[i : i+1])
	}
}

func TestHeaderVerificationEIP1559Finalized(t *testing.T) {
	// Create a simple chain to verify
	var (
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		key3, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = crypto.PubkeyToAddress(key2.PublicKey)
		addr3   = crypto.PubkeyToAddress(key3.PublicKey)
		testdb  = rawdb.NewMemoryDatabase()
		gspec   = &Genesis{
			Config: params.EIP1559FinalizedChainConfig,
			Alloc: GenesisAlloc{addr1: {Balance: new(big.Int).SetUint64((params.EIP1559InitialBaseFee * params.TxGas * 2) + 11000)},
				addr2: {Balance: new(big.Int).SetUint64((params.EIP1559InitialBaseFee * params.TxGas) + 1000)}},
			BaseFee: new(big.Int).SetUint64(params.EIP1559InitialBaseFee)}
		genesis   = gspec.MustCommit(testdb)
		signer    = types.HomesteadSigner{}
		blocks, _ = GenerateChain(params.EIP1559FinalizedChainConfig, genesis, ethash.NewFaker(), testdb, 5, func(i int, gen *BlockGen) {
			switch i {
			case 0:
				// In block 1, addr1 sends addr2 some ether.
				tx, _ := types.SignTx(types.NewTransaction(gen.TxNonce(addr1), addr2, big.NewInt(10000), params.TxGas, nil, nil, new(big.Int), new(big.Int).SetUint64(params.EIP1559InitialBaseFee)), signer, key1)
				gen.AddTx(tx)
			case 1:
				// In block 2, addr1 sends some more ether to addr2.
				// addr2 attempts to pass it on to addr3 using a EIP1559 transaction
				tx1, _ := types.SignTx(types.NewTransaction(gen.TxNonce(addr1), addr2, big.NewInt(1000), params.TxGas, nil, nil, new(big.Int), new(big.Int).SetUint64(params.EIP1559InitialBaseFee)), signer, key1)
				tx2, _ := types.SignTx(types.NewTransaction(gen.TxNonce(addr2), addr3, big.NewInt(1000), params.TxGas, nil, nil, new(big.Int), new(big.Int).SetUint64(params.EIP1559InitialBaseFee)), signer, key2)
				gen.AddTx(tx1)
				gen.AddTx(tx2)
			case 2:
				// Block 3 is empty but was mined by addr3.
				gen.SetCoinbase(addr3)
				gen.SetExtra([]byte("yeehaw"))
			case 3:
				// Block 4 includes blocks 2 and 3 as uncle headers (with modified extra data).
				b2 := gen.PrevBlock(1).Header()
				b2.Extra = []byte("foo")
				gen.AddUncle(b2)
				b3 := gen.PrevBlock(2).Header()
				b3.Extra = []byte("foo")
				gen.AddUncle(b3)
			}
		})
	)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	// Run the header checker for blocks one-by-one, checking for both valid and invalid nonces
	chain, _ := NewBlockChain(testdb, nil, params.EIP1559FinalizedChainConfig, ethash.NewFaker(), vm.Config{}, nil, nil)
	defer chain.Stop()

	for i := 0; i < len(blocks); i++ {
		for j, valid := range []bool{true, false} {
			var results <-chan error

			if valid {
				engine := ethash.NewFaker()
				_, results = engine.VerifyHeaders(chain, []*types.Header{headers[i]}, []bool{true})
			} else {
				engine := ethash.NewFakeFailer(headers[i].Number.Uint64())
				_, results = engine.VerifyHeaders(chain, []*types.Header{headers[i]}, []bool{true})
			}
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
		chain.InsertChain(blocks[i : i+1])
	}
}

// Tests that concurrent header verification works, for both good and bad blocks.
func TestHeaderConcurrentVerification2(t *testing.T)  { testHeaderConcurrentVerification(t, 2) }
func TestHeaderConcurrentVerification8(t *testing.T)  { testHeaderConcurrentVerification(t, 8) }
func TestHeaderConcurrentVerification32(t *testing.T) { testHeaderConcurrentVerification(t, 32) }

func testHeaderConcurrentVerification(t *testing.T, threads int) {
	// Create a simple chain to verify
	var (
		testdb    = rawdb.NewMemoryDatabase()
		gspec     = &Genesis{Config: params.TestChainConfig}
		genesis   = gspec.MustCommit(testdb)
		blocks, _ = GenerateChain(params.TestChainConfig, genesis, ethash.NewFaker(), testdb, 8, nil)
	)
	headers := make([]*types.Header, len(blocks))
	seals := make([]bool, len(blocks))

	for i, block := range blocks {
		headers[i] = block.Header()
		seals[i] = true
	}
	// Set the number of threads to verify on
	old := runtime.GOMAXPROCS(threads)
	defer runtime.GOMAXPROCS(old)

	// Run the header checker for the entire block chain at once both for a valid and
	// also an invalid chain (enough if one arbitrary block is invalid).
	for i, valid := range []bool{true, false} {
		var results <-chan error

		if valid {
			chain, _ := NewBlockChain(testdb, nil, params.TestChainConfig, ethash.NewFaker(), vm.Config{}, nil, nil)
			_, results = chain.engine.VerifyHeaders(chain, headers, seals)
			chain.Stop()
		} else {
			chain, _ := NewBlockChain(testdb, nil, params.TestChainConfig, ethash.NewFakeFailer(uint64(len(headers)-1)), vm.Config{}, nil, nil)
			_, results = chain.engine.VerifyHeaders(chain, headers, seals)
			chain.Stop()
		}
		// Wait for all the verification results
		checks := make(map[int]error)
		for j := 0; j < len(blocks); j++ {
			select {
			case result := <-results:
				checks[j] = result

			case <-time.After(time.Second):
				t.Fatalf("test %d.%d: verification timeout", i, j)
			}
		}
		// Check nonce check validity
		for j := 0; j < len(blocks); j++ {
			want := valid || (j < len(blocks)-2) // We chose the last-but-one nonce in the chain to fail
			if (checks[j] == nil) != want {
				t.Errorf("test %d.%d: validity mismatch: have %v, want %v", i, j, checks[j], want)
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
			t.Fatalf("test %d: unexpected result returned: %v", i, result)
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func TestHeaderConcurrentVerificationEIP15592(t *testing.T) {
	testHeaderConcurrentVerificationEIP1559(t, 2)
}
func TestHeaderConcurrentVerificationEIP15598(t *testing.T) {
	testHeaderConcurrentVerificationEIP1559(t, 8)
}
func TestHeaderConcurrentVerificationEIP155932(t *testing.T) {
	testHeaderConcurrentVerificationEIP1559(t, 32)
}

func testHeaderConcurrentVerificationEIP1559(t *testing.T, threads int) {
	// Create a simple chain to verify
	var (
		testdb    = rawdb.NewMemoryDatabase()
		gspec     = &Genesis{Config: params.EIP1559ChainConfig, BaseFee: new(big.Int)}
		genesis   = gspec.MustCommit(testdb)
		blocks, _ = GenerateChain(params.EIP1559ChainConfig, genesis, ethash.NewFaker(), testdb, 8, nil)
	)
	headers := make([]*types.Header, len(blocks))
	seals := make([]bool, len(blocks))

	for i, block := range blocks {
		headers[i] = block.Header()
		seals[i] = true
	}
	// Set the number of threads to verify on
	old := runtime.GOMAXPROCS(threads)
	defer runtime.GOMAXPROCS(old)

	// Run the header checker for the entire block chain at once both for a valid and
	// also an invalid chain (enough if one arbitrary block is invalid).
	for i, valid := range []bool{true, false} {
		var results <-chan error

		if valid {
			chain, _ := NewBlockChain(testdb, nil, params.EIP1559ChainConfig, ethash.NewFaker(), vm.Config{}, nil, nil)
			_, results = chain.engine.VerifyHeaders(chain, headers, seals)
			chain.Stop()
		} else {
			chain, _ := NewBlockChain(testdb, nil, params.EIP1559ChainConfig, ethash.NewFakeFailer(uint64(len(headers)-1)), vm.Config{}, nil, nil)
			_, results = chain.engine.VerifyHeaders(chain, headers, seals)
			chain.Stop()
		}
		// Wait for all the verification results
		checks := make(map[int]error)
		for j := 0; j < len(blocks); j++ {
			select {
			case result := <-results:
				checks[j] = result

			case <-time.After(time.Second):
				t.Fatalf("test %d.%d: verification timeout", i, j)
			}
		}
		// Check nonce check validity
		for j := 0; j < len(blocks); j++ {
			want := valid || (j < len(blocks)-2) // We chose the last-but-one nonce in the chain to fail
			if (checks[j] == nil) != want {
				t.Errorf("test %d.%d: validity mismatch: have %v, want %v", i, j, checks[j], want)
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
			t.Fatalf("test %d: unexpected result returned: %v", i, result)
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func TestHeaderConcurrentVerificationEIP1559Finalized2(t *testing.T) {
	testHeaderConcurrentVerificationEIP1559Finalized(t, 2)
}
func TestHeaderConcurrentVerificationEIP1559Finalized8(t *testing.T) {
	testHeaderConcurrentVerificationEIP1559Finalized(t, 8)
}
func TestHeaderConcurrentVerificationEIP1559Finalized32(t *testing.T) {
	testHeaderConcurrentVerificationEIP1559Finalized(t, 32)
}

func testHeaderConcurrentVerificationEIP1559Finalized(t *testing.T, threads int) {
	// Create a simple chain to verify
	var (
		testdb    = rawdb.NewMemoryDatabase()
		gspec     = &Genesis{Config: params.EIP1559FinalizedChainConfig, BaseFee: new(big.Int)}
		genesis   = gspec.MustCommit(testdb)
		blocks, _ = GenerateChain(params.EIP1559FinalizedChainConfig, genesis, ethash.NewFaker(), testdb, 8, nil)
	)
	headers := make([]*types.Header, len(blocks))
	seals := make([]bool, len(blocks))

	for i, block := range blocks {
		headers[i] = block.Header()
		seals[i] = true
	}
	// Set the number of threads to verify on
	old := runtime.GOMAXPROCS(threads)
	defer runtime.GOMAXPROCS(old)

	// Run the header checker for the entire block chain at once both for a valid and
	// also an invalid chain (enough if one arbitrary block is invalid).
	for i, valid := range []bool{true, false} {
		var results <-chan error

		if valid {
			chain, _ := NewBlockChain(testdb, nil, params.EIP1559FinalizedChainConfig, ethash.NewFaker(), vm.Config{}, nil, nil)
			_, results = chain.engine.VerifyHeaders(chain, headers, seals)
			chain.Stop()
		} else {
			chain, _ := NewBlockChain(testdb, nil, params.EIP1559FinalizedChainConfig, ethash.NewFakeFailer(uint64(len(headers)-1)), vm.Config{}, nil, nil)
			_, results = chain.engine.VerifyHeaders(chain, headers, seals)
			chain.Stop()
		}
		// Wait for all the verification results
		checks := make(map[int]error)
		for j := 0; j < len(blocks); j++ {
			select {
			case result := <-results:
				checks[j] = result

			case <-time.After(time.Second):
				t.Fatalf("test %d.%d: verification timeout", i, j)
			}
		}
		// Check nonce check validity
		for j := 0; j < len(blocks); j++ {
			want := valid || (j < len(blocks)-2) // We chose the last-but-one nonce in the chain to fail
			if (checks[j] == nil) != want {
				t.Errorf("test %d.%d: validity mismatch: have %v, want %v", i, j, checks[j], want)
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
			t.Fatalf("test %d: unexpected result returned: %v", i, result)
		case <-time.After(25 * time.Millisecond):
		}
	}
}

// Tests that aborting a header validation indeed prevents further checks from being
// run, as well as checks that no left-over goroutines are leaked.
func TestHeaderConcurrentAbortion2(t *testing.T)  { testHeaderConcurrentAbortion(t, 2) }
func TestHeaderConcurrentAbortion8(t *testing.T)  { testHeaderConcurrentAbortion(t, 8) }
func TestHeaderConcurrentAbortion32(t *testing.T) { testHeaderConcurrentAbortion(t, 32) }

func testHeaderConcurrentAbortion(t *testing.T, threads int) {
	// Create a simple chain to verify
	var (
		testdb    = rawdb.NewMemoryDatabase()
		gspec     = &Genesis{Config: params.TestChainConfig}
		genesis   = gspec.MustCommit(testdb)
		blocks, _ = GenerateChain(params.TestChainConfig, genesis, ethash.NewFaker(), testdb, 1024, nil)
	)
	headers := make([]*types.Header, len(blocks))
	seals := make([]bool, len(blocks))

	for i, block := range blocks {
		headers[i] = block.Header()
		seals[i] = true
	}
	// Set the number of threads to verify on
	old := runtime.GOMAXPROCS(threads)
	defer runtime.GOMAXPROCS(old)

	// Start the verifications and immediately abort
	chain, _ := NewBlockChain(testdb, nil, params.TestChainConfig, ethash.NewFakeDelayer(time.Millisecond), vm.Config{}, nil, nil)
	defer chain.Stop()

	abort, results := chain.engine.VerifyHeaders(chain, headers, seals)
	close(abort)

	// Deplete the results channel
	verified := 0
	for depleted := false; !depleted; {
		select {
		case result := <-results:
			if result != nil {
				t.Errorf("header %d: validation failed: %v", verified, result)
			}
			verified++
		case <-time.After(50 * time.Millisecond):
			depleted = true
		}
	}
	// Check that abortion was honored by not processing too many POWs
	if verified > 2*threads {
		t.Errorf("verification count too large: have %d, want below %d", verified, 2*threads)
	}
}

// TestCalcGasLimitAndBaseFee tests that CalcGasLimitAndBaseFee() returns the correct values
func TestCalcGasLimitAndBaseFee(t *testing.T) {
	testConditions := []struct {
		// Test inputs
		config                *params.ChainConfig
		eip1559Block          *big.Int
		eip1559FinalizedBlock *big.Int
		parentGasLimit        uint64
		parentGasUsed         uint64
		parentBaseFee         *big.Int
		parentBlockNumber     *big.Int
		// Expected results
		gasLimit uint64
		baseFee  *big.Int
	}{
		{
			// Before activation GasLimit is calculated using the legacy function and BaseFee is nil
			params.TestChainConfig,
			nil,
			nil,
			8000000,
			8000000,
			nil,
			big.NewInt(5),
			8000000,
			nil,
		}, {
			// At the EIP1559 initialization block the GasLimit is split evenly between the two pools and BaseFee is the initial value
			params.EIP1559ChainConfig,
			new(big.Int).SetUint64(params.EIP1559ForkBlockNumber),
			nil,
			8000000,
			8000000,
			big.NewInt(1100000000),
			new(big.Int).SetUint64(params.EIP1559ForkBlockNumber - 1),
			params.MaxGasEIP1559 / 2,
			new(big.Int).SetUint64(params.EIP1559InitialBaseFee),
		},
		// After initialization the GasLimit and BaseFee are set according to their functions
		// Half way between initialization and finalization we should be at a 25 : 75 legacy : eip1559 split
		{
			params.EIP1559ChainConfig,
			new(big.Int).SetUint64(params.EIP1559ForkBlockNumber),
			nil,
			8000000,
			10000000,
			new(big.Int).SetUint64(params.EIP1559InitialBaseFee),
			new(big.Int).SetUint64((params.EIP1559ForkBlockNumber + (params.EIP1559ForkFinalizedBlockNumber-params.EIP1559ForkBlockNumber)/2) - 1),
			(params.MaxGasEIP1559 * 3) / 4,
			new(big.Int).SetUint64(params.EIP1559InitialBaseFee),
		},
		{
			params.EIP1559ChainConfig,
			new(big.Int).SetUint64(params.EIP1559ForkBlockNumber),
			nil,
			8000000,
			7000000,
			new(big.Int).SetUint64(params.EIP1559InitialBaseFee),
			new(big.Int).SetUint64((params.EIP1559ForkBlockNumber + (params.EIP1559ForkFinalizedBlockNumber-params.EIP1559ForkBlockNumber)/2) - 1),
			(params.MaxGasEIP1559 * 3) / 4,
			new(big.Int).SetUint64(962500000),
		},
		{
			params.EIP1559ChainConfig,
			new(big.Int).SetUint64(params.EIP1559ForkBlockNumber),
			nil,
			8000000,
			10000000,
			big.NewInt(1100000000),
			new(big.Int).SetUint64((params.EIP1559ForkBlockNumber + (params.EIP1559ForkFinalizedBlockNumber-params.EIP1559ForkBlockNumber)/2) - 1),
			(params.MaxGasEIP1559 * 3) / 4,
			new(big.Int).SetUint64(1100000000),
		},
		{
			params.EIP1559ChainConfig,
			new(big.Int).SetUint64(params.EIP1559ForkBlockNumber),
			nil,
			8000000,
			9000000,
			big.NewInt(1100000000),
			new(big.Int).SetUint64((params.EIP1559ForkBlockNumber + (params.EIP1559ForkFinalizedBlockNumber-params.EIP1559ForkBlockNumber)/2) - 1),
			(params.MaxGasEIP1559 * 3) / 4,
			new(big.Int).SetUint64(1086250000),
		},
		// At and beyond EIP1559 finalization the GasLimit (for the EIP1559 pool) is the entire MaxGasEIP1559
		{
			params.EIP1559FinalizedChainConfig,
			new(big.Int).SetUint64(params.EIP1559ForkBlockNumber),
			new(big.Int).SetUint64(params.EIP1559ForkFinalizedBlockNumber),
			8000000,
			9000000,
			big.NewInt(1086250000),
			new(big.Int).SetUint64(params.EIP1559ForkFinalizedBlockNumber - 1),
			params.MaxGasEIP1559,
			new(big.Int).SetUint64(1072671875),
		},
		{
			params.EIP1559FinalizedChainConfig,
			new(big.Int).SetUint64(params.EIP1559ForkBlockNumber),
			new(big.Int).SetUint64(params.EIP1559ForkFinalizedBlockNumber),
			8000000,
			9000000,
			big.NewInt(1072671875),
			new(big.Int).SetUint64(params.EIP1559ForkFinalizedBlockNumber + 1),
			params.MaxGasEIP1559,
			new(big.Int).SetUint64(1059263476),
		},
		{
			params.EIP1559FinalizedChainConfig,
			new(big.Int).SetUint64(params.EIP1559ForkBlockNumber),
			new(big.Int).SetUint64(params.EIP1559ForkFinalizedBlockNumber),
			8000000,
			params.TargetGasUsed + 1000,
			big.NewInt(1059263476),
			new(big.Int).SetUint64(params.EIP1559ForkFinalizedBlockNumber + 10000),
			params.MaxGasEIP1559,
			new(big.Int).SetUint64(1059276716),
		},
		{
			params.EIP1559FinalizedChainConfig,
			new(big.Int).SetUint64(params.EIP1559ForkBlockNumber),
			new(big.Int).SetUint64(params.EIP1559ForkFinalizedBlockNumber),
			8000000,
			params.MaxGasEIP1559,
			big.NewInt(1059276716),
			new(big.Int).SetUint64(params.EIP1559ForkFinalizedBlockNumber + 10000),
			params.MaxGasEIP1559,
			new(big.Int).SetUint64(1191686305),
		},
		{
			params.EIP1559FinalizedChainConfig,
			new(big.Int).SetUint64(params.EIP1559ForkBlockNumber),
			new(big.Int).SetUint64(params.EIP1559ForkFinalizedBlockNumber),
			8000000,
			0,
			big.NewInt(1049238967),
			new(big.Int).SetUint64(params.EIP1559ForkFinalizedBlockNumber + 10000),
			params.MaxGasEIP1559,
			new(big.Int).SetUint64(918084097),
		},
	}
	for i, test := range testConditions {
		config := *test.config
		config.EIP1559Block = test.eip1559Block
		config.EIP1559FinalizedBlock = test.eip1559FinalizedBlock
		parentHeader := &types.Header{}
		parentHeader.GasLimit = test.parentGasLimit
		parentHeader.GasUsed = test.parentGasUsed
		parentHeader.BaseFee = test.parentBaseFee
		parentHeader.Number = test.parentBlockNumber
		parentBlock := types.NewBlockWithHeader(parentHeader)
		gasLimit, baseFee := CalcGasLimitAndBaseFee(&config, parentBlock, parentHeader.GasLimit, parentHeader.GasLimit)
		if gasLimit != test.gasLimit {
			t.Errorf("test %d expected GasLimit %d got %d", i+1, test.gasLimit, gasLimit)
		}
		if baseFee == nil && test.baseFee != nil {
			t.Errorf("test %d expected BaseFee %d got nil", i+1, test.baseFee)
		} else if baseFee != nil && baseFee.Cmp(test.baseFee) != 0 {
			t.Errorf("test %d expected BaseFee %d got %d", i+1, test.baseFee.Uint64(), baseFee.Uint64())
		}
	}
}

// Copyright 2019 The go-ethereum Authors
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

package simulated

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

var _ bind.ContractBackend = (Client)(nil)

var (
	testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddr   = crypto.PubkeyToAddress(testKey.PublicKey)
)

func simTestBackend(testAddr common.Address) *Backend {
	return NewBackend(
		core.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000000000)},
		},
	)
}

func newTx(sim *Backend, key *ecdsa.PrivateKey) (*types.Transaction, error) {
	client := sim.Client()

	// create a signed transaction to send
	head, _ := client.HeaderByNumber(context.Background(), nil) // Should be child's, good enough
	gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(params.GWei))
	addr := crypto.PubkeyToAddress(key.PublicKey)
	chainid, _ := client.ChainID(context.Background())
	nonce, err := client.PendingNonceAt(context.Background(), addr)
	if err != nil {
		return nil, err
	}
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainid,
		Nonce:     nonce,
		GasTipCap: big.NewInt(params.GWei),
		GasFeeCap: gasPrice,
		Gas:       21000,
		To:        &addr,
	})
	return types.SignTx(tx, types.LatestSignerForChainID(chainid), key)
}

func TestNewBackend(t *testing.T) {
	sim := NewBackend(core.GenesisAlloc{})
	defer sim.Close()

	client := sim.Client()
	num, err := client.BlockNumber(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if num != 0 {
		t.Fatalf("expected 0 got %v", num)
	}
	// Create a block
	sim.Commit()
	num, err = client.BlockNumber(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if num != 1 {
		t.Fatalf("expected 1 got %v", num)
	}
}

func TestAdjustTime(t *testing.T) {
	sim := NewBackend(core.GenesisAlloc{})
	defer sim.Close()

	client := sim.Client()
	block1, _ := client.BlockByNumber(context.Background(), nil)

	// Create a block
	if err := sim.AdjustTime(time.Minute); err != nil {
		t.Fatal(err)
	}
	block2, _ := client.BlockByNumber(context.Background(), nil)
	prevTime := block1.Time()
	newTime := block2.Time()
	if newTime-prevTime != uint64(time.Minute) {
		t.Errorf("adjusted time not equal to 60 seconds. prev: %v, new: %v", prevTime, newTime)
	}
}

func TestSendTransaction(t *testing.T) {
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	signedTx, err := newTx(sim, testKey)
	if err != nil {
		t.Errorf("could not create transaction: %v", err)
	}
	// send tx to simulated backend
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		t.Errorf("could not add tx to pending block: %v", err)
	}
	sim.Commit()
	block, err := client.BlockByNumber(ctx, big.NewInt(1))
	if err != nil {
		t.Errorf("could not get block at height 1: %v", err)
	}

	if signedTx.Hash() != block.Transactions()[0].Hash() {
		t.Errorf("did not commit sent transaction. expected hash %v got hash %v", block.Transactions()[0].Hash(), signedTx.Hash())
	}
}

// TestFork check that the chain length after a reorg is correct.
// Steps:
//  1. Save the current block which will serve as parent for the fork.
//  2. Mine n blocks with n ∈ [0, 20].
//  3. Assert that the chain length is n.
//  4. Fork by using the parent block as ancestor.
//  5. Mine n+1 blocks which should trigger a reorg.
//  6. Assert that the chain length is n+1.
//     Since Commit() was called 2n+1 times in total,
//     having a chain length of just n+1 means that a reorg occurred.
func TestFork(t *testing.T) {
	t.Parallel()
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	// 1.
	parent, _ := client.HeaderByNumber(ctx, nil)

	// 2.
	n := int(rand.Int31n(21))
	for i := 0; i < n; i++ {
		sim.Commit()
	}

	// 3.
	b, _ := client.BlockNumber(ctx)
	if b != uint64(n) {
		t.Error("wrong chain length")
	}

	// 4.
	sim.Fork(parent.Hash())

	// 5.
	for i := 0; i < n+1; i++ {
		sim.Commit()
	}

	// 6.
	b, _ = client.BlockNumber(ctx)
	if b != uint64(n+1) {
		t.Error("wrong chain length")
	}
}

// TestForkResendTx checks that re-sending a TX after a fork
// is possible and does not cause a "nonce mismatch" panic.
// Steps:
//  1. Save the current block which will serve as parent for the fork.
//  2. Send a transaction.
//  3. Check that the TX is included in block 1.
//  4. Fork by using the parent block as ancestor.
//  5. Mine a block, Re-send the transaction and mine another one.
//  6. Check that the TX is now included in block 2.
func TestForkResendTx(t *testing.T) {
	t.Parallel()
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	// 1.
	parent, _ := client.HeaderByNumber(ctx, nil)

	// 2.
	tx, err := newTx(sim, testKey)
	if err != nil {
		t.Fatalf("could not create transaction: %v", err)
	}
	client.SendTransaction(ctx, tx)
	sim.Commit()

	// 3.
	receipt, _ := client.TransactionReceipt(ctx, tx.Hash())
	if h := receipt.BlockNumber.Uint64(); h != 1 {
		t.Errorf("TX included in wrong block: %d", h)
	}

	// 4.
	if err := sim.Fork(parent.Hash()); err != nil {
		t.Errorf("forking: %v", err)
	}

	// 5.
	sim.Commit()
	if err := client.SendTransaction(ctx, tx); err != nil {
		t.Fatalf("sending transaction: %v", err)
	}
	sim.Commit()
	receipt, _ = client.TransactionReceipt(ctx, tx.Hash())
	if h := receipt.BlockNumber.Uint64(); h != 2 {
		t.Errorf("TX included in wrong block: %d", h)
	}
}

func TestCommitReturnValue(t *testing.T) {
	t.Parallel()
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	// Test if Commit returns the correct block hash
	h1 := sim.Commit()
	cur, _ := client.HeaderByNumber(ctx, nil)
	if h1 != cur.Hash() {
		t.Error("Commit did not return the hash of the last block.")
	}

	// Create a block in the original chain (containing a transaction to force different block hashes)
	head, _ := client.HeaderByNumber(ctx, nil) // Should be child's, good enough
	gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(1))
	_tx := types.NewTransaction(0, testAddr, big.NewInt(1000), params.TxGas, gasPrice, nil)
	tx, _ := types.SignTx(_tx, types.HomesteadSigner{}, testKey)
	client.SendTransaction(ctx, tx)

	h2 := sim.Commit()

	// Create another block in the original chain
	sim.Commit()

	// Fork at the first bock
	if err := sim.Fork(h1); err != nil {
		t.Errorf("forking: %v", err)
	}

	// Test if Commit returns the correct block hash after the reorg
	h2fork := sim.Commit()
	if h2 == h2fork {
		t.Error("The block in the fork and the original block are the same block!")
	}
	if header, err := client.HeaderByHash(ctx, h2fork); err != nil || header == nil {
		t.Error("Could not retrieve the just created block (side-chain)")
	}
}

// TestAdjustTimeAfterFork ensures that after a fork, AdjustTime uses the pending fork
// block's parent rather than the canonical head's parent.
func TestAdjustTimeAfterFork(t *testing.T) {
	t.Parallel()
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	sim := simTestBackend(testAddr)
	defer sim.Close()

	client := sim.Client()
	ctx := context.Background()

	sim.Commit() // h1
	h1, _ := client.HeaderByNumber(ctx, nil)

	sim.Commit() // h2
	sim.Fork(h1.Hash())
	sim.AdjustTime(1 * time.Second)
	sim.Commit()

	head, _ := client.HeaderByNumber(ctx, nil)
	if head.Number.Uint64() == 2 && head.ParentHash != h1.Hash() {
		t.Errorf("failed to build block on fork")
	}
}

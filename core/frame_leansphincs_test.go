// Copyright 2026 The go-ethereum Authors
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
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto/leansphincs"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func TestEIP8141LeanSphincs(t *testing.T) {
	config := *params.MergedTestChainConfig
	zero := uint64(0)
	config.AmsterdamTime, config.BogotaTime = &zero, &zero
	engine := beacon.New(ethash.NewFaker())
	sender := common.HexToAddress("0x000000000000000000000000000000000000aabb")
	seed := bytes.Repeat([]byte{0x42}, leansphincs.SeedSize)
	pub, err := leansphincs.KeyGen(seed)
	if err != nil {
		t.Fatal(err)
	}
	// The account approves only a leanSPHINCS entry with its full public key.
	code := program.New().Push(types.FrameTxSchemeLeanSphincs).Push(1).Push(0).Op(vm.SIGPARAM, vm.EQ).
		Push(pub).Push(0).Push(0).Op(vm.SIGPARAM, vm.EQ, vm.AND)
	code.Push(code.Size()+4).Op(vm.JUMPI, vm.INVALID, vm.JUMPDEST)
	code.Push(3).Push(0).Push(0).Op(vm.APPROVE)
	alloc := SystemContractAllocs()
	alloc[sender] = types.Account{Code: code.Bytes(), Balance: new(big.Int).Mul(big.NewInt(1), big.NewInt(params.Ether))}
	alloc[params.FrameTxExpiryVerifier] = types.Account{Code: params.FrameTxExpiryVerifierCode, Balance: big.NewInt(0)}
	gspec := &Genesis{Config: &config, Alloc: alloc}
	frameTx := &types.FrameTx{
		ChainID: uint256.MustFromBig(config.ChainID), Sender: sender,
		Frames: []types.Frame{{Mode: types.ModeVerify, Flags: types.ApproveExecutionAndPayment,
			GasLimits: types.Limits{Execution: 100_000}, Value: uint256.NewInt(0)}},
		Signatures: types.SignatureList{{Scheme: types.FrameTxSchemeLeanSphincs,
			Signer: pub, Signature: make([]byte, leansphincs.SignatureSize)}},
		Fees: types.Fees{MaxFee: uint256.NewInt(1e16), MaxPriorityFeePerGas: uint256.NewInt(2)},
	}
	sigHash := types.LatestSigner(&config).Hash(types.NewTx(frameTx))
	frameTx.Signatures[0].Signature, err = leansphincs.Sign(seed, sigHash[:])
	if err != nil {
		t.Fatal(err)
	}
	tx := types.NewTx(frameTx)
	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(_ int, b *BlockGen) {
		b.SetParentBeaconRoot(common.Hash{})
		b.AddTx(tx)
	})
	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), gspec, engine, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: %v", n, err)
	}
	state, err := chain.State()
	if err != nil {
		t.Fatal(err)
	}
	if nonce := state.GetNonce(sender); nonce != 1 {
		t.Fatalf("nonce: got %d, want 1", nonce)
	}
	receipts := chain.GetReceiptsByHash(blocks[0].Hash())
	if len(receipts) != 1 || receipts[0].Payer == nil || *receipts[0].Payer != sender || len(receipts[0].FrameReceipts) != 1 || receipts[0].FrameReceipts[0].Status != 1 {
		t.Fatalf("leanSPHINCS frame did not approve sender and payment: %+v", receipts)
	}
}

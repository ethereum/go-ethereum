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

package framepool

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto/leansphincs"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// TestLeanSphincsAdmission admits a transaction whose only signature is a
// leanSPHINCS entry, through policy, prefix classification, signature
// verification and the prefix simulation against the sender's wallet code.
func TestLeanSphincsAdmission(t *testing.T) {
	seed := bytes.Repeat([]byte{0x42}, leansphincs.SeedSize)
	pub, err := leansphincs.KeyGen(seed)
	if err != nil {
		t.Fatal(err)
	}
	// The wallet approves only a leanSPHINCS entry carrying its own key.
	code := program.New().Push(types.FrameTxSchemeLeanSphincs).Push(1).Push(0).Op(vm.SIGPARAM, vm.EQ).
		Push(pub).Push(0).Push(0).Op(vm.SIGPARAM, vm.EQ, vm.AND)
	code.Push(code.Size()+4).Op(vm.JUMPI, vm.INVALID, vm.JUMPDEST)
	code.Push(3).Push(0).Push(0).Op(vm.APPROVE)

	sender := poolAddress(t, 1)
	pool, _, _ := setupFramePool(t, 10, func(statedb *state.StateDB) {
		statedb.SetCode(sender, code.Bytes(), tracing.CodeChangeUnspecified)
	})
	inner := &types.FrameTx{
		ChainID: uint256.MustFromBig(params.AllDevChainProtocolChanges.ChainID), Sender: sender,
		Frames: []types.Frame{verifyFrame(types.ApproveExecutionAndPayment)},
		// The signature hash commits to the signature's length.
		Signatures: types.SignatureList{{Scheme: types.FrameTxSchemeLeanSphincs, Signer: pub, Signature: make([]byte, leansphincs.SignatureSize)}},
		Fees:       types.Fees{MaxPriorityFeePerGas: uint256.NewInt(2)},
	}
	maxGas, err := types.FrameTxMaxGas(inner.Frames, inner.Signatures, inner.Sender, inner.NonceKeys, inner.NonceSeq)
	if err != nil {
		t.Fatal(err)
	}
	inner.Fees.MaxFee = new(uint256.Int).Mul(uint256.NewInt(20), uint256.NewInt(maxGas))
	hash := types.LatestSigner(params.AllDevChainProtocolChanges).Hash(types.NewTx(inner))
	if inner.Signatures[0].Signature, err = leansphincs.Sign(seed, hash[:]); err != nil {
		t.Fatal(err)
	}
	tx := types.NewTx(inner)
	if err := pool.ValidateTxBasics(tx); err != nil {
		t.Fatalf("stateless validation: %v", err)
	}
	addPoolTx(t, pool, tx, nil)
	assertLive(t, pool, tx, true)

	// A signature over another hash fails before any state is read.
	inner = poolTxData(tx)
	inner.Signatures[0].Signature, _ = leansphincs.Sign(seed, bytes.Repeat([]byte{1}, 32))
	if err := pool.ValidateTxBasics(types.NewTx(inner)); !errors.Is(err, types.ErrFrameTxInvalidSignature) {
		t.Fatalf("forged signature: got %v, want %v", err, types.ErrFrameTxInvalidSignature)
	}
}

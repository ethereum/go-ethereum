// Copyright (c) 2018 Tomochain
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package randomize

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	epocNumber = int64(12)
	key, _     = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr       = crypto.PubkeyToAddress(key.PublicKey)
	byte0      = make([][32]byte, epocNumber)
	acc1Key, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc1Addr   = crypto.PubkeyToAddress(acc1Key.PublicKey)
)

func TestRandomize(t *testing.T) {
	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(100000000000000)}})
	transactOpts := bind.NewKeyedTransactor(key)
	transactOpts.GasLimit = 1000000

	randomizeAddress, randomize, err := DeployRandomize(transactOpts, contractBackend, big.NewInt(2))
	t.Log("contract address", randomizeAddress.String())
	if err != nil {
		t.Fatalf("can't deploy root registry: %v", err)
	}
	contractBackend.Commit()

	d := time.Now().Add(1000 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()
	code, _ := contractBackend.CodeAt(ctx, randomizeAddress, nil)
	t.Log("contract code", common.ToHex(code))
	f := func(key, val common.Hash) bool {
		t.Log(key.Hex(), val.Hex())
		return true
	}
	contractBackend.ForEachStorageAt(ctx, randomizeAddress, nil, f)
	s, err := randomize.SetSecret(byte0)
	if err != nil {
		t.Fatalf("can't set secret: %v", err)
	}
	t.Log("tx data", s)
	contractBackend.Commit()
}

func TestSendTxRandomizeSecretAndOpening(t *testing.T) {
	genesis := core.GenesisAlloc{acc1Addr: {Balance: big.NewInt(1000000000000)}}
	backend := backends.NewSimulatedBackend(genesis)
	backend.Commit()
	signer := types.HomesteadSigner{}
	ctx := context.Background()

	transactOpts := bind.NewKeyedTransactor(acc1Key)
	transactOpts.GasLimit = 4200000
	epocNumber := uint64(900)
	randomizeAddr, randomizeContract, err := DeployRandomize(transactOpts, backend, new(big.Int).SetInt64(0))
	if err != nil {
		t.Fatalf("Can't deploy randomize SC: %v", err)
	}
	backend.Commit()

	nonce := uint64(1)
	randomizeKeyValue := contracts.RandStringByte(32)
	tx, err := contracts.BuildTxSecretRandomize(nonce, randomizeAddr, epocNumber, randomizeKeyValue)
	if err != nil {
		t.Fatalf("Can't create tx randomize secret: %v", err)
	}
	tx, err = types.SignTx(tx, signer, acc1Key)
	if err != nil {
		t.Fatalf("Can't sign tx randomize secret: %v", err)
	}

	err = backend.SendTransaction(ctx, tx)
	if err != nil {
		t.Fatalf("Can't send tx for create randomize secret: %v", err)
	}
	backend.Commit()
	// Increment nonce.
	nonce++
	// Set opening.
	tx, err = contracts.BuildTxOpeningRandomize(nonce, randomizeAddr, randomizeKeyValue)
	if err != nil {
		t.Fatalf("Can't create tx randomize opening: %v", err)
	}
	tx, err = types.SignTx(tx, signer, acc1Key)
	if err != nil {
		t.Fatalf("Can't sign tx randomize opening: %v", err)
	}

	err = backend.SendTransaction(ctx, tx)
	if err != nil {
		t.Fatalf("Can't send tx for create randomize opening: %v", err)
	}
	backend.Commit()

	// Get randomize secret from SC.
	secretsArr, err := randomizeContract.GetSecret(acc1Addr)
	if err != nil {
		t.Fatalf("Can't get secret from SC: %v", err)
	}
	if len(secretsArr) <= 0 {
		t.Error("Empty get secrets from SC", err)
	}

	// Decrypt randomize from SC.
	secrets, err := randomizeContract.GetSecret(acc1Addr)
	if err != nil {
		t.Error("Fail get secrets from randomize", err)
	}
	opening, err := randomizeContract.GetOpening(acc1Addr)
	if err != nil {
		t.Fatalf("Can't get secret from SC: %v", err)
	}
	randomizes, err := contracts.DecryptRandomizeFromSecretsAndOpening(secrets, opening)
	t.Log("randomizes", randomizes)
	if err != nil {
		t.Error("Can't decrypt secret and opening", err)
	}
	if len(randomizes) != 901 {
		t.Error("Randomize length not match", "length", len(randomizes))
	}
}

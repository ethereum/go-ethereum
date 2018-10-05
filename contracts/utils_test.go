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

package contracts

import (
	"context"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/blocksigner"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

func TestSendTxSign(t *testing.T) {
	acc1Key, _ := crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc2Key, _ := crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	acc3Key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	acc4Key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee04aefe388d1e14474d32c45c72ce7b7a")
	acc1Addr := crypto.PubkeyToAddress(acc1Key.PublicKey)
	acc2Addr := crypto.PubkeyToAddress(acc2Key.PublicKey)
	acc3Addr := crypto.PubkeyToAddress(acc3Key.PublicKey)
	acc4Addr := crypto.PubkeyToAddress(acc4Key.PublicKey)
	accounts := []common.Address{acc2Addr, acc3Addr, acc4Addr}
	keys := []*ecdsa.PrivateKey{acc2Key, acc3Key, acc4Key}

	signer := types.HomesteadSigner{}
	genesis := core.GenesisAlloc{acc1Addr: {Balance: big.NewInt(1000000000)}}
	backend := backends.NewSimulatedBackend(genesis)
	backend.Commit()
	ctx := context.Background()

	transactOpts := bind.NewKeyedTransactor(acc1Key)
	blockSignerAddr, blockSigner, err := blocksigner.DeployBlockSigner(transactOpts, backend, big.NewInt(99))
	if err != nil {
		t.Fatalf("Can't get block signer: %v", err)
	}
	backend.Commit()

	nonces := make(map[*ecdsa.PrivateKey]int)
	oldBlocks := make(map[common.Hash]common.Address)

	signTx := func(ctx context.Context, backend *backends.SimulatedBackend, signer types.HomesteadSigner, nonces map[*ecdsa.PrivateKey]int, accKey *ecdsa.PrivateKey, blockNumber *big.Int, blockHash common.Hash) *types.Transaction {
		tx, _ := types.SignTx(CreateTxSign(blockNumber, blockHash, uint64(nonces[accKey]), blockSignerAddr), signer, accKey)
		backend.SendTransaction(ctx, tx)
		backend.Commit()
		nonces[accKey]++

		return tx
	}

	// Tx sign for signer.
	signCount := int64(0)
	blockHashes := make([]common.Hash, 10)
	for i := int64(0); i < 10; i++ {
		blockHash := randomHash()
		blockHashes[i] = blockHash
		randIndex := rand.Intn(len(keys))
		accKey := keys[randIndex]
		signTx(ctx, backend, signer, nonces, accKey, new(big.Int).SetInt64(i), blockHash)
		oldBlocks[blockHash] = accounts[randIndex]
		signCount++

		// Tx sign for validators.
		for _, key := range keys {
			if key != accKey {
				signTx(ctx, backend, signer, nonces, key, new(big.Int).SetInt64(i), blockHash)
				signCount++
			}
		}
	}

	for _, blockHash := range blockHashes {
		signers, err := blockSigner.GetSigners(blockHash)
		if err != nil {
			t.Fatalf("Can't get signers: %v", err)
		}

		if signers[0].String() != oldBlocks[blockHash].String() {
			t.Errorf("Tx sign for block signer not match %v - %v", signers[0].String(), oldBlocks[blockHash].String())
		}

		if len(signers) != len(keys) {
			t.Error("Tx sign for block validators not match")
		}
	}
}

// Generate random string.
func randomHash() common.Hash {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789"
	var b common.Hash
	for i := range b {
		rand.Seed(time.Now().UnixNano())
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return b
}

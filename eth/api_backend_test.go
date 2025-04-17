// Copyright 2025 The go-ethereum Authors
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

package eth

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/blobpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/txpool/locals"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var (
	key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	address = crypto.PubkeyToAddress(key.PublicKey)
	funds   = big.NewInt(1000_000_000_000_000)
	gspec   = &core.Genesis{
		Config: params.MergedTestChainConfig,
		Alloc: types.GenesisAlloc{
			address: {Balance: funds},
		},
		Difficulty: common.Big0,
		BaseFee:    big.NewInt(params.InitialBaseFee),
	}
	signer = types.LatestSignerForChainID(gspec.Config.ChainID)
)

func initBackend(withLocal bool) *EthAPIBackend {
	var (
		// Create a database pre-initialize with a genesis block
		db     = rawdb.NewMemoryDatabase()
		engine = beacon.New(ethash.NewFaker())
	)
	chain, _ := core.NewBlockChain(db, nil, gspec, nil, engine, vm.Config{}, nil)

	txconfig := legacypool.DefaultConfig
	txconfig.Journal = "" // Don't litter the disk with test journals

	blobPool := blobpool.New(blobpool.Config{Datadir: ""}, chain, nil)
	legacyPool := legacypool.New(txconfig, chain)
	txpool, _ := txpool.New(txconfig.PriceLimit, chain, []txpool.SubPool{legacyPool, blobPool})

	eth := &Ethereum{
		blockchain: chain,
		txPool:     txpool,
	}
	if withLocal {
		eth.localTxTracker = locals.New("", time.Minute, gspec.Config, txpool)
	}
	return &EthAPIBackend{
		eth: eth,
	}
}

func makeTx(nonce uint64, gasPrice *big.Int, amount *big.Int, key *ecdsa.PrivateKey) *types.Transaction {
	if gasPrice == nil {
		gasPrice = big.NewInt(params.GWei)
	}
	if amount == nil {
		amount = big.NewInt(1000)
	}
	tx, _ := types.SignTx(types.NewTransaction(nonce, common.Address{0x00}, amount, params.TxGas, gasPrice, nil), signer, key)
	return tx
}

type unsignedAuth struct {
	nonce uint64
	key   *ecdsa.PrivateKey
}

func pricedSetCodeTx(nonce uint64, gaslimit uint64, gasFee, tip *uint256.Int, key *ecdsa.PrivateKey, unsigned []unsignedAuth) *types.Transaction {
	var authList []types.SetCodeAuthorization
	for _, u := range unsigned {
		auth, _ := types.SignSetCode(u.key, types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(gspec.Config.ChainID),
			Address: common.Address{0x42},
			Nonce:   u.nonce,
		})
		authList = append(authList, auth)
	}
	return pricedSetCodeTxWithAuth(nonce, gaslimit, gasFee, tip, key, authList)
}

func pricedSetCodeTxWithAuth(nonce uint64, gaslimit uint64, gasFee, tip *uint256.Int, key *ecdsa.PrivateKey, authList []types.SetCodeAuthorization) *types.Transaction {
	return types.MustSignNewTx(key, signer, &types.SetCodeTx{
		ChainID:    uint256.MustFromBig(gspec.Config.ChainID),
		Nonce:      nonce,
		GasTipCap:  tip,
		GasFeeCap:  gasFee,
		Gas:        gaslimit,
		To:         common.Address{},
		Value:      uint256.NewInt(100),
		Data:       nil,
		AccessList: nil,
		AuthList:   authList,
	})
}

func TestSendTx(t *testing.T) {
	testSendTx(t, false)
	testSendTx(t, true)
}

func testSendTx(t *testing.T, withLocal bool) {
	b := initBackend(withLocal)

	txA := pricedSetCodeTx(0, 250000, uint256.NewInt(params.GWei), uint256.NewInt(params.GWei), key, []unsignedAuth{
		{
			nonce: 0,
			key:   key,
		},
	})
	b.SendTx(context.Background(), txA)

	txB := makeTx(1, nil, nil, key)
	err := b.SendTx(context.Background(), txB)

	if withLocal {
		if err != nil {
			t.Fatalf("Unexpected error sending tx: %v", err)
		}
	} else {
		if !errors.Is(err, txpool.ErrInflightTxLimitReached) {
			t.Fatalf("Unexpected error, want: %v, got: %v", txpool.ErrInflightTxLimitReached, err)
		}
	}
}

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
	"reflect"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/ethash"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/txpool"
	"github.com/XinFinOrg/XDPoSChain/core/txpool/legacypool"
	"github.com/XinFinOrg/XDPoSChain/core/txpool/locals"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/params"
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

func initBackend(t *testing.T, withLocal bool) *EthAPIBackend {
	t.Helper()

	var (
		// Create a database pre-initialize with a genesis block
		db     = rawdb.NewMemoryDatabase()
		engine = ethash.NewFaker()
	)
	chain, err := core.NewBlockChain(db, nil, gspec, engine, vm.Config{})
	if err != nil {
		t.Fatalf("failed to create blockchain: %v", err)
	}

	txconfig := legacypool.DefaultConfig
	txconfig.Journal = "" // Don't litter the disk with test journals

	legacyPool := legacypool.New(txconfig, chain)
	txpool, err := txpool.New(txconfig.PriceLimit, chain, []txpool.SubPool{legacyPool})
	if err != nil {
		// Ensure we don't leak the blockchain goroutines if txpool creation fails.
		chain.Stop()
		t.Fatalf("failed to create txpool: %v", err)
	}

	eth := &Ethereum{
		blockchain: chain,
		txPool:     txpool,
	}
	if withLocal {
		eth.localTxTracker = locals.New("", time.Minute, gspec.Config, txpool)
	}
	t.Cleanup(func() {
		if eth.localTxTracker != nil {
			if err := eth.localTxTracker.Stop(); err != nil {
				t.Errorf("failed to stop local tx tracker: %v", err)
			}
		}
		if err := txpool.Close(); err != nil {
			t.Errorf("failed to close txpool: %v", err)
		}
		chain.Stop()
	})

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
	b := initBackend(t, withLocal)

	txA := pricedSetCodeTx(0, 250000, uint256.NewInt(params.GWei), uint256.NewInt(params.GWei), key, []unsignedAuth{{nonce: 0, key: key}})
	if err := b.SendTx(context.Background(), txA); err != nil {
		t.Fatalf("Failed to submit tx: %v", err)
	}
	for {
		pending, _ := b.TxPool().ContentFrom(address)
		if len(pending) == 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

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

func TestSendTxWithLocalPermanentErrorNotTracked(t *testing.T) {
	b := initBackend(t, true)
	if b.eth.localTxTracker == nil {
		t.Fatal("expected local tx tracker to be configured")
	}
	// Force txpool min tip above tx gas price so submission fails permanently.
	if err := b.TxPool().SetGasTip(big.NewInt(params.GWei + 1)); err != nil {
		t.Fatalf("failed to set gas tip: %v", err)
	}

	tx := makeTx(0, big.NewInt(params.GWei), nil, key)
	err := b.SendTx(context.Background(), tx)
	if !errors.Is(err, txpool.ErrTxGasPriceTooLow) {
		t.Fatalf("unexpected error, want: %v, got: %v", txpool.ErrTxGasPriceTooLow, err)
	}

	tracked := reflect.ValueOf(b.eth.localTxTracker).Elem().FieldByName("all").Len()
	if tracked != 0 {
		t.Fatalf("unexpected tracked tx count: have %d, want 0", tracked)
	}
}

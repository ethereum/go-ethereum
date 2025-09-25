// Copyright 2016 The go-ethereum Authors
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

package bind_test

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/params"
)

var waitDeployedTests = map[string]struct {
	code        string
	gas         uint64
	wantAddress common.Address
	wantErr     error
}{
	"successful deploy": {
		code:        `6060604052600a8060106000396000f360606040526008565b00`,
		gas:         3000000,
		wantAddress: common.HexToAddress("0x3a220f351252089d385b29beca14e27f204c296a"),
	},
	"empty code": {
		code:        ``,
		gas:         300000,
		wantErr:     bind.ErrNoCodeAfterDeploy,
		wantAddress: common.HexToAddress("0x3a220f351252089d385b29beca14e27f204c296a"),
	},
}

func TestWaitDeployed(t *testing.T) {
	t.Parallel()
	for name, test := range waitDeployedTests {
		backend := simulated.NewBackend(
			types.GenesisAlloc{
				crypto.PubkeyToAddress(testKey.PublicKey): {Balance: big.NewInt(10000000000000000)},
			},
		)
		defer backend.Close()

		// Create the transaction
		head, _ := backend.Client().HeaderByNumber(context.Background(), nil) // Should be child's, good enough
		gasPrice := new(big.Int).Add(head.BaseFee, big.NewInt(params.GWei))

		tx := types.NewContractCreation(0, big.NewInt(0), test.gas, gasPrice, common.FromHex(test.code))
		tx, _ = types.SignTx(tx, types.LatestSignerForChainID(big.NewInt(1337)), testKey)

		// Wait for it to get mined in the background.
		var (
			err     error
			address common.Address
			mined   = make(chan struct{})
			ctx     = context.Background()
		)
		go func() {
			address, err = bind.WaitDeployed(ctx, backend.Client(), tx.Hash())
			close(mined)
		}()

		// Send and mine the transaction.
		if err := backend.Client().SendTransaction(ctx, tx); err != nil {
			t.Errorf("test %q: failed to send transaction: %v", name, err)
		}
		backend.Commit()

		select {
		case <-mined:
			if err != test.wantErr {
				t.Errorf("test %q: error mismatch: want %q, got %q", name, test.wantErr, err)
			}
			if address != test.wantAddress {
				t.Errorf("test %q: unexpected contract address %s", name, address.Hex())
			}
		case <-time.After(2 * time.Second):
			t.Errorf("test %q: timeout", name)
		}
	}
}

func TestWaitDeployedCornerCases(t *testing.T) {
	var (
		backend = simulated.NewBackend(
			types.GenesisAlloc{
				crypto.PubkeyToAddress(testKey.PublicKey): {Balance: big.NewInt(10000000000000000)},
			},
		)
		head, _     = backend.Client().HeaderByNumber(t.Context(), nil) // Should be child's, good enough
		gasPrice    = new(big.Int).Add(head.BaseFee, big.NewInt(1))
		signer      = types.LatestSigner(params.AllDevChainProtocolChanges)
		code        = common.FromHex("6060604052600a8060106000396000f360606040526008565b00")
		ctx, cancel = context.WithCancel(t.Context())
	)
	defer backend.Close()

	// 1. WaitDeploy on a transaction that does not deploy a contract, verify it
	// returns an error.
	tx := types.MustSignNewTx(testKey, signer, &types.LegacyTx{
		Nonce:    0,
		To:       &common.Address{0x01},
		Gas:      300000,
		GasPrice: gasPrice,
		Data:     code,
	})
	if err := backend.Client().SendTransaction(ctx, tx); err != nil {
		t.Errorf("failed to send transaction: %q", err)
	}
	backend.Commit()
	if _, err := bind.WaitDeployed(ctx, backend.Client(), tx.Hash()); err != bind.ErrNoAddressInReceipt {
		t.Errorf("error mismatch: want %q, got %q, ", bind.ErrNoAddressInReceipt, err)
	}

	// 2. Create a contract, but cancel the WaitDeploy before it is mined.
	tx = types.MustSignNewTx(testKey, signer, &types.LegacyTx{
		Nonce:    1,
		Gas:      300000,
		GasPrice: gasPrice,
		Data:     code,
	})

	// Wait in another thread so that we can quickly cancel it after submitting
	// the transaction.
	done := make(chan struct{})
	go func() {
		defer close(done)
		want := errors.New("context canceled")
		_, err := bind.WaitDeployed(ctx, backend.Client(), tx.Hash())
		if err == nil || errors.Is(want, err) {
			t.Errorf("error mismatch: want %v, got %v", want, err)
		}
	}()

	if err := backend.Client().SendTransaction(ctx, tx); err != nil {
		t.Errorf("failed to send transaction: %q", err)
	}
	cancel()

	// Wait for goroutine to exit or for a timeout.
	select {
	case <-done:
	case <-time.After(time.Second * 2):
		t.Fatalf("failed to cancel wait deploy")
	}
}

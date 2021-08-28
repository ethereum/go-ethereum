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

package backends

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

// LimitedBackend represents a non-geth client which does not support the full set of
// JSON RPC calls that get supports
type LimitedBackend struct {
	*SimulatedBackend
}

// SuggestGasTipCap returns an error, simulating an Ethereum node which does not support
// the non-standard eth_maxPriorityFeePerGas
func (lc *LimitedBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return nil, errors.New("Method not found")
}

// Tests binding to non-geth nodes which do not support eth_maxPriorityFeePerGas
func TestOtherNodes(t *testing.T) {
	nilBytecode := hexutil.MustDecode("0x606060405260068060106000396000f3606060405200")
	nilABI := abi.ABI{}

	// Generate a new random account and a funded simulator
	key, _ := crypto.GenerateKey()
	auth, _ := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))

	// Override certain functions in the simulator
	sim := &LimitedBackend{
		NewSimulatedBackend(core.GenesisAlloc{auth.From: {Balance: big.NewInt(10000000000000000)}}, 10000000),
	}
	defer sim.Close()

	opts := &bind.TransactOpts{
		From:   auth.From,
		Signer: auth.Signer,
	}

	// Should be able to deploy a contract to a backend which does not support eth_maxPriorityFeePerGas
	_, _, _, err := bind.DeployContract(opts, nilABI, nilBytecode, sim)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

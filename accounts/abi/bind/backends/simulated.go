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

package backends

import (
	"context"
	"os"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/keystore"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/ethclient/simulated"
	"github.com/XinFinOrg/XDPoSChain/params"
)

// SimulatedBackend is a simulated blockchain.
// Deprecated: use package github.com/XinFinOrg/XDPoSChain/ethclient/simulated instead.
type SimulatedBackend struct {
	*simulated.Backend
}

// Client returns a client that accesses the simulated chain.
func (b *SimulatedBackend) Client() simulated.Client {
	return b.Backend.Client()
}

// Fork sets the head to a new block, which is based on the provided parentHash.
func (b *SimulatedBackend) Fork(ctx context.Context, parentHash common.Hash) error {
	return b.Backend.Fork(parentHash)
}

// NewXDCSimulatedBackend creates a new backend for testing purpose.
func NewXDCSimulatedBackend(alloc types.GenesisAlloc, gasLimit uint64, chainConfig *params.ChainConfig) *SimulatedBackend {
	b := simulated.New(alloc, gasLimit, chainConfig)
	return &SimulatedBackend{
		Backend: b,
	}
}

// NewXDCSimulatedBackend creates a new backend for testing purpose.
//
// A simulated backend always uses chainID 1337.
//
// Deprecated: please use simulated.Backend from package
// github.com/XinFinOrg/XDPoSChain/ethclient/simulated instead.
func NewSimulatedBackend(alloc core.GenesisAlloc, gasLimit uint64) *SimulatedBackend {
	b := simulated.New(alloc, gasLimit, params.AllEthashProtocolChanges)
	return &SimulatedBackend{
		Backend: b,
	}
}

func SimulateWalletAddressAndSignFn() (common.Address, func(account accounts.Account, hash []byte) ([]byte, error), error) {
	veryLightScryptN := 2
	veryLightScryptP := 1
	dir, _ := os.MkdirTemp("", "eth-SimulateWalletAddressAndSignFn-test")
	defer os.RemoveAll(dir)

	ks := keystore.NewKeyStore(dir, veryLightScryptN, veryLightScryptP)
	pass := "" // not used but required by API
	a1, err := ks.NewAccount(pass)
	if err != nil {
		return common.Address{}, nil, err
	}
	if err := ks.Unlock(a1, ""); err != nil {
		return a1.Address, nil, err
	}
	return a1.Address, ks.SignHash, nil
}

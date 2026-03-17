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

package eth

import (
	"context"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/hexutil"
	"github.com/XinFinOrg/XDPoSChain/rpc"
)

// EthereumAPI provides an API to access Ethereum full node-related information.
type EthereumAPI struct {
	e *Ethereum
}

// NewPublicEthereumAPI creates a new Ethereum protocol API for full nodes.
func NewEthereumAPI(e *Ethereum) *EthereumAPI {
	return &EthereumAPI{e}
}

// Etherbase is the address that mining rewards will be send to
func (api *EthereumAPI) Etherbase() (common.Address, error) {
	return api.e.Etherbase()
}

// Coinbase is the address that mining rewards will be send to (alias for Etherbase)
func (api *EthereumAPI) Coinbase() (common.Address, error) {
	return api.Etherbase()
}

// Hashrate returns the POW hashrate
func (api *EthereumAPI) Hashrate() hexutil.Uint64 {
	return hexutil.Uint64(api.e.Miner().HashRate())
}

// Mining returns an indication if this node is currently mining.
func (api *EthereumAPI) Mining() bool {
	return api.e.IsStaking()
}

func (api *EthereumAPI) ChainId() hexutil.Uint64 {
	chainID := new(big.Int)
	if config := api.e.blockchain.Config(); config.IsEIP155(api.e.blockchain.CurrentBlock().Number) {
		chainID = config.ChainID
	}
	return (hexutil.Uint64)(chainID.Uint64())
}

// GetOwner return masternode owner of the given coinbase address
func (api *EthereumAPI) GetOwnerByCoinbase(ctx context.Context, coinbase common.Address, blockNr rpc.BlockNumber) (common.Address, error) {
	statedb, _, err := api.e.APIBackend.StateAndHeaderByNumber(ctx, blockNr)
	if err != nil {
		return common.Address{}, err
	}
	return statedb.GetOwner(coinbase), nil
}

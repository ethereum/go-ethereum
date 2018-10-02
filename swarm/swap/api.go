// Copyright 2018 The go-ethereum Authors
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

package swap

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

var (
	ErrNoSuchPeerAccounting = errors.New("No accounting with that peer")
)

//This is the API definition to access swarm swap accounting data via RPC
type API struct {
	swap *Swap
}

//TODO: define metrics
//Get metrics about swap for this node
type SwapMetrics struct {
}

//Create a new API instance
func NewAPI(swap *Swap) *API {
	return &API{swap: swap}
}

//Get the balance for this node with a specific peer
func (swapapi *API) BalanceWithPeer(ctx context.Context, peer enode.ID) (balance *big.Int, err error) {
	balance = swapapi.swap.peers[peer]
	if balance == nil {
		err = ErrNoSuchPeerAccounting
	}
	return
}

//Get the overall balance of the node
//Iterates over all peers this node is having accounted interaction
//and just adds up balances.
//It assumes that if a disfavorable balance is represented as a negative value
func (swapapi *API) Balance(ctx context.Context) (balance *big.Int, err error) {
	balance = big.NewInt(0)
	for _, peerBalance := range swapapi.swap.peers {
		balance.Add(balance, peerBalance)
	}
	return
}

//Just return the Swap metrics
func (swapapi *API) GetSwapMetrics() (*SwapMetrics, error) {
	return nil, nil
}

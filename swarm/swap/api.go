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

	"github.com/ethereum/go-ethereum/p2p/enode"
)

var (
	ErrNoSuchPeerAccounting = errors.New("No accounting with that peer")
)

//This is the API definition to access swarm swap accounting data via RPC
type API struct {
	swap *Swap
}

//Get metrics about swap for this node
//The current metrics are for accounted message types only
//(i.e. BytesTransferred is amount of bytes sent but only for a
//accounted message types)
type Metrics struct {
	BalanceCredited uint64
	BalanceDebited  uint64
	BytesCredited   uint64
	BytesDebited    uint64
	MsgCredited     uint64
	MsgDebited      uint64
	ChequesIssued   uint64
	ChequesReceived uint64
	PeerDrops       uint64
	SelfDrops       uint64
}

//Create a new API instance
func NewAPI(swap *Swap) *API {
	return &API{swap: swap}
}

//Get the balance for this node with a specific peer
func (swapapi *API) BalanceWithPeer(ctx context.Context, peer enode.ID) (balance int64, err error) {
	var ok bool
	balance, ok = swapapi.swap.balances[peer]
	if !ok {
		err = ErrNoSuchPeerAccounting
	}
	return
}

//Get the overall balance of the node
//Iterates over all peers this node is having accounted interaction
//and just adds up balances.
//It assumes that a disfavorable balance is represented as a negative value
func (swapapi *API) Balance(ctx context.Context) (balance int64, err error) {
	balance = 0
	for _, peerBalance := range swapapi.swap.balances {
		balance += peerBalance
	}
	return
}

//Just return the Swap metrics
func (swapapi *API) GetSwapMetricsForPeer(ctx context.Context, peer enode.ID) (*Metrics, error) {
	var ok bool
	var metrics *Metrics
	metrics, ok = swapapi.swap.metrics[peer]
	if !ok {
		return nil, ErrNoSuchPeerAccounting
	}
	return metrics, nil
}

//Just return the Swap metrics
func (swapapi *API) GetSwapMetrics(ctx context.Context, peer enode.ID) (*Metrics, error) {
	var metrics *Metrics

	for _, m := range swapapi.swap.metrics {
		metrics.BalanceCredited += m.BalanceCredited
		metrics.BalanceDebited += m.BalanceDebited
		metrics.BytesCredited += m.BytesCredited
		metrics.BytesDebited += m.BytesDebited
		metrics.ChequesIssued += m.ChequesIssued
		metrics.ChequesReceived += m.ChequesReceived
		metrics.MsgCredited += m.MsgCredited
		metrics.MsgDebited += m.MsgDebited
		metrics.PeerDrops += m.PeerDrops
		metrics.SelfDrops += m.SelfDrops
	}

	return metrics, nil
}

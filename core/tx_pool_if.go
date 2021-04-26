// Copyright 2021 The go-ethereum Authors
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

package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

var _ TxPoolIf = (*TxPool)(nil)

type TxPoolIf interface {
	Stop()
	SubscribeNewTxsEvent(ch chan<- NewTxsEvent) event.Subscription
	GasPrice() *big.Int
	SetGasPrice(price *big.Int)
	Nonce(addr common.Address) uint64
	Stats() (int, int)
	Content() (map[common.Address]types.Transactions, map[common.Address]types.Transactions)
	Pending() (map[common.Address]types.Transactions, error)
	Locals() []common.Address
	AddLocal(tx *types.Transaction) error
	AddRemotes(txs []*types.Transaction) []error
	AddRemotesSync(txs []*types.Transaction) []error
	Status(hashes []common.Hash) []TxStatus
	Get(hash common.Hash) *types.Transaction
}

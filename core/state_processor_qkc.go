// Copyright 2026-2027 The quarkchain Authors
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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
)

// ErrDepositOrder is returned when a block has a cross-shard deposit after a
// transaction that is not a deposit.
var ErrDepositOrder = errors.New("deposit after a non-deposit transaction")

// checkDepositOrder checks that the cross-shard deposits of a block, which the
// consensus layer forces into it, all come before its other transactions.
func checkDepositOrder(txs types.Transactions) error {
	for i := 1; i < len(txs); i++ {
		if txs[i].Type() == types.DepositTxType && txs[i-1].Type() != types.DepositTxType {
			return fmt.Errorf("%w: tx %d [%v]", ErrDepositOrder, i, txs[i].Hash().Hex())
		}
	}
	return nil
}

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

package locals

import (
	"errors"

	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
)

// IsTemporaryReject determines whether the given error indicates a temporary
// reason to reject a transaction from being included in the txpool. The result
// may change if the txpool's state changes later.
func IsTemporaryReject(err error) bool {
	switch {
	case errors.Is(err, legacypool.ErrOutOfOrderTxFromDelegated):
		return true
	case errors.Is(err, txpool.ErrInflightTxLimitReached):
		return true
	case errors.Is(err, legacypool.ErrAuthorityReserved):
		return true
	case errors.Is(err, txpool.ErrUnderpriced):
		return true
	case errors.Is(err, legacypool.ErrTxPoolOverflow):
		return true
	case errors.Is(err, legacypool.ErrFutureReplacePending):
		return true
	default:
		return false
	}
}

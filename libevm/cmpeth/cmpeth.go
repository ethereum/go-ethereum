// Copyright 2025 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

// Package cmpeth provides ETH-specific options for the cmp package.
package cmpeth

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ava-labs/libevm/core/types"
)

// CompareHeadersByHash returns an option to compare Headers based on
// [types.Header.Hash] equality.
func CompareHeadersByHash() cmp.Option {
	return cmp.Comparer(func(a, b *types.Header) bool {
		return a.Hash() == b.Hash()
	})
}

// CompareTransactionsByBinary returns an option to compare Transactions based
// on [types.Transaction.MarshalBinary] equality. Two nil pointers are
// considered equal.
//
// If MarshalBinary() returns an error, it will be reported with
// [testing.TB.Fatal].
func CompareTransactionsByBinary(tb testing.TB) cmp.Option {
	tb.Helper()
	return cmp.Comparer(func(a, b *types.Transaction) bool {
		tb.Helper()

		if a == nil && b == nil {
			return true
		}
		if a == nil || b == nil {
			return false
		}

		return bytes.Equal(marshalTxBinary(tb, a), marshalTxBinary(tb, b))
	})
}

func marshalTxBinary(tb testing.TB, tx *types.Transaction) []byte {
	tb.Helper()
	buf, err := tx.MarshalBinary()
	if err != nil {
		tb.Fatalf("%T.MarshalBinary() error %v", tx, err)
	}
	return buf
}

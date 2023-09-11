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

package types

import (
	"errors"

	"github.com/ethereum/go-ethereum/rlp"
)

// IsLegacyStoredReceipts tries to parse the RLP-encoded blob
// first as an array of v3 stored receipt, then v4 stored receipt and
// returns true if successful.
func IsLegacyStoredReceipts(raw []byte) (bool, error) {
	var v3 []v3StoredReceiptRLP
	if err := rlp.DecodeBytes(raw, &v3); err == nil {
		return true, nil
	}
	var v4 []v4StoredReceiptRLP
	if err := rlp.DecodeBytes(raw, &v4); err == nil {
		return true, nil
	}
	var v5 []storedReceiptRLP
	// Check to see valid fresh stored receipt
	if err := rlp.DecodeBytes(raw, &v5); err == nil {
		return false, nil
	}
	return false, errors.New("value is not a valid receipt encoding")
}

// ConvertLegacyStoredReceipts takes the RLP encoding of an array of legacy
// stored receipts and returns a fresh RLP-encoded stored receipt.
func ConvertLegacyStoredReceipts(raw []byte) ([]byte, error) {
	var receipts []ReceiptForStorage
	if err := rlp.DecodeBytes(raw, &receipts); err != nil {
		return nil, err
	}
	return rlp.EncodeToBytes(&receipts)
}

// Copyright 2024 The go-ethereum Authors
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
	"fmt"
)

const (
	depositRequestSize = 192
)

// DepositLogToRequest unpacks a serialized DepositEvent.
func DepositLogToRequest(data []byte) ([]byte, error) {
	if len(data) != 576 {
		return nil, fmt.Errorf("deposit wrong length: want 576, have %d", len(data))
	}

	request := make([]byte, depositRequestSize)
	const (
		pubkeyOffset         = 0
		withdrawalCredOffset = pubkeyOffset + 48
		amountOffset         = withdrawalCredOffset + 32
		signatureOffset      = amountOffset + 8
		indexOffset          = signatureOffset + 96
	)
	// The ABI encodes the position of dynamic elements first. Since there are 5
	// elements, skip over the positional data. The first 32 bytes of dynamic
	// elements also encode their actual length. Skip over that value too.
	b := 32*5 + 32
	// PublicKey is the first element. ABI encoding pads values to 32 bytes, so
	// despite BLS public keys being length 48, the value length here is 64. Then
	// skip over the next length value.
	copy(request[pubkeyOffset:], data[b:b+48])
	b += 48 + 16 + 32
	// WithdrawalCredentials is 32 bytes. Read that value then skip over next
	// length.
	copy(request[withdrawalCredOffset:], data[b:b+32])
	b += 32 + 32
	// Amount is 8 bytes, but it is padded to 32. Skip over it and the next
	// length.
	copy(request[amountOffset:], data[b:b+8])
	b += 8 + 24 + 32
	// Signature is 96 bytes. Skip over it and the next length.
	copy(request[signatureOffset:], data[b:b+96])
	b += 96 + 32
	// Index is 8 bytes.
	copy(request[indexOffset:], data[b:b+8])
	return request, nil
}

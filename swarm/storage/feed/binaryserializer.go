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

package feed

import "github.com/ethereum/go-ethereum/common/hexutil"

type binarySerializer interface {
	binaryPut(serializedData []byte) error
	binaryLength() int
	binaryGet(serializedData []byte) error
}

// Values interface represents a string key-value store
// useful for building query strings
type Values interface {
	Get(key string) string
	Set(key, value string)
}

type valueSerializer interface {
	FromValues(values Values) error
	AppendValues(values Values)
}

// Hex serializes the structure and converts it to a hex string
func Hex(bin binarySerializer) string {
	b := make([]byte, bin.binaryLength())
	bin.binaryPut(b)
	return hexutil.Encode(b)
}

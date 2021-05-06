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

package abi

import (
	"github.com/ethereum/go-ethereum/abi"
	"github.com/ethereum/go-ethereum/common"
)

// MakeTopics converts a filter query argument list into a filter topic set.
func MakeTopics(query ...[]interface{}) ([][]common.Hash, error) {
	return abi.MakeTopics(query...)
}

// ParseTopics converts the indexed topic fields into actual log field values.
func ParseTopics(out interface{}, fields Arguments, topics []common.Hash) error {
	return abi.ParseTopics(out, fields, topics)
}

// ParseTopicsIntoMap converts the indexed topic field-value pairs into map key-value pairs.
func ParseTopicsIntoMap(out map[string]interface{}, fields Arguments, topics []common.Hash) error {
	return abi.ParseTopicsIntoMap(out, fields, topics)
}

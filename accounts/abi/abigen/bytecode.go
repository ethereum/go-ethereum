// Copyright 2026 The go-ethereum Authors
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

package abigen

import (
	"fmt"
	"strings"
)

// normalizeBytecode strips the optional 0x prefix and validates deployment
// bytecode before it is interpolated into generated Go source. Besides hex
// digits, Solidity library placeholders of the form __$<hex>$__ are allowed.
func normalizeBytecode(bytecode string) (string, error) {
	bytecode = strings.TrimPrefix(strings.TrimSpace(bytecode), "0x")
	for pos := 0; pos < len(bytecode); {
		if isHexByte(bytecode[pos]) {
			pos++
			continue
		}
		if strings.HasPrefix(bytecode[pos:], "__$") {
			rest := bytecode[pos+3:]
			end := strings.Index(rest, "$__")
			if end <= 0 {
				return "", fmt.Errorf("malformed library placeholder at offset %d", pos)
			}
			for i := 0; i < end; i++ {
				if !isHexByte(rest[i]) {
					return "", fmt.Errorf("malformed library placeholder at offset %d", pos)
				}
			}
			pos += 3 + end + 3
			continue
		}
		return "", fmt.Errorf("invalid character at offset %d", pos)
	}
	return bytecode, nil
}

func isHexByte(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

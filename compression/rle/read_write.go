// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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

// Package rle implements the run-length encoding used for Ethereum data.
package rle

import (
	"bytes"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
)

const (
	token             byte = 0xfe
	emptyShaToken          = 0xfd
	emptyListShaToken      = 0xfe
	tokenToken             = 0xff
)

var empty = crypto.Sha3([]byte(""))
var emptyList = crypto.Sha3([]byte{0x80})

func Decompress(dat []byte) ([]byte, error) {
	buf := new(bytes.Buffer)

	for i := 0; i < len(dat); i++ {
		if dat[i] == token {
			if i+1 < len(dat) {
				switch dat[i+1] {
				case emptyShaToken:
					buf.Write(empty)
				case emptyListShaToken:
					buf.Write(emptyList)
				case tokenToken:
					buf.WriteByte(token)
				default:
					buf.Write(make([]byte, int(dat[i+1]-2)))
				}
				i++
			} else {
				return nil, errors.New("error reading bytes. token encountered without proceeding bytes")
			}
		} else {
			buf.WriteByte(dat[i])
		}
	}

	return buf.Bytes(), nil
}

func compressChunk(dat []byte) (ret []byte, n int) {
	switch {
	case dat[0] == token:
		return []byte{token, tokenToken}, 1
	case len(dat) > 1 && dat[0] == 0x0 && dat[1] == 0x0:
		j := 0
		for j <= 254 && j < len(dat) {
			if dat[j] != 0 {
				break
			}
			j++
		}
		return []byte{token, byte(j + 2)}, j
	case len(dat) >= 32:
		if dat[0] == empty[0] && bytes.Compare(dat[:32], empty) == 0 {
			return []byte{token, emptyShaToken}, 32
		} else if dat[0] == emptyList[0] && bytes.Compare(dat[:32], emptyList) == 0 {
			return []byte{token, emptyListShaToken}, 32
		}
		fallthrough
	default:
		return dat[:1], 1
	}
}

func Compress(dat []byte) []byte {
	buf := new(bytes.Buffer)

	i := 0
	for i < len(dat) {
		b, n := compressChunk(dat[i:])
		buf.Write(b)
		i += n
	}

	return buf.Bytes()
}

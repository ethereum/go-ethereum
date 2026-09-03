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

//go:build womir

package main

import "unsafe"

// These match the WOMIR guest-io imports (env module).
// Protocol: __hint_input prepares next item, __hint_buffer reads words.
// Each item has format: [byte_len_u32_le, ...data_words_padded_to_4bytes]
//
//go:wasmimport env __hint_input
func hintInput()

//go:wasmimport env __hint_buffer
func hintBuffer(ptr unsafe.Pointer, numWords uint32)
func readWord() uint32 {
	var buf [4]byte
	hintBuffer(unsafe.Pointer(&buf[0]), 1)
	return uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
}
func readBytes() []byte {
	hintInput()
	byteLen := readWord()
	numWords := (byteLen + 3) / 4
	data := make([]byte, numWords*4)
	hintBuffer(unsafe.Pointer(&data[0]), numWords)
	return data[:byteLen]
}

// getInput reads the RLP-encoded Payload from the WOMIR hint stream.
func getInput() []byte {
	return readBytes()
}

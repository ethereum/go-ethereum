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

//go:build example
// +build example

package main

import (
	_ "embed"
)

// payloadRlp is the RLP-encoded Payload of Hoodi block 0x1192c3: the chain ID,
// the block and its witness.
//
//go:embed 1192c3_payload.rlp
var payloadRlp []byte

// getInput is a platform-specific function that will recover the input payload
// and returns it as a slice. It is expected to be an RLP-encoded Payload structure
// that contains the witness and the block.
// This is a demo version, that is intended to run on a regular computer, so it
// returns an embedded payload for a small Hoodi block.
func getInput() []byte {
	return payloadRlp
}

// Copyright 2023 The go-ethereum Authors
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

// Package kzg4844 implements the KZG crypto for EIP-4844.
package kzg4844

import (
	"embed"
	"encoding/json"

	gokzg4844 "github.com/crate-crypto/go-kzg-4844"
)

//go:embed trusted_setup.json
var content embed.FS

func init() {
	config, err := content.ReadFile("trusted_setup.json")
	if err != nil {
		panic(err)
	}
	params := new(gokzg4844.JSONTrustedSetup)
	if err := json.Unmarshal(config, params); err != nil {
		panic(err)
	}
	if err := initKZG(params); err != nil {
		panic(err)
	}
}

// Blob represents a 4844 data blob.
type Blob [131072]byte

// Commitment is a serialized commitment to a polynomial.
type Commitment [48]byte

// Proof is a serialized commitment to the quotient polynomial.
type Proof [48]byte

// Point is a BLS field element.
type Point [32]byte

// Claim is a claimed evaluation value in a specific point.
type Claim [32]byte

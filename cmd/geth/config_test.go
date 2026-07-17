// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"testing"

	"github.com/ethereum/go-ethereum/eth/ethconfig"
)

func TestEngineMaxReorgDepthConfigRoundTrip(t *testing.T) {
	original := ethconfig.Defaults
	original.EngineMaxReorgDepth = 0

	encoded, err := tomlSettings.Marshal(&original)
	if err != nil {
		t.Fatalf("failed to encode config: %v", err)
	}
	decoded := ethconfig.Defaults
	if err := tomlSettings.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("failed to decode config: %v", err)
	}
	if decoded.EngineMaxReorgDepth != original.EngineMaxReorgDepth {
		t.Fatalf("reorg depth changed across config round-trip: have %d, want %d", decoded.EngineMaxReorgDepth, original.EngineMaxReorgDepth)
	}
}

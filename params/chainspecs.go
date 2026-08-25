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

package params

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed chainspecs
var chainspecs embed.FS

func readChainSpec(filename string) *ChainConfig {
	f, err := chainspecs.Open(filename)
	if err != nil {
		panic(fmt.Sprintf("Could not open chainspec for %s: %v", filename, err))
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	spec := &ChainConfig{}
	err = decoder.Decode(&spec)
	if err != nil {
		panic(fmt.Sprintf("Could not parse chainspec for %s: %v", filename, err))
	}
	return spec
}

var BloatnetChainConfig = readChainSpec("chainspecs/bloatnet.json")

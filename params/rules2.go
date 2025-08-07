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

package params

import (
	"github.com/ethereum/go-ethereum/params/forks"
)

// Rules captures the activations of forks at a particular block height.
type Rules2 struct {
	active map[forks.Fork]bool
}

func (cfg *Config2) Rules(blockNum uint64, blockTime uint64) Rules2 {
	r := Rules2{
		active: make(map[forks.Fork]bool, len(cfg.activation)),
	}
	return r
}

// Active reports whether the given fork is active.
func (r *Rules2) Active(f forks.Fork) bool {
	return r.active[f]
}

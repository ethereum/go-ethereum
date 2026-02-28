// Copyright 2019 The go-ethereum Authors
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

package les

import (
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

type ulc struct {
	keys     map[string]bool
	fraction int
}

// newULC creates and returns an ultra light client instance.
func newULC(servers []string, fraction int) (*ulc, error) {
	keys := make(map[string]bool)
	for _, id := range servers {
		node, err := enode.Parse(enode.ValidSchemes, id)
		if err != nil {
			log.Warn("Failed to parse trusted server", "id", id, "err", err)
			continue
		}
		keys[node.ID().String()] = true
	}
	if len(keys) == 0 {
		return nil, errors.New("no trusted servers")
	}
	return &ulc{
		keys:     keys,
		fraction: fraction,
	}, nil
}

// trusted return an indicator that whether the specified peer is trusted.
func (u *ulc) trusted(p enode.ID) bool {
	return u.keys[p.String()]
}

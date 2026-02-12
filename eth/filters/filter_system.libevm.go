// Copyright 2026 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package filters

import (
	"github.com/ava-labs/libevm/core/types"
)

// BloomOverrider is an optional extension to [Backend], allowing arbitrary
// bloom filters to be returned for a header. If not implemented,
// [types.Header.Bloom] is used instead.
type BloomOverrider interface {
	OverrideHeaderBloom(*types.Header) types.Bloom
}

func maybeOverrideBloom(header *types.Header, backend Backend) types.Bloom {
	if bo, ok := backend.(BloomOverrider); ok {
		return bo.OverrideHeaderBloom(header)
	}
	return header.Bloom
}

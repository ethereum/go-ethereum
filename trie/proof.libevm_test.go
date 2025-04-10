// Copyright 2025 the libevm authors.
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

package trie

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
)

func TestRangeProofKeysWithDifferentLengths(t *testing.T) {
	var (
		root  = common.HexToHash("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
		start = common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")
		keys  = [][]byte{
			common.Hex2Bytes("1000000000000000000000000000000"),
			common.Hex2Bytes("1000000000000000000000000000000000000000000000000000000000000000"),
		}
		values = [][]byte{
			common.Hex2Bytes("02"),
			common.Hex2Bytes("03"),
		}
	)
	_, err := VerifyRangeProof(
		root,
		start,
		keys,
		values,
		nil, // force it to use stacktrie
	)
	require.ErrorIs(t, err, errKeysHaveDifferentLengths)
}

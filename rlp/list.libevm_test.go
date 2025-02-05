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

package rlp

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeListToBuffer(t *testing.T) {
	vals := []uint{1, 2, 3, 4, 5}

	want, err := EncodeToBytes(vals)
	require.NoErrorf(t, err, "EncodeToBytes(%T{%[1]v})", vals)

	var got bytes.Buffer
	buf := NewEncoderBuffer(&got)
	err = EncodeListToBuffer(buf, vals)
	require.NoErrorf(t, err, "EncodeListToBuffer(..., %T{%[1]v})", vals)
	require.NoErrorf(t, buf.Flush(), "%T.Flush()", buf)

	assert.Equal(t, want, got.Bytes(), "EncodeListToBuffer(..., %T{%[1]v})", vals)
}

func TestDecodeList(t *testing.T) {
	vals := []uint{0, 1, 42, 314159}

	rlp, err := EncodeToBytes(vals)
	require.NoErrorf(t, err, "EncodeToBytes(%T{%[1]v})", vals)

	s := NewStream(bytes.NewReader(rlp), 0)
	got, err := DecodeList[uint](s)
	require.NoErrorf(t, err, "DecodeList[%T]()", vals[0])

	require.Equal(t, len(vals), len(got), "number of values returned by DecodeList()")
	for i, gotPtr := range got {
		assert.Equalf(t, vals[i], *gotPtr, "DecodeList()[%d]", i)
	}
}

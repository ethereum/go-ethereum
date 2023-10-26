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

package algorand

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func packValue(v any) (any, error) {
	data, err := pack(reflect.ValueOf(v))
	if err != nil {
		return nil, err
	}

	var vv reflect.Value
	switch v.(type) {
	case int:
		vv = reflect.New(reflect.TypeOf(int64(0)))
	case uint:
		vv = reflect.New(reflect.TypeOf(uint64(0)))
	default:
		vv = reflect.New(reflect.TypeOf(v))
	}
	err = unpack(data, vv.Interface())
	if err != nil {
		return nil, err
	}
	return vv.Elem().Interface(), nil
}

func TestPackBool(t *testing.T) {
	v, err := packValue(true)
	require.NoError(t, err)
	require.Equal(t, true, v)

	v, err = packValue(false)
	require.NoError(t, err)
	require.Equal(t, false, v)
}

func TestPackString(t *testing.T) {
	v, err := packValue("hello")
	require.NoError(t, err)
	require.Equal(t, "hello", v)
}

func TestPackByte(t *testing.T) {
	v, err := packValue(byte('a'))
	require.NoError(t, err)
	require.Equal(t, byte('a'), v)
}

func TestPackInteger(t *testing.T) {
	i := 1000000

	v, err := packValue(int(i))
	require.NoError(t, err)
	require.Equal(t, int64(i), v)

	v, err = packValue(int8(i))
	require.NoError(t, err)
	require.Equal(t, int8(i), v)

	v, err = packValue(int16(i))
	require.NoError(t, err)
	require.Equal(t, int16(i), v)

	v, err = packValue(int32(i))
	require.NoError(t, err)
	require.Equal(t, int32(i), v)

	v, err = packValue(int64(i))
	require.NoError(t, err)
	require.Equal(t, int64(i), v)

	v, err = packValue(uint(i))
	require.NoError(t, err)
	require.Equal(t, uint64(i), v)

	v, err = packValue(uint8(i))
	require.NoError(t, err)
	require.Equal(t, uint8(i), v)

	v, err = packValue(uint16(i))
	require.NoError(t, err)
	require.Equal(t, uint16(i), v)

	v, err = packValue(uint32(i))
	require.NoError(t, err)
	require.Equal(t, uint32(i), v)

	v, err = packValue(uint64(i))
	require.NoError(t, err)
	require.Equal(t, uint64(i), v)
}

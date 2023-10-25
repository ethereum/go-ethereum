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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestGetCmdType(t *testing.T) {
	data, err := common.ParseHexOrString("0x0000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	cmd, err := getCmdTypeFromRawInput(data)
	require.NoError(t, err)
	require.Equal(t, AccountCmd, CmdType(cmd))
}

func TestDecodeInput(t *testing.T) {
	data, err := common.ParseHexOrString("0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000006416d6f756e740000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003a3733373737373737373737373737373737373737373737373737373737373737373737373737373737373737373737373737375546454a324349000000000000")
	require.NoError(t, err)
	input, err := UnpackInput(data)
	require.NoError(t, err)
	require.Equal(t, AccountCmd, input.GetCmdType())
	// 737777777777777777777777777777777777777777777777777UFEJ2CI is the address of RewardsPool in the Algorand mainnet.
	require.Equal(t, "737777777777777777777777777777777777777777777777777UFEJ2CI", input.(*AccountInput).Address)
	require.Equal(t, "Amount", input.(*AccountInput).FieldName)
}

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

package ethtest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/crypto"
	"github.com/ava-labs/libevm/params"
)

func TestDeterministicPrivateKey(t *testing.T) {
	tests := []struct {
		seed []byte
		// Specific values are random, but we lock them in to ensure
		// deterministic generation.
		want common.Address
	}{
		{
			seed: nil,
			want: common.HexToAddress("0x9cce34F7aB185c7ABA1b7C8140d620B4BDA941d6"),
		},
		{
			seed: []byte{0},
			want: common.HexToAddress("0xa385D2E939787Af0B304512b2b6d56364F1722FA"),
		},
		{
			seed: []byte{1},
			want: common.HexToAddress("0x3Eea25034397B249a3eD8614BB4d0533e5b03594"),
		},
	}

	signer := types.LatestSigner(params.MergedTestChainConfig)

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			key := UNSAFEDeterministicPrivateKey(t, tt.seed)

			t.Run("address_from_pubkey", func(t *testing.T) {
				got := crypto.PubkeyToAddress(key.PublicKey)
				require.Equal(t, tt.want, got, "crypto.PubKeyToAddress(UNSAFEDeterministicPrivateKey())")
			})

			t.Run("address_via_sender_recovery", func(t *testing.T) {
				got, err := types.Sender(
					signer,
					types.MustSignNewTx(key, signer, &types.LegacyTx{}),
				)
				require.NoError(t, err, "types.Sender(...)")
				require.Equal(t, tt.want, got, "types.Sender(..., types.MustSignNewTx(UNSAFEDeterministicPrivateKey(), ....))")
			})
		})
	}
}

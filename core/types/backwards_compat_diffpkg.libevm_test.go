// Copyright 2024-2025 the libevm authors.
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

package types_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/libevm/ethtest"
	"github.com/ava-labs/libevm/rlp"
)

func TestHeaderRLPBackwardsCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		register func()
	}{
		{
			name:     "no registered extras",
			register: func() {},
		},
		{
			name: "no-op header hooks",
			register: func() {
				RegisterExtras[
					NOOPHeaderHooks, *NOOPHeaderHooks,
					NOOPBlockBodyHooks, *NOOPBlockBodyHooks,
					struct{},
				]()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			TestOnlyClearRegisteredExtras()
			defer TestOnlyClearRegisteredExtras()
			tt.register()
			testHeaderRLPBackwardsCompatibility(t)
		})
	}
}

//nolint:thelper
func testHeaderRLPBackwardsCompatibility(t *testing.T) {
	// This is a deliberate change-detector test that locks in backwards
	// compatibility of RLP encoding.
	rng := ethtest.NewPseudoRand(42)

	const numExtraBytes = 16
	hdr := &Header{
		ParentHash:  rng.Hash(),
		UncleHash:   rng.Hash(),
		Coinbase:    rng.Address(),
		Root:        rng.Hash(),
		TxHash:      rng.Hash(),
		ReceiptHash: rng.Hash(),
		Bloom:       rng.Bloom(),
		Difficulty:  rng.Uint256().ToBig(),
		Number:      rng.BigUint64(),
		GasLimit:    rng.Uint64(),
		GasUsed:     rng.Uint64(),
		Time:        rng.Uint64(),
		Extra:       rng.Bytes(numExtraBytes),
		MixDigest:   rng.Hash(),
		Nonce:       rng.BlockNonce(),

		BaseFee:          rng.BigUint64(),
		WithdrawalsHash:  rng.HashPtr(),
		BlobGasUsed:      rng.Uint64Ptr(),
		ExcessBlobGas:    rng.Uint64Ptr(),
		ParentBeaconRoot: rng.HashPtr(),
	}
	t.Logf("%T:\n%+v", hdr, hdr)

	// WARNING: changing this hex might break backwards compatibility of RLP
	// encoding (i.e. block hashes might change)!
	const wantHex = `f9029aa01a571e7e4d774caf46053201cfe0001b3c355ffcc93f510e671e8809741f0eeda0756095410506ec72a2c287fe83ebf68efb0be177e61acec1c985277e90e52087941bfc3bc193012ba58912c01fb35a3454831a8971a00bc9f064144eb5965c5e5d1020f9f90392e7e06ded9225966abc7c754b410e61a0d942eab201424f4320ec1e1ffa9390baf941629b9349977b5d48e0502dbb9386a035d9d550a9c113f78689b4c161c4605609bb57b83061914c42ad244daa7fc38eb901004b31d39ae246d689f23176d679a62ff328f530407cbafd0146f45b2ed635282e2812f2705bfffe52576a6fb31df817f29efac71fa56b8e133334079f8e2a8fd2055451571021506f27190adb52a1313f6d28c77d66ae1aa3d3d6757a762476f4c8a2b7b2a37079a4b6a15d1bc44161190c82d5e1c8b55e05c7354f1e5f6512924c941fb3d93667dc3a8c304a3c164e6525dfc99b5f474110c5059485732153e20300c3482832d07b65f97958360da414cb438ce252aec6c2718d155798390a6c6782181d1bac1dd64cd956332b008412ddc735f2994e297c8a088c6bb4c637542295ba3cbc3cd399c8127076f4d834d74d5b11a36b6d02e2fe3a583216aa4ccea0f052df9a96e7a454256bebabdfc38c429079f25913e0f1d7416b2f056c4a115f88b85f0e9fd6d25717881f03d9985060087c88a2c54269dfd07ca388eb8f974b42a412da90c757012bf5479896165caf573cf82fb3a0aa10f6ebf6b62bef8ed36b8ea3d4b1ddb80c99afafa37cb8f3393eb6d802f5bc886c8cd6bcd168a7e0886d5b1345d948b818a0061a7182ff228a4e66bade4717e6f4d318ac98fca12a053af6f98805a764fb5d8890ed9cab2c5229908891c7e2f71857c77ca0523cb6f654ef3fc7294c7768cddd9ccf4bcda3066d382675f37dd1a18507b5fb`
	wantRLP, err := hex.DecodeString(wantHex)
	require.NoError(t, err, "hex.DecodeString()")

	t.Run("Encode", func(t *testing.T) {
		got, err := rlp.EncodeToBytes(hdr)
		require.NoErrorf(t, err, "rlp.EncodeToBytes(%T)", hdr)
		assert.Equalf(t, wantRLP, got, "rlp.EncodeToBytes(%T)", hdr)
	})

	t.Run("Decode", func(t *testing.T) {
		got := new(Header)
		err := rlp.DecodeBytes(wantRLP, got)
		require.NoErrorf(t, err, "rlp.DecodeBytes(..., %T)", hdr)
		assert.Equal(t, hdr, got)
	})
}

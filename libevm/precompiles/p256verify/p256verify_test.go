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

package p256verify

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/libevm"
	"github.com/ava-labs/libevm/libevm/ethtest"
	"github.com/ava-labs/libevm/libevm/hookstest"
	"github.com/ava-labs/libevm/params"

	_ "embed"
)

var _ vm.PrecompiledContract = Precompile{}

// ulerdoganTestCase is the test case from
// https://github.com/ulerdogan/go-ethereum/blob/cec0b058115282168c5afc5197de3f6b5479dc4a/core/vm/testdata/precompiles/p256Verify.json,
// copied under LGPL. See the respective commit for copyright and license
// information.
const ulerdoganTestCase = `4cee90eb86eaa050036147a12d49004b6b9c72bd725d39d4785011fe190f0b4da73bd4903f0ce3b639bbbf6e8e80d16931ff4bcf5993d58468e8fb19086e8cac36dbcd03009df8c59286b162af3bd7fcc0450c9aa81be5d10d312af6c66b1d604aebd3099c618202fcfe16ae7770b0c49ab5eadf74b754204a3bb6060e44eff37618b065f9832de4ca6ca971a7a1adc826d0f7c00181a5fb2ddf79ae00b4e10e`

//go:embed testdata/ecdsa_secp256r1_sha256_test.json
var wycheproofECDSASHA256 []byte

type testCase struct {
	name        string
	in          []byte
	wantSuccess bool
}

func signAndPack(tb testing.TB, priv *ecdsa.PrivateKey, hash [32]byte) []byte {
	tb.Helper()
	r, s, err := ecdsa.Sign(rand.Reader, priv, hash[:])
	require.NoError(tb, err, "ecdsa.Sign()")
	return Pack(hash, r, s, &priv.PublicKey)
}

func TestPrecompile(t *testing.T) {
	assert.Equal(t, params.P256VerifyGas, Precompile{}.RequiredGas(nil), "RequiredGas()")

	tests := []testCase{
		{
			name: "empty_input",
		},
		{
			name: "input_too_short",
			in:   make([]byte, inputLen-1),
		},
		{
			name: "input_too_long",
			in:   make([]byte, inputLen+1),
		},
		{
			name: "pub_key_at_infinity",
			in:   make([]byte, inputLen),
		},
		{
			name: "pub_key_not_on_curve",
			in:   []byte{inputLen - 1: 1},
		},
		{
			name:        "ulerdogan",
			in:          common.Hex2Bytes(ulerdoganTestCase),
			wantSuccess: true,
		},
	}

	for range 50 {
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err, "ecdsa.GenerateKey(elliptic.P256(), crypto/rand.Reader)")

		for range 50 {
			var toSign [32]byte
			_, err := rand.Read(toSign[:])
			require.NoErrorf(t, err, "crypto/rand.Read(%T)", toSign)

			in := signAndPack(t, priv, toSign)
			tests = append(tests, testCase{
				name:        "fuzz_valid",
				in:          in,
				wantSuccess: true,
			})
			corrupt := slices.Clone(in)
			corrupt[0]++ // different signed hash
			tests = append(tests, testCase{
				name: "fuzz_invalid",
				in:   corrupt,
			})
		}
	}

	tests = append(tests, wycheproofTestCases(t)...)
	if t.Failed() {
		return
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Precompile{}.Run(tt.in)
			require.NoError(t, err, "Run() always returns nil, even on verification failure")

			var want []byte
			if tt.wantSuccess {
				want = common.LeftPadBytes([]byte{1}, 32)
			}
			assert.Equal(t, want, got)
		})
	}
}

type jsonHex []byte

var _ json.Unmarshaler = (*jsonHex)(nil)

func (j *jsonHex) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	*j = b
	return nil
}

func wycheproofTestCases(t *testing.T) []testCase {
	t.Helper()

	var raw struct {
		Groups []struct {
			Key struct {
				X jsonHex `json:"wx"`
				Y jsonHex `json:"wy"`
			}
			Tests []struct {
				ID       int `json:"tcId"`
				Comment  string
				Preimage jsonHex `json:"msg"`
				ASNSig   jsonHex `json:"sig"`
				Result   string
			} `json:"tests"`
		} `json:"testGroups"`
	}
	require.NoError(t, json.Unmarshal(wycheproofECDSASHA256, &raw))

	var cases []testCase
	for _, group := range raw.Groups {
		key := &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(group.Key.X),
			Y:     new(big.Int).SetBytes(group.Key.Y),
		}

		for _, test := range group.Tests {
			t.Run(fmt.Sprintf("parse_test_%d", test.ID), func(t *testing.T) {
				// Many of the invalid cases are due to ASN1-specific problems,
				// which aren't of concern to us.
				include := test.Result == "valid" ||
					strings.Contains(test.Comment, "r or s") ||
					strings.Contains(test.Comment, "r and s") ||
					slices.Contains(
						[]int{
							// Special cases of r and/or s.
							286, 294, 295, 303, 304, 340, 341,
							342, 343, 356, 357, 358, 359,
						},
						test.ID,
					)

				include = include && !slices.Contains(
					// These cases have negative r or s value(s) with the same
					// absolute value(s) as valid signatures. Packing and then
					// unpacking via [big.Int.Bytes] therefore converts them to
					// the valid, positive values that pass verification and
					// raise false-positive test errors.
					[]int{133, 139, 140},
					test.ID,
				)
				if !include {
					return
				}

				var rs [2]*big.Int
				rest, err := asn1.Unmarshal(test.ASNSig, &rs)
				if err != nil || len(rest) > 0 {
					return
				}
				if rs[0].BitLen() > 256 || rs[1].BitLen() > 256 {
					return
				}
				cases = append(cases, testCase{
					name:        fmt.Sprintf("wycheproof_ecdsa_secp256r1_sha256_%d", test.ID),
					in:          Pack(sha256.Sum256(test.Preimage), rs[0], rs[1], key),
					wantSuccess: test.Result == "valid",
				})
			})
		}
	}
	t.Logf("%d Wycheproof cases", len(cases))
	return cases
}

func BenchmarkPrecompile(b *testing.B) {
	in := common.Hex2Bytes(ulerdoganTestCase)
	var p Precompile

	for range b.N {
		// Explicitly drop return values to placate the linter. The error is
		// always nil and the input is tested above.
		_, _ = p.Run(in)
	}
}

func TestViaEVM(t *testing.T) {
	addr := common.Address{42}
	hooks := hookstest.Stub{
		PrecompileOverrides: map[common.Address]libevm.PrecompiledContract{
			addr: Precompile{},
		},
	}
	hooks.Register(t)

	_, evm := ethtest.NewZeroEVM(t)
	in := common.Hex2Bytes(ulerdoganTestCase)

	got, _, err := evm.Call(vm.AccountRef{}, addr, in, 25000, uint256.NewInt(0))
	require.NoError(t, err)
	assert.Equal(t, []byte{31: 1}, got)
}

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

	"github.com/google/go-cmp/cmp"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	. "github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/libevm/cmpeth"
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
				RegisterExtras[NOOPHeaderHooks, *NOOPHeaderHooks, struct{}]()
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

func TestBodyRLPBackwardsCompatibility(t *testing.T) {
	newTx := func(nonce uint64) *Transaction { return NewTx(&LegacyTx{Nonce: nonce}) }
	newHdr := func(hashLow byte) *Header { return &Header{ParentHash: common.Hash{hashLow}} }
	newWithdraw := func(idx uint64) *Withdrawal { return &Withdrawal{Index: idx} }

	// We build up test-case [Body] instances from the power set of each of
	// these components.
	txMatrix := [][]*Transaction{
		nil, {}, // Must be equivalent for non-optional field
		{newTx(1)},
		{newTx(2), newTx(3)}, // Demonstrates nested lists
	}
	uncleMatrix := [][]*Header{
		nil, {},
		{newHdr(1)},
		{newHdr(2), newHdr(3)},
	}
	withdrawMatrix := [][]*Withdrawal{
		nil, {}, // Must be different for optional field
		{newWithdraw(1)},
		{newWithdraw(2), newWithdraw(3)},
	}

	var bodies []*Body
	for _, tx := range txMatrix {
		for _, u := range uncleMatrix {
			for _, w := range withdrawMatrix {
				bodies = append(bodies, &Body{tx, u, w})
			}
		}
	}

	for _, body := range bodies {
		t.Run("", func(t *testing.T) {
			t.Logf("\n%s", pretty.Sprint(body))

			// The original [Body] doesn't implement [rlp.Encoder] nor
			// [rlp.Decoder] so we can use a methodless equivalent as the gold
			// standard.
			type withoutMethods Body
			wantRLP, err := rlp.EncodeToBytes((*withoutMethods)(body))
			require.NoErrorf(t, err, "rlp.EncodeToBytes([%T with methods stripped])", body)

			t.Run("Encode", func(t *testing.T) {
				got, err := rlp.EncodeToBytes(body)
				require.NoErrorf(t, err, "rlp.EncodeToBytes(%#v)", body)
				assert.Equalf(t, wantRLP, got, "rlp.EncodeToBytes(%#v)", body)
			})

			t.Run("Decode", func(t *testing.T) {
				got := new(Body)
				err := rlp.DecodeBytes(wantRLP, got)
				require.NoErrorf(t, err, "rlp.DecodeBytes(%v, %T)", wantRLP, got)

				want := body
				// Regular RLP decoding will never leave these non-optional
				// fields nil.
				if want.Transactions == nil {
					want.Transactions = []*Transaction{}
				}
				if want.Uncles == nil {
					want.Uncles = []*Header{}
				}

				opts := cmp.Options{
					cmpeth.CompareHeadersByHash(),
					cmpeth.CompareTransactionsByBinary(t),
				}
				if diff := cmp.Diff(body, got, opts); diff != "" {
					t.Errorf("rlp.DecodeBytes(rlp.EncodeToBytes(%#v)) diff (-want +got):\n%s", body, diff)
				}
			})
		})
	}
}

// cChainBodyExtras carries the same additional fields as the Avalanche C-Chain
// (ava-labs/coreth) [Body] and implements [BodyHooks] to achieve equivalent RLP
// {en,de}coding.
type cChainBodyExtras struct {
	Version uint32
	ExtData *[]byte
}

var _ BodyHooks = (*cChainBodyExtras)(nil)

func (e *cChainBodyExtras) AppendRLPFields(b rlp.EncoderBuffer, _ bool) error {
	b.WriteUint64(uint64(e.Version))

	var data []byte
	if e.ExtData != nil {
		data = *e.ExtData
	}
	b.WriteBytes(data)

	return nil
}

func (e *cChainBodyExtras) DecodeExtraRLPFields(s *rlp.Stream) error {
	if err := s.Decode(&e.Version); err != nil {
		return err
	}

	buf, err := s.Bytes()
	if err != nil {
		return err
	}
	if len(buf) > 0 {
		e.ExtData = &buf
	} else {
		// Respect the `rlp:"nil"` field tag.
		e.ExtData = nil
	}

	return nil
}

func TestBodyRLPCChainCompat(t *testing.T) {
	// The inputs to this test were used to generate the expected RLP with
	// ava-labs/coreth. This serves as both an example of how to use [BodyHooks]
	// and a test of compatibility.

	t.Cleanup(func() {
		TestOnlyRegisterBodyHooks(NOOPBodyHooks{})
	})

	body := &Body{
		Transactions: []*Transaction{
			NewTx(&LegacyTx{
				Nonce: 42,
				To:    common.PointerTo(common.HexToAddress(`decafc0ffeebad`)),
			}),
		},
		Uncles: []*Header{ /* RLP encoding differs in ava-labs/coreth */ },
	}

	const version = 314159
	tests := []struct {
		name  string
		extra *cChainBodyExtras
		// WARNING: changing these values might break backwards compatibility of
		// RLP encoding!
		wantRLPHex string
	}{
		{
			extra: &cChainBodyExtras{
				Version: version,
			},
			wantRLPHex: `e5dedd2a80809400000000000000000000000000decafc0ffeebad8080808080c08304cb2f80`,
		},
		{
			extra: &cChainBodyExtras{
				Version: version,
				ExtData: &[]byte{1, 4, 2, 8, 5, 7},
			},
			wantRLPHex: `ebdedd2a80809400000000000000000000000000decafc0ffeebad8080808080c08304cb2f86010402080507`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantRLP, err := hex.DecodeString(tt.wantRLPHex)
			require.NoErrorf(t, err, "hex.DecodeString(%q)", tt.wantRLPHex)

			t.Run("Encode", func(t *testing.T) {
				TestOnlyRegisterBodyHooks(tt.extra)
				got, err := rlp.EncodeToBytes(body)
				require.NoErrorf(t, err, "rlp.EncodeToBytes(%+v)", body)
				assert.Equalf(t, wantRLP, got, "rlp.EncodeToBytes(%+v)", body)
			})

			t.Run("Decode", func(t *testing.T) {
				var extra cChainBodyExtras
				TestOnlyRegisterBodyHooks(&extra)

				got := new(Body)
				err := rlp.DecodeBytes(wantRLP, got)
				require.NoErrorf(t, err, "rlp.DecodeBytes(%#x, %T)", wantRLP, got)
				assert.Equal(t, tt.extra, &extra, "rlp.DecodeBytes(%#x, [%T as registered extra in %T carrier])", wantRLP, &extra, got)

				opts := cmp.Options{
					cmpeth.CompareHeadersByHash(),
					cmpeth.CompareTransactionsByBinary(t),
				}
				if diff := cmp.Diff(body, got, opts); diff != "" {
					t.Errorf("rlp.DecodeBytes(%#x, [%T while carrying registered %T extra payload]) diff (-want +got):\n%s", wantRLP, got, &extra, diff)
				}
			})
		})
	}
}

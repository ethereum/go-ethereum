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

package types

import (
	"encoding/hex"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/rlp"
)

func TestBodyRLPBackwardsCompatibility(t *testing.T) {
	newTx := func(nonce uint64) *Transaction { return NewTx(&LegacyTx{Nonce: nonce}) }
	newHdr := func(hashLow byte) *Header { return &Header{ParentHash: common.Hash{hashLow}} }
	newWithdraw := func(idx uint64) *Withdrawal { return &Withdrawal{Index: idx} }

	// We build up test-case [Body] instances from the Cartesian product of each
	// of these components.
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
					cmp.Comparer((*Header).equalHash),
					cmp.Comparer((*Transaction).equalHash),
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

func (e *cChainBodyExtras) RLPFieldsForEncoding(b *Body) *rlp.Fields {
	// The Avalanche C-Chain uses all of the geth required fields (but none of
	// the optional ones) so there's no need to explicitly list them. This
	// pattern might not be ideal for readability but is used here for
	// demonstrative purposes.
	//
	// All new fields will always be tagged as optional for backwards
	// compatibility so this is safe to do, but only for the required fields.
	return &rlp.Fields{
		Required: append(
			NOOPBodyHooks{}.RLPFieldsForEncoding(b).Required,
			e.Version, e.ExtData,
		),
	}
}

func (e *cChainBodyExtras) RLPFieldPointersForDecoding(b *Body) *rlp.Fields {
	// An alternative to the pattern used above is to explicitly list all
	// fields for better introspection.
	return &rlp.Fields{
		Required: []any{
			&b.Transactions,
			&b.Uncles,
			&e.Version,
			rlp.Nillable(&e.ExtData), // equivalent to `rlp:"nil"`
		},
	}
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
			name: "nil_ExtData",
			extra: &cChainBodyExtras{
				Version: version,
			},
			wantRLPHex: `e5dedd2a80809400000000000000000000000000decafc0ffeebad8080808080c08304cb2f80`,
		},
		{
			name: "non-nil_ExtData",
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
					cmp.Comparer((*Header).equalHash),
					cmp.Comparer((*Transaction).equalHash),
				}
				if diff := cmp.Diff(body, got, opts); diff != "" {
					t.Errorf("rlp.DecodeBytes(%#x, [%T while carrying registered %T extra payload]) diff (-want +got):\n%s", wantRLP, got, &extra, diff)
				}
			})
		})
	}
}

// equalHash reports whether `a` and `b` have equal hashes. It allows for nil
// arguments, returning `true` if both are nil, `false` if only one is nil,
// otherwise `a.Hash() == b.Hash()`.
func equalHash[
	T any, P interface {
		Hash() common.Hash
		*T
	},
](a, b P) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Hash() == b.Hash()
}

func (h *Header) equalHash(hh *Header) bool           { return equalHash(h, hh) }
func (tx *Transaction) equalHash(u *Transaction) bool { return equalHash(tx, u) }

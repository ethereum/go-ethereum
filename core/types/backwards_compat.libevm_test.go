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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/rlp"
)

func newTx(nonce uint64) *Transaction    { return NewTx(&LegacyTx{Nonce: nonce}) }
func newHdr(parentHashHigh byte) *Header { return &Header{ParentHash: common.Hash{parentHashHigh}} }
func newWithdraw(idx uint64) *Withdrawal { return &Withdrawal{Index: idx} }

func blockBodyRLPTestInputs() []*Body {
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
				bodies = append(bodies, &Body{tx, u, w, nil /* extra field */})
			}
		}
	}
	return bodies
}

func TestBodyRLPBackwardsCompatibility(t *testing.T) {
	for _, body := range blockBodyRLPTestInputs() {
		t.Run("", func(t *testing.T) {
			t.Cleanup(func() {
				if t.Failed() {
					t.Logf("\n%s", pretty.Sprint(body))
				}
			})

			// The original [Body] doesn't implement [rlp.Encoder] nor
			// [rlp.Decoder] so we can use a methodless equivalent as the gold
			// standard.
			type withoutMethods Body
			wantRLP, err := rlp.EncodeToBytes((*withoutMethods)(body))
			require.NoErrorf(t, err, "rlp.EncodeToBytes([%T with methods stripped])", body)

			t.Run("Encode", func(t *testing.T) {
				got, err := rlp.EncodeToBytes(body)
				require.NoErrorf(t, err, "rlp.EncodeToBytes(%T)", body)
				assert.Equalf(t, wantRLP, got, "rlp.EncodeToBytes(%T)", body)
			})

			t.Run("Decode", func(t *testing.T) {
				got := new(Body)
				err := rlp.DecodeBytes(wantRLP, got)
				require.NoErrorf(
					t, err, "rlp.DecodeBytes(rlp.EncodeToBytes(%T), %T) resulted in %s",
					(*withoutMethods)(body), got, pretty.Sprint(got),
				)

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
					cmpopts.IgnoreUnexported(Body{}),
				}
				if diff := cmp.Diff(want, got, opts); diff != "" {
					t.Errorf("rlp.DecodeBytes(rlp.EncodeToBytes(%T)) diff (-want +got):\n%s", (*withoutMethods)(body), diff)
				}
			})
		})
	}
}

func TestBlockRLPBackwardsCompatibility(t *testing.T) {
	TestOnlyClearRegisteredExtras()
	t.Cleanup(TestOnlyClearRegisteredExtras)

	RegisterExtras[
		NOOPHeaderHooks, *NOOPHeaderHooks,
		NOOPBlockBodyHooks, *NOOPBlockBodyHooks, // types under test
		struct{},
	]()

	// Note that there are also a number of tests in `block_test.go` that ensure
	// backwards compatibility as [NOOPBlockBodyHooks] are used by default when
	// nothing is registered (the above registration is only for completeness).

	for _, body := range blockBodyRLPTestInputs() {
		t.Run("", func(t *testing.T) {
			// [Block] doesn't export most of its fields so uses [extblock] as a
			// proxy for RLP encoding, which is what we therefore use as the
			// backwards-compatible gold standard.
			hdr := newHdr(99)
			block := extblock{
				Header:      hdr,
				Txs:         body.Transactions,
				Uncles:      body.Uncles,
				Withdrawals: body.Withdrawals,
			}

			// We've added [extblock.EncodeRLP] and [extblock.DecodeRLP] for our
			// hooks.
			type withoutMethods extblock

			wantRLP, err := rlp.EncodeToBytes(withoutMethods(block))
			require.NoErrorf(t, err, "rlp.EncodeToBytes([%T with methods stripped])", block)

			// Our input to RLP might not be the canonical RLP output.
			var wantBlock extblock
			err = rlp.DecodeBytes(wantRLP, (*withoutMethods)(&wantBlock))
			require.NoErrorf(t, err, "rlp.DecodeBytes(..., [%T with methods stripped])", &wantBlock)

			t.Run("Encode", func(t *testing.T) {
				b := NewBlockWithHeader(hdr).WithBody(*body).WithWithdrawals(body.Withdrawals)
				got, err := rlp.EncodeToBytes(b)
				require.NoErrorf(t, err, "rlp.EncodeToBytes(%T)", b)

				assert.Equalf(t, wantRLP, got, "expect %T RLP identical to that from %T struct stripped of methods", got, extblock{})
			})

			t.Run("Decode", func(t *testing.T) {
				var gotBlock Block
				err := rlp.DecodeBytes(wantRLP, &gotBlock)
				require.NoErrorf(t, err, "rlp.DecodeBytes(..., %T)", &gotBlock)

				got := extblock{
					gotBlock.Header(),
					gotBlock.Transactions(),
					gotBlock.Uncles(),
					gotBlock.Withdrawals(),
					nil, // unexported libevm hooks
				}

				opts := cmp.Options{
					cmp.Comparer((*Header).equalHash),
					cmp.Comparer((*Transaction).equalHash),
					cmpopts.IgnoreUnexported(extblock{}),
				}
				if diff := cmp.Diff(wantBlock, got, opts); diff != "" {
					t.Errorf("rlp.DecodeBytes([RLP from %T stripped of methods], ...) diff (-want +got):\n%s", extblock{}, diff)
				}
			})
		})
	}
}

// cChainBodyExtras carries the same additional fields as the Avalanche C-Chain
// (ava-labs/coreth) [Body] and implements [BlockBodyHooks] to achieve
// equivalent RLP {en,de}coding.
//
// It is not intended as a full test of ava-labs/coreth existing functionality,
// which should be implemented when that module consumes libevm, but as proof of
// equivalence of the [rlp.Fields] approach.
type cChainBodyExtras struct {
	Version uint32
	ExtData *[]byte
}

var _ BlockBodyHooks = (*cChainBodyExtras)(nil)

func (e *cChainBodyExtras) BodyRLPFieldsForEncoding(b *Body) *rlp.Fields {
	// The Avalanche C-Chain uses all of the geth required fields (but none of
	// the optional ones) so there's no need to explicitly list them. This
	// pattern might not be ideal for readability but is used here for
	// demonstrative purposes.
	//
	// All new fields will always be tagged as optional for backwards
	// compatibility so this is safe to do, but only for the required fields.
	return &rlp.Fields{
		Required: append(
			NOOPBlockBodyHooks{}.BodyRLPFieldsForEncoding(b).Required,
			e.Version, e.ExtData,
		),
	}
}

func (e *cChainBodyExtras) BodyRLPFieldPointersForDecoding(b *Body) *rlp.Fields {
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

// See [cChainBodyExtras] intent.

func (e *cChainBodyExtras) Copy() *cChainBodyExtras {
	panic("unimplemented")
}

func (e *cChainBodyExtras) BlockRLPFieldsForEncoding(b *BlockRLPProxy) *rlp.Fields {
	panic("unimplemented")
}

func (e *cChainBodyExtras) BlockRLPFieldPointersForDecoding(b *BlockRLPProxy) *rlp.Fields {
	panic("unimplemented")
}

func TestBodyRLPCChainCompat(t *testing.T) {
	// The inputs to this test were used to generate the expected RLP with
	// ava-labs/coreth. This serves as both an example of how to use [BodyHooks]
	// and a test of compatibility.
	TestOnlyClearRegisteredExtras()
	t.Cleanup(TestOnlyClearRegisteredExtras)
	extras := RegisterExtras[
		NOOPHeaderHooks, *NOOPHeaderHooks,
		cChainBodyExtras, *cChainBodyExtras,
		struct{},
	]()

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
				extras.Body.Set(body, tt.extra)
				got, err := rlp.EncodeToBytes(body)
				require.NoErrorf(t, err, "rlp.EncodeToBytes(%+v)", body)
				assert.Equalf(t, wantRLP, got, "rlp.EncodeToBytes(%+v)", body)
			})

			t.Run("Decode", func(t *testing.T) {
				var extra cChainBodyExtras
				got := new(Body)
				extras.Body.Set(got, &extra)
				err := rlp.DecodeBytes(wantRLP, got)
				require.NoErrorf(t, err, "rlp.DecodeBytes(%#x, %T)", wantRLP, got)
				assert.Equal(t, tt.extra, &extra, "rlp.DecodeBytes(%#x, [%T as registered extra in %T carrier])", wantRLP, &extra, got)

				opts := cmp.Options{
					cmp.Comparer((*Header).equalHash),
					cmp.Comparer((*Transaction).equalHash),
					cmpopts.IgnoreUnexported(Body{}),
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

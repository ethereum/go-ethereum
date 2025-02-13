// Copyright 2024 the libevm authors.
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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/crypto"
	"github.com/ava-labs/libevm/libevm/ethtest"
	"github.com/ava-labs/libevm/libevm/pseudo"
	"github.com/ava-labs/libevm/rlp"
)

type stubHeaderHooks struct {
	suffix                                   []byte
	gotRawJSONToUnmarshal, gotRawRLPToDecode []byte
	setHeaderToOnUnmarshalOrDecode           Header
	accessor                                 pseudo.Accessor[*Header, *stubHeaderHooks]
	toCopy                                   *stubHeaderHooks

	errMarshal, errUnmarshal, errEncode, errDecode error
}

func fakeHeaderJSON(h *Header, suffix []byte) []byte {
	return []byte(fmt.Sprintf(`"%#x:%#x"`, h.ParentHash, suffix))
}

func fakeHeaderRLP(h *Header, suffix []byte) []byte {
	return append(crypto.Keccak256(h.ParentHash[:]), suffix...)
}

func (hh *stubHeaderHooks) EncodeJSON(h *Header) ([]byte, error) {
	return fakeHeaderJSON(h, hh.suffix), hh.errMarshal
}

func (hh *stubHeaderHooks) DecodeJSON(h *Header, b []byte) error {
	hh.gotRawJSONToUnmarshal = b
	*h = hh.setHeaderToOnUnmarshalOrDecode
	return hh.errUnmarshal
}

func (hh *stubHeaderHooks) EncodeRLP(h *Header, w io.Writer) error {
	if _, err := w.Write(fakeHeaderRLP(h, hh.suffix)); err != nil {
		return err
	}
	return hh.errEncode
}

func (hh *stubHeaderHooks) DecodeRLP(h *Header, s *rlp.Stream) error {
	r, err := s.Raw()
	if err != nil {
		return err
	}
	hh.gotRawRLPToDecode = r
	*h = hh.setHeaderToOnUnmarshalOrDecode
	return hh.errDecode
}

func (hh *stubHeaderHooks) PostCopy(dst *Header) {
	hh.accessor.Set(dst, hh.toCopy)
}

func TestHeaderHooks(t *testing.T) {
	TestOnlyClearRegisteredExtras()
	defer TestOnlyClearRegisteredExtras()

	extras := RegisterExtras[
		stubHeaderHooks, *stubHeaderHooks,
		NOOPBlockBodyHooks, *NOOPBlockBodyHooks,
		struct{},
	]()
	rng := ethtest.NewPseudoRand(13579)

	suffix := rng.Bytes(8)
	hdr := &Header{
		ParentHash: rng.Hash(),
	}
	extras.Header.Get(hdr).suffix = append([]byte{}, suffix...)

	t.Run("MarshalJSON", func(t *testing.T) {
		got, err := json.Marshal(hdr)
		require.NoError(t, err, "json.Marshal(%T)", hdr)
		assert.Equal(t, fakeHeaderJSON(hdr, suffix), got)
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		hdr := new(Header)
		stub := &stubHeaderHooks{
			setHeaderToOnUnmarshalOrDecode: Header{
				Extra: []byte("can you solve this puzzle? 0xbda01b6cf56c303bd3f581599c0d5c0b"),
			},
		}
		extras.Header.Set(hdr, stub)

		input := fmt.Sprintf("%q", "hello, JSON world")
		err := json.Unmarshal([]byte(input), hdr)
		require.NoErrorf(t, err, "json.Unmarshal()")

		assert.Equal(t, input, string(stub.gotRawJSONToUnmarshal), "raw JSON received by hook")
		assert.Equal(t, &stub.setHeaderToOnUnmarshalOrDecode, hdr, "%T after JSON unmarshalling with hook", hdr)
	})

	t.Run("EncodeRLP", func(t *testing.T) {
		got, err := rlp.EncodeToBytes(hdr)
		require.NoError(t, err, "rlp.EncodeToBytes(%T)", hdr)
		assert.Equal(t, fakeHeaderRLP(hdr, suffix), got)
	})

	t.Run("DecodeRLP", func(t *testing.T) {
		input, err := rlp.EncodeToBytes(rng.Bytes(8))
		require.NoError(t, err)

		hdr := new(Header)
		stub := &stubHeaderHooks{
			setHeaderToOnUnmarshalOrDecode: Header{
				Extra: []byte("arr4n was here"),
			},
		}
		extras.Header.Set(hdr, stub)
		err = rlp.DecodeBytes(input, hdr)
		require.NoErrorf(t, err, "rlp.DecodeBytes(%#x)", input)

		assert.Equal(t, input, stub.gotRawRLPToDecode, "raw RLP received by hooks")
		assert.Equalf(t, &stub.setHeaderToOnUnmarshalOrDecode, hdr, "%T after RLP decoding with hook", hdr)
	})

	t.Run("PostCopy", func(t *testing.T) {
		hdr := new(Header)
		stub := &stubHeaderHooks{
			accessor: extras.Header,
			toCopy: &stubHeaderHooks{
				suffix: []byte("copied"),
			},
		}
		extras.Header.Set(hdr, stub)

		got := extras.Header.Get(CopyHeader(hdr))
		assert.Equal(t, stub.toCopy, got)
	})

	t.Run("error_propagation", func(t *testing.T) {
		errMarshal := errors.New("whoops")
		errUnmarshal := errors.New("is it broken?")
		errEncode := errors.New("uh oh")
		errDecode := errors.New("something bad happened")

		hdr := new(Header)
		setStub := func() {
			extras.Header.Set(hdr, &stubHeaderHooks{
				errMarshal:   errMarshal,
				errUnmarshal: errUnmarshal,
				errEncode:    errEncode,
				errDecode:    errDecode,
			})
		}

		setStub()
		// The { } blocks are defensive, avoiding accidentally having the wrong
		// error checked in a future refactor. The verbosity is acceptable for
		// clarity in tests.
		{
			_, err := json.Marshal(hdr)
			assert.ErrorIs(t, err, errMarshal, "via json.Marshal()") //nolint:testifylint // require is inappropriate here as we wish to keep going
		}
		{
			err := json.Unmarshal([]byte("{}"), hdr)
			assert.Equal(t, errUnmarshal, err, "via json.Unmarshal()")
		}

		setStub() // [stubHeaderHooks] completely overrides the Header
		{
			err := rlp.Encode(io.Discard, hdr)
			assert.Equal(t, errEncode, err, "via rlp.Encode()")
		}
		{
			err := rlp.DecodeBytes([]byte{0}, hdr)
			assert.Equal(t, errDecode, err, "via rlp.DecodeBytes()")
		}
	})
}

type blockPayload struct {
	NOOPBlockBodyHooks
	x int
}

func (p *blockPayload) Copy() *blockPayload {
	return &blockPayload{x: p.x}
}

func TestBlockWithX(t *testing.T) {
	TestOnlyClearRegisteredExtras()
	t.Cleanup(TestOnlyClearRegisteredExtras)

	extras := RegisterExtras[
		NOOPHeaderHooks, *NOOPHeaderHooks,
		blockPayload, *blockPayload,
		struct{},
	]()

	typ := reflect.TypeOf(&Block{})
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i).Name
		if method == "Withdrawals" || !strings.HasPrefix(method, "With") {
			continue
		}

		block := NewBlockWithHeader(&Header{})
		const initialPayload = int(42)
		payload := &blockPayload{
			x: initialPayload,
		}
		extras.Block.Set(block, payload)

		t.Run(method, func(t *testing.T) {
			var newBlock *Block

			switch method {
			case "WithBody":
				var body Body
				extras.Body.Set(&body, payload)
				newBlock = block.WithBody(body)
			case "WithSeal":
				newBlock = block.WithSeal(&Header{})
			case "WithWithdrawals":
				newBlock = block.WithWithdrawals(nil)
			default:
				t.Fatalf("method call not implemented: %s", method)
			}

			payload.x++
			// This specifically uses `require` instead of `assert` because a
			// failure here invalidates the next test, which demonstrates a deep
			// copy.
			require.Equalf(t, initialPayload+1, extras.Block.Get(block).x, "%T payload %T after modification via pointer", block, payload)

			switch got := extras.Block.Get(newBlock); got.x {
			case initialPayload: // expected
			case 0:
				t.Errorf("%T payload %T got zero value; the payload was probably not copied, resulting in a default being created", newBlock, got)
			case initialPayload + 1:
				t.Errorf("%T payload %T got same value as modified original; the payload was probably shallow copied", newBlock, got)
			default:
				t.Errorf("%T payload %T got %d, want %d; this is unexpected even as an error so you're on your own here", newBlock, got, got.x, initialPayload)
			}
		})
	}
}

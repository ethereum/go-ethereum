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
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/crypto"
	"github.com/ava-labs/libevm/libevm/ethtest"
	"github.com/ava-labs/libevm/rlp"
)

type stubHeaderHooks struct {
	rlpSuffix           []byte
	gotRawRLPToDecode   []byte
	setHeaderToOnDecode Header

	errEncode, errDecode error
}

func fakeHeaderRLP(h *Header, suffix []byte) []byte {
	return append(crypto.Keccak256(h.ParentHash[:]), suffix...)
}

func (hh *stubHeaderHooks) EncodeRLP(h *Header, w io.Writer) error {
	if _, err := w.Write(fakeHeaderRLP(h, hh.rlpSuffix)); err != nil {
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
	*h = hh.setHeaderToOnDecode
	return hh.errDecode
}

func TestHeaderHooks(t *testing.T) {
	TestOnlyClearRegisteredExtras()
	defer TestOnlyClearRegisteredExtras()

	extras := RegisterExtras[stubHeaderHooks, *stubHeaderHooks, struct{}]()
	rng := ethtest.NewPseudoRand(13579)

	t.Run("EncodeRLP", func(t *testing.T) {
		suffix := rng.Bytes(8)

		hdr := &Header{
			ParentHash: rng.Hash(),
		}
		extras.Header.Get(hdr).rlpSuffix = append([]byte{}, suffix...)

		got, err := rlp.EncodeToBytes(hdr)
		require.NoError(t, err, "rlp.EncodeToBytes(%T)", hdr)
		assert.Equal(t, fakeHeaderRLP(hdr, suffix), got)
	})

	t.Run("DecodeRLP", func(t *testing.T) {
		input, err := rlp.EncodeToBytes(rng.Bytes(8))
		require.NoError(t, err)

		hdr := new(Header)
		stub := &stubHeaderHooks{
			setHeaderToOnDecode: Header{
				Extra: []byte("arr4n was here"),
			},
		}
		extras.Header.Set(hdr, stub)
		err = rlp.DecodeBytes(input, hdr)
		require.NoErrorf(t, err, "rlp.DecodeBytes(%#x)", input)

		assert.Equal(t, input, stub.gotRawRLPToDecode, "raw RLP received by hooks")
		assert.Equalf(t, &stub.setHeaderToOnDecode, hdr, "%T after RLP decoding with hook", hdr)
	})

	t.Run("error_propagation", func(t *testing.T) {
		errEncode := errors.New("uh oh")
		errDecode := errors.New("something bad happened")

		hdr := new(Header)
		extras.Header.Set(hdr, &stubHeaderHooks{
			errEncode: errEncode,
			errDecode: errDecode,
		})

		assert.Equal(t, errEncode, rlp.Encode(io.Discard, hdr), "via rlp.Encode()")
		assert.Equal(t, errDecode, rlp.DecodeBytes([]byte{0}, hdr), "via rlp.DecodeBytes()")
	})
}

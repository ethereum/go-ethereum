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

package abi

import (
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/crypto"
)

func TestEventPackingRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		abiJSON      string
		eventName    string
		args         []any
		wantTopics   []common.Hash
		wantData     []byte
		wantUnpacked any // MUST be a pointer
	}{
		{
			name: "received",
			abiJSON: `[{
				"type": "event",
				"name": "received",
				"anonymous": false,
				"inputs": [
					{"indexed": false, "name": "sender", "type": "address"},
					{"indexed": false, "name": "amount", "type": "uint256"},
					{"indexed": false, "name": "memo", "type": "bytes"}
				]
			}]`,
			eventName: "received",
			args: []any{
				common.HexToAddress("0x376c47978271565f56DEB45495afa69E59c16Ab2"),
				big.NewInt(1),
				[]byte{0x88},
			},
			wantTopics: []common.Hash{
				crypto.Keccak256Hash([]byte("received(address,uint256,bytes)")),
			},
			wantData: common.Hex2Bytes("000000000000000000000000376c47978271565f56deb45495afa69e59c16ab20000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000018800000000000000000000000000000000000000000000000000000000000000"),
			wantUnpacked: &struct {
				Sender common.Address
				Amount *big.Int
				Memo   []byte
			}{
				common.HexToAddress("0x376c47978271565f56DEB45495afa69E59c16Ab2"),
				big.NewInt(1),
				[]byte{0x88},
			},
		},
		{
			name: "anonymous",
			abiJSON: `[{
				"type": "event",
				"name": "received",
				"anonymous": true,
				"inputs": [
					{"indexed": false, "name": "sender", "type": "address"},
					{"indexed": false, "name": "amount", "type": "uint256"},
					{"indexed": false, "name": "memo", "type": "bytes"}
				]
			}]`,
			eventName: "received",
			args: []any{
				common.HexToAddress("0x376c47978271565f56DEB45495afa69E59c16Ab2"),
				big.NewInt(1),
				[]byte{0x88},
			},
			wantTopics: nil,
			wantData:   common.Hex2Bytes("000000000000000000000000376c47978271565f56deb45495afa69e59c16ab20000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000018800000000000000000000000000000000000000000000000000000000000000"),
			wantUnpacked: &struct {
				Sender common.Address
				Amount *big.Int
				Memo   []byte
			}{
				common.HexToAddress("0x376c47978271565f56DEB45495afa69E59c16Ab2"),
				big.NewInt(1),
				[]byte{0x88},
			},
		},
		{
			name: "Transfer",
			abiJSON: `[{
				"type": "event",
				"name": "Transfer",
				"anonymous": false,
				"inputs": [
					{"indexed": true, "name": "from", "type": "address"},
					{"indexed": true, "name": "to", "type": "address"},
					{"indexed": false, "name": "value", "type": "uint256"}
				]
			}]`,
			eventName: "Transfer",
			args: []any{
				common.HexToAddress("0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"),
				common.HexToAddress("0x376c47978271565f56DEB45495afa69E59c16Ab2"),
				big.NewInt(100),
			},
			wantTopics: []common.Hash{
				crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")),
				common.HexToHash("0x0000000000000000000000008db97c7cece249c2b98bdc0226cc4c2a57bf52fc"),
				common.HexToHash("0x000000000000000000000000376c47978271565f56deb45495afa69e59c16ab2"),
			},
			wantData: common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000064"),
			wantUnpacked: &struct {
				Value *big.Int
			}{
				big.NewInt(100),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			abi, err := JSON(strings.NewReader(test.abiJSON))
			require.NoErrorf(t, err, "JSON(%s)", test.abiJSON)

			t.Run("pack", func(t *testing.T) {
				topics, data, err := abi.PackEvent(test.eventName, test.args...)
				require.NoErrorf(t, err, "%T.PackEvent(%q, %v...)", abi, test.eventName, test.args)

				assert.Equal(t, test.wantTopics, topics, "topics")
				assert.Equal(t, test.wantData, data, "data")
			})

			t.Run("unpack", func(t *testing.T) {
				typ := reflect.TypeOf(test.wantUnpacked)
				require.Equal(t, reflect.Pointer, typ.Kind(), "unpacking type MUST be a pointer")

				got := reflect.New(typ.Elem()).Interface()
				require.NoError(t, abi.UnpackInputIntoInterface(got, test.eventName, test.wantData))

				if diff := cmp.Diff(test.wantUnpacked, got, compareBigInts()); diff != "" {
					t.Errorf("%T.UnpackInputIntoInterface(%T) diff (-want +got):\n%s", abi, got, diff)
				}
			})
		})
	}
}

// receiveFuncInput matches the input signature of the "receive" method defined
// by [receiveFuncABI].
type receiveFuncInput struct {
	Sender common.Address
	Amount *big.Int
	Memo   []byte
}

var receiveFuncABI ABI

func init() {
	var err error
	receiveFuncABI, err = JSON(strings.NewReader(`
[{
  "type": "function",
  "name": "receive",
  "inputs": [
    {
      "name": "sender",
      "type": "address"
    },
    {
      "name": "amount",
      "type": "uint256"
    },
    {
      "name": "memo",
      "type": "bytes"
    }
  ],
  "outputs": [
    {
      "name": "isAllowed",
      "type": "bool"
    },
    {
      "name": "randomNumber",
      "type": "uint64"
    }
  ]
}]
`))
	if err != nil {
		panic(err)
	}
}

func TestUnpackInputIntoInterface(t *testing.T) {
	tests := []struct {
		name              string
		extraPaddingBytes int
	}{
		{
			name: "No extra padding to input data",
		},
		{
			name:              "Valid input data with 32 extra bytes",
			extraPaddingBytes: 32,
		},
		{
			name:              "Valid input data with 64 extra bytes",
			extraPaddingBytes: 64,
		},
		{
			name:              "Valid input data with 33 extra bytes",
			extraPaddingBytes: 33,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			abi := receiveFuncABI
			const method = "receive"

			input := receiveFuncInput{
				Sender: common.Address{2},
				Amount: big.NewInt(100),
				Memo:   []byte("hello"),
			}

			args := []any{input.Sender, input.Amount, input.Memo}
			packed, err := abi.Pack(method, args...)
			require.NoErrorf(t, err, "%T.Pack(%q, %v...)", abi, method, args)

			// skip 4 byte selector
			data := append(packed[4:], make([]byte, test.extraPaddingBytes)...)

			var got receiveFuncInput
			require.NoErrorf(t, abi.UnpackInputIntoInterface(&got, method, data), "%T.UnpackInputIntoInterface()", abi)

			if diff := cmp.Diff(input, got, compareBigInts()); diff != "" {
				t.Errorf("%T.Pack() -> %T.UnpackInputIntoInterface(%T, ...) round-trip diff (-want +got):\n%s", abi, abi, got, diff)
			}
		})
	}
}

func TestPackOutput(t *testing.T) {
	abi := receiveFuncABI
	const (
		method       = "receive"
		boolReturn   = true
		uint64Return = uint64(42)
	)
	want := []any{boolReturn, uint64Return}

	packed, err := abi.PackOutput(method, boolReturn, uint64Return)
	require.NoErrorf(t, err, "%T.PackOutput(%q, %v, %v)", abi, method, boolReturn, uint64Return)

	m := abi.Methods["receive"]
	got, err := m.Outputs.Unpack(packed)
	require.NoErrorf(t, err, "%T.Outputs.Unpack(%T.PackOutput())", m, abi)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("%T.PackOutput() -> %T.Outputs.Unpack() round-trip diff (-want +got):\n%s", abi, m, diff)
	}
}

func compareBigInts() cmp.Option {
	return cmp.Comparer(func(a, b *big.Int) bool {
		switch aN, bN := a == nil, b == nil; {
		case aN != bN:
			return false
		case aN && bN:
			return true
		default:
			return a.Cmp(b) == 0
		}
	})
}

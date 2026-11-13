// Copyright 2019 The go-ethereum Authors
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

package apitypes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBytesPadding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Type   string
		Input  []byte
		Output []byte // nil => error
	}{
		{
			// Fail on wrong length
			Type:   "bytes20",
			Input:  []byte{},
			Output: nil,
		},
		{
			Type:   "bytes1",
			Input:  []byte{1},
			Output: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			Type:   "bytes1",
			Input:  []byte{1, 2},
			Output: nil,
		},
		{
			Type:   "bytes7",
			Input:  []byte{1, 2, 3, 4, 5, 6, 7},
			Output: []byte{1, 2, 3, 4, 5, 6, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			Type:   "bytes32",
			Input:  []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			Output: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		},
		{
			Type:   "bytes32",
			Input:  []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33},
			Output: nil,
		},
	}

	d := TypedData{}
	for i, test := range tests {
		val, err := d.EncodePrimitiveValue(test.Type, test.Input, 1)
		if test.Output == nil {
			if err == nil {
				t.Errorf("test %d: expected error, got no error (result %x)", i, val)
			}
		} else {
			if err != nil {
				t.Errorf("test %d: expected no error, got %v", i, err)
			}
			if len(val) != 32 {
				t.Errorf("test %d: expected len 32, got %d", i, len(val))
			}
			if !bytes.Equal(val, test.Output) {
				t.Errorf("test %d: expected %x, got %x", i, test.Output, val)
			}
		}
	}
}

func TestParseAddress(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Input  interface{}
		Output []byte // nil => error
	}{
		{
			Input:  [20]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, 0x11, 0x12, 0x13, 0x14},
			Output: common.FromHex("0x0000000000000000000000000102030405060708090A0B0C0D0E0F1011121314"),
		},
		{
			Input:  "0x0102030405060708090A0B0C0D0E0F1011121314",
			Output: common.FromHex("0x0000000000000000000000000102030405060708090A0B0C0D0E0F1011121314"),
		},
		{
			Input:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, 0x11, 0x12, 0x13, 0x14},
			Output: common.FromHex("0x0000000000000000000000000102030405060708090A0B0C0D0E0F1011121314"),
		},
		// Various error-cases:
		{Input: "0x000102030405060708090A0B0C0D0E0F1011121314"}, // too long string
		{Input: "0x01"}, // too short string
		{Input: ""},
		{Input: [32]byte{}},       // too long fixed-size array
		{Input: [21]byte{}},       // too long fixed-size array
		{Input: make([]byte, 19)}, // too short slice
		{Input: make([]byte, 21)}, // too long slice
		{Input: nil},
	}

	d := TypedData{}
	for i, test := range tests {
		val, err := d.EncodePrimitiveValue("address", test.Input, 1)
		if test.Output == nil {
			if err == nil {
				t.Errorf("test %d: expected error, got no error (result %x)", i, val)
			}
			continue
		}
		if err != nil {
			t.Errorf("test %d: expected no error, got %v", i, err)
		}
		if have, want := len(val), 32; have != want {
			t.Errorf("test %d: have len %d, want %d", i, have, want)
		}
		if !bytes.Equal(val, test.Output) {
			t.Errorf("test %d: want %x, have %x", i, test.Output, val)
		}
	}
}

func TestParseBytes(t *testing.T) {
	t.Parallel()
	for i, tt := range []struct {
		v   interface{}
		exp []byte
	}{
		{"0x", []byte{}},
		{"0x1234", []byte{0x12, 0x34}},
		{[]byte{12, 34}, []byte{12, 34}},
		{hexutil.Bytes([]byte{12, 34}), []byte{12, 34}},
		{"1234", nil},    // not a proper hex-string
		{"0x01233", nil}, // nibbles should be rejected
		{"not a hex string", nil},
		{15, nil},
		{nil, nil},
		{[2]byte{12, 34}, []byte{12, 34}},
		{[8]byte{12, 34, 56, 78, 90, 12, 34, 56}, []byte{12, 34, 56, 78, 90, 12, 34, 56}},
		{[16]byte{12, 34, 56, 78, 90, 12, 34, 56, 12, 34, 56, 78, 90, 12, 34, 56}, []byte{12, 34, 56, 78, 90, 12, 34, 56, 12, 34, 56, 78, 90, 12, 34, 56}},
	} {
		out, ok := parseBytes(tt.v)
		if tt.exp == nil {
			if ok || out != nil {
				t.Errorf("test %d: expected !ok, got ok = %v with out = %x", i, ok, out)
			}
			continue
		}
		if !ok {
			t.Errorf("test %d: expected ok got !ok", i)
		}
		if !bytes.Equal(out, tt.exp) {
			t.Errorf("test %d: expected %x got %x", i, tt.exp, out)
		}
	}
}

func TestParseInteger(t *testing.T) {
	t.Parallel()
	for i, tt := range []struct {
		t   string
		v   interface{}
		exp *big.Int
	}{
		{"uint32", "-123", nil},
		{"int32", "-123", big.NewInt(-123)},
		{"int32", big.NewInt(-124), big.NewInt(-124)},
		{"uint32", "0xff", big.NewInt(0xff)},
		{"int8", "0xffff", nil},
	} {
		res, err := parseInteger(tt.t, tt.v)
		if tt.exp == nil && res == nil {
			continue
		}
		if tt.exp == nil && res != nil {
			t.Errorf("test %d, got %v, expected nil", i, res)
			continue
		}
		if tt.exp != nil && res == nil {
			t.Errorf("test %d, got '%v', expected %v", i, err, tt.exp)
			continue
		}
		if tt.exp.Cmp(res) != 0 {
			t.Errorf("test %d, got %v expected %v", i, res, tt.exp)
		}
	}
}

func TestConvertStringDataToSlice(t *testing.T) {
	t.Parallel()
	slice := []string{"a", "b", "c"}
	var it interface{} = slice
	_, err := convertDataToSlice(it)
	if err != nil {
		t.Fatal(err)
	}
}

func TestConvertUint256DataToSlice(t *testing.T) {
	t.Parallel()
	slice := []*math.HexOrDecimal256{
		math.NewHexOrDecimal256(1),
		math.NewHexOrDecimal256(2),
		math.NewHexOrDecimal256(3),
	}
	var it interface{} = slice
	_, err := convertDataToSlice(it)
	if err != nil {
		t.Fatal(err)
	}
}

func TestConvertAddressDataToSlice(t *testing.T) {
	t.Parallel()
	slice := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
		common.HexToAddress("0x0000000000000000000000000000000000000003"),
	}
	var it interface{} = slice
	_, err := convertDataToSlice(it)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTypedDataArrayValidate(t *testing.T) {
	t.Parallel()

	type testDataInput struct {
		Name        string           `json:"name"`
		Domain      TypedDataDomain  `json:"domain"`
		PrimaryType string           `json:"primaryType"`
		Types       Types            `json:"types"`
		Message     TypedDataMessage `json:"data"`
		Digest      string           `json:"digest"`
	}
	fc, err := os.ReadFile("./testdata/typed-data.json")
	require.NoError(t, err, "error reading test data file")

	var tests []testDataInput
	err = json.Unmarshal(fc, &tests)
	require.NoError(t, err, "error unmarshalling test data file contents")

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			td := TypedData{
				Types:       tc.Types,
				PrimaryType: tc.PrimaryType,
				Domain:      tc.Domain,
				Message:     tc.Message,
			}

			domainSeparator, tErr := td.HashStruct("EIP712Domain", td.Domain.Map())
			assert.NoError(t, tErr, "failed to hash domain separator: %v", tErr)

			messageHash, tErr := td.HashStruct(td.PrimaryType, td.Message)
			assert.NoError(t, tErr, "failed to hash message: %v", tErr)

			digest := crypto.Keccak256Hash(fmt.Appendf(nil, "%s%s%s", "\x19\x01", string(domainSeparator), string(messageHash)))
			assert.Equal(t, tc.Digest, digest.String(), "digest doesn't not match")

			assert.NoError(t, td.validate(), "validation failed", tErr)
		})
	}
}

func TestTypedDataArrayShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		typ    string
		value  string // JSON
		digest string // empty => error
	}{
		// Values matching their declared type, digests from @metamask/eth-sig-util.
		{"uint256[]", `[1,2]`, "0x9c9967ec3784119776f1457211397f2966aa0f79373ec299bfe0219b5a8cbc89"},
		{"uint256[][]", `[[1,2],[3]]`, "0x48a56be876dad2e5a6e25ad5b49c6ca74ba8769120794ea088b77acd7d114ff3"},
		{"uint256[2][]", `[[1,2],[3,4]]`, "0x25925b83064f3bb2cdb7408af9518f455ba7dd4ead44e0812f72a3ab8f99dee2"},
		{"uint256[][2]", `[[1],[2,3]]`, "0xe264c687826045a2010aa525e28a8045e60a005cc303fedd6e8a247f616955f5"},
		{"uint256[2]", `[1,2]`, "0x2a5075318e6f411c6742ad68d80c9cfd110d52466f3a6ed94f97992d02f67a87"},
		{"P[1]", `[{"v":1}]`, "0x57bc77817c84e0c122f0ccd22610a1b784bfa827ac87e0d7f6267ebfb2510010"},
		{"P[][]", `[[{"v":1}],[{"v":2},{"v":3}]]`, "0x7899f85c190f86609dda7ac98f56b035f5a5e146b65c0e2c412ba309413770a7"},
		{"bytes[2][]", `[["0x01","0x0203"]]`, "0x2fa52b6daf335717fd28a4e7e3dfd81336bfd784b5fb9531b98a193302ca841e"},
		{"bytes2[]", `["0x0102"]`, "0xae54038a0cc257005234ad15f28098a1f1ad079047fe5eb457e8aa7ca552af3f"},

		// Null elements.
		{"uint256[]", `[null]`, ""},
		{"P[]", `[null]`, ""},
		{"uint256[][]", `[[1],null]`, ""},

		// Wrong number of dimensions.
		{"uint256[]", `[[1,2],[3]]`, ""},
		{"uint256[][]", `[1,2]`, ""},
		{"P[]", `[[{"v":1}]]`, ""},
		{"P[][]", `[{"v":1}]`, ""},

		// Wrong fixed length.
		{"uint256[2]", `[1]`, ""},
		{"uint256[2]", `[1,2,3]`, ""},
		{"P[1]", `[{"v":1},{"v":2}]`, ""},
		{"uint256[2][]", `[[1,2],[3]]`, ""},
		{"uint256[][2]", `[[1]]`, ""},

		// Malformed array types.
		{"uint256[x]", `[1]`, ""},
		{"uint256[x][]", `[]`, ""},
	}
	for i, test := range tests {
		var value interface{}
		if err := json.Unmarshal([]byte(test.value), &value); err != nil {
			t.Fatalf("test %d: invalid JSON: %v", i, err)
		}
		digest, err := arrayShapeDigest(test.typ, value)
		if test.digest == "" {
			if err == nil {
				t.Errorf("test %d (%s %s): expected error, got digest %x", i, test.typ, test.value, digest)
			}
			continue
		}
		if err != nil {
			t.Errorf("test %d (%s %s): unexpected error: %v", i, test.typ, test.value, err)
		} else if have := hexutil.Encode(digest); have != test.digest {
			t.Errorf("test %d (%s %s): digest mismatch: have %s, want %s", i, test.typ, test.value, have, test.digest)
		}
	}

	// Go byte types stay primitive values inside arrays of bytesN.
	want := "0xae54038a0cc257005234ad15f28098a1f1ad079047fe5eb457e8aa7ca552af3f"
	for _, item := range []interface{}{[]byte{1, 2}, hexutil.Bytes{1, 2}, [2]byte{1, 2}} {
		digest, err := arrayShapeDigest("bytes2[]", []interface{}{item})
		if err != nil {
			t.Errorf("bytes2[] with %T: unexpected error: %v", item, err)
		} else if have := hexutil.Encode(digest); have != want {
			t.Errorf("bytes2[] with %T: digest mismatch: have %s, want %s", item, have, want)
		}
	}
}

func arrayShapeDigest(typ string, value interface{}) ([]byte, error) {
	chainId := math.HexOrDecimal256(*big.NewInt(1))
	td := TypedData{
		Types: Types{
			"EIP712Domain": {{Name: "name", Type: "string"}, {Name: "version", Type: "string"}, {Name: "chainId", Type: "uint256"}},
			"P":            {{Name: "v", Type: "uint256"}},
			"M":            {{Name: "x", Type: typ}},
		},
		PrimaryType: "M",
		Domain:      TypedDataDomain{Name: "Shape", Version: "1", ChainId: &chainId},
		Message:     TypedDataMessage{"x": value},
	}
	digest, _, err := TypedDataAndHash(td)
	return digest, err
}

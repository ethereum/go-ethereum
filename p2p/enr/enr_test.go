// Copyright 2017 The go-ethereum Authors
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

package enr

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	privkey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	pubkey     = &privkey.PublicKey
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func randomString(strlen int) string {
	b := make([]byte, strlen)
	rnd.Read(b)
	return string(b)
}

// TestGetSetID tests encoding/decoding and setting/getting of the ID key.
func TestGetSetID(t *testing.T) {
	id := ID("someid")
	var r Record
	r.Set(id)

	var id2 ID
	require.NoError(t, r.Load(&id2))
	assert.Equal(t, id, id2)
}

// TestGetSetIP4 tests encoding/decoding and setting/getting of the IP key.
func TestGetSetIP4(t *testing.T) {
	ip := IP{192, 168, 0, 3}
	var r Record
	r.Set(ip)

	var ip2 IP
	require.NoError(t, r.Load(&ip2))
	assert.Equal(t, ip, ip2)
}

// TestGetSetIP6 tests encoding/decoding and setting/getting of the IP key.
func TestGetSetIP6(t *testing.T) {
	ip := IP{0x20, 0x01, 0x48, 0x60, 0, 0, 0x20, 0x01, 0, 0, 0, 0, 0, 0, 0x00, 0x68}
	var r Record
	r.Set(ip)

	var ip2 IP
	require.NoError(t, r.Load(&ip2))
	assert.Equal(t, ip, ip2)
}

// TestGetSetDiscPort tests encoding/decoding and setting/getting of the DiscPort key.
func TestGetSetUDP(t *testing.T) {
	port := UDP(30309)
	var r Record
	r.Set(port)

	var port2 UDP
	require.NoError(t, r.Load(&port2))
	assert.Equal(t, port, port2)
}

// TestGetSetSecp256k1 tests encoding/decoding and setting/getting of the Secp256k1 key.
func TestGetSetSecp256k1(t *testing.T) {
	var r Record
	if err := SignV4(&r, privkey); err != nil {
		t.Fatal(err)
	}

	var pk Secp256k1
	require.NoError(t, r.Load(&pk))
	assert.EqualValues(t, pubkey, &pk)
}

func TestLoadErrors(t *testing.T) {
	var r Record
	ip4 := IP{127, 0, 0, 1}
	r.Set(ip4)

	// Check error for missing keys.
	var udp UDP
	err := r.Load(&udp)
	if !IsNotFound(err) {
		t.Error("IsNotFound should return true for missing key")
	}
	assert.Equal(t, &KeyError{Key: udp.ENRKey(), Err: errNotFound}, err)

	// Check error for invalid keys.
	var list []uint
	err = r.Load(WithEntry(ip4.ENRKey(), &list))
	kerr, ok := err.(*KeyError)
	if !ok {
		t.Fatalf("expected KeyError, got %T", err)
	}
	assert.Equal(t, kerr.Key, ip4.ENRKey())
	assert.Error(t, kerr.Err)
	if IsNotFound(err) {
		t.Error("IsNotFound should return false for decoding errors")
	}
}

// TestSortedGetAndSet tests that Set produced a sorted pairs slice.
func TestSortedGetAndSet(t *testing.T) {
	type pair struct {
		k string
		v uint32
	}

	for _, tt := range []struct {
		input []pair
		want  []pair
	}{
		{
			input: []pair{{"a", 1}, {"c", 2}, {"b", 3}},
			want:  []pair{{"a", 1}, {"b", 3}, {"c", 2}},
		},
		{
			input: []pair{{"a", 1}, {"c", 2}, {"b", 3}, {"d", 4}, {"a", 5}, {"bb", 6}},
			want:  []pair{{"a", 5}, {"b", 3}, {"bb", 6}, {"c", 2}, {"d", 4}},
		},
		{
			input: []pair{{"c", 2}, {"b", 3}, {"d", 4}, {"a", 5}, {"bb", 6}},
			want:  []pair{{"a", 5}, {"b", 3}, {"bb", 6}, {"c", 2}, {"d", 4}},
		},
	} {
		var r Record
		for _, i := range tt.input {
			r.Set(WithEntry(i.k, &i.v))
		}
		for i, w := range tt.want {
			// set got's key from r.pair[i], so that we preserve order of pairs
			got := pair{k: r.pairs[i].k}
			assert.NoError(t, r.Load(WithEntry(w.k, &got.v)))
			assert.Equal(t, w, got)
		}
	}
}

// TestDirty tests record signature removal on setting of new key/value pair in record.
func TestDirty(t *testing.T) {
	var r Record

	if r.Signed() {
		t.Error("Signed returned true for zero record")
	}
	if _, err := rlp.EncodeToBytes(r); err != errEncodeUnsigned {
		t.Errorf("expected errEncodeUnsigned, got %#v", err)
	}

	require.NoError(t, SignV4(&r, privkey))
	if !r.Signed() {
		t.Error("Signed return false for signed record")
	}
	_, err := rlp.EncodeToBytes(r)
	assert.NoError(t, err)

	r.SetSeq(3)
	if r.Signed() {
		t.Error("Signed returned true for modified record")
	}
	if _, err := rlp.EncodeToBytes(r); err != errEncodeUnsigned {
		t.Errorf("expected errEncodeUnsigned, got %#v", err)
	}
}

// TestGetSetOverwrite tests value overwrite when setting a new value with an existing key in record.
func TestGetSetOverwrite(t *testing.T) {
	var r Record

	ip := IP{192, 168, 0, 3}
	r.Set(ip)

	ip2 := IP{192, 168, 0, 4}
	r.Set(ip2)

	var ip3 IP
	require.NoError(t, r.Load(&ip3))
	assert.Equal(t, ip2, ip3)
}

// TestSignEncodeAndDecode tests signing, RLP encoding and RLP decoding of a record.
func TestSignEncodeAndDecode(t *testing.T) {
	var r Record
	r.Set(UDP(30303))
	r.Set(IP{127, 0, 0, 1})
	require.NoError(t, SignV4(&r, privkey))

	blob, err := rlp.EncodeToBytes(r)
	require.NoError(t, err)

	var r2 Record
	require.NoError(t, rlp.DecodeBytes(blob, &r2))
	assert.Equal(t, r, r2)

	blob2, err := rlp.EncodeToBytes(r2)
	require.NoError(t, err)
	assert.Equal(t, blob, blob2)
}

func TestNodeAddr(t *testing.T) {
	var r Record
	if addr := r.NodeAddr(); addr != nil {
		t.Errorf("wrong address on empty record: got %v, want %v", addr, nil)
	}

	require.NoError(t, SignV4(&r, privkey))
	expected := "a448f24c6d18e575453db13171562b71999873db5b286df957af199ec94617f7"
	assert.Equal(t, expected, hex.EncodeToString(r.NodeAddr()))
}

var pyRecord, _ = hex.DecodeString("f884b8407098ad865b00a582051940cb9cf36836572411a47278783077011599ed5cd16b76f2635f4e234738f30813a89eb9137e3e3df5266e3a1f11df72ecf1145ccb9c01826964827634826970847f00000189736563703235366b31a103ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd31388375647082765f")

// TestPythonInterop checks that we can decode and verify a record produced by the Python
// implementation.
func TestPythonInterop(t *testing.T) {
	var r Record
	if err := rlp.DecodeBytes(pyRecord, &r); err != nil {
		t.Fatalf("can't decode: %v", err)
	}

	var (
		wantAddr, _ = hex.DecodeString("a448f24c6d18e575453db13171562b71999873db5b286df957af199ec94617f7")
		wantSeq     = uint64(1)
		wantIP      = IP{127, 0, 0, 1}
		wantUDP     = UDP(30303)
	)
	if r.Seq() != wantSeq {
		t.Errorf("wrong seq: got %d, want %d", r.Seq(), wantSeq)
	}
	if addr := r.NodeAddr(); !bytes.Equal(addr, wantAddr) {
		t.Errorf("wrong addr: got %x, want %x", addr, wantAddr)
	}
	want := map[Entry]interface{}{new(IP): &wantIP, new(UDP): &wantUDP}
	for k, v := range want {
		desc := fmt.Sprintf("loading key %q", k.ENRKey())
		if assert.NoError(t, r.Load(k), desc) {
			assert.Equal(t, k, v, desc)
		}
	}
}

// TestRecordTooBig tests that records bigger than SizeLimit bytes cannot be signed.
func TestRecordTooBig(t *testing.T) {
	var r Record
	key := randomString(10)

	// set a big value for random key, expect error
	r.Set(WithEntry(key, randomString(SizeLimit)))
	if err := SignV4(&r, privkey); err != errTooBig {
		t.Fatalf("expected to get errTooBig, got %#v", err)
	}

	// set an acceptable value for random key, expect no error
	r.Set(WithEntry(key, randomString(100)))
	require.NoError(t, SignV4(&r, privkey))
}

// TestSignEncodeAndDecodeRandom tests encoding/decoding of records containing random key/value pairs.
func TestSignEncodeAndDecodeRandom(t *testing.T) {
	var r Record

	// random key/value pairs for testing
	pairs := map[string]uint32{}
	for i := 0; i < 10; i++ {
		key := randomString(7)
		value := rnd.Uint32()
		pairs[key] = value
		r.Set(WithEntry(key, &value))
	}

	require.NoError(t, SignV4(&r, privkey))
	_, err := rlp.EncodeToBytes(r)
	require.NoError(t, err)

	for k, v := range pairs {
		desc := fmt.Sprintf("key %q", k)
		var got uint32
		buf := WithEntry(k, &got)
		require.NoError(t, r.Load(buf), desc)
		require.Equal(t, v, got, desc)
	}
}

func BenchmarkDecode(b *testing.B) {
	var r Record
	for i := 0; i < b.N; i++ {
		rlp.DecodeBytes(pyRecord, &r)
	}
	b.StopTimer()
	r.NodeAddr()
}

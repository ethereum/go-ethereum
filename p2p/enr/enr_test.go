// Copyright 2015 The go-ethereum Authors
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
	"math/rand"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	privkeyHex = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
)

var rnd *rand.Rand

func init() {
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// TestGetSetID tests encoding/decoding and setting/getting of the enr.ID type
func TestGetSetID(t *testing.T) {
	id := ID("someid")
	var r Record
	r.Set(id)

	var id2 ID

	_, err := r.Load(&id2)
	if err != nil {
		t.Fatal(err)
	}

	if id != id2 {
		t.Fatalf("got %#v, expected %#v", id2, id)
	}
}

// TestGetSetIP4 tests encoding/decoding and setting/getting of the enr.IP4 type
func TestGetSetIP4(t *testing.T) {
	ip := IP4(net.IP{192, 168, 0, 3})
	var r Record
	r.Set(ip)

	var ip2 IP4

	_, err := r.Load(&ip2)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(ip, ip2) != 0 {
		t.Fatalf("got %#v, expected %#v", ip2, ip)
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
			r.Set(WithKey(i.k, &i.v))
		}
		for i, w := range tt.want {
			// set got's key from r.pair[i], so that we preserve order of pairs
			got := pair{k: r.pairs[i].k}
			if ok, err := r.Load(WithKey(w.k, &got.v)); !ok || err != nil {
				t.Fatal(err)
			}

			if got != w {
				t.Fatalf("expected %#v, got %#v", w, got)
			}
		}
	}
}

// TestGetSetIP6 tests encoding/decoding and setting/getting of the enr.IP6 type
func TestGetSetIP6(t *testing.T) {
	ip := IP6(net.IP{0x20, 0x01, 0x48, 0x60, 0, 0, 0x20, 0x01, 0, 0, 0, 0, 0, 0, 0x00, 0x68})
	var r Record
	r.Set(ip)

	var ip2 IP6

	_, err := r.Load(&ip2)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(ip, ip2) != 0 {
		t.Fatalf("got %#v, expected %#v", ip2, ip)
	}
}

// TestGetSetDiscPort tests encoding/decoding and setting/getting of the enr.DiscPort type
func TestGetSetDiscPort(t *testing.T) {
	port := DiscPort(30309)
	var r Record
	r.Set(port)

	var port2 DiscPort

	_, err := r.Load(&port2)
	if err != nil {
		t.Fatal(err)
	}

	if port != port2 {
		t.Fatalf("got %#v, expected %#v", port2, port)
	}
}

// TestGetSetSecp256k1 tests encoding/decoding and setting/getting of the enr.Secp256k1 type
func TestGetSetSecp256k1(t *testing.T) {
	privkey, err := crypto.HexToECDSA(privkeyHex)
	if err != nil {
		t.Fatal(err)
	}

	var r Record

	err = r.Sign(privkey)
	if err != nil {
		t.Fatal(err)
	}

	var pk Secp256k1

	_, err = r.Load(&pk)
	if err != nil {
		t.Fatal(err)
	}

	got := (*btcec.PublicKey)(&pk).SerializeCompressed()
	expected := (*btcec.PublicKey)(&privkey.PublicKey).SerializeCompressed()
	if bytes.Compare(got, expected) != 0 {
		t.Fatalf("got %#v, expected %#v", got, expected)
	}
}

// TestDirty tests record signature removal on setting of new key/value pair in record.
func TestDirty(t *testing.T) {
	privkey, err := crypto.HexToECDSA(privkeyHex)
	if err != nil {
		t.Fatal(err)
	}

	var r Record

	err = r.Sign(privkey)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := rlp.EncodeToBytes(r); err != nil {
		t.Fatal(err)
	}

	r.SetSeq(3)

	if _, err := rlp.EncodeToBytes(r); err == nil {
		t.Fatal("expected err, got nil")
	}
}

// TestGetSetOverwrite tests value overwrite when setting a new value with an existing key in record.
func TestGetSetOverwrite(t *testing.T) {
	var r Record

	ip := IP4(net.IP{192, 168, 0, 3})
	r.Set(ip)

	ip2 := IP4(net.IP{192, 168, 0, 4})
	r.Set(ip2)

	var ip3 IP4

	_, err := r.Load(&ip3)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(ip2, ip3) != 0 {
		t.Fatalf("got %#v, expected %#v", ip2, ip3)
	}
}

// TestSignEncodeAndDecode tests signing, RLP encoding and RLP decoding of a record.
func TestSignEncodeAndDecode(t *testing.T) {
	privkey, err := crypto.HexToECDSA(privkeyHex)
	if err != nil {
		t.Fatal(err)
	}

	var r Record
	port := DiscPort(30303)
	r.Set(port)

	ipv4 := IP4(net.ParseIP("127.0.0.1"))
	r.Set(ipv4)

	err = r.Sign(privkey)
	if err != nil {
		t.Fatal(err)
	}

	blob, err := rlp.EncodeToBytes(r)
	if err != nil {
		t.Fatal(err)
	}

	var r2 Record
	err = rlp.DecodeBytes(blob, &r2)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(r, r2) {
		t.Errorf("records not deep equal ; got\n%#v, expected\n%#v", r2, r)
	}

	blob2, err := rlp.EncodeToBytes(r2)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(blob, blob2) != 0 {
		t.Errorf("serialised records not equal ; got\n%#v, expected\n%#v", blob2, blob)
	}
}

// TestNodeAddress tests that record returns correct node address - keccak256 hash of the public key.
func TestNodeAddress(t *testing.T) {
	privkey, err := crypto.HexToECDSA(privkeyHex)
	if err != nil {
		t.Fatal(err)
	}

	var r Record

	err = r.Sign(privkey)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := r.NodeAddr()
	if err != nil {
		t.Fatal(err)
	}

	expected := "caaa1485d83b18b32ed9ad666026151bf0cae8a0a88c857ae2d4c5be2daa6726"
	got := hex.EncodeToString(addr)
	if got != expected {
		t.Errorf("got\n%#v, expected\n%#v", got, expected)
	}
}

// TestPythonInterop tests that Go implementation can successfully RLP decode a record produced by Python implementation.
func TestPythonInterop(t *testing.T) {
	enc, _ := hex.DecodeString("f896b840638a54215d80a6713c8d523a6adc4e6e73652d859103a36b700851cb0e61b66b8ebfc1a610c57d732ec6e0a8f06a9a7a28df5051ece514702ff9cdff0b11f454018664697363763582765f82696490736563703235366b312d6b656363616b83697034847f00000189736563703235366b31a103ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd3138")
	var r Record
	if err := rlp.DecodeBytes(enc, &r); err != nil {
		t.Fatalf("can't decode: %v", err)
	}

	var (
		wantAddr, _  = hex.DecodeString("caaa1485d83b18b32ed9ad666026151bf0cae8a0a88c857ae2d4c5be2daa6726")
		wantSeq      = uint32(1)
		wantIP       = IP4(net.ParseIP("127.0.0.1").To4())
		wantDiscport = DiscPort(30303)
	)
	if r.Seq() != wantSeq {
		t.Errorf("wrong seq: got %d, want %d", r.Seq(), wantSeq)
	}
	if addr, _ := r.NodeAddr(); !bytes.Equal(addr, wantAddr) {
		t.Errorf("wrong addr: got %x, want %x", addr, wantAddr)
	}
	want := map[Key]interface{}{new(IP4): &wantIP, new(DiscPort): &wantDiscport}
	for k, v := range want {
		if _, err := r.Load(k); err != nil {
			t.Errorf("can't load %q: %v", k.ENRKey(), err)
		} else if !reflect.DeepEqual(k, v) {
			t.Errorf("wrong %q: got %v, want %v", k.ENRKey(), k, v)
		}
	}
}

// TestRecordTooBig tests that records bigger than SizeLimit bytes cannot be signed.
func TestRecordTooBig(t *testing.T) {
	privkey, err := crypto.HexToECDSA(privkeyHex)
	if err != nil {
		t.Fatal(err)
	}

	var r Record

	key := randomString(10)

	// set a big value for random key, expect error
	r.Set(WithKey(key, randomString(300)))
	err = r.Sign(privkey)
	if err != errTooBig {
		t.Fatalf("expected to get errTooBig, got %#v", err)
	}

	// set an acceptable value for random key, expect no error
	r.Set(WithKey(key, randomString(100)))
	err = r.Sign(privkey)
	if err != nil {
		t.Fatal(err)
	}
}

// TestSignEncodeAndDecodeRandom tests encoding/decoding of records containing random key/value pairs.
func TestSignEncodeAndDecodeRandom(t *testing.T) {
	privkey, err := crypto.HexToECDSA(privkeyHex)
	if err != nil {
		t.Fatal(err)
	}

	var r Record

	// random key/value pairs for testing
	pairs := map[string]uint32{}

	for i := 0; i < 10; i++ {
		key := randomString(7)
		value := rnd.Uint32()

		pair := WithKey(key, &value)
		r.Set(pair)

		pairs[key] = value
	}

	if r.Sign(privkey); err != nil {
		t.Fatal(err)
	}

	_, err = rlp.EncodeToBytes(r)
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range pairs {
		var got uint32
		buf := WithKey(k, &got)

		if ok, err := r.Load(buf); !ok || err != nil {
			t.Fatal(ok, err)
		}

		if got != v {
			t.Fatalf("got %#v, expected %#v", got, v)
		}
	}
}

func randomString(strlen int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := range result {
		result[i] = chars[rnd.Intn(len(chars))]
	}
	return string(result)
}

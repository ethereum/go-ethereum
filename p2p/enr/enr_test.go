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
	"net"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	privkeyHex = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
)

func TestGetSetID(t *testing.T) {
	id := "someid"
	e := NewENR()
	e.SetID(id)

	got, err := e.GetID()
	if err != nil {
		t.Fatalf("error: %#v", err)
	}

	if got != id {
		t.Fatalf("got %#v, expected %#v", got, id)
	}
}

func TestGetSetIP4(t *testing.T) {
	ip := net.IP{192, 168, 0, 3}
	e := NewENR()
	e.SetIPv4(ip)

	got, err := e.GetIPv4()
	if err != nil {
		t.Fatalf("error: %#v", err)
	}

	if !got.Equal(ip) {
		t.Fatalf("got %#v, expected %#v", got, ip)
	}
}

func TestGetSetIP6(t *testing.T) {
	ip := net.IP{0x20, 0x01, 0x48, 0x60, 0, 0, 0x20, 0x01, 0, 0, 0, 0, 0, 0, 0x00, 0x68}
	e := NewENR()
	e.SetIPv6(ip)

	got, err := e.GetIPv6()
	if err != nil {
		t.Fatalf("error: %#v", err)
	}

	if !got.Equal(ip) {
		t.Fatalf("got %#v, expected %#v", got, ip)
	}
}

func TestGetSetDiscv5(t *testing.T) {
	port := uint32(30309)
	e := NewENR()

	err := e.SetDiscv5(port)
	if err != nil {
		t.Fatalf("error: %#v", err)
	}

	got, err := e.GetDiscv5()
	if err != nil {
		t.Fatalf("error: %#v", err)
	}

	if got != port {
		t.Fatalf("got %#v, expected %#v", got, port)
	}
}

func TestGetSetSecp256k1(t *testing.T) {
	privkey, err := crypto.HexToECDSA(privkeyHex)
	if err != nil {
		t.Fatalf("error: %#v", err)
	}

	e := NewENR()

	err = e.Sign(privkey)
	if err != nil {
		t.Fatalf("error: %#v", err)
	}

	got, err := e.GetSecp256k1()
	if err != nil {
		t.Fatalf("error: %#v", err)
	}

	expected := (*btcec.PublicKey)(&privkey.PublicKey).SerializeCompressed()
	if bytes.Compare(got, expected) != 0 {
		t.Fatalf("got %#v, expected %#v", got, expected)
	}
}

func TestDirty(t *testing.T) {
	privkey, err := crypto.HexToECDSA(privkeyHex)
	if err != nil {
		t.Fatalf("error: %#v", err)
	}

	e := NewENR()

	err = e.Sign(privkey)
	if err != nil {
		t.Fatalf("error: %#v", err)
	}

	if _, err := e.Encode(); err != nil {
		t.Fatalf("error: %#v", err)
	}

	e.SetRaw([]byte(`some key`), []byte(`some value`))

	if _, err := e.Encode(); err == nil {
		t.Fatal("expected err, got nil")
	}
}

func TestSignEncodeAndDecode(t *testing.T) {
	privkey, err := crypto.HexToECDSA(privkeyHex)
	if err != nil {
		t.Fatal(err)
	}

	e := NewENR()
	e.SetDiscv5(30303)
	e.SetIPv4(net.ParseIP("127.0.0.1"))

	err = e.Sign(privkey)
	if err != nil {
		t.Fatal(err)
	}

	record, err := e.Encode()
	if err != nil {
		t.Fatal(err)
	}

	e2 := NewENR()

	err = e2.Decode(record)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(e, e2) {
		t.Errorf("got\n%#v, expected\n%#v", e2, e)
	}

	expectedRecord := "b8415571f9a36b1e26c366745894656dd1565033cdeda18d330c9a9bac67dfc3e786556b0490e509372fa9db5abf418accd895467e8ff047bbdc147789bef71a2cc401f8560186646973637635840000765f82696490736563703235366b312d6b656363616b83697034847f00000189736563703235366b31a103ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd3138"

	got := hex.EncodeToString(record)
	if got != expectedRecord {
		t.Errorf("got\n%#v, expected\n%#v", got, expectedRecord)
	}

	blob, err := e2.Encode()
	if err != nil {
		t.Fatal(err)
	}

	got = hex.EncodeToString(blob)
	if got != expectedRecord {
		t.Errorf("got\n%#v, expected\n%#v", got, expectedRecord)
	}
}

func TestNodeAddress(t *testing.T) {
	privkey, err := crypto.HexToECDSA(privkeyHex)
	if err != nil {
		t.Fatal(err)
	}

	e := NewENR()

	err = e.Sign(privkey)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := e.NodeAddress()
	if err != nil {
		t.Fatal(err)
	}

	expected := "caaa1485d83b18b32ed9ad666026151bf0cae8a0a88c857ae2d4c5be2daa6726"
	got := hex.EncodeToString(addr)
	if got != expected {
		t.Errorf("got\n%#v, expected\n%#v", got, expected)
	}
}

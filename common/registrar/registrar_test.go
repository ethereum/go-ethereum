// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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

package registrar

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type testBackend struct {
	// contracts mock
	contracts map[string](map[string]string)
}

var (
	text     = "test"
	codehash = common.StringToHash("1234")
	hash     = common.BytesToHash(crypto.Sha3([]byte(text)))
	url      = "bzz://bzzhash/my/path/contr.act"
)

func NewTestBackend() *testBackend {
	HashRegAddr = common.BigToAddress(common.Big0).Hex() //[2:]
	UrlHintAddr = common.BigToAddress(common.Big1).Hex() //[2:]
	self := &testBackend{}
	self.contracts = make(map[string](map[string]string))

	self.contracts[HashRegAddr[2:]] = make(map[string]string)
	key := storageAddress(storageMapping(storageIdx2Addr(1), codehash[:]))
	self.contracts[HashRegAddr[2:]][key] = hash.Hex()

	self.contracts[UrlHintAddr[2:]] = make(map[string]string)
	mapaddr := storageMapping(storageIdx2Addr(1), hash[:])

	key = storageAddress(storageFixedArray(mapaddr, storageIdx2Addr(0)))
	self.contracts[UrlHintAddr[2:]][key] = common.ToHex([]byte(url))
	key = storageAddress(storageFixedArray(mapaddr, storageIdx2Addr(1)))
	self.contracts[UrlHintAddr[2:]][key] = "0x0"
	return self
}

func (self *testBackend) StorageAt(ca, sa string) (res string) {
	c := self.contracts[ca]
	if c == nil {
		return
	}
	res = c[sa]
	return
}

func (self *testBackend) Transact(fromStr, toStr, nonceStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, error) {
	return "", nil
}

func (self *testBackend) Call(fromStr, toStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, string, error) {
	return "", "", nil
}

func TestSetGlobalRegistrar(t *testing.T) {
	b := NewTestBackend()
	res := New(b)
	_, err := res.SetGlobalRegistrar("addresshex", common.BigToAddress(common.Big1))
	if err != nil {
		t.Errorf("unexpected error: %v'", err)
	}
}

func TestHashToHash(t *testing.T) {
	b := NewTestBackend()
	res := New(b)
	// res.SetHashReg()

	got, err := res.HashToHash(codehash)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	} else {
		if got != hash {
			t.Errorf("incorrect result, expected '%v', got '%v'", hash.Hex(), got.Hex())
		}
	}
}

func TestHashToUrl(t *testing.T) {
	b := NewTestBackend()
	res := New(b)
	// res.SetUrlHint()

	got, err := res.HashToUrl(hash)
	if err != nil {
		t.Errorf("expected 	 error, got %v", err)
	} else {
		if got != url {
			t.Errorf("incorrect result, expected '%v', got '%s'", url, got)
		}
	}
}

package resolver

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
	codehash = common.RightPadString("1234", 64)
	hash     = common.Bytes2Hex(crypto.Sha3([]byte(text)))
	url      = "bzz://bzzhash/my/path/contr.act"
)

func NewTestBackend() *testBackend {
	self := &testBackend{}
	self.contracts = make(map[string](map[string]string))

	self.contracts[HashRegContractAddress] = make(map[string]string)
	key := storageAddress(1, common.Hex2Bytes(codehash))
	self.contracts[HashRegContractAddress][key] = hash

	self.contracts[URLHintContractAddress] = make(map[string]string)
	key = storageAddress(1, common.Hex2Bytes(hash))
	self.contracts[URLHintContractAddress][key] = url

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

func TestKeyToContentHash(t *testing.T) {
	b := NewTestBackend()
	res := New(b, URLHintContractAddress, HashRegContractAddress)
	chash := common.Hash{}
	copy(chash[:], common.Hex2BytesFixed(codehash, 32))
	got, err := res.KeyToContentHash(chash)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	} else {
		if common.Bytes2Hex(got[:]) != hash {
			t.Errorf("incorrect result, expected %x, got %x: ", hash, common.Bytes2Hex(got[:]))
		}
	}
}

func TestContentHashToUrl(t *testing.T) {
	b := NewTestBackend()
	res := New(b, URLHintContractAddress, HashRegContractAddress)
	chash := common.Hash{}
	copy(chash[:], common.Hex2Bytes(hash))
	got, err := res.ContentHashToUrl(chash)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	} else {
		if string(got[:]) != url {
			t.Errorf("incorrect result, expected %v, got %s: ", url, string(got[:]))
		}
	}
}

func TestKeyToUrl(t *testing.T) {
}

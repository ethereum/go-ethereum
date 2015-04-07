package resolver

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

/*
Resolver implements the Ethereum DNS mapping
HashReg : Key Hash (hash of domain name or contract code) -> Content Hash
UrlHint : Content Hash -> Url Hint

The resolver is meant to be called by the roundtripper transport implementation
of a url scheme
*/
const (
	URLHintContractAddress = core.ContractAddrURLhint
	HashRegContractAddress = core.ContractAddrHashReg
)

type Resolver struct {
	backend                Backend
	urlHintContractAddress string
	hashRegContractAddress string
}

type Backend interface {
	StorageAt(string, string) string
}

func New(eth Backend, uhca, nrca string) *Resolver {
	return &Resolver{eth, uhca, nrca}
}

func (self *Resolver) KeyToContentHash(khash common.Hash) (chash common.Hash, err error) {
	// look up in hashReg
	key := storageAddress(1, khash[:])
	hash := self.backend.StorageAt("0x"+self.hashRegContractAddress, key)

	if hash == "0x0" || len(hash) < 3 {
		err = fmt.Errorf("GetHashReg: content hash not found")
		return
	}

	copy(chash[:], common.Hex2BytesFixed(hash[2:], 32))
	return
}

func (self *Resolver) ContentHashToUrl(chash common.Hash) (uri string, err error) {
	// look up in URL reg
	key := storageAddress(1, chash[:])
	hex := self.backend.StorageAt("0x"+self.urlHintContractAddress, key)
	uri = string(common.Hex2Bytes(hex[2:]))
	l := len(uri)
	for (l > 0) && (uri[l-1] == 0) {
		l--
	}
	uri = uri[:l]

	if l == 0 {
		err = fmt.Errorf("GetURLhint: URL hint not found")
	}
	return
}

func (self *Resolver) KeyToUrl(key common.Hash) (uri string, hash common.Hash, err error) {
	// look up in urlHint
	hash, err = self.KeyToContentHash(key)
	if err != nil {
		return
	}
	uri, err = self.ContentHashToUrl(hash)
	return
}

func storageAddress(varidx uint32, key []byte) string {
	data := make([]byte, 64)
	binary.BigEndian.PutUint32(data[60:64], varidx)
	copy(data[0:32], key[0:32])
	return "0x" + common.Bytes2Hex(crypto.Sha3(data))
}

package resolver

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

/*
Resolver implements the Ethereum DNS mapping
NameReg : Domain Name (or Code hash of Contract) -> Content Hash
UrlHint : Content Hash -> Url Hint

The resolver is meant to be called by the roundtripper transport implementation
of a url scheme
*/
const (
	urlHintContractAddress = "urlhint"
	nameRegContractAddress = "nameReg"
)

type Resolver struct {
	backend                Backend
	urlHintContractAddress string
	nameRegContractAddress string
}

type Backend interface {
	StorageAt(string, string) string
}

func New(eth Backend, uhca, nrca string) *Resolver {
	return &Resolver{eth, uhca, nrca}
}

func (self *Resolver) NameToContentHash(name string) (chash common.Hash, err error) {
	// look up in nameReg
	key := storageAddress(0, common.Hex2Bytes(name))
	hash := self.backend.StorageAt(self.nameRegContractAddress, key)
	copy(chash[:], common.Hex2Bytes(hash))
	return
}

func (self *Resolver) ContentHashToUrl(chash common.Hash) (uri string, err error) {
	// look up in nameReg
	key := storageAddress(0, chash[:])
	uri = self.backend.StorageAt(self.urlHintContractAddress, key)
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

func (self *Resolver) NameToUrl(name string) (uri string, hash common.Hash, err error) {
	// look up in urlHint
	hash, err = self.NameToContentHash(name)
	if err != nil {
		return
	}
	uri, err = self.ContentHashToUrl(hash)
	return
}

func storageAddress(varidx uint32, key []byte) string {
	data := make([]byte, 64)
	binary.BigEndian.PutUint32(data[28:32], varidx)
	copy(data[32:64], key[0:32])
	return common.Bytes2Hex(crypto.Sha3(data))
}

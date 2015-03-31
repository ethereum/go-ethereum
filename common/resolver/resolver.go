package resolver

import (
	"encoding/binary"
	"fmt"
	// "net/url"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/xeth"
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
	xeth                   *xeth.XEth
	urlHintContractAddress string
	nameRegContractAddress string
}

func New(_xeth *xeth.XEth, uhca, nrca string) *Resolver {
	return &Resolver{_xeth, uhca, nrca}
}

func (self *Resolver) NameToContentHash(name string) (hash common.Hash, err error) {
	// look up in nameReg
	hashbytes := self.xeth.StorageAt(self.nameRegContractAddress, storageAddress(0, common.Hex2Bytes(name)))
	copy(hash[:], hashbytes[:32])
	return
}

func (self *Resolver) ContentHashToUrl(hash common.Hash) (uri string, err error) {
	// look up in nameReg

	urlHex := self.xeth.StorageAt(self.urlHintContractAddress, storageAddress(0, hash.Bytes()))
	uri = string(common.Hex2Bytes(urlHex))
	l := len(uri)
	for (l > 0) && (uri[l-1] == 0) {
		l--
	}
	uri = uri[:l]
	if l == 0 {
		err = fmt.Errorf("GetURLhint: URL hint not found")
	}
	// rawurl := fmt.Sprintf("bzz://%x/my/path/mycontract.s	ud", hash[:])
	// mime type?
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

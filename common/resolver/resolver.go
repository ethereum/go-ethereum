package resolver

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	xe "github.com/ethereum/go-ethereum/xeth"
)

/*
Resolver implements the Ethereum DNS mapping
HashReg : Key Hash (hash of domain name or contract code) -> Content Hash
UrlHint : Content Hash -> Url Hint

The resolver is meant to be called by the roundtripper transport implementation
of a url scheme
*/

// contract addresses will be hardcoded after they're created
var URLHintContractAddress string = "0000000000000000000000000000000000000000000000000000000000001234"
var HashRegContractAddress string = "0000000000000000000000000000000000000000000000000000000000005678"

func CreateContracts(xeth *xe.XEth, addr string) {
	var err error
	URLHintContractAddress, err = xeth.Transact(addr, "", "", "100000000000", "1000000", "100000", ContractCodeURLhint)
	if err != nil {
		panic(err)
	}
	HashRegContractAddress, err = xeth.Transact(addr, "", "", "100000000000", "1000000", "100000", ContractCodeHashReg)
	if err != nil {
		panic(err)
	}
}

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
	key := storageAddress(storageMapping(storageIdx2Addr(1), khash[:]))
	hash := self.backend.StorageAt(self.hashRegContractAddress, key)

	if hash == "0x0" || len(hash) < 3 {
		err = fmt.Errorf("GetHashReg: content hash not found")
		return
	}

	copy(chash[:], common.Hex2BytesFixed(hash[2:], 32))
	return
}

func (self *Resolver) ContentHashToUrl(chash common.Hash) (uri string, err error) {
	// look up in URL reg
	var str string = " "
	var idx uint32
	for len(str) > 0 {
		mapaddr := storageMapping(storageIdx2Addr(1), chash[:])
		key := storageAddress(storageFixedArray(mapaddr, storageIdx2Addr(idx)))
		hex := self.backend.StorageAt(self.urlHintContractAddress, key)
		str = string(common.Hex2Bytes(hex[2:]))
		l := len(str)
		for (l > 0) && (str[l-1] == 0) {
			l--
		}
		str = str[:l]
		uri = uri + str
		idx++
	}

	if len(uri) == 0 {
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

func storageIdx2Addr(varidx uint32) []byte {
	data := make([]byte, 32)
	binary.BigEndian.PutUint32(data[28:32], varidx)
	return data
}

func storageMapping(addr, key []byte) []byte {
	data := make([]byte, 64)
	copy(data[0:32], key[0:32])
	copy(data[32:64], addr[0:32])
	return crypto.Sha3(data)
}

func storageFixedArray(addr, idx []byte) []byte {
	var carry byte
	for i := 31; i >= 0; i-- {
		var b byte = addr[i] + idx[i] + carry
		if b < addr[i] {
			carry = 1
		} else {
			carry = 0
		}
		addr[i] = b
	}
	return addr
}

func storageAddress(addr []byte) string {
	return common.ToHex(addr)
}

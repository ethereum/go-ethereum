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
	"encoding/binary"
	"fmt"
	"math/big"
	"regexp"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

/*
Registrar implements the Ethereum name registrar services mapping
- arbitrary strings to ethereum addresses
- hashes to hashes
- hashes to arbitrary strings
(likely will provide lookup service for all three)

The Registrar is used by
* the roundtripper transport implementation of
url schemes to resolve domain names and services that register these names
* contract info retrieval (NatSpec).

The Registrar uses 3 contracts on the blockchain:
* GlobalRegistrar: Name (string) -> Address (Owner)
* HashReg : Key Hash (hash of domain name or contract code) -> Content Hash
* UrlHint : Content Hash -> Url Hint

These contracts are (currently) not included in the genesis block.
Each Set<X> needs to be called once on each blockchain/network once.

Contract addresses need to be set (HashReg and UrlHint retrieved from the global
registrar the first time any Registrar method is called in a client session

So the caller needs to make sure the relevant environment initialised the desired
contracts
*/
var (
	UrlHintAddr         = "0x0"
	HashRegAddr         = "0x0"
	GlobalRegistrarAddr = "0x0"
	// GlobalRegistrarAddr = "0xc6d9d2cd449a754c494264e1809c50e34d64562b"

	zero = regexp.MustCompile("^(0x)?0*$")
)

const (
	trueHex  = "0000000000000000000000000000000000000000000000000000000000000001"
	falseHex = "0000000000000000000000000000000000000000000000000000000000000000"
)

func abiSignature(s string) string {
	return common.ToHex(crypto.Sha3([]byte(s))[:4])
}

var (
	HashRegName = "HashReg"
	UrlHintName = "UrlHint"

	registerContentHashAbi = abiSignature("register(uint256,uint256)")
	registerUrlAbi         = abiSignature("register(uint256,uint8,uint256)")
	setOwnerAbi            = abiSignature("setowner()")
	reserveAbi             = abiSignature("reserve(bytes32)")
	resolveAbi             = abiSignature("addr(bytes32)")
	registerAbi            = abiSignature("setAddress(bytes32,address,bool)")
	addressAbiPrefix       = falseHex[:24]
)

// Registrar's backend is defined as an interface (implemented by xeth, but could be remote)
type Backend interface {
	StorageAt(string, string) string
	Transact(fromStr, toStr, nonceStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, error)
	Call(fromStr, toStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, string, error)
}

// TODO Registrar should also just implement The Resolver and Registry interfaces
// Simplify for now.
type VersionedRegistrar interface {
	Resolver(*big.Int) *Registrar
	Registry() *Registrar
}

type Registrar struct {
	backend Backend
}

func New(b Backend) (res *Registrar) {
	res = &Registrar{b}
	return
}

func (self *Registrar) SetGlobalRegistrar(namereg string, addr common.Address) (txhash string, err error) {
	if namereg != "" {
		GlobalRegistrarAddr = namereg
		return
	}
	if GlobalRegistrarAddr == "0x0" || GlobalRegistrarAddr == "0x" {
		if (addr == common.Address{}) {
			err = fmt.Errorf("GlobalRegistrar address not found and sender for creation not given")
			return
		} else {
			txhash, err = self.backend.Transact(addr.Hex(), "", "", "", "800000", "", GlobalRegistrarCode)
			if err != nil {
				err = fmt.Errorf("GlobalRegistrar address not found and sender for creation failed: %v", err)
				return
			}
		}
	}
	return
}

func (self *Registrar) SetHashReg(hashreg string, addr common.Address) (txhash string, err error) {
	if hashreg != "" {
		HashRegAddr = hashreg
	} else {
		if !zero.MatchString(HashRegAddr) {
			return
		}
		nameHex, extra := encodeName(HashRegName, 2)
		hashRegAbi := resolveAbi + nameHex + extra
		glog.V(logger.Detail).Infof("\ncall HashRegAddr %v with %v\n", GlobalRegistrarAddr, hashRegAbi)
		var res string
		res, _, err = self.backend.Call("", GlobalRegistrarAddr, "", "", "", hashRegAbi)
		if len(res) >= 40 {
			HashRegAddr = "0x" + res[len(res)-40:len(res)]
		}
		if err != nil || zero.MatchString(HashRegAddr) {
			if (addr == common.Address{}) {
				err = fmt.Errorf("HashReg address not found and sender for creation not given")
				return
			}

			txhash, err = self.backend.Transact(addr.Hex(), "", "", "", "", "", HashRegCode)
			if err != nil {
				err = fmt.Errorf("HashReg address not found and sender for creation failed: %v", err)
			}
			glog.V(logger.Detail).Infof("created HashRegAddr @ txhash %v\n", txhash)
		} else {
			glog.V(logger.Detail).Infof("HashRegAddr found at @ %v\n", HashRegAddr)
			return
		}
	}

	return
}

func (self *Registrar) SetUrlHint(urlhint string, addr common.Address) (txhash string, err error) {
	if urlhint != "" {
		UrlHintAddr = urlhint
	} else {
		if !zero.MatchString(UrlHintAddr) {
			return
		}
		nameHex, extra := encodeName(UrlHintName, 2)
		urlHintAbi := resolveAbi + nameHex + extra
		glog.V(logger.Detail).Infof("UrlHint address query data: %s to %s", urlHintAbi, GlobalRegistrarAddr)
		var res string
		res, _, err = self.backend.Call("", GlobalRegistrarAddr, "", "", "", urlHintAbi)
		if len(res) >= 40 {
			UrlHintAddr = "0x" + res[len(res)-40:len(res)]
		}
		if err != nil || zero.MatchString(UrlHintAddr) {
			if (addr == common.Address{}) {
				err = fmt.Errorf("UrlHint address not found and sender for creation not given")
				return
			}
			txhash, err = self.backend.Transact(addr.Hex(), "", "", "", "210000", "", UrlHintCode)
			if err != nil {
				err = fmt.Errorf("UrlHint address not found and sender for creation failed: %v", err)
			}
			glog.V(logger.Detail).Infof("created UrlHint @ txhash %v\n", txhash)
		} else {
			glog.V(logger.Detail).Infof("UrlHint found @ %v\n", HashRegAddr)
			return
		}
	}

	return
}

// ReserveName(from, name) reserves name for the sender address in the globalRegistrar
// the tx needs to be mined to take effect
func (self *Registrar) ReserveName(address common.Address, name string) (txh string, err error) {
	nameHex, extra := encodeName(name, 2)
	abi := reserveAbi + nameHex + extra
	glog.V(logger.Detail).Infof("Reserve data: %s", abi)
	return self.backend.Transact(
		address.Hex(),
		GlobalRegistrarAddr,
		"", "", "", "",
		abi,
	)
}

// SetAddressToName(from, name, addr) will set the Address to address for name
// in the globalRegistrar using from as the sender of the transaction
// the tx needs to be mined to take effect
func (self *Registrar) SetAddressToName(from common.Address, name string, address common.Address) (txh string, err error) {
	nameHex, extra := encodeName(name, 6)
	addrHex := encodeAddress(address)

	abi := registerAbi + nameHex + addrHex + trueHex + extra
	glog.V(logger.Detail).Infof("SetAddressToName data: %s to %s ", abi, GlobalRegistrarAddr)

	return self.backend.Transact(
		from.Hex(),
		GlobalRegistrarAddr,
		"", "", "", "",
		abi,
	)
}

// NameToAddr(from, name) queries the registrar for the address on name
func (self *Registrar) NameToAddr(from common.Address, name string) (address common.Address, err error) {
	nameHex, extra := encodeName(name, 2)
	abi := resolveAbi + nameHex + extra
	glog.V(logger.Detail).Infof("NameToAddr data: %s", abi)
	res, _, err := self.backend.Call(
		from.Hex(),
		GlobalRegistrarAddr,
		"", "", "",
		abi,
	)
	if err != nil {
		return
	}
	address = common.HexToAddress(res)
	return
}

// called as first step in the registration process on HashReg
func (self *Registrar) SetOwner(address common.Address) (txh string, err error) {
	return self.backend.Transact(
		address.Hex(),
		HashRegAddr,
		"", "", "", "",
		setOwnerAbi,
	)
}

// registers some content hash to a key/code hash
// e.g., the contract Info combined Json Doc's ContentHash
// to CodeHash of a contract or hash of a domain
func (self *Registrar) SetHashToHash(address common.Address, codehash, dochash common.Hash) (txh string, err error) {
	_, err = self.SetOwner(address)
	if err != nil {
		return
	}
	codehex := common.Bytes2Hex(codehash[:])
	dochex := common.Bytes2Hex(dochash[:])

	data := registerContentHashAbi + codehex + dochex
	glog.V(logger.Detail).Infof("SetHashToHash data: %s sent  to %v\n", data, HashRegAddr)
	return self.backend.Transact(
		address.Hex(),
		HashRegAddr,
		"", "", "", "",
		data,
	)
}

// SetUrlToHash(from, hash, url) registers a url to a content hash so that the content can be fetched
// address is used as sender for the transaction and will be the owner of a new
// registry entry on first time use
// FIXME: silently doing nothing if sender is not the owner
// note that with content addressed storage, this step is no longer necessary
func (self *Registrar) SetUrlToHash(address common.Address, hash common.Hash, url string) (txh string, err error) {
	hashHex := common.Bytes2Hex(hash[:])
	var urlHex string
	urlb := []byte(url)
	var cnt byte
	n := len(urlb)

	for n > 0 {
		if n > 32 {
			n = 32
		}
		urlHex = common.Bytes2Hex(urlb[:n])
		urlb = urlb[n:]
		n = len(urlb)
		bcnt := make([]byte, 32)
		bcnt[31] = cnt
		data := registerUrlAbi +
			hashHex +
			common.Bytes2Hex(bcnt) +
			common.Bytes2Hex(common.Hex2BytesFixed(urlHex, 32))
		txh, err = self.backend.Transact(
			address.Hex(),
			UrlHintAddr,
			"", "", "", "",
			data,
		)
		if err != nil {
			return
		}
		cnt++
	}
	return
}

// HashToHash(key) resolves contenthash for key (a hash) using HashReg
// resolution is costless non-transactional
// implemented as direct retrieval from  db
func (self *Registrar) HashToHash(khash common.Hash) (chash common.Hash, err error) {
	// look up in hashReg
	at := HashRegAddr[2:]
	key := storageAddress(storageMapping(storageIdx2Addr(1), khash[:]))
	hash := self.backend.StorageAt(at, key)

	if hash == "0x0" || len(hash) < 3 || (hash == common.Hash{}.Hex()) {
		err = fmt.Errorf("content hash not found for '%v'", khash.Hex())
		return
	}
	copy(chash[:], common.Hex2BytesFixed(hash[2:], 32))
	return
}

// HashToUrl(contenthash) resolves the url for contenthash using UrlHint
// resolution is costless non-transactional
// implemented as direct retrieval from  db
// if we use content addressed storage, this step is no longer necessary
func (self *Registrar) HashToUrl(chash common.Hash) (uri string, err error) {
	// look up in URL reg
	var str string = " "
	var idx uint32
	for len(str) > 0 {
		mapaddr := storageMapping(storageIdx2Addr(1), chash[:])
		key := storageAddress(storageFixedArray(mapaddr, storageIdx2Addr(idx)))
		hex := self.backend.StorageAt(UrlHintAddr[2:], key)
		str = string(common.Hex2Bytes(hex[2:]))
		l := 0
		for (l < len(str)) && (str[l] == 0) {
			l++
		}

		str = str[l:]
		uri = uri + str
		idx++
	}

	if len(uri) == 0 {
		err = fmt.Errorf("GetURLhint: URL hint not found for '%v'", chash.Hex())
	}
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
	sha := crypto.Sha3(data)
	return sha
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

func encodeAddress(address common.Address) string {
	return addressAbiPrefix + address.Hex()[2:]
}

func encodeName(name string, index uint8) (string, string) {
	extra := common.Bytes2Hex([]byte(name))
	if len(name) > 32 {
		return fmt.Sprintf("%064x", index), extra
	}
	return extra + falseHex[len(extra):], ""
}

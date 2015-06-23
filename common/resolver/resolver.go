package resolver

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

/*
Resolver implements the Ethereum DNS mapping
HashReg : Key Hash (hash of domain name or contract code) -> Content Hash
UrlHint : Content Hash -> Url Hint

The resolver is meant to be called by the roundtripper transport implementation
of a url scheme
*/

// // contract addresses will be hardcoded after they're created
var (
	UrlHintContractAddress = "0x0"
	HashRegContractAddress = "0x0"
)

const (
	txValue    = "0"
	txGas      = "100000"
	txGasPrice = "1000000000000"
)

func abiSignature(s string) string {
	return common.ToHex(crypto.Sha3([]byte(s))[:4])
}

var (
	HashReg = "HashReg"
	UrlHint = "UrlHint"

	registerContentHashAbi = abiSignature("register(uint256,uint256)")
	registerUrlAbi         = abiSignature("register(uint256,uint8,uint256)")
	setOwnerAbi            = abiSignature("setowner()")
	reserveAbi             = abiSignature("reserve(bytes32)")
	resolveAbi             = abiSignature("addr(bytes32)")
	registerAbi            = abiSignature("setAddress(bytes32,address,bool)")
	addressAbiPrefix       = string(make([]byte, 24))
)

type Backend interface {
	StorageAt(string, string) string
	Transact(fromStr, toStr, nonceStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, error)
	Call(fromStr, toStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, string, error)
}

type Resolver struct {
	backend Backend
}

func New(eth Backend) (res *Resolver) {
	res = &Resolver{eth}
	res.setContracts()
	return
}

func (self *Resolver) setContracts() {
	var err error
	if HashRegContractAddress != "0x0" {
		return
	}
	// reset iff error anywhere
	defer func() {
		if err != nil {
			HashRegContractAddress = "0x0"
		}
	}()
	hashRegAbi := registerAbi + string(common.Hex2BytesFixed(HashRegContractAddress[2:], 32))
	HashRegContractAddress, _, err = self.backend.Call("", GlobalRegistrarAddr, "", "", "", hashRegAbi)
	if err != nil {
		err = fmt.Errorf("HashReg address not found: %v", err)
		return
	}

	if UrlHintContractAddress != "0x0" {
		return
	}
	// reset iff error anywhere
	defer func() {
		if err != nil {
			UrlHintContractAddress = "0x0"
		}
	}()

	urlHintAbi := registerAbi + string(common.Hex2BytesFixed(UrlHintContractAddress[2:], 32))
	UrlHintContractAddress, _, err = self.backend.Call("", GlobalRegistrarAddr, "", "", "", urlHintAbi)
	if err != nil {
		err = fmt.Errorf("UrlHint address not found: %v", err)
		return
	}

	glog.V(logger.Detail).Infof("HashReg @ %v\nUrlHint @ %v\n", HashRegContractAddress, UrlHintContractAddress)
}

// This can be safely called from tests to or private chains to create
// new HashReg and UrlHint contracts (requires transaction)
// It does nothing if addresses are set
func (self *Resolver) CreateContracts(addr common.Address) (hashReg, urlHint string, err error) {
	if HashRegContractAddress != "0x0" {
		err = fmt.Errorf("HashReg already exists at %v", HashRegContractAddress)
		return
	}
	hashReg, err = self.backend.Transact(addr.Hex(), "", "", txValue, txGas, txGasPrice, ContractCodeHashReg)
	if err != nil {
		return
	}
	_, err = self.Reserve(addr, HashReg)
	if err != nil {
		return
	}
	_, err = self.RegisterAddress(addr, HashReg, common.HexToAddress(hashReg))
	if err != nil {
		return
	}

	if UrlHintContractAddress != "0x0" {
		err = fmt.Errorf("UrlHint already exists at %v", UrlHintContractAddress)
		return
	}
	urlHint, err = self.backend.Transact(addr.Hex(), "", "", txValue, txGas, txGasPrice, ContractCodeURLhint)
	if err != nil {
		return
	}
	_, err = self.Reserve(addr, UrlHint)
	if err != nil {
		return
	}
	_, err = self.RegisterAddress(addr, UrlHint, common.HexToAddress(urlHint))
	if err != nil {
		return
	}
	HashRegContractAddress = hashReg
	UrlHintContractAddress = urlHint
	glog.V(logger.Detail).Infof("HashReg @ %v\nUrlHint @ %v\n", HashRegContractAddress, UrlHintContractAddress)

	return
}

// Reserve(from, name) reserves name for the sender address in the globalRegistrar
// the tx needs to be mined to take effect
func (self *Resolver) Reserve(address common.Address, name string) (txh string, err error) {
	nameHex, extra := encodeName(name, 6)
	abi := reserveAbi + nameHex + extra
	return self.backend.Transact(
		address.Hex(),
		GlobalRegistrarAddr,
		"", txValue, txGas, txGasPrice,
		abi,
	)
}

// RegisterAddress(from, name, addr) will set the Address to address for name
// in the globalRegistrar using from as the sender of the transaction
// the tx needs to be mined to take effect
func (self *Resolver) RegisterAddress(from common.Address, name string, address common.Address) (txh string, err error) {
	nameHex, extra := encodeName(name, 6)
	addrHex := encodeAddress(address)

	trueHex := make([]byte, 64)
	trueHex[63] = 1

	abi := registerAbi + nameHex + addrHex + extra
	return self.backend.Transact(
		from.Hex(),
		GlobalRegistrarAddr,
		"", txValue, txGas, txGasPrice,
		abi,
	)
}

// NameToAddr(from, name) queries the registrar for the address on
func (self *Resolver) NameToAddr(from common.Address, name string) (address common.Address, err error) {
	nameHex, extra := encodeName(name, 2)
	abi := resolveAbi + nameHex + extra
	res, _, err := self.backend.Call(
		from.Hex(),
		GlobalRegistrarAddr,
		txValue, txGas, txGasPrice,
		abi,
	)
	if err != nil {
		return
	}
	address = common.HexToAddress(res)
	return
}

// called as first step in the registration process on HashReg
func (self *Resolver) SetOwner(address common.Address) (txh string, err error) {
	return self.backend.Transact(
		address.Hex(),
		HashRegContractAddress,
		"", txValue, txGas, txGasPrice,
		setOwnerAbi,
	)
}

// registers some content hash to a key/code hash
// e.g., the contract Info combined Json Doc's ContentHash
// to CodeHash of a contract or hash of a domain
// kept
func (self *Resolver) RegisterContentHash(address common.Address, codehash, dochash common.Hash) (txh string, err error) {
	_, err = self.SetOwner(address)
	if err != nil {
		return
	}
	codehex := common.Bytes2Hex(codehash[:])
	dochex := common.Bytes2Hex(dochash[:])

	data := registerContentHashAbi + codehex + dochex
	return self.backend.Transact(
		address.Hex(),
		HashRegContractAddress,
		"", txValue, txGas, txGasPrice,
		data,
	)
}

// registers a url to a content hash so that the content can be fetched
// address is used as sender for the transaction and will be the owner of a new
// registry entry on first time use
// FIXME: silently doing nothing if sender is not the owner
// note that with content addressed storage, this step is no longer necessary
// it could be purely
func (self *Resolver) RegisterUrl(address common.Address, hash common.Hash, url string) (txh string, err error) {
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
			UrlHintContractAddress,
			"", txValue, txGas, txGasPrice,
			data,
		)
		if err != nil {
			return
		}
		cnt++
	}
	return
}

func (self *Resolver) RegisterAddrWithUrl(address common.Address, codehash, dochash common.Hash, url string) (txh string, err error) {

	_, err = self.RegisterContentHash(address, codehash, dochash)
	if err != nil {
		return
	}
	return self.RegisterUrl(address, dochash, url)
}

// resolution is costless non-transactional
// implemented as direct retrieval from  db
func (self *Resolver) KeyToContentHash(khash common.Hash) (chash common.Hash, err error) {
	// look up in hashReg
	at := HashRegContractAddress[2:]
	key := storageAddress(storageMapping(storageIdx2Addr(1), khash[:]))
	hash := self.backend.StorageAt(at, key)

	if hash == "0x0" || len(hash) < 3 {
		err = fmt.Errorf("content hash not found for '%v'", khash.Hex())
		return
	}
	copy(chash[:], common.Hex2BytesFixed(hash[2:], 32))
	return
}

// retrieves the url-hint for the content hash -
// if we use content addressed storage, this step is no longer necessary
func (self *Resolver) ContentHashToUrl(chash common.Hash) (uri string, err error) {
	// look up in URL reg
	var str string = " "
	var idx uint32
	for len(str) > 0 {
		mapaddr := storageMapping(storageIdx2Addr(1), chash[:])
		key := storageAddress(storageFixedArray(mapaddr, storageIdx2Addr(idx)))
		hex := self.backend.StorageAt(UrlHintContractAddress[2:], key)
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
		err = fmt.Errorf("GetURLhint: URL hint not found for '%v'", chash.Hex())
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

func encodeName(name string, index uint8) (nameHex, extra string) {
	nameHexBytes := make([]byte, 64)
	if len(name) > 32 {
		nameHexBytes[63] = byte(index)
		extra = common.Bytes2Hex([]byte(name))
	} else {
		copy(nameHexBytes, []byte(name))
	}
	return string(nameHexBytes), extra
}

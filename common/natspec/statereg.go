package natspec

import (
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/xeth"
	"io/ioutil"
	"net/http"
)

type StateReg struct {
	xeth             *xeth.XEth
	caURL, caNatSpec string //contract addresses
}

func NewStateReg(_xeth *xeth.XEth) (self *StateReg) {

	self.xeth = _xeth
	self.testCreateContracts()
	return

}

const codeURLhint = "0x33600081905550609c8060136000396000f30060003560e060020a900480632f926732" +
	"14601f578063f39ec1f714603157005b602b6004356024356044565b60006000f35b603a" +
	"600435607f565b8060005260206000f35b600054600160a060020a031633600160a06002" +
	"0a031614606257607b565b8060016000848152602001908152602001600020819055505b" +
	"5050565b60006001600083815260200190815260200160002054905091905056"

const codeNatSpec = "0x33600081905550609c8060136000396000f30060003560e060020a900480632f926732" +
	"14601f578063f39ec1f714603157005b602b6004356024356044565b60006000f35b603a" +
	"600435607f565b8060005260206000f35b600054600160a060020a031633600160a06002" +
	"0a031614606257607b565b8060016000848152602001908152602001600020819055505b" +
	"5050565b60006001600083815260200190815260200160002054905091905056"

func (self *StateReg) testCreateContracts() {

	var err error
	self.caURL, err = self.xeth.Transact(self.xeth.Coinbase(), "", "100000", "", self.xeth.DefaultGas().String(), codeURLhint)
	if err != nil {
		panic(err)
	}
	self.caNatSpec, err = self.xeth.Transact(self.xeth.Coinbase(), "", "100000", "", self.xeth.DefaultGas().String(), codeNatSpec)
	if err != nil {
		panic(err)
	}

}

func (self *StateReg) GetURLhint(hash string) (url string, err error) {

	url_hex := self.xeth.StorageAt(self.caURL, storageAddress(0, common.Hex2Bytes(hash)))
	url = string(common.Hex2Bytes(url_hex))
	l := len(url)
	for (l > 0) && (url[l-1] == 0) {
		l--
	}
	url = url[:l]
	if l == 0 {
		err = fmt.Errorf("GetURLhint: URL hint not found")
	}
	return

}

func storageAddress(varidx uint32, key []byte) string {
	data := make([]byte, 64)
	binary.BigEndian.PutUint32(data[28:32], varidx)
	copy(data[32:64], key[0:32])
	return common.Bytes2Hex(crypto.Sha3(data))
}

func (self *StateReg) GetNatSpec(codehash string) (hash string, err error) {

	hash = self.xeth.StorageAt(self.caNatSpec, storageAddress(0, common.Hex2Bytes(codehash)))
	return

}

func (self *StateReg) GetContent(hash string) (content []byte, err error) {

	// get URL
	url, err := self.GetURLhint(hash)
	if err != nil {
		return
	}

	// retrieve content
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		return
	}
	content, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// check hash

	if common.Bytes2Hex(crypto.Sha3(content)) != hash {
		content = nil
		err = fmt.Errorf("GetContent error: content hash mismatch")
	}

	return

}

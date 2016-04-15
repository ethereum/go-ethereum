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

// +build ignore

package natspec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/httpclient"
	"github.com/ethereum/go-ethereum/common/registrar"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/robertkrimen/otto"
)

type abi2method map[[8]byte]*method

type NatSpec struct {
	jsvm       *otto.Otto
	abiDocJson []byte
	userDoc    userDoc
	tx, data   string
}

// main entry point for to get natspec notice for a transaction
// the implementation is frontend friendly in that it always gives back
// a notice that is safe to display
// :FIXME: the second return value is an error, which can be used to fine-tune bahaviour
func GetNotice(xeth *xeth.XEth, tx string, http *httpclient.HTTPClient) (notice string) {
	ns, err := New(xeth, tx, http)
	if err != nil {
		if ns == nil {
			return getFallbackNotice(fmt.Sprintf("no NatSpec info found for contract: %v", err), tx)
		} else {
			return getFallbackNotice(fmt.Sprintf("invalid NatSpec info: %v", err), tx)
		}
	}

	notice, err = ns.Notice()
	if err != nil {
		return getFallbackNotice(fmt.Sprintf("NatSpec notice error: %v", err), tx)
	}

	return
}

func getFallbackNotice(comment, tx string) string {
	return fmt.Sprintf("About to submit transaction (%s): %s", comment, tx)
}

type transaction struct {
	To   string `json:"to"`
	Data string `json:"data"`
}

type jsonTx struct {
	Params []transaction `json:"params"`
}

type contractInfo struct {
	Source        string          `json:"source"`
	Language      string          `json:"language"`
	Version       string          `json:"compilerVersion"`
	AbiDefinition json.RawMessage `json:"abiDefinition"`
	UserDoc       userDoc         `json:"userDoc"`
	DeveloperDoc  json.RawMessage `json:"developerDoc"`
}

func New(xeth *xeth.XEth, jsontx string, http *httpclient.HTTPClient) (self *NatSpec, err error) {

	// extract contract address from tx
	var tx jsonTx
	err = json.Unmarshal([]byte(jsontx), &tx)
	if err != nil {
		return
	}
	t := tx.Params[0]
	contractAddress := t.To

	content, err := FetchDocsForContract(contractAddress, xeth, http)
	if err != nil {
		return
	}

	self, err = NewWithDocs(content, jsontx, t.Data)
	return
}

// also called by admin.contractInfo.get
func FetchDocsForContract(contractAddress string, xeth *xeth.XEth, client *httpclient.HTTPClient) (content []byte, err error) {
	// retrieve contract hash from state
	codehex := xeth.CodeAt(contractAddress)
	codeb := xeth.CodeAtBytes(contractAddress)

	if codehex == "0x" {
		err = fmt.Errorf("contract (%v) not found", contractAddress)
		return
	}
	codehash := common.BytesToHash(crypto.Keccak256(codeb))
	// set up nameresolver with natspecreg + urlhint contract addresses
	reg := registrar.New(xeth)

	// resolve host via HashReg/UrlHint Resolver
	hash, err := reg.HashToHash(codehash)
	if err != nil {
		return
	}
	if client.HasScheme("bzz") {
		content, err = client.Get("bzz://"+hash.Hex()[2:], "")
		if err == nil { // non-fatal
			return
		}
		err = nil
		//falling back to urlhint
	}

	uri, err := reg.HashToUrl(hash)
	if err != nil {
		return
	}

	// get content via http client and authenticate content using hash
	content, err = client.GetAuthContent(uri, hash)
	if err != nil {
		return
	}
	return
}

func NewWithDocs(infoDoc []byte, tx string, data string) (self *NatSpec, err error) {

	var contract contractInfo
	err = json.Unmarshal(infoDoc, &contract)
	if err != nil {
		return
	}

	self = &NatSpec{
		jsvm:       otto.New(),
		abiDocJson: []byte(contract.AbiDefinition),
		userDoc:    contract.UserDoc,
		tx:         tx,
		data:       data,
	}

	// load and require natspec js (but it is meant to be protected environment)
	_, err = self.jsvm.Run(natspecJS)
	if err != nil {
		return
	}
	_, err = self.jsvm.Run("var natspec = require('natspec');")
	return
}

// type abiDoc []method

// type method struct {
// 	Name   string  `json:name`
// 	Inputs []input `json:inputs`
// 	abiKey [8]byte
// }

// type input struct {
// 	Name string `json:name`
// 	Type string `json:type`
// }

// json skeleton for abi doc (contract method definitions)
type method struct {
	Notice string `json:notice`
	name   string
}

type userDoc struct {
	Methods map[string]*method `json:methods`
}

func (self *NatSpec) makeAbi2method(abiKey [8]byte) (meth *method) {
	for signature, m := range self.userDoc.Methods {
		name := strings.Split(signature, "(")[0]
		hash := []byte(common.Bytes2Hex(crypto.Keccak256([]byte(signature))))
		var key [8]byte
		copy(key[:], hash[:8])
		if bytes.Equal(key[:], abiKey[:]) {
			meth = m
			meth.name = name
			return
		}
	}
	return
}

func (self *NatSpec) Notice() (notice string, err error) {
	var abiKey [8]byte
	if len(self.data) < 10 {
		err = fmt.Errorf("Invalid transaction data")
		return
	}
	copy(abiKey[:], self.data[2:10])
	meth := self.makeAbi2method(abiKey)

	if meth == nil {
		err = fmt.Errorf("abi key does not match any method")
		return
	}
	notice, err = self.noticeForMethod(self.tx, meth.name, meth.Notice)
	return
}

func (self *NatSpec) noticeForMethod(tx string, name, expression string) (notice string, err error) {

	if _, err = self.jsvm.Run("var transaction = " + tx + ";"); err != nil {
		return "", fmt.Errorf("natspec.js error setting transaction: %v", err)
	}

	if _, err = self.jsvm.Run("var abi = " + string(self.abiDocJson) + ";"); err != nil {
		return "", fmt.Errorf("natspec.js error setting abi: %v", err)
	}

	if _, err = self.jsvm.Run("var method = '" + name + "';"); err != nil {
		return "", fmt.Errorf("natspec.js error setting method: %v", err)
	}

	if _, err = self.jsvm.Run("var expression = \"" + expression + "\";"); err != nil {
		return "", fmt.Errorf("natspec.js error setting expression: %v", err)
	}

	self.jsvm.Run("var call = {method: method,abi: abi,transaction: transaction};")
	value, err := self.jsvm.Run("natspec.evaluateExpression(expression, call);")
	if err != nil {
		return "", fmt.Errorf("natspec.js error evaluating expression: %v", err)
	}
	evalError := "Natspec evaluation failed, wrong input params"
	if value.String() == evalError {
		return "", fmt.Errorf("natspec.js error evaluating expression: wrong input params in expression '%s'", expression)
	}
	if len(value.String()) == 0 {
		return "", fmt.Errorf("natspec.js error evaluating expression")
	}

	return value.String(), nil

}

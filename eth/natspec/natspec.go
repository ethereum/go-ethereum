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

package natspec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/registrar"
	"github.com/ethereum/go-ethereum/node"
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
	"github.com/robertkrimen/otto"
)

type abi2method map[[8]byte]*method

type NatSpec struct {
	backend Backend
	http    *node.HTTPClient
	reg     *registrar.Registrar
}

type Backend interface {
	GetCode(address common.Address, blockNr rpc.BlockNumber) (string, error)
}

func New(backend Backend, http *node.HTTPClient, reg *registrar.Registrar) *NatSpec {
	return &NatSpec{backend, http, reg}
}

type contractInfo struct {
	jsvm       *otto.Otto
	abiDocJson []byte
	userDoc    userDoc
	tx         *SendTxArgs
}

type SendTxArgs struct {
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	Gas      *rpc.HexNumber `json:"gas"`
	GasPrice *rpc.HexNumber `json:"gasPrice"`
	Value    *rpc.HexNumber `json:"value"`
	Data     string         `json:"data"`
	Nonce    *rpc.HexNumber `json:"nonce"`
}

func (self *NatSpec) GetNatSpec(tx *SendTxArgs) (string, error) {
	to := tx.To
	// in principle this could be cached, but practically swarm takes care of that
	info, err := self.GetContractInfo(to)
	if err != nil {
		if info == nil {
			return "", fmt.Errorf("no contract info found for %v: %v", to, err)
		} else {
			return "", fmt.Errorf("invalid contract info for %v: %v", to, err)
		}
	}
	ns, err := newContractInfo(info, tx)
	if err != nil {
		return "", fmt.Errorf("invalid contract info for %v: %v", to, err)
	}
	notice, err := ns.notice()
	if err != nil {
		return "", fmt.Errorf("NatSpec notice error for contract %v: %v", to, err)
	}

	return notice, nil
}

type ContractInfo struct {
	Source        string          `json:"source"`
	Language      string          `json:"language"`
	Version       string          `json:"compilerVersion"`
	AbiDefinition json.RawMessage `json:"abiDefinition"`
	UserDoc       userDoc         `json:"userDoc"`
	DeveloperDoc  json.RawMessage `json:"developerDoc"`
}

func (self *NatSpec) GetContractInfo(to common.Address) (*ContractInfo, error) {
	// retrieve contract hash from state
	codehex, err := self.backend.GetCode(to, -1)
	if err != nil || len(codehex) <= 3 {
		return nil, fmt.Errorf("contract (%v) not found", to)
	}
	codehash := common.BytesToHash(crypto.Sha3(common.FromHex(codehex)))

	// resolve host via HashReg/UrlHint Resolver
	hash, err := self.reg.HashToHash(codehash)
	if err != nil {
		return nil, fmt.Errorf("no content hash registered for contract %v: %v", to, err)
	}
	var data []byte
	if self.http.HasScheme("bzz") {
		data, err = self.http.GetBody("bzz://" + hash.Hex()[2:])
		if err != nil { // non-fatal
			data = nil
		}
	}

	//falling back to urlhint
	if data == nil {
		uri, err := self.reg.HashToUrl(hash)
		if err != nil {
			return nil, err
		}

		// get content via http client and authenticate content using hash
		data, err = self.http.GetAuthBody(uri, hash)
		if err != nil {
			return nil, err
		}
	}

	var info ContractInfo
	err = json.Unmarshal(data, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func newContractInfo(info *ContractInfo, tx *SendTxArgs) (self *contractInfo, err error) {

	self = &contractInfo{
		jsvm:       otto.New(),
		abiDocJson: []byte(info.AbiDefinition),
		userDoc:    info.UserDoc,
		tx:         tx,
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

func (self *contractInfo) makeAbi2method(abiKey [8]byte) (meth *method) {
	for signature, m := range self.userDoc.Methods {
		name := strings.Split(signature, "(")[0]
		hash := []byte(common.Bytes2Hex(crypto.Sha3([]byte(signature))))
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

func (self *contractInfo) notice() (notice string, err error) {
	var abiKey [8]byte
	if len(self.tx.Data) < 10 {
		err = fmt.Errorf("Invalid transaction data")
		return
	}
	copy(abiKey[:], self.tx.Data[2:10])
	meth := self.makeAbi2method(abiKey)

	if meth == nil {
		err = fmt.Errorf("abi key does not match any method")
		return
	}
	notice, err = self.noticeForMethod(meth.name, meth.Notice)
	return
}

func (self *contractInfo) noticeForMethod(name, expression string) (notice string, err error) {
	tx, err := json.Marshal(self.tx)
	if err != nil {
		return "", nil
	}

	if _, err = self.jsvm.Run("var transaction = " + string(tx) + ";"); err != nil {
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
		return "", fmt.Errorf("natspec.js error evaluating expression: empty notice")
	}

	return value.String(), nil

}

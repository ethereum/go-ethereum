package natspec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/robertkrimen/otto"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/docserver"
	"github.com/ethereum/go-ethereum/common/resolver"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/xeth"
)

type abi2method map[[8]byte]*method

type NatSpec struct {
	jsvm                    *otto.Otto
	userDocJson, abiDocJson []byte
	userDoc                 userDoc
	tx, data                string
	// abiDoc abiDoc
}

func getFallbackNotice(comment, tx string) string {

	return "About to submit transaction (" + comment + "): " + tx

}

func GetNotice(xeth *xeth.XEth, tx string, http *docserver.DocServer) (notice string) {

	ns, err := New(xeth, tx, http)
	if err != nil {
		if ns == nil {
			return getFallbackNotice("no NatSpec info found for contract", tx)
		} else {
			return getFallbackNotice("invalid NatSpec info", tx)
		}
	}

	notice, err2 := ns.Notice()

	if err2 != nil {
		return getFallbackNotice("NatSpec notice error \""+err2.Error()+"\"", tx)
	}

	return

}

func New(xeth *xeth.XEth, tx string, http *docserver.DocServer) (self *NatSpec, err error) {

	// extract contract address from tx

	var obj map[string]json.RawMessage
	err = json.Unmarshal([]byte(tx), &obj)
	if err != nil {
		return
	}
	var tmp []map[string]string
	err = json.Unmarshal(obj["params"], &tmp)
	if err != nil {
		return
	}
	contractAddress := tmp[0]["to"]

	// retrieve contract hash from state
	if !xeth.IsContract(contractAddress) {
		err = fmt.Errorf("NatSpec error: contract not found")
		return
	}
	codehex := xeth.CodeAt(contractAddress)
	codeHash := common.BytesToHash(crypto.Sha3(common.Hex2Bytes(codehex[2:])))
	// parse out host/domain

	// set up nameresolver with natspecreg + urlhint contract addresses
	res := resolver.New(
		xeth,
		resolver.URLHintContractAddress,
		resolver.HashRegContractAddress,
	)

	// resolve host via HashReg/UrlHint Resolver
	uri, hash, err := res.KeyToUrl(codeHash)
	if err != nil {
		return
	}

	// get content via http client and authenticate content using hash
	content, err := http.GetAuthContent(uri, hash)
	if err != nil {
		return
	}

	// get abi, userdoc
	var obj2 map[string]json.RawMessage
	err = json.Unmarshal(content, &obj2)
	if err != nil {
		return
	}

	abi := []byte(obj2["abi"])
	userdoc := []byte(obj2["userdoc"])

	self, err = NewWithDocs(abi, userdoc, tx)
	return
}

func NewWithDocs(abiDocJson, userDocJson []byte, tx string) (self *NatSpec, err error) {

	var obj map[string]json.RawMessage
	err = json.Unmarshal([]byte(tx), &obj)
	if err != nil {
		return
	}
	var tmp []map[string]string
	err = json.Unmarshal(obj["params"], &tmp)
	if err != nil {
		return
	}
	data := tmp[0]["data"]

	self = &NatSpec{
		jsvm:        otto.New(),
		abiDocJson:  abiDocJson,
		userDocJson: userDocJson,
		tx:          tx,
		data:        data,
	}

	// load and require natspec js (but it is meant to be protected environment)
	_, err = self.jsvm.Run(natspecJS)
	if err != nil {
		return
	}
	_, err = self.jsvm.Run("var natspec = require('natspec');")
	if err != nil {
		return
	}

	err = json.Unmarshal(userDocJson, &self.userDoc)
	// err = parseAbiJson(abiDocJson, &self.abiDoc)

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

func (self *NatSpec) Notice() (notice string, err error) {
	var abiKey [8]byte
	if len(self.data) < 10 {
		err = fmt.Errorf("Invalid transaction data")
		return
	}
	copy(abiKey[:], self.data[2:10])
	meth := self.makeAbi2method(abiKey)

	if meth == nil {
		err = fmt.Errorf("abi key %x does not match any method %v")
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

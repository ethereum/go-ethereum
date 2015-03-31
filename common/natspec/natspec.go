package natspec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/robertkrimen/otto"
	"strings"

	"github.com/ethereum/go-ethereum/common"
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

// TODO: should initialise with abi and userdoc jsons
func New(xeth *xeth.XEth, tx string) (self *NatSpec, err error) {

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
	codeHash := xeth.CodeAt(contractAddress)

	// retrieve natspec info content hash

	statereg := NewStateReg(xeth)

	natspecHash, err1 := statereg.GetNatSpec(codeHash)
	if err1 != nil {
		return nil, err1
	}

	// retrieve content

	content, err2 := statereg.GetContent(natspecHash)
	if err2 != nil {
		return nil, err2
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

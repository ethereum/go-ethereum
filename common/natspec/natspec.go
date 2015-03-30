package natspec

import (
	"encoding/json"
	"fmt"
	"github.com/robertkrimen/otto"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type abi2method map[[8]byte]*method

type NatSpec struct {
	jsvm                    *otto.Otto
	methods                 abi2method
	userDocJson, abiDocJson []byte
	userDoc                 userDoc
	// abiDoc abiDoc
}

// TODO: should initialise with abi and userdoc jsons
func New() (self *NatSpec, err error) {
	// fetch abi, userdoc
	abi := []byte("")
	userdoc := []byte("")
	return NewWithDocs(abi, userdoc)
}

func NewWithDocs(abiDocJson, userDocJson []byte) (self *NatSpec, err error) {

	self = &NatSpec{
		jsvm:        otto.New(),
		abiDocJson:  abiDocJson,
		userDocJson: userDocJson,
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
	if self.methods != nil {
		meth = self.methods[abiKey]
		return
	}
	self.methods = make(abi2method)
	for signature, m := range self.userDoc.Methods {
		name := strings.Split(signature, "(")[0]
		hash := []byte(common.Bytes2Hex(crypto.Sha3([]byte(signature))))
		var key [8]byte
		copy(key[:], hash[:8])
		self.methods[key] = meth
		meth.name = name
		if key == abiKey {
			meth = m
		}
	}
	return
}

func (self *NatSpec) Notice(tx string, abi string) (notice string, err error) {
	var abiKey [8]byte
	copy(abiKey[:], []byte(abi)[:8])
	meth := self.makeAbi2method(abiKey)
	if meth == nil {
		err = fmt.Errorf("abi key %x does not match any method %v")
		return
	}
	notice, err = self.noticeForMethod(tx, meth.name, meth.Notice)
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

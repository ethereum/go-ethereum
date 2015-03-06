package natspec

import (
	// "encoding/json"
	// "fmt"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/javascript"
	"io/ioutil"
)

type NatSpec struct {
	jsre *javascript.JSRE
}

func NewNATSpec(ethereum *eth.Ethereum, transaction string) (self *NatSpec, err error) {

	self = new(NatSpec)
	self.jsre = javascript.NewJSRE(ethereum)
	//self.jsre.LoadExtFile("/home/fefe/go-ethereum/ethutil/natspec/natspec.js")
	code, err := ioutil.ReadFile("natspec.js")
	if err != nil {
		return
	}

	_, err = self.jsre.Run(string(code))
	if err != nil {
		return
	}
	_, err = self.jsre.Run("var natspec = require('natspec');")
	if err != nil {
		return
	}

	self.jsre.Run("var transaction = " + transaction + ";")

	return
}

func (self *NatSpec) SetDescription(desc string) (err error) {

	_, err = self.jsre.Run("var expression = \"" + desc + "\";")
	return

}

func (self *NatSpec) SetABI(abi string) (err error) {

	_, err = self.jsre.Run("var abi = " + abi + ";")
	return

}

func (self *NatSpec) SetMethod(method string) (err error) {

	_, err = self.jsre.Run("var method = '" + method + "';")
	return

}

func (self *NatSpec) Parse() string {

	self.jsre.Run("var call = {method: method,abi: abi,transaction: transaction};")
	value, err := self.jsre.Run("natspec.evaluateExpression(expression, call);")
	if err != nil {
		return err.Error()
	}
	return value.String()

}

package natspec

import (
	"github.com/obscuren/otto"
	"io/ioutil"
)

type NatSpec struct {
	jsvm *otto.Otto
}

func NewNATSpec(transaction string) (self *NatSpec, err error) {

	self = new(NatSpec)
	self.jsvm = otto.New()
	code, err := ioutil.ReadFile("natspec.js")
	if err != nil {
		return
	}

	_, err = self.jsvm.Run(string(code))
	if err != nil {
		return
	}
	_, err = self.jsvm.Run("var natspec = require('natspec');")
	if err != nil {
		return
	}

	self.jsvm.Run("var transaction = " + transaction + ";")

	return
}

func (self *NatSpec) SetDescription(desc string) (err error) {

	_, err = self.jsvm.Run("var expression = \"" + desc + "\";")
	return

}

func (self *NatSpec) SetABI(abi string) (err error) {

	_, err = self.jsvm.Run("var abi = " + abi + ";")
	return

}

func (self *NatSpec) SetMethod(method string) (err error) {

	_, err = self.jsvm.Run("var method = '" + method + "';")
	return

}

func (self *NatSpec) Parse() string {

	self.jsvm.Run("var call = {method: method,abi: abi,transaction: transaction};")
	value, err := self.jsvm.Run("natspec.evaluateExpression(expression, call);")
	if err != nil {
		return err.Error()
	}
	return value.String()

}

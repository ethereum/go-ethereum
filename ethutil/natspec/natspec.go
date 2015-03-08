package natspec

import (
	"fmt"
	"github.com/obscuren/otto"
)

type NatSpec struct {
	jsvm *otto.Otto
}

// TODO: should initialise with abi and userdoc jsons
func NewNATSpec() (self *NatSpec, err error) {

	self = new(NatSpec)
	self.jsvm = otto.New()

	_, err = self.jsvm.Run(natspecJS)
	if err != nil {
		return
	}
	_, err = self.jsvm.Run("var natspec = require('natspec');")
	if err != nil {
		return
	}

	return
}

func (self *NatSpec) Notice(transaction, abi, method, expression string) (string, error) {
	var err error
	if _, err = self.jsvm.Run("var transaction = " + transaction + ";"); err != nil {
		return "", fmt.Errorf("natspec.js error setting transaction: %v", err)
	}

	if _, err = self.jsvm.Run("var abi = " + abi + ";"); err != nil {
		return "", fmt.Errorf("natspec.js error setting abi: %v", err)
	}

	if _, err = self.jsvm.Run("var method = '" + method + "';"); err != nil {
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

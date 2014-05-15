package main

import (
	"bufio"
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/robertkrimen/otto"
	"os"
)

type JSConsole struct {
	vm  *otto.Otto
	lib *ethpub.PEthereum
}

func NewJSConsole(ethereum *eth.Ethereum) *JSConsole {
	return &JSConsole{vm: otto.New(), lib: ethpub.NewPEthereum(ethereum)}
}

func (self *JSConsole) Start() {
	self.initBindings()

	fmt.Println("Eth JavaScript console")
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("eth >>> ")
		str, _, err := reader.ReadLine()
		if err != nil {
			fmt.Println("Error reading input", err)
		} else {
			self.ParseInput(string(str))
		}
	}
}

func (self *JSConsole) ParseInput(code string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("[native] error", r)
		}
	}()

	value, err := self.vm.Run(code)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(value)
}

func (self *JSConsole) initBindings() {
	t := &JSWrapper{self.lib, self.vm}

	self.vm.Set("eth", t)
}

// The JS wrapper attempts to wrap the PEthereum object and returns
// proper javascript objects
type JSWrapper struct {
	*ethpub.PEthereum
	vm *otto.Otto
}

func (self *JSWrapper) GetKey() otto.Value {
	return self.toVal(self.PEthereum.GetKey())
}

func (self *JSWrapper) GetStateObject(addr string) otto.Value {
	return self.toVal(self.PEthereum.GetStateObject(addr))
}

func (self *JSWrapper) Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr string) otto.Value {
	r, err := self.PEthereum.Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr)
	if err != nil {
		fmt.Println(err)

		return otto.UndefinedValue()
	}

	return self.toVal(r)
}

func (self *JSWrapper) Create(key, valueStr, gasStr, gasPriceStr, initStr, bodyStr string) otto.Value {
	r, err := self.PEthereum.Create(key, valueStr, gasStr, gasPriceStr, initStr, bodyStr)

	if err != nil {
		fmt.Println(err)

		return otto.UndefinedValue()
	}

	return self.toVal(r)
}

// Wrapper function
func (self *JSWrapper) toVal(v interface{}) otto.Value {
	result, err := self.vm.ToValue(v)

	if err != nil {
		fmt.Println(err)

		return otto.UndefinedValue()
	}

	return result
}

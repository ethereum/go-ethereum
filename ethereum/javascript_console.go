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

	fmt.Println("Eth JS Console")
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("eth >>> ")
		str, _, err := reader.ReadLine()
		if err != nil {
			fmt.Println("Error reading input", err)
		} else {
			if string(str) == "quit" {
				return
			}

			self.ParseInput(string(str))
		}
	}
}

func (self *JSConsole) ParseInput(code string) {
	value, err := self.vm.Run(code)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(value)
}

type OtherStruct struct {
	Test string
}

type JSWrapper struct {
	pub *ethpub.PEthereum
	vm  *otto.Otto
}

func (self *JSWrapper) GetKey() otto.Value {
	result, err := self.vm.ToValue(self.pub.GetKey())
	if err != nil {
		fmt.Println(err)

		return otto.UndefinedValue()
	}

	return result

}

func (self *JSConsole) initBindings() {
	t := &JSWrapper{self.lib, self.vm}

	self.vm.Set("eth", t)
}

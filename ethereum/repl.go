package main

import (
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/obscuren/otto"
)

type Repl interface {
	Start()
	Stop()
}

type JSRepl struct {
	re *JSRE

	prompt string
}

func NewJSRepl(ethereum *eth.Ethereum) *JSRepl {
	return &JSRepl{re: NewJSRE(ethereum), prompt: "> "}
}

func (self *JSRepl) Start() {
	self.read()
}

func (self *JSRepl) Stop() {
	self.re.Stop()
}

func (self *JSRepl) parseInput(code string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("[native] error", r)
		}
	}()

	value, err := self.re.Run(code)
	if err != nil {
		fmt.Println(err)
		return
	}

	self.PrintValue(value)
}

// The JSEthereum object attempts to wrap the PEthereum object and returns
// meaningful javascript objects
type JSBlock struct {
	*ethpub.PBlock
	eth *JSEthereum
}

func (self *JSBlock) GetTransaction(hash string) otto.Value {
	return self.eth.toVal(self.PBlock.GetTransaction(hash))
}

type JSEthereum struct {
	*ethpub.PEthereum
	vm *otto.Otto
}

func (self *JSEthereum) GetBlock(hash string) otto.Value {
	return self.toVal(&JSBlock{self.PEthereum.GetBlock(hash), self})
}

func (self *JSEthereum) GetPeers() otto.Value {
	return self.toVal(self.PEthereum.GetPeers())
}

func (self *JSEthereum) GetKey() otto.Value {
	return self.toVal(self.PEthereum.GetKey())
}

func (self *JSEthereum) GetStateObject(addr string) otto.Value {
	return self.toVal(self.PEthereum.GetStateObject(addr))
}

func (self *JSEthereum) GetStateKeyVals(addr string) otto.Value {
	return self.toVal(self.PEthereum.GetStateObject(addr).StateKeyVal(false))
}

func (self *JSEthereum) Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr string) otto.Value {
	r, err := self.PEthereum.Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr)
	if err != nil {
		fmt.Println(err)

		return otto.UndefinedValue()
	}

	return self.toVal(r)
}

func (self *JSEthereum) Create(key, valueStr, gasStr, gasPriceStr, scriptStr string) otto.Value {
	r, err := self.PEthereum.Create(key, valueStr, gasStr, gasPriceStr, scriptStr)

	if err != nil {
		fmt.Println(err)

		return otto.UndefinedValue()
	}

	return self.toVal(r)
}

func (self *JSEthereum) toVal(v interface{}) otto.Value {
	result, err := self.vm.ToValue(v)

	if err != nil {
		fmt.Println("Value unknown:", err)

		return otto.UndefinedValue()
	}

	return result
}

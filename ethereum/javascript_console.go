package main

import (
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/robertkrimen/otto"
)

type Repl interface {
	Start()
}

type JSRE struct {
	ethereum *eth.Ethereum
	vm       *otto.Otto
	lib      *ethpub.PEthereum

	blockChan  chan ethutil.React
	changeChan chan ethutil.React
	quitChan   chan bool

	objectCb map[string][]otto.Value
}

func NewJSRE(ethereum *eth.Ethereum) *JSRE {
	re := &JSRE{
		ethereum,
		otto.New(),
		ethpub.NewPEthereum(ethereum),
		make(chan ethutil.React, 1),
		make(chan ethutil.React, 1),
		make(chan bool),
		make(map[string][]otto.Value),
	}

	// We have to make sure that, whoever calls this, calls "Stop"
	go re.mainLoop()

	re.Bind("eth", &JSEthereum{re.lib, re.vm})
	t, _ := re.vm.Get("eth")
	t.Object().Set("watch", func(call otto.FunctionCall) otto.Value {
		addr, _ := call.Argument(0).ToString()
		cb := call.Argument(1)

		re.objectCb[addr] = append(re.objectCb[addr], cb)

		event := "object:" + string(ethutil.FromHex(addr))
		ethereum.Reactor().Subscribe(event, re.changeChan)

		return otto.UndefinedValue()
	})

	return re
}

func (self *JSRE) Stop() {
	// Kill the main loop
	self.quitChan <- true

	close(self.blockChan)
	close(self.quitChan)
	close(self.changeChan)
}

func (self *JSRE) mainLoop() {
	// Subscribe to events
	reactor := self.ethereum.Reactor()
	reactor.Subscribe("newBlock", self.blockChan)

out:
	for {
		select {
		case <-self.quitChan:
			break out
		case block := <-self.blockChan:
			if _, ok := block.Resource.(*ethchain.Block); ok {
			}
		case object := <-self.changeChan:
			if stateObject, ok := object.Resource.(*ethchain.StateObject); ok {
				for _, cb := range self.objectCb[ethutil.Hex(stateObject.Address())] {
					val, _ := self.vm.ToValue(ethpub.NewPStateObject(stateObject))
					cb.Call(cb, val)
				}
			} else if storageObject, ok := object.Resource.(*ethchain.StorageState); ok {
				fmt.Println(storageObject)
			}
		}
	}
}

func (self *JSRE) Bind(name string, v interface{}) {
	self.vm.Set(name, v)
}

func (self *JSRE) Run(code string) (otto.Value, error) {
	return self.vm.Run(code)
}

type JSRepl struct {
	re *JSRE
}

func NewJSRepl(ethereum *eth.Ethereum) *JSRepl {
	return &JSRepl{re: NewJSRE(ethereum)}
}

func (self *JSRepl) Start() {
	fmt.Println("Eth JavaScript console")
	self.read()
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

	fmt.Println(value)
}

// The JSEthereum object attempts to wrap the PEthereum object and returns
// meaningful javascript objects
type JSEthereum struct {
	*ethpub.PEthereum
	vm *otto.Otto
}

func (self *JSEthereum) GetKey() otto.Value {
	return self.toVal(self.PEthereum.GetKey())
}

func (self *JSEthereum) GetStateObject(addr string) otto.Value {
	return self.toVal(self.PEthereum.GetStateObject(addr))
}

func (self *JSEthereum) Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr string) otto.Value {
	r, err := self.PEthereum.Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr)
	if err != nil {
		fmt.Println(err)

		return otto.UndefinedValue()
	}

	return self.toVal(r)
}

func (self *JSEthereum) Create(key, valueStr, gasStr, gasPriceStr, initStr, bodyStr string) otto.Value {
	r, err := self.PEthereum.Create(key, valueStr, gasStr, gasPriceStr, initStr, bodyStr)

	if err != nil {
		fmt.Println(err)

		return otto.UndefinedValue()
	}

	return self.toVal(r)
}

func (self *JSEthereum) toVal(v interface{}) otto.Value {
	result, err := self.vm.ToValue(v)

	if err != nil {
		fmt.Println(err)

		return otto.UndefinedValue()
	}

	return result
}

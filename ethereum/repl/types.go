package ethrepl

import (
	"fmt"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/obscuren/otto"
)

type JSStateObject struct {
	*ethpub.PStateObject
	eth *JSEthereum
}

func (self *JSStateObject) EachStorage(call otto.FunctionCall) otto.Value {
	cb := call.Argument(0)
	self.PStateObject.EachStorage(func(key string, value *ethutil.Value) {
		value.Decode()

		cb.Call(self.eth.toVal(self), self.eth.toVal(key), self.eth.toVal(ethutil.Bytes2Hex(value.Bytes())))
	})

	return otto.UndefinedValue()
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
	return self.toVal(&JSStateObject{self.PEthereum.GetStateObject(addr), self})
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

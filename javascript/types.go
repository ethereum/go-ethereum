package javascript

import (
	"fmt"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/ui"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/obscuren/otto"
)

type JSStateObject struct {
	*xeth.JSObject
	eth *JSEthereum
}

func (self *JSStateObject) EachStorage(call otto.FunctionCall) otto.Value {
	cb := call.Argument(0)

	it := self.JSObject.Trie().Iterator()
	for it.Next() {
		cb.Call(self.eth.toVal(self), self.eth.toVal(ethutil.Bytes2Hex(it.Key)), self.eth.toVal(ethutil.Bytes2Hex(it.Value)))
	}

	return otto.UndefinedValue()
}

// The JSEthereum object attempts to wrap the PEthereum object and returns
// meaningful javascript objects
type JSBlock struct {
	*xeth.JSBlock
	eth *JSEthereum
}

func (self *JSBlock) GetTransaction(hash string) otto.Value {
	return self.eth.toVal(self.JSBlock.GetTransaction(hash))
}

type JSLog struct {
	Address string   `json:address`
	Topics  []string `json:topics`
	Number  int32    `json:number`
	Data    string   `json:data`
}

func NewJSLog(log state.Log) JSLog {
	return JSLog{
		Address: ethutil.Bytes2Hex(log.Address()),
		Topics:  nil, //ethutil.Bytes2Hex(log.Address()),
		Number:  0,
		Data:    ethutil.Bytes2Hex(log.Data()),
	}
}

type JSEthereum struct {
	*xeth.JSXEth
	vm       *otto.Otto
	ethereum *eth.Ethereum
}

func (self *JSEthereum) Block(v interface{}) otto.Value {
	if number, ok := v.(int64); ok {
		return self.toVal(&JSBlock{self.JSXEth.BlockByNumber(int32(number)), self})
	} else if hash, ok := v.(string); ok {
		return self.toVal(&JSBlock{self.JSXEth.BlockByHash(hash), self})
	}

	return otto.UndefinedValue()
}

func (self *JSEthereum) Peers() otto.Value {
	return self.toVal(self.JSXEth.Peers())
}

func (self *JSEthereum) Key() otto.Value {
	return self.toVal(self.JSXEth.Key())
}

func (self *JSEthereum) GetStateObject(addr string) otto.Value {
	return self.toVal(&JSStateObject{xeth.NewJSObject(self.JSXEth.World().SafeGet(ethutil.Hex2Bytes(addr))), self})
}

func (self *JSEthereum) Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr string) otto.Value {
	r, err := self.JSXEth.Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr)
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

func (self *JSEthereum) Messages(object map[string]interface{}) otto.Value {
	filter := ui.NewFilterFromMap(object, self.ethereum)

	logs := filter.Find()
	var jslogs []JSLog
	for _, m := range logs {
		jslogs = append(jslogs, NewJSLog(m))
	}

	v, _ := self.vm.ToValue(jslogs)

	return v
}

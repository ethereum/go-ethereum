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

type JSMessage struct {
	To        string `json:"to"`
	From      string `json:"from"`
	Input     string `json:"input"`
	Output    string `json:"output"`
	Path      int    `json:"path"`
	Origin    string `json:"origin"`
	Timestamp int32  `json:"timestamp"`
	Coinbase  string `json:"coinbase"`
	Block     string `json:"block"`
	Number    int32  `json:"number"`
}

func NewJSMessage(message *state.Message) JSMessage {
	return JSMessage{
		To:        ethutil.Bytes2Hex(message.To),
		From:      ethutil.Bytes2Hex(message.From),
		Input:     ethutil.Bytes2Hex(message.Input),
		Output:    ethutil.Bytes2Hex(message.Output),
		Path:      message.Path,
		Origin:    ethutil.Bytes2Hex(message.Origin),
		Timestamp: int32(message.Timestamp),
		Coinbase:  ethutil.Bytes2Hex(message.Origin),
		Block:     ethutil.Bytes2Hex(message.Block),
		Number:    int32(message.Number.Int64()),
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

	messages := filter.Find()
	var msgs []JSMessage
	for _, m := range messages {
		msgs = append(msgs, NewJSMessage(m))
	}

	v, _ := self.vm.ToValue(msgs)

	return v
}

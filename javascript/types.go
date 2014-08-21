package javascript

import (
	"fmt"

	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethpipe"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/obscuren/otto"
)

type JSStateObject struct {
	*ethpipe.JSObject
	eth *JSEthereum
}

func (self *JSStateObject) EachStorage(call otto.FunctionCall) otto.Value {
	cb := call.Argument(0)
	self.JSObject.EachStorage(func(key string, value *ethutil.Value) {
		value.Decode()

		cb.Call(self.eth.toVal(self), self.eth.toVal(key), self.eth.toVal(ethutil.Bytes2Hex(value.Bytes())))
	})

	return otto.UndefinedValue()
}

// The JSEthereum object attempts to wrap the PEthereum object and returns
// meaningful javascript objects
type JSBlock struct {
	*ethpipe.JSBlock
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

func NewJSMessage(message *ethstate.Message) JSMessage {
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
	*ethpipe.JSPipe
	vm       *otto.Otto
	ethereum *eth.Ethereum
}

func (self *JSEthereum) GetBlock(hash string) otto.Value {
	return self.toVal(&JSBlock{self.JSPipe.BlockByHash(hash), self})
}

func (self *JSEthereum) GetPeers() otto.Value {
	return self.toVal(self.JSPipe.Peers())
}

func (self *JSEthereum) GetKey() otto.Value {
	return self.toVal(self.JSPipe.Key())
}

func (self *JSEthereum) GetStateObject(addr string) otto.Value {
	return self.toVal(&JSStateObject{ethpipe.NewJSObject(self.JSPipe.World().SafeGet(ethutil.Hex2Bytes(addr))), self})
}

func (self *JSEthereum) Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr string) otto.Value {
	r, err := self.JSPipe.Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr)
	if err != nil {
		fmt.Println(err)

		return otto.UndefinedValue()
	}

	return self.toVal(r)
}

func (self *JSEthereum) Create(key, valueStr, gasStr, gasPriceStr, scriptStr string) otto.Value {
	r, err := self.JSPipe.Transact(key, "", valueStr, gasStr, gasPriceStr, scriptStr)

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
	filter := ethchain.NewFilterFromMap(object, self.ethereum)

	messages := filter.Find()
	var msgs []JSMessage
	for _, m := range messages {
		msgs = append(msgs, NewJSMessage(m))
	}

	v, _ := self.vm.ToValue(msgs)

	return v
}

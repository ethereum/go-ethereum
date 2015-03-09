package javascript

import (
	"fmt"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/obscuren/otto"
)

/*
JEth provides the the actual bindings for xeth (extended ethereum)
*/
type jeth struct {
	xeth  *xeth.XEth
	toVal func(v interface{}) otto.Value
}

func (self *jeth) IsMining(call otto.FunctionCall) otto.Value {
	return self.toVal(self.xeth.IsMining())
}

func (self *jeth) IsListening(call otto.FunctionCall) otto.Value {
	return self.toVal(self.xeth.IsListening())
}

func (self *jeth) IsContract(call otto.FunctionCall) otto.Value {
	address, err := call.Argument(0).ToString()
	if err != nil {
		return otto.UndefinedValue()
	}
	return self.toVal(self.xeth.IsContract(address))
}

func (self *jeth) GetCoinbase(call otto.FunctionCall) otto.Value {
	return self.toVal(self.xeth.Coinbase())
}

func (self *jeth) SetMining(call otto.FunctionCall) otto.Value {
	shouldmine, err := call.Argument(0).ToBoolean()
	if err != nil {
		return otto.UndefinedValue()
	}
	mining := self.xeth.SetMining(shouldmine)
	return self.toVal(mining)
}

func (self *jeth) SuggestPeer(call otto.FunctionCall) otto.Value {
	nodeURL, err := call.Argument(0).ToString()
	if err != nil {
		return otto.FalseValue()
	}
	if err := self.xeth.SuggestPeer(nodeURL); err != nil {
		return otto.FalseValue()
	}
	return otto.TrueValue()
}

func (self *jeth) Import(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) == 0 {
		fmt.Println("err: require file name")
		return otto.FalseValue()
	}

	fn, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	if err := self.xeth.Import(fn); err != nil {
		return otto.FalseValue()
	}

	return otto.TrueValue()
}

func (self *jeth) Export(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) == 0 {
		fmt.Println("err: require file name")
		return otto.FalseValue()
	}

	fn, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	if err := self.xeth.Export(fn); err != nil {
		return otto.FalseValue()
	}

	return otto.TrueValue()
}

// func (self *jethRE) DumpBlock(call otto.FunctionCall) otto.Value {
//  var dump state.World
//  var err error

//  if len(call.ArgumentList) > 0 {
//    if call.Argument(0).IsNumber() {
//      num, _ := call.Argument(0).ToInteger()
//      dump, err = self.xeth.DumpBlockByNumber(int32(num))
//    } else if call.Argument(0).IsString() {
//      hash, _ := call.Argument(0).ToString()
//      dump, err = self.xeth.DumpBlockByHash(hash)
//    } else {
//      fmt.Println("invalid argument for dump. Either hex string or number")
//      return otto.UndefinedValue()
//    }
//  } else {
//    dump, err = self.xeth.DumpBlockByNumber(-1)
//  }
//  if err != nil {
//    fmt.Println(err)
//    return otto.UndefinedValue()
//  }

//  return self.toVal(dump)

// }

// import (
// 	"fmt"
// 	"github.com/ethereum/go-ethereum/ethutil"
// 	"github.com/ethereum/go-ethereum/state"
// 	"github.com/ethereum/go-ethereum/xeth"
// 	"github.com/obscuren/otto"
// )

// type JSStateObject struct {
// 	*xeth.Object
// 	eth *JSEthereum
// }

// func (self *JSStateObject) EachStorage(call otto.FunctionCall) otto.Value {
// 	cb := call.Argument(0)

// 	it := self.Object.Trie().Iterator()
// 	for it.Next() {
// 		cb.Call(self.eth.toVal(self), self.eth.toVal(ethutil.Bytes2Hex(it.Key)), self.eth.toVal(ethutil.Bytes2Hex(it.Value)))
// 	}

// 	return otto.UndefinedValue()
// }

// // The JSEthereum object attempts to wrap the PEthereum object and returns
// // meaningful javascript objects
// type JSBlock struct {
// 	*xeth.Block
// 	eth *JSEthereum
// }

// func (self *JSBlock) GetTransaction(hash string) otto.Value {
// 	return self.eth.toVal(self.Block.GetTransaction(hash))
// }

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

func (self *jeth) Block(call otto.FunctionCall) otto.Value {
	var block *xeth.Block
	if len(call.ArgumentList) > 0 {
		if call.Argument(0).IsNumber() {
			num, _ := call.Argument(0).ToInteger()
			block = self.xeth.BlockByNumber(int32(num))
		} else if call.Argument(0).IsString() {
			hash, _ := call.Argument(0).ToString()
			block = self.xeth.BlockByHash(hash)
		} else {
			fmt.Println("invalid argument for dump. Either hex string or number")
			return otto.UndefinedValue()
		}
	} else {
		block = self.xeth.BlockByNumber(-1)
	}
	if block == nil {
		fmt.Println("block not found")
		return otto.UndefinedValue()
	}

	return self.toVal(block)
}

// func (self *JSEthereum) GetStateObject(addr string) otto.Value {
// 	return self.toVal(&JSStateObject{self.XEth.State().SafeGet(addr), self})
// }

// func (self *JSEthereum) Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr string) otto.Value {
// 	r, err := self.XEth.Transact(recipient, valueStr, gasStr, gasPriceStr, dataStr)
// 	if err != nil {
// 		fmt.Println(err)

// 		return otto.UndefinedValue()
// 	}

// 	return self.toVal(r)
// }

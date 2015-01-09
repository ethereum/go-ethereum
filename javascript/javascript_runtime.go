package javascript

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/obscuren/otto"
)

var jsrelogger = logger.NewLogger("JSRE")

type JSRE struct {
	ethereum *eth.Ethereum
	Vm       *otto.Otto
	pipe     *xeth.JSXEth

	events event.Subscription

	objectCb map[string][]otto.Value
}

func (jsre *JSRE) LoadExtFile(path string) {
	result, err := ioutil.ReadFile(path)
	if err == nil {
		jsre.Vm.Run(result)
	} else {
		jsrelogger.Infoln("Could not load file:", path)
	}
}

func (jsre *JSRE) LoadIntFile(file string) {
	assetPath := path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "cmd", "mist", "assets", "ext")
	jsre.LoadExtFile(path.Join(assetPath, file))
}

func NewJSRE(ethereum *eth.Ethereum) *JSRE {
	re := &JSRE{
		ethereum,
		otto.New(),
		xeth.NewJSXEth(ethereum),
		nil,
		make(map[string][]otto.Value),
	}

	// Init the JS lib
	re.Vm.Run(jsLib)

	// Load extra javascript files
	re.LoadIntFile("string.js")
	re.LoadIntFile("big.js")

	// Subscribe to events
	mux := ethereum.EventMux()
	re.events = mux.Subscribe(core.NewBlockEvent{})

	// We have to make sure that, whoever calls this, calls "Stop"
	go re.mainLoop()

	re.Bind("eth", &JSEthereum{re.pipe, re.Vm, ethereum})

	re.initStdFuncs()

	jsrelogger.Infoln("started")

	return re
}

func (self *JSRE) Bind(name string, v interface{}) {
	self.Vm.Set(name, v)
}

func (self *JSRE) Run(code string) (otto.Value, error) {
	return self.Vm.Run(code)
}

func (self *JSRE) Require(file string) error {
	if len(filepath.Ext(file)) == 0 {
		file += ".js"
	}

	fh, err := os.Open(file)
	if err != nil {
		return err
	}

	content, _ := ioutil.ReadAll(fh)
	self.Run("exports = {};(function() {" + string(content) + "})();")

	return nil
}

func (self *JSRE) Stop() {
	self.events.Unsubscribe()
	jsrelogger.Infoln("stopped")
}

func (self *JSRE) mainLoop() {
	for _ = range self.events.Chan() {
	}
}

func (self *JSRE) initStdFuncs() {
	t, _ := self.Vm.Get("eth")
	eth := t.Object()
	eth.Set("watch", self.watch)
	eth.Set("addPeer", self.addPeer)
	eth.Set("require", self.require)
	eth.Set("stopMining", self.stopMining)
	eth.Set("startMining", self.startMining)
	eth.Set("execBlock", self.execBlock)
	eth.Set("dump", self.dump)
	eth.Set("export", self.export)
}

/*
 * The following methods are natively implemented javascript functions
 */

func (self *JSRE) dump(call otto.FunctionCall) otto.Value {
	var block *types.Block

	if len(call.ArgumentList) > 0 {
		if call.Argument(0).IsNumber() {
			num, _ := call.Argument(0).ToInteger()
			block = self.ethereum.ChainManager().GetBlockByNumber(uint64(num))
		} else if call.Argument(0).IsString() {
			hash, _ := call.Argument(0).ToString()
			block = self.ethereum.ChainManager().GetBlock(ethutil.Hex2Bytes(hash))
		} else {
			fmt.Println("invalid argument for dump. Either hex string or number")
		}

		if block == nil {
			fmt.Println("block not found")

			return otto.UndefinedValue()
		}

	} else {
		block = self.ethereum.ChainManager().CurrentBlock()
	}

	statedb := state.New(block.Root(), self.ethereum.Db())
	v, _ := self.Vm.ToValue(statedb.Dump())

	return v
}

func (self *JSRE) stopMining(call otto.FunctionCall) otto.Value {
	v, _ := self.Vm.ToValue(utils.StopMining(self.ethereum))
	return v
}

func (self *JSRE) startMining(call otto.FunctionCall) otto.Value {
	v, _ := self.Vm.ToValue(utils.StartMining(self.ethereum))
	return v
}

// eth.watch
func (self *JSRE) watch(call otto.FunctionCall) otto.Value {
	addr, _ := call.Argument(0).ToString()
	var storageAddr string
	var cb otto.Value
	var storageCallback bool
	if len(call.ArgumentList) > 2 {
		storageCallback = true
		storageAddr, _ = call.Argument(1).ToString()
		cb = call.Argument(2)
	} else {
		cb = call.Argument(1)
	}

	if storageCallback {
		self.objectCb[addr+storageAddr] = append(self.objectCb[addr+storageAddr], cb)

		// event := "storage:" + string(ethutil.Hex2Bytes(addr)) + ":" + string(ethutil.Hex2Bytes(storageAddr))
		// self.ethereum.EventMux().Subscribe(event, self.changeChan)
	} else {
		self.objectCb[addr] = append(self.objectCb[addr], cb)

		// event := "object:" + string(ethutil.Hex2Bytes(addr))
		// self.ethereum.EventMux().Subscribe(event, self.changeChan)
	}

	return otto.UndefinedValue()
}

func (self *JSRE) addPeer(call otto.FunctionCall) otto.Value {
	host, err := call.Argument(0).ToString()
	if err != nil {
		return otto.FalseValue()
	}
	self.ethereum.SuggestPeer(host)

	return otto.TrueValue()
}

func (self *JSRE) require(call otto.FunctionCall) otto.Value {
	file, err := call.Argument(0).ToString()
	if err != nil {
		return otto.UndefinedValue()
	}
	if err := self.Require(file); err != nil {
		fmt.Println("err:", err)
		return otto.UndefinedValue()
	}

	t, _ := self.Vm.Get("exports")

	return t
}

func (self *JSRE) execBlock(call otto.FunctionCall) otto.Value {
	hash, err := call.Argument(0).ToString()
	if err != nil {
		return otto.UndefinedValue()
	}

	err = utils.BlockDo(self.ethereum, ethutil.Hex2Bytes(hash))
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	return otto.TrueValue()
}

func (self *JSRE) export(call otto.FunctionCall) otto.Value {
	fn, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	data := self.ethereum.ChainManager().Export()

	if err := ethutil.WriteFile(fn, data); err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	return otto.TrueValue()
}

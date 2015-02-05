/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Jeffrey Wilcke <i@jev.io>
 */
package main

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event/filter"
	"github.com/ethereum/go-ethereum/javascript"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/obscuren/qml"
)

type memAddr struct {
	Num   string
	Value string
}

// UI Library that has some basic functionality exposed
type UiLib struct {
	*xeth.XEth
	engine    *qml.Engine
	eth       *eth.Ethereum
	connected bool
	assetPath string
	// The main application window
	win      *qml.Window
	Db       *Debugger
	DbWindow *DebuggerWindow

	jsEngine *javascript.JSRE

	filterCallbacks map[int][]int
	filterManager   *filter.FilterManager

	miner *miner.Miner
}

func NewUiLib(engine *qml.Engine, eth *eth.Ethereum, assetPath string) *UiLib {
	lib := &UiLib{XEth: xeth.New(eth), engine: engine, eth: eth, assetPath: assetPath, jsEngine: javascript.NewJSRE(eth), filterCallbacks: make(map[int][]int)} //, filters: make(map[int]*xeth.JSFilter)}
	lib.miner = miner.New(eth.KeyManager().Address(), eth)
	lib.filterManager = filter.NewFilterManager(eth.EventMux())
	go lib.filterManager.Start()

	return lib
}

func (self *UiLib) Notef(args []interface{}) {
	guilogger.Infoln(args...)
}

func (self *UiLib) PastPeers() *ethutil.List {
	return ethutil.NewList([]string{})
	//return ethutil.NewList(eth.PastPeers())
}

func (self *UiLib) ImportTx(rlpTx string) {
	tx := types.NewTransactionFromBytes(ethutil.Hex2Bytes(rlpTx))
	err := self.eth.TxPool().Add(tx)
	if err != nil {
		guilogger.Infoln("import tx failed ", err)
	}
}

func (self *UiLib) EvalJavascriptFile(path string) {
	self.jsEngine.LoadExtFile(path[7:])
}

func (self *UiLib) EvalJavascriptString(str string) string {
	value, err := self.jsEngine.Run(str)
	if err != nil {
		return err.Error()
	}

	return fmt.Sprintf("%v", value)
}

func (ui *UiLib) OpenQml(path string) {
	container := NewQmlApplication(path[7:], ui)
	app := NewExtApplication(container, ui)

	go app.run()
}

func (ui *UiLib) OpenHtml(path string) {
	container := NewHtmlApplication(path, ui)
	app := NewExtApplication(container, ui)

	go app.run()
}

func (ui *UiLib) OpenBrowser() {
	ui.OpenHtml("file://" + ui.AssetPath("ext/home.html"))
}

func (ui *UiLib) Muted(content string) {
	component, err := ui.engine.LoadFile(ui.AssetPath("qml/muted.qml"))
	if err != nil {
		guilogger.Debugln(err)

		return
	}
	win := component.CreateWindow(nil)
	go func() {
		path := "file://" + ui.AssetPath("muted/index.html")
		win.Set("url", path)

		win.Show()
		win.Wait()
	}()
}

func (ui *UiLib) Connect(button qml.Object) {
	if !ui.connected {
		ui.eth.Start(SeedNode)
		ui.connected = true
		button.Set("enabled", false)
	}
}

func (ui *UiLib) ConnectToPeer(addr string, hexid string) {
	id, err := discover.HexID(hexid)
	if err != nil {
		guilogger.Errorf("bad node ID: %v", err)
		return
	}
	if err := ui.eth.SuggestPeer(addr, id); err != nil {
		guilogger.Infoln(err)
	}
}

func (ui *UiLib) AssetPath(p string) string {
	return path.Join(ui.assetPath, p)
}

func (self *UiLib) StartDbWithContractAndData(contractHash, data string) {
	dbWindow := NewDebuggerWindow(self)
	object := self.eth.ChainManager().State().GetStateObject(ethutil.Hex2Bytes(contractHash))
	if len(object.Code) > 0 {
		dbWindow.SetCode(ethutil.Bytes2Hex(object.Code))
	}
	dbWindow.SetData(data)

	dbWindow.Show()
}

func (self *UiLib) StartDbWithCode(code string) {
	dbWindow := NewDebuggerWindow(self)
	dbWindow.SetCode(code)
	dbWindow.Show()
}

func (self *UiLib) StartDebugger() {
	dbWindow := NewDebuggerWindow(self)

	dbWindow.Show()
}

func (self *UiLib) Transact(params map[string]interface{}) (string, error) {
	object := mapToTxParams(params)

	return self.XEth.Transact(
		object["to"],
		object["value"],
		object["gas"],
		object["gasPrice"],
		object["data"],
	)
}

func (self *UiLib) Compile(code string) (string, error) {
	bcode, err := ethutil.Compile(code, false)
	if err != nil {
		return err.Error(), err
	}

	return ethutil.Bytes2Hex(bcode), err
}

func (self *UiLib) Call(params map[string]interface{}) (string, error) {
	object := mapToTxParams(params)

	return self.XEth.Execute(
		object["to"],
		object["value"],
		object["gas"],
		object["gasPrice"],
		object["data"],
	)
}

func (self *UiLib) AddLocalTransaction(to, data, gas, gasPrice, value string) int {
	return self.miner.AddLocalTx(&miner.LocalTx{
		To:       ethutil.Hex2Bytes(to),
		Data:     ethutil.Hex2Bytes(data),
		Gas:      gas,
		GasPrice: gasPrice,
		Value:    value,
	}) - 1
}

func (self *UiLib) RemoveLocalTransaction(id int) {
	self.miner.RemoveLocalTx(id)
}

func (self *UiLib) SetGasPrice(price string) {
	self.miner.MinAcceptedGasPrice = ethutil.Big(price)
}

func (self *UiLib) SetExtra(extra string) {
	self.miner.Extra = extra
}

func (self *UiLib) ToggleMining() bool {
	if !self.miner.Mining() {
		self.miner.Start()

		return true
	} else {
		self.miner.Stop()

		return false
	}
}

func (self *UiLib) ToHex(data string) string {
	return "0x" + ethutil.Bytes2Hex([]byte(data))
}

func (self *UiLib) ToAscii(data string) string {
	start := 0
	if len(data) > 1 && data[0:2] == "0x" {
		start = 2
	}
	return string(ethutil.Hex2Bytes(data[start:]))
}

/// Ethereum filter methods
func (self *UiLib) NewFilter(object map[string]interface{}, view *qml.Common) (id int) {
	/* TODO remove me
	filter := qt.NewFilterFromMap(object, self.eth)
	filter.MessageCallback = func(messages state.Messages) {
		view.Call("messages", xeth.ToMessages(messages), id)
	}
	id = self.filterManager.InstallFilter(filter)
	return id
	*/
	return 0
}

func (self *UiLib) NewFilterString(typ string, view *qml.Common) (id int) {
	/* TODO remove me
	filter := core.NewFilter(self.eth)
	filter.BlockCallback = func(block *types.Block) {
		view.Call("messages", "{}", id)
	}
	id = self.filterManager.InstallFilter(filter)
	return id
	*/
	return 0
}

func (self *UiLib) Messages(id int) *ethutil.List {
	/* TODO remove me
	filter := self.filterManager.GetFilter(id)
	if filter != nil {
		messages := xeth.ToMessages(filter.Find())

		return messages
	}
	*/

	return ethutil.EmptyList()
}

func (self *UiLib) ReadFile(p string) string {
	content, err := ioutil.ReadFile(self.AssetPath(path.Join("ext", p)))
	if err != nil {
		guilogger.Infoln("error reading file", p, ":", err)
	}
	return string(content)
}

func (self *UiLib) UninstallFilter(id int) {
	self.filterManager.UninstallFilter(id)
}

func mapToTxParams(object map[string]interface{}) map[string]string {
	// Default values
	if object["from"] == nil {
		object["from"] = ""
	}
	if object["to"] == nil {
		object["to"] = ""
	}
	if object["value"] == nil {
		object["value"] = ""
	}
	if object["gas"] == nil {
		object["gas"] = ""
	}
	if object["gasPrice"] == nil {
		object["gasPrice"] = ""
	}

	var dataStr string
	var data []string
	if list, ok := object["data"].(*qml.List); ok {
		list.Convert(&data)
	} else if str, ok := object["data"].(string); ok {
		data = []string{str}
	}

	for _, str := range data {
		if ethutil.IsHex(str) {
			str = str[2:]

			if len(str) != 64 {
				str = ethutil.LeftPadString(str, 64)
			}
		} else {
			str = ethutil.Bytes2Hex(ethutil.LeftPadBytes(ethutil.Big(str).Bytes(), 32))
		}

		dataStr += str
	}
	object["data"] = dataStr

	conv := make(map[string]string)
	for key, value := range object {
		if v, ok := value.(string); ok {
			conv[key] = v
		}
	}

	return conv
}

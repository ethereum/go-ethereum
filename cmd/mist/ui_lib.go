// Copyright (c) 2013-2014, Jeffrey Wilcke. All rights reserved.
//
// This library is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public
// License as published by the Free Software Foundation; either
// version 2.1 of the License, or (at your option) any later version.
//
// This library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this library; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston,
// MA 02110-1301  USA

package main

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event/filter"
	"github.com/ethereum/go-ethereum/javascript"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/ui/qt"
	"github.com/ethereum/go-ethereum/xeth"
	"gopkg.in/qml.v1"
)

type memAddr struct {
	Num   string
	Value string
}

// UI Library that has some basic functionality exposed
type UiLib struct {
	*xeth.JSXEth
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
	lib := &UiLib{JSXEth: xeth.NewJSXEth(eth), engine: engine, eth: eth, assetPath: assetPath, jsEngine: javascript.NewJSRE(eth), filterCallbacks: make(map[int][]int)} //, filters: make(map[int]*xeth.JSFilter)}
	lib.miner = miner.New(eth.KeyManager().Address(), eth)
	lib.filterManager = filter.NewFilterManager(eth.EventMux())

	return lib
}

func (self *UiLib) Notef(args []interface{}) {
	guilogger.Infoln(args...)
}

func (self *UiLib) LookupDomain(domain string) string {
	world := self.World()

	if len(domain) > 32 {
		domain = string(crypto.Sha3([]byte(domain)))
	}
	data := world.Config().Get("DnsReg").StorageString(domain).Bytes()

	// Left padded = A record, Right padded = CNAME
	if len(data) > 0 && data[0] == 0 {
		data = bytes.TrimLeft(data, "\x00")
		var ipSlice []string
		for _, d := range data {
			ipSlice = append(ipSlice, strconv.Itoa(int(d)))
		}

		return strings.Join(ipSlice, ".")
	} else {
		data = bytes.TrimRight(data, "\x00")

		return string(data)
	}
}

func (self *UiLib) LookupName(addr string) string {
	var (
		nameReg = self.World().Config().Get("NameReg")
		lookup  = nameReg.Storage(ethutil.Hex2Bytes(addr))
	)

	if lookup.Len() != 0 {
		return strings.Trim(lookup.Str(), "\x00")
	}

	return addr
}

func (self *UiLib) LookupAddress(name string) string {
	var (
		nameReg = self.World().Config().Get("NameReg")
		lookup  = nameReg.Storage(ethutil.RightPadBytes([]byte(name), 32))
	)

	if lookup.Len() != 0 {
		return ethutil.Bytes2Hex(lookup.Bytes())
	}

	return ""
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
		ui.eth.Start(true)
		ui.connected = true
		button.Set("enabled", false)
	}
}

func (ui *UiLib) ConnectToPeer(addr string) {
	ui.eth.SuggestPeer(addr)
}

func (ui *UiLib) AssetPath(p string) string {
	return path.Join(ui.assetPath, p)
}

func (self *UiLib) StartDbWithContractAndData(contractHash, data string) {
	dbWindow := NewDebuggerWindow(self)
	object := self.eth.ChainManager().State().GetStateObject(ethutil.Hex2Bytes(contractHash))
	if len(object.Code) > 0 {
		dbWindow.SetCode("0x" + ethutil.Bytes2Hex(object.Code))
	}
	dbWindow.SetData("0x" + data)

	dbWindow.Show()
}

func (self *UiLib) StartDbWithCode(code string) {
	dbWindow := NewDebuggerWindow(self)
	dbWindow.SetCode("0x" + code)
	dbWindow.Show()
}

func (self *UiLib) StartDebugger() {
	dbWindow := NewDebuggerWindow(self)

	dbWindow.Show()
}

func (self *UiLib) NewFilter(object map[string]interface{}) (id int) {
	filter := qt.NewFilterFromMap(object, self.eth)
	filter.MessageCallback = func(messages state.Messages) {
		self.win.Root().Call("invokeFilterCallback", xeth.ToJSMessages(messages), id)
	}
	id = self.filterManager.InstallFilter(filter)
	return id
}

func (self *UiLib) NewFilterString(typ string) (id int) {
	filter := core.NewFilter(self.eth)
	filter.BlockCallback = func(block *types.Block) {
		if self.win != nil && self.win.Root() != nil {
			self.win.Root().Call("invokeFilterCallback", "{}", id)
		} else {
			fmt.Println("QML is lagging")
		}
	}
	id = self.filterManager.InstallFilter(filter)
	return id
}

func (self *UiLib) Messages(id int) *ethutil.List {
	filter := self.filterManager.GetFilter(id)
	if filter != nil {
		messages := xeth.ToJSMessages(filter.Find())

		return messages
	}

	return ethutil.EmptyList()
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

func (self *UiLib) Transact(params map[string]interface{}) (string, error) {
	object := mapToTxParams(params)

	return self.JSXEth.Transact(
		object["from"],
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

	return self.JSXEth.Execute(
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

/*
// XXX Refactor me & MOVE
func (self *Ethereum) InstallFilter(filter *core.Filter) (id int) {
	return self.filterManager.InstallFilter(filter)
}
func (self *Ethereum) UninstallFilter(id int)        { self.filterManager.UninstallFilter(id) }
func (self *Ethereum) GetFilter(id int) *core.Filter { return self.filterManager.GetFilter(id) }
*/

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
	"io/ioutil"
	"path"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/event/filter"
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
	win *qml.Window

	filterCallbacks map[int][]int
	filterManager   *filter.FilterManager
}

func NewUiLib(engine *qml.Engine, eth *eth.Ethereum, assetPath, libPath string) *UiLib {
	x := xeth.New(eth, nil)
	lib := &UiLib{
		XEth:            x,
		engine:          engine,
		eth:             eth,
		assetPath:       assetPath,
		filterCallbacks: make(map[int][]int),
	}
	lib.filterManager = filter.NewFilterManager(eth.EventMux())
	go lib.filterManager.Start()

	return lib
}

func (self *UiLib) Notef(args []interface{}) {
	guilogger.Infoln(args...)
}

func (self *UiLib) ImportTx(rlpTx string) {
	tx := types.NewTransactionFromBytes(common.Hex2Bytes(rlpTx))
	err := self.eth.TxPool().Add(tx)
	if err != nil {
		guilogger.Infoln("import tx failed ", err)
	}
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
		ui.eth.Start()
		ui.connected = true
		button.Set("enabled", false)
	}
}

func (ui *UiLib) ConnectToPeer(nodeURL string) {
	if err := ui.eth.AddPeer(nodeURL); err != nil {
		guilogger.Infoln("AddPeer error: " + err.Error())
	}
}

func (ui *UiLib) AssetPath(p string) string {
	return path.Join(ui.assetPath, p)
}

func (self *UiLib) Transact(params map[string]interface{}) (string, error) {
	object := mapToTxParams(params)

	return self.XEth.Transact(
		object["from"],
		object["to"],
		object["value"],
		object["gas"],
		object["gasPrice"],
		object["data"],
	)
}

func (self *UiLib) Call(params map[string]interface{}) (string, error) {
	object := mapToTxParams(params)

	return self.XEth.Call(
		object["from"],
		object["to"],
		object["value"],
		object["gas"],
		object["gasPrice"],
		object["data"],
	)
}

func (self *UiLib) AddLocalTransaction(to, data, gas, gasPrice, value string) int {
	return 0
	/*
		return self.miner.AddLocalTx(&miner.LocalTx{
			To:       common.Hex2Bytes(to),
			Data:     common.Hex2Bytes(data),
			Gas:      gas,
			GasPrice: gasPrice,
			Value:    value,
		}) - 1
	*/
}

func (self *UiLib) RemoveLocalTransaction(id int) {
	//self.miner.RemoveLocalTx(id)
}

func (self *UiLib) ToggleMining() bool {
	if !self.eth.IsMining() {
		err := self.eth.StartMining()
		return err == nil
	} else {
		self.eth.StopMining()
		return false
	}
}

func (self *UiLib) ToHex(data string) string {
	return "0x" + common.Bytes2Hex([]byte(data))
}

func (self *UiLib) ToAscii(data string) string {
	start := 0
	if len(data) > 1 && data[0:2] == "0x" {
		start = 2
	}
	return string(common.Hex2Bytes(data[start:]))
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

func (self *UiLib) Messages(id int) *common.List {
	/* TODO remove me
	filter := self.filterManager.GetFilter(id)
	if filter != nil {
		messages := xeth.ToMessages(filter.Find())

		return messages
	}
	*/

	return common.EmptyList()
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
		if common.IsHex(str) {
			str = str[2:]

			if len(str) != 64 {
				str = common.LeftPadString(str, 64)
			}
		} else {
			str = common.Bytes2Hex(common.LeftPadBytes(common.Big(str).Bytes(), 32))
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

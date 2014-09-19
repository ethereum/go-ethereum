package main

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethpipe"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/go-ethereum/javascript"
	"gopkg.in/qml.v1"
)

type memAddr struct {
	Num   string
	Value string
}

// UI Library that has some basic functionality exposed
type UiLib struct {
	*ethpipe.JSPipe
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
	//filters         map[int]*ethpipe.JSFilter
}

func NewUiLib(engine *qml.Engine, eth *eth.Ethereum, assetPath string) *UiLib {
	return &UiLib{JSPipe: ethpipe.NewJSPipe(eth), engine: engine, eth: eth, assetPath: assetPath, jsEngine: javascript.NewJSRE(eth), filterCallbacks: make(map[int][]int)} //, filters: make(map[int]*ethpipe.JSFilter)}
}

func (self *UiLib) Notef(args []interface{}) {
	logger.Infoln(args...)
}

func (self *UiLib) LookupDomain(domain string) string {
	world := self.World()

	if len(domain) > 32 {
		domain = string(ethcrypto.Sha3Bin([]byte(domain)))
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

func (self *UiLib) PastPeers() *ethutil.List {
	return ethutil.NewList(eth.PastPeers())
}

func (self *UiLib) ImportTx(rlpTx string) {
	tx := ethchain.NewTransactionFromBytes(ethutil.Hex2Bytes(rlpTx))
	self.eth.TxPool().QueueTransaction(tx)
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
		logger.Debugln(err)

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
	ui.eth.ConnectToPeer(addr)
}

func (ui *UiLib) AssetPath(p string) string {
	return path.Join(ui.assetPath, p)
}

func (self *UiLib) StartDbWithContractAndData(contractHash, data string) {
	dbWindow := NewDebuggerWindow(self)
	object := self.eth.StateManager().CurrentState().GetStateObject(ethutil.Hex2Bytes(contractHash))
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

func (self *UiLib) NewFilter(object map[string]interface{}) int {
	filter, id := self.eth.InstallFilter(object)
	filter.MessageCallback = func(messages ethstate.Messages) {
		self.win.Root().Call("invokeFilterCallback", ethpipe.ToJSMessages(messages), id)
	}

	return id
}

func (self *UiLib) NewFilterString(typ string) int {
	filter, id := self.eth.InstallFilter(nil)
	filter.BlockCallback = func(block *ethchain.Block) {
		self.win.Root().Call("invokeFilterCallback", "{}", id)
	}

	return id
}

func (self *UiLib) Messages(id int) *ethutil.List {
	filter := self.eth.GetFilter(id)
	if filter != nil {
		messages := filter.Find()

		return ethpipe.ToJSMessages(messages)
	}

	return ethutil.EmptyList()
}

func (self *UiLib) UninstallFilter(id int) {
	self.eth.UninstallFilter(id)
}

func (self *UiLib) Transact(object map[string]interface{}) (*ethpipe.JSReceipt, error) {
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

	return self.JSPipe.Transact(
		object["from"].(string),
		object["to"].(string),
		object["value"].(string),
		object["gas"].(string),
		object["gasPrice"].(string),
		dataStr,
	)
}

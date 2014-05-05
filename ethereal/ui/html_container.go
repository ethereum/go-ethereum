package ethui

import (
	"errors"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/go-qml/qml"
	"path/filepath"
)

type HtmlApplication struct {
	win     *qml.Window
	webView qml.Object
	engine  *qml.Engine
	lib     *UiLib
	path    string
}

func NewHtmlApplication(path string, lib *UiLib) *HtmlApplication {
	engine := qml.NewEngine()

	return &HtmlApplication{engine: engine, lib: lib, path: path}

}

func (app *HtmlApplication) Create() error {
	component, err := app.engine.LoadFile(app.lib.AssetPath("qml/webapp.qml"))
	if err != nil {
		return err
	}

	if filepath.Ext(app.path) == "eth" {
		return errors.New("Ethereum package not yet supported")

		// TODO
		ethutil.OpenPackage(app.path)
	}

	win := component.CreateWindow(nil)
	win.Set("url", app.path)
	webView := win.ObjectByName("webView")

	app.win = win
	app.webView = webView

	return nil
}

func (app *HtmlApplication) Engine() *qml.Engine {
	return app.engine
}

func (app *HtmlApplication) Window() *qml.Window {
	return app.win
}

func (app *HtmlApplication) NewBlock(block *ethchain.Block) {
	b := &ethpub.PBlock{Number: int(block.BlockInfo().Number), Hash: ethutil.Hex(block.Hash())}
	app.webView.Call("onNewBlockCb", b)
}

func (app *HtmlApplication) ObjectChanged(stateObject *ethchain.StateObject) {
	app.webView.Call("onObjectChangeCb", ethpub.NewPStateObject(stateObject))
}

func (app *HtmlApplication) StorageChanged(storageObject *ethchain.StorageState) {
	app.webView.Call("onStorageChangeCb", ethpub.NewPStorageState(storageObject))
}

func (app *HtmlApplication) Destroy() {
	app.engine.Destroy()
}

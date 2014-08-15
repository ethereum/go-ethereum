package main

import (
	"fmt"
	"runtime"

	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
	"gopkg.in/qml.v1"
)

type QmlApplication struct {
	win    *qml.Window
	engine *qml.Engine
	lib    *UiLib
	path   string
}

func NewQmlApplication(path string, lib *UiLib) *QmlApplication {
	engine := qml.NewEngine()
	return &QmlApplication{engine: engine, path: path, lib: lib}
}

func (app *QmlApplication) Create() error {
	path := string(app.path)

	// For some reason for windows we get /c:/path/to/something, windows doesn't like the first slash but is fine with the others so we are removing it
	if app.path[0] == '/' && runtime.GOOS == "windows" {
		path = app.path[1:]
	}

	component, err := app.engine.LoadFile(path)
	if err != nil {
		logger.Warnln(err)
	}
	app.win = component.CreateWindow(nil)

	return nil
}

func (app *QmlApplication) Destroy() {
	app.engine.Destroy()
}

func (app *QmlApplication) NewWatcher(quitChan chan bool) {
}

// Events
func (app *QmlApplication) NewBlock(block *ethchain.Block) {
	pblock := &ethpub.PBlock{Number: int(block.BlockInfo().Number), Hash: ethutil.Bytes2Hex(block.Hash())}
	app.win.Call("onNewBlockCb", pblock)
}

func (app *QmlApplication) ObjectChanged(stateObject *ethstate.StateObject) {
	app.win.Call("onObjectChangeCb", ethpub.NewPStateObject(stateObject))
}

func (app *QmlApplication) StorageChanged(storageObject *ethstate.StorageState) {
	app.win.Call("onStorageChangeCb", ethpub.NewPStorageState(storageObject))
}

func (self *QmlApplication) Messages(msgs ethstate.Messages, id string) {
	fmt.Println("IMPLEMENT QML APPLICATION MESSAGES METHOD")
}

// Getters
func (app *QmlApplication) Engine() *qml.Engine {
	return app.engine
}
func (app *QmlApplication) Window() *qml.Window {
	return app.win
}

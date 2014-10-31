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
	"fmt"
	"runtime"

	"github.com/ethereum/go-ethereum/chain"
	"github.com/ethereum/go-ethereum/ethstate"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/xeth"
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
		guilogger.Warnln(err)
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
func (app *QmlApplication) NewBlock(block *chain.Block) {
	pblock := &xeth.JSBlock{Number: int(block.BlockInfo().Number), Hash: ethutil.Bytes2Hex(block.Hash())}
	app.win.Call("onNewBlockCb", pblock)
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

func (app *QmlApplication) Post(data string, s int) {}

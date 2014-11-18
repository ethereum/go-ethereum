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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"github.com/ethereum/go-ethereum/chain/types"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/javascript"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/howeyc/fsnotify"
	"gopkg.in/qml.v1"
)

type HtmlApplication struct {
	win     *qml.Window
	webView qml.Object
	engine  *qml.Engine
	lib     *UiLib
	path    string
	watcher *fsnotify.Watcher
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
		//ethutil.OpenPackage(app.path)
	}

	win := component.CreateWindow(nil)
	win.Set("url", app.path)
	webView := win.ObjectByName("webView")

	app.win = win
	app.webView = webView

	return nil
}

func (app *HtmlApplication) RootFolder() string {
	folder, err := url.Parse(app.path)
	if err != nil {
		return ""
	}
	return path.Dir(ethutil.WindonizePath(folder.RequestURI()))
}
func (app *HtmlApplication) RecursiveFolders() []os.FileInfo {
	files, _ := ioutil.ReadDir(app.RootFolder())
	var folders []os.FileInfo
	for _, file := range files {
		if file.IsDir() {
			folders = append(folders, file)
		}
	}
	return folders
}

func (app *HtmlApplication) NewWatcher(quitChan chan bool) {
	var err error

	app.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		guilogger.Infoln("Could not create new auto-reload watcher:", err)
		return
	}
	err = app.watcher.Watch(app.RootFolder())
	if err != nil {
		guilogger.Infoln("Could not start auto-reload watcher:", err)
		return
	}
	for _, folder := range app.RecursiveFolders() {
		fullPath := app.RootFolder() + "/" + folder.Name()
		app.watcher.Watch(fullPath)
	}

	go func() {
	out:
		for {
			select {
			case <-quitChan:
				app.watcher.Close()
				break out
			case <-app.watcher.Event:
				//guilogger.Debugln("Got event:", ev)
				app.webView.Call("reload")
			case err := <-app.watcher.Error:
				// TODO: Do something here
				guilogger.Infoln("Watcher error:", err)
			}
		}
	}()

}

func (app *HtmlApplication) Engine() *qml.Engine {
	return app.engine
}

func (app *HtmlApplication) Window() *qml.Window {
	return app.win
}

func (app *HtmlApplication) NewBlock(block *types.Block) {
	b := &xeth.JSBlock{Number: int(block.BlockInfo().Number), Hash: ethutil.Bytes2Hex(block.Hash())}
	app.webView.Call("onNewBlockCb", b)
}

func (self *HtmlApplication) Messages(messages state.Messages, id string) {
	var msgs []javascript.JSMessage
	for _, m := range messages {
		msgs = append(msgs, javascript.NewJSMessage(m))
	}

	b, _ := json.Marshal(msgs)

	self.webView.Call("onWatchedCb", string(b), id)
}

func (app *HtmlApplication) Destroy() {
	app.engine.Destroy()
}

func (app *HtmlApplication) Post(data string, seed int) {
	fmt.Println("about to call 'post'")
	app.webView.Call("post", seed, data)
}

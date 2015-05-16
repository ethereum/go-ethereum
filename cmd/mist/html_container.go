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
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/howeyc/fsnotify"
	"github.com/obscuren/qml"
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
		//common.OpenPackage(app.path)
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
	return filepath.Dir(common.WindonizePath(folder.RequestURI()))
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
	b := &xeth.Block{Number: int(block.NumberU64()), Hash: block.Hash().Hex()}
	app.webView.Call("onNewBlockCb", b)
}

func (app *HtmlApplication) Destroy() {
	app.engine.Destroy()
}

func (app *HtmlApplication) Post(data string, seed int) {
	fmt.Println("about to call 'post'")
	app.webView.Call("post", seed, data)
}

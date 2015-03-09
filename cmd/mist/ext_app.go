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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/obscuren/qml"
)

type AppContainer interface {
	Create() error
	Destroy()

	Window() *qml.Window
	Engine() *qml.Engine

	NewBlock(*types.Block)
	NewWatcher(chan bool)
	Post(string, int)
}

type ExtApplication struct {
	*xeth.XEth
	eth core.Backend

	events          event.Subscription
	watcherQuitChan chan bool

	filters map[string]*core.Filter

	container AppContainer
	lib       *UiLib
}

func NewExtApplication(container AppContainer, lib *UiLib) *ExtApplication {
	return &ExtApplication{
		XEth:            xeth.New(lib.eth),
		eth:             lib.eth,
		watcherQuitChan: make(chan bool),
		filters:         make(map[string]*core.Filter),
		container:       container,
		lib:             lib,
	}
}

func (app *ExtApplication) run() {
	// Set the "eth" api on to the containers context
	context := app.container.Engine().Context()
	context.SetVar("eth", app)
	context.SetVar("ui", app.lib)

	err := app.container.Create()
	if err != nil {
		guilogger.Errorln(err)
		return
	}

	// Call the main loop
	go app.mainLoop()

	app.container.NewWatcher(app.watcherQuitChan)

	win := app.container.Window()
	win.Show()
	win.Wait()

	app.stop()
}

func (app *ExtApplication) stop() {
	app.events.Unsubscribe()

	// Kill the main loop
	app.watcherQuitChan <- true

	app.container.Destroy()
}

func (app *ExtApplication) mainLoop() {
	for ev := range app.events.Chan() {
		switch ev := ev.(type) {
		case core.NewBlockEvent:
			app.container.NewBlock(ev.Block)

			/* TODO remove
			case state.Messages:
				for id, filter := range app.filters {
					msgs := filter.FilterMessages(ev)
					if len(msgs) > 0 {
						app.container.Messages(msgs, id)
					}
				}
			*/
		}
	}
}

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
	"encoding/json"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/javascript"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/ui/qt"
	"github.com/ethereum/go-ethereum/xeth"
	"gopkg.in/qml.v1"
)

type AppContainer interface {
	Create() error
	Destroy()

	Window() *qml.Window
	Engine() *qml.Engine

	NewBlock(*types.Block)
	NewWatcher(chan bool)
	Messages(state.Messages, string)
	Post(string, int)
}

type ExtApplication struct {
	*xeth.JSXEth
	eth core.EthManager

	events          event.Subscription
	watcherQuitChan chan bool

	filters map[string]*core.Filter

	container AppContainer
	lib       *UiLib
}

func NewExtApplication(container AppContainer, lib *UiLib) *ExtApplication {
	return &ExtApplication{
		JSXEth:          xeth.NewJSXEth(lib.eth),
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

	// Subscribe to events
	mux := app.lib.eth.EventMux()
	app.events = mux.Subscribe(core.NewBlockEvent{}, state.Messages(nil))

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

		case state.Messages:
			for id, filter := range app.filters {
				msgs := filter.FilterMessages(ev)
				if len(msgs) > 0 {
					app.container.Messages(msgs, id)
				}
			}
		}
	}
}

func (self *ExtApplication) Watch(filterOptions map[string]interface{}, identifier string) {
	self.filters[identifier] = qt.NewFilterFromMap(filterOptions, self.eth)
}

func (self *ExtApplication) GetMessages(object map[string]interface{}) string {
	filter := qt.NewFilterFromMap(object, self.eth)

	messages := filter.Find()
	var msgs []javascript.JSMessage
	for _, m := range messages {
		msgs = append(msgs, javascript.NewJSMessage(m))
	}

	b, err := json.Marshal(msgs)
	if err != nil {
		return "{\"error\":" + err.Error() + "}"
	}

	return string(b)
}

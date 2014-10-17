package main

import (
	"encoding/json"

	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethpipe"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/event"
	"github.com/ethereum/go-ethereum/javascript"
	"gopkg.in/qml.v1"
)

type AppContainer interface {
	Create() error
	Destroy()

	Window() *qml.Window
	Engine() *qml.Engine

	NewBlock(*ethchain.Block)
	NewWatcher(chan bool)
	Messages(ethstate.Messages, string)
	Post(string, int)
}

type ExtApplication struct {
	*ethpipe.JSPipe
	eth ethchain.EthManager

	events          event.Subscription
	watcherQuitChan chan bool

	filters map[string]*ethchain.Filter

	container AppContainer
	lib       *UiLib
}

func NewExtApplication(container AppContainer, lib *UiLib) *ExtApplication {
	return &ExtApplication{
		JSPipe:          ethpipe.NewJSPipe(lib.eth),
		eth:             lib.eth,
		watcherQuitChan: make(chan bool),
		filters:         make(map[string]*ethchain.Filter),
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
		logger.Errorln(err)
		return
	}

	// Subscribe to events
	mux := app.lib.eth.EventMux()
	app.events = mux.Subscribe(ethchain.NewBlockEvent{}, ethstate.Messages(nil))

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
		case ethchain.NewBlockEvent:
			app.container.NewBlock(ev.Block)

		case ethstate.Messages:
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
	self.filters[identifier] = ethchain.NewFilterFromMap(filterOptions, self.eth)
}

func (self *ExtApplication) GetMessages(object map[string]interface{}) string {
	filter := ethchain.NewFilterFromMap(object, self.eth)

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

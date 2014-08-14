package main

import (
	"encoding/json"

	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethreact"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/go-ethereum/javascript"
	"github.com/go-qml/qml"
)

type AppContainer interface {
	Create() error
	Destroy()

	Window() *qml.Window
	Engine() *qml.Engine

	NewBlock(*ethchain.Block)
	ObjectChanged(*ethstate.StateObject)
	StorageChanged(*ethstate.StorageState)
	NewWatcher(chan bool)
}

type ExtApplication struct {
	*ethpub.PEthereum
	eth ethchain.EthManager

	blockChan       chan ethreact.Event
	changeChan      chan ethreact.Event
	quitChan        chan bool
	watcherQuitChan chan bool

	container        AppContainer
	lib              *UiLib
	registeredEvents []string
}

func NewExtApplication(container AppContainer, lib *UiLib) *ExtApplication {
	app := &ExtApplication{
		ethpub.NewPEthereum(lib.eth),
		lib.eth,
		make(chan ethreact.Event, 100),
		make(chan ethreact.Event, 100),
		make(chan bool),
		make(chan bool),
		container,
		lib,
		nil,
	}

	return app
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

	// Call the main loop
	go app.mainLoop()

	// Subscribe to events
	reactor := app.lib.eth.Reactor()
	reactor.Subscribe("newBlock", app.blockChan)

	app.container.NewWatcher(app.watcherQuitChan)

	win := app.container.Window()
	win.Show()
	win.Wait()

	app.stop()
}

func (app *ExtApplication) stop() {
	// Clean up
	reactor := app.lib.eth.Reactor()
	reactor.Unsubscribe("newBlock", app.blockChan)
	for _, event := range app.registeredEvents {
		reactor.Unsubscribe(event, app.changeChan)
	}

	// Kill the main loop
	app.quitChan <- true
	app.watcherQuitChan <- true

	close(app.blockChan)
	close(app.quitChan)
	close(app.changeChan)

	app.container.Destroy()
}

func (app *ExtApplication) mainLoop() {
out:
	for {
		select {
		case <-app.quitChan:
			break out
		case block := <-app.blockChan:
			if block, ok := block.Resource.(*ethchain.Block); ok {
				app.container.NewBlock(block)
			}
		case object := <-app.changeChan:
			if stateObject, ok := object.Resource.(*ethstate.StateObject); ok {
				app.container.ObjectChanged(stateObject)
			} else if storageObject, ok := object.Resource.(*ethstate.StorageState); ok {
				app.container.StorageChanged(storageObject)
			}
		}
	}

}

func (app *ExtApplication) Watch(addr, storageAddr string) {
	var event string
	if len(storageAddr) == 0 {
		event = "object:" + string(ethutil.Hex2Bytes(addr))
		app.lib.eth.Reactor().Subscribe(event, app.changeChan)
	} else {
		event = "storage:" + string(ethutil.Hex2Bytes(addr)) + ":" + string(ethutil.Hex2Bytes(storageAddr))
		app.lib.eth.Reactor().Subscribe(event, app.changeChan)
	}

	app.registeredEvents = append(app.registeredEvents, event)
}

func (self *ExtApplication) GetMessages(object map[string]interface{}) string {
	filter := ethchain.NewFilter(self.eth)

	if object["earliest"] != nil {
		earliest := object["earliest"]
		if e, ok := earliest.(string); ok {
			filter.SetEarliestBlock(ethutil.Hex2Bytes(e))
		} else {
			filter.SetEarliestBlock(earliest)
		}
	}

	if object["latest"] != nil {
		latest := object["latest"]
		if l, ok := latest.(string); ok {
			filter.SetLatestBlock(ethutil.Hex2Bytes(l))
		} else {
			filter.SetLatestBlock(latest)
		}
	}
	if object["to"] != nil {
		filter.AddTo(ethutil.Hex2Bytes(object["to"].(string)))
	}
	if object["from"] != nil {
		filter.AddFrom(ethutil.Hex2Bytes(object["from"].(string)))
	}
	if object["max"] != nil {
		filter.SetMax(object["max"].(int))
	}
	if object["skip"] != nil {
		filter.SetSkip(object["skip"].(int))
	}

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

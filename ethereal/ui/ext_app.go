package ethui

import (
	"fmt"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/go-qml/qml"
)

type AppContainer interface {
	Create() error
	Destroy()

	Window() *qml.Window
	Engine() *qml.Engine

	NewBlock(*ethchain.Block)
	ObjectChanged(*ethchain.StateObject)
	StorageChanged(*ethchain.StorageState)
}

type ExtApplication struct {
	*ethpub.PEthereum

	blockChan  chan ethutil.React
	changeChan chan ethutil.React
	quitChan   chan bool

	container        AppContainer
	lib              *UiLib
	registeredEvents []string
}

func NewExtApplication(container AppContainer, lib *UiLib) *ExtApplication {
	app := &ExtApplication{
		ethpub.NewPEthereum(lib.eth.StateManager(), lib.eth.BlockChain(), lib.eth.TxPool()),
		make(chan ethutil.React, 1),
		make(chan ethutil.React, 1),
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
		fmt.Println(err)

		return
	}

	// Call the main loop
	go app.mainLoop()

	// Subscribe to events
	reactor := app.lib.eth.Reactor()
	reactor.Subscribe("newBlock", app.blockChan)

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
			if stateObject, ok := object.Resource.(*ethchain.StateObject); ok {
				app.container.ObjectChanged(stateObject)
			} else if storageObject, ok := object.Resource.(*ethchain.StorageState); ok {
				app.container.StorageChanged(storageObject)
			}
		}
	}

}

func (app *ExtApplication) Watch(addr, storageAddr string) {
	var event string
	if len(storageAddr) == 0 {
		event = "object:" + string(ethutil.FromHex(addr))
		app.lib.eth.Reactor().Subscribe(event, app.changeChan)
	} else {
		event = "storage:" + string(ethutil.FromHex(addr)) + ":" + string(ethutil.FromHex(storageAddr))
		app.lib.eth.Reactor().Subscribe(event, app.changeChan)
	}

	app.registeredEvents = append(app.registeredEvents, event)
}

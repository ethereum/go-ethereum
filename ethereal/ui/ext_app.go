package ethui

import (
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/go-ethereum/utils"
	"github.com/go-qml/qml"
	"math/big"
	"strings"
)

type AppContainer interface {
	Create() error
	Destroy()

	Window() *qml.Window
	Engine() *qml.Engine

	NewBlock(*ethchain.Block)
	ObjectChanged(*ethchain.StateObject)
	StorageChanged(*ethchain.StateObject, []byte, *big.Int)
}

type ExtApplication struct {
	*QEthereum

	blockChan  chan ethutil.React
	changeChan chan ethutil.React
	quitChan   chan bool

	container        AppContainer
	lib              *UiLib
	registeredEvents []string
}

func NewExtApplication(container AppContainer, lib *UiLib) *ExtApplication {
	app := &ExtApplication{
		NewQEthereum(lib.eth),
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
			} else if _, ok := object.Resource.(*big.Int); ok {
				//
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

type QEthereum struct {
	stateManager *ethchain.StateManager
	blockChain   *ethchain.BlockChain
	txPool       *ethchain.TxPool
}

func NewQEthereum(eth *eth.Ethereum) *QEthereum {
	return &QEthereum{
		eth.StateManager(),
		eth.BlockChain(),
		eth.TxPool(),
	}
}

func (lib *QEthereum) GetBlock(hexHash string) *QBlock {
	hash := ethutil.FromHex(hexHash)

	block := lib.blockChain.GetBlock(hash)

	return &QBlock{Number: int(block.BlockInfo().Number), Hash: ethutil.Hex(block.Hash())}
}

func (lib *QEthereum) GetKey() string {
	return ethutil.Hex(ethutil.Config.Db.GetKeys()[0].Address())
}

func (lib *QEthereum) GetStateObject(address string) *QStateObject {
	stateObject := lib.stateManager.ProcState().GetContract(ethutil.FromHex(address))
	if stateObject != nil {
		return NewQStateObject(stateObject)
	}

	// See GetStorage for explanation on "nil"
	return NewQStateObject(nil)
}

func (lib *QEthereum) Watch(addr, storageAddr string) {
	//	lib.stateManager.Watch(ethutil.FromHex(addr), ethutil.FromHex(storageAddr))
}

func (lib *QEthereum) CreateTx(key, recipient, valueStr, gasStr, gasPriceStr, dataStr string) (string, error) {
	return lib.Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr)
}

func (lib *QEthereum) Transact(key, recipient, valueStr, gasStr, gasPriceStr, dataStr string) (string, error) {
	var hash []byte
	var contractCreation bool
	if len(recipient) == 0 {
		contractCreation = true
	} else {
		hash = ethutil.FromHex(recipient)
	}

	keyPair := ethutil.Config.Db.GetKeys()[0]
	value := ethutil.Big(valueStr)
	gas := ethutil.Big(gasStr)
	gasPrice := ethutil.Big(gasPriceStr)
	var tx *ethchain.Transaction
	// Compile and assemble the given data
	if contractCreation {
		// Compile script
		mainScript, initScript, err := utils.CompileScript(dataStr)
		if err != nil {
			return "", err
		}

		tx = ethchain.NewContractCreationTx(value, gas, gasPrice, mainScript, initScript)
	} else {
		lines := strings.Split(dataStr, "\n")
		var data []byte
		for _, line := range lines {
			data = append(data, ethutil.BigToBytes(ethutil.Big(line), 256)...)
		}

		tx = ethchain.NewTransactionMessage(hash, value, gas, gasPrice, data)
	}
	acc := lib.stateManager.GetAddrState(keyPair.Address())
	tx.Nonce = acc.Nonce
	tx.Sign(keyPair.PrivateKey)
	lib.txPool.QueueTransaction(tx)

	if contractCreation {
		ethutil.Config.Log.Infof("Contract addr %x", tx.Hash()[12:])
	} else {
		ethutil.Config.Log.Infof("Tx hash %x", tx.Hash())
	}

	return ethutil.Hex(tx.Hash()), nil
}

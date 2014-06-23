package ethui

import (
	"bytes"
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/go-ethereum/utils"
	"github.com/go-qml/qml"
	"math/big"
	"strings"
	"time"
)

var logger = ethlog.NewLogger("GUI")

type Gui struct {
	// The main application window
	win *qml.Window
	// QML Engine
	engine    *qml.Engine
	component *qml.Common
	// The ethereum interface
	eth *eth.Ethereum

	// The public Ethereum library
	lib   *EthLib
	uiLib *UiLib

	txDb *ethdb.LDBDatabase

	addr []byte

	pub *ethpub.PEthereum
	logLevel ethlog.LogLevel
}

// Create GUI, but doesn't start it
func New(ethereum *eth.Ethereum, logLevel ethlog.LogLevel) *Gui {
	lib := &EthLib{stateManager: ethereum.StateManager(), blockChain: ethereum.BlockChain(), txPool: ethereum.TxPool()}
	db, err := ethdb.NewLDBDatabase("tx_database")
	if err != nil {
		panic(err)
	}

	// On first run we won't have any keys yet, so this would crash.
	// Therefor we check if we are ready to actually start this process
	var addr []byte
	if ethutil.GetKeyRing().Len() != 0 {
		addr = ethutil.GetKeyRing().Get(0).Address()
	}

	pub := ethpub.NewPEthereum(ethereum)

	return &Gui{eth: ethereum, lib: lib, txDb: db, addr: addr, pub: pub, logLevel: logLevel}
}

func (gui *Gui) Start(assetPath string) {
	const version = "0.5.0 RC14"

	defer gui.txDb.Close()

	// Register ethereum functions
	qml.RegisterTypes("Ethereum", 1, 0, []qml.TypeSpec{{
		Init: func(p *ethpub.PBlock, obj qml.Object) { p.Number = 0; p.Hash = "" },
	}, {
		Init: func(p *ethpub.PTx, obj qml.Object) { p.Value = ""; p.Hash = ""; p.Address = "" },
	}, {
		Init: func(p *ethpub.KeyVal, obj qml.Object) { p.Key = ""; p.Value = "" },
	}})

	ethutil.Config.SetClientString("Ethereal")

	// Create a new QML engine
	gui.engine = qml.NewEngine()
	context := gui.engine.Context()

	// Expose the eth library and the ui library to QML
	context.SetVar("eth", gui)
	context.SetVar("pub", gui.pub)
	gui.uiLib = NewUiLib(gui.engine, gui.eth, assetPath)
	context.SetVar("ui", gui.uiLib)

	// Load the main QML interface
	data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))

	var win *qml.Window
	var err error
	if len(data) == 0 {
		win, err = gui.showKeyImport(context)
	} else {
		win, err = gui.showWallet(context)
		ethlog.AddLogSystem(gui)
	}
	if err != nil {
		logger.Errorln("asset not found: you can set an alternative asset path on the command line using option 'asset_path'", err)

		panic(err)
	}

	logger.Infoln("Starting GUI")

	win.Show()
	win.Wait()

	gui.eth.Stop()
}

func (gui *Gui) ToggleMining() {
	var txt string
	if gui.eth.Mining {
		utils.StopMining(gui.eth)
		txt = "Start mining"
	} else {
		utils.StartMining(gui.eth)
		txt = "Stop mining"
	}

	gui.win.Root().Set("miningButtonText", txt)
}

func (gui *Gui) showWallet(context *qml.Context) (*qml.Window, error) {
	component, err := gui.engine.LoadFile(gui.uiLib.AssetPath("qml/wallet.qml"))
	if err != nil {
		return nil, err
	}

	win := gui.createWindow(component)

	gui.setInitialBlockChain()
	gui.loadAddressBook()
	gui.readPreviousTransactions()
	gui.setPeerInfo()

	go gui.update()

	return win, nil
}

func (gui *Gui) showKeyImport(context *qml.Context) (*qml.Window, error) {
	context.SetVar("lib", gui.lib)
	component, err := gui.engine.LoadFile(gui.uiLib.AssetPath("qml/first_run.qml"))
	if err != nil {
		return nil, err
	}

	return gui.createWindow(component), nil
}

func (gui *Gui) createWindow(comp qml.Object) *qml.Window {
	win := comp.CreateWindow(nil)

	gui.win = win
	gui.uiLib.win = win

	db := &Debugger{gui.win, make(chan bool), make(chan bool), true, false}
	gui.lib.Db = db
	gui.uiLib.Db = db

	return gui.win
}
func (gui *Gui) setInitialBlockChain() {
	sBlk := gui.eth.BlockChain().LastBlockHash
	blk := gui.eth.BlockChain().GetBlock(sBlk)
	for ; blk != nil; blk = gui.eth.BlockChain().GetBlock(sBlk) {
		sBlk = blk.PrevHash

		// Loop through all transactions to see if we missed any while being offline
		for _, tx := range blk.Transactions() {
			if bytes.Compare(tx.Sender(), gui.addr) == 0 || bytes.Compare(tx.Recipient, gui.addr) == 0 {
				if ok, _ := gui.txDb.Get(tx.Hash()); ok == nil {
					gui.txDb.Put(tx.Hash(), tx.RlpEncode())
				}

			}
		}

		gui.processBlock(blk, true)
	}
}

type address struct {
	Name, Address string
}

var namereg = ethutil.FromHex("bb5f186604d057c1c5240ca2ae0f6430138ac010")

func (gui *Gui) loadAddressBook() {
	gui.win.Root().Call("clearAddress")
	stateObject := gui.eth.StateManager().CurrentState().GetStateObject(namereg)
	if stateObject != nil {
		stateObject.State().EachStorage(func(name string, value *ethutil.Value) {
			gui.win.Root().Call("addAddress", struct{ Name, Address string }{name, ethutil.Hex(value.Bytes())})
		})
	}
}

func (gui *Gui) readPreviousTransactions() {
	it := gui.txDb.Db().NewIterator(nil, nil)
	for it.Next() {
		tx := ethchain.NewTransactionFromBytes(it.Value())

		var inout string
		if bytes.Compare(tx.Sender(), gui.addr) == 0 {
			inout = "send"
		} else {
			inout = "recv"
		}

		gui.win.Root().Call("addTx", ethpub.NewPTx(tx), inout)

	}
	it.Release()
}

func (gui *Gui) processBlock(block *ethchain.Block, initial bool) {
	gui.win.Root().Call("addBlock", ethpub.NewPBlock(block), initial)
}

func (gui *Gui) setWalletValue(amount, unconfirmedFunds *big.Int) {
	var str string
	if unconfirmedFunds != nil {
		pos := "+"
		if unconfirmedFunds.Cmp(big.NewInt(0)) < 0 {
			pos = "-"
		}
		val := ethutil.CurrencyToString(new(big.Int).Abs(ethutil.BigCopy(unconfirmedFunds)))
		str = fmt.Sprintf("%v (%s %v)", ethutil.CurrencyToString(amount), pos, val)
	} else {
		str = fmt.Sprintf("%v", ethutil.CurrencyToString(amount))
	}

	gui.win.Root().Call("setWalletValue", str)
}

// Simple go routine function that updates the list of peers in the GUI
func (gui *Gui) update() {
	reactor := gui.eth.Reactor()

	blockChan := make(chan ethutil.React, 1)
	txChan := make(chan ethutil.React, 1)
	objectChan := make(chan ethutil.React, 1)
	peerChan := make(chan ethutil.React, 1)

	reactor.Subscribe("newBlock", blockChan)
	reactor.Subscribe("newTx:pre", txChan)
	reactor.Subscribe("newTx:post", txChan)
	reactor.Subscribe("object:"+string(namereg), objectChan)
	reactor.Subscribe("peerList", peerChan)

	ticker := time.NewTicker(5 * time.Second)

	state := gui.eth.StateManager().TransState()

	unconfirmedFunds := new(big.Int)
	gui.win.Root().Call("setWalletValue", fmt.Sprintf("%v", ethutil.CurrencyToString(state.GetAccount(gui.addr).Amount)))

	for {
		select {
		case b := <-blockChan:
			block := b.Resource.(*ethchain.Block)
			gui.processBlock(block, false)
			if bytes.Compare(block.Coinbase, gui.addr) == 0 {
				gui.setWalletValue(gui.eth.StateManager().CurrentState().GetAccount(gui.addr).Amount, nil)
			}

		case txMsg := <-txChan:
			tx := txMsg.Resource.(*ethchain.Transaction)

			if txMsg.Event == "newTx:pre" {
				object := state.GetAccount(gui.addr)

				if bytes.Compare(tx.Sender(), gui.addr) == 0 {
					gui.win.Root().Call("addTx", ethpub.NewPTx(tx), "send")
					gui.txDb.Put(tx.Hash(), tx.RlpEncode())

					unconfirmedFunds.Sub(unconfirmedFunds, tx.Value)
				} else if bytes.Compare(tx.Recipient, gui.addr) == 0 {
					gui.win.Root().Call("addTx", ethpub.NewPTx(tx), "recv")
					gui.txDb.Put(tx.Hash(), tx.RlpEncode())

					unconfirmedFunds.Add(unconfirmedFunds, tx.Value)
				}

				gui.setWalletValue(object.Amount, unconfirmedFunds)
			} else {
				object := state.GetAccount(gui.addr)
				if bytes.Compare(tx.Sender(), gui.addr) == 0 {
					object.SubAmount(tx.Value)
				} else if bytes.Compare(tx.Recipient, gui.addr) == 0 {
					object.AddAmount(tx.Value)
				}

				gui.setWalletValue(object.Amount, nil)

				state.UpdateStateObject(object)
			}
		case <-objectChan:
			gui.loadAddressBook()
		case <-peerChan:
			gui.setPeerInfo()
		case <-ticker.C:
			gui.setPeerInfo()
		}
	}
}

func (gui *Gui) setPeerInfo() {
	gui.win.Root().Call("setPeers", fmt.Sprintf("%d / %d", gui.eth.PeerCount(), gui.eth.MaxPeers))

	gui.win.Root().Call("resetPeers")
	for _, peer := range gui.pub.GetPeers() {
		gui.win.Root().Call("addPeer", peer)
	}
}

func (gui *Gui) RegisterName(name string) {
	keyPair := ethutil.GetKeyRing().Get(0)
	name = fmt.Sprintf("\"%s\"\n1", name)
	gui.pub.Transact(ethutil.Hex(keyPair.PrivateKey), "namereg", "1000", "1000000", "150", name)
}

func (gui *Gui) Transact(recipient, value, gas, gasPrice, data string) (*ethpub.PReceipt, error) {
	keyPair := ethutil.GetKeyRing().Get(0)

	return gui.pub.Transact(ethutil.Hex(keyPair.PrivateKey), recipient, value, gas, gasPrice, data)
}

func (gui *Gui) Create(recipient, value, gas, gasPrice, data string) (*ethpub.PReceipt, error) {
	keyPair := ethutil.GetKeyRing().Get(0)

	return gui.pub.Transact(ethutil.Hex(keyPair.PrivateKey), recipient, value, gas, gasPrice, data)
}

func (gui *Gui) ChangeClientId(id string) {
	ethutil.Config.SetIdentifier(id)
}

func (gui *Gui) ClientId() string {
	return ethutil.Config.Identifier
}

// functions that allow Gui to implement interface ethlog.LogSystem
func (gui *Gui) SetLogLevel(level ethlog.LogLevel) {
	gui.logLevel = level
}

func (gui *Gui) GetLogLevel() ethlog.LogLevel {
	return gui.logLevel
}

func (gui *Gui) Println(v ...interface{}) {
	gui.printLog(fmt.Sprintln(v...))
}

func (gui *Gui) Printf(format string, v ...interface{}) {
	gui.printLog(fmt.Sprintf(format, v...))
}

// Print function that logs directly to the GUI
func (gui *Gui) printLog(s string) {
	str := strings.TrimRight(s, "\n")
	lines := strings.Split(str, "\n")
	for _, line := range lines {
		gui.win.Root().Call("addLog", line)
	}
}

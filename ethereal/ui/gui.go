package ethui

import (
	"bytes"
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/go-qml/qml"
	"github.com/obscuren/mutan"
	"math/big"
	"strings"
)

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
}

// Create GUI, but doesn't start it
func New(ethereum *eth.Ethereum) *Gui {
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

	return &Gui{eth: ethereum, lib: lib, txDb: db, addr: addr, pub: pub}
}

func (gui *Gui) Start(assetPath string) {
	defer gui.txDb.Close()

	// Register ethereum functions
	qml.RegisterTypes("Ethereum", 1, 0, []qml.TypeSpec{{
		Init: func(p *ethpub.PBlock, obj qml.Object) { p.Number = 0; p.Hash = "" },
	}, {
		Init: func(p *ethpub.PTx, obj qml.Object) { p.Value = ""; p.Hash = ""; p.Address = "" },
	}})

	ethutil.Config.SetClientString(fmt.Sprintf("/Ethereal v%s", "0.5.0 RC4"))
	ethutil.Config.Log.Infoln("[GUI] Starting GUI")
	// Create a new QML engine
	gui.engine = qml.NewEngine()
	context := gui.engine.Context()

	// Expose the eth library and the ui library to QML
	context.SetVar("eth", gui)
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
	}
	if err != nil {
		ethutil.Config.Log.Infoln("FATAL: asset not found: you can set an alternative asset path on on the command line using option 'asset_path'")

		panic(err)
	}

	win.Show()
	win.Wait()

	gui.eth.Stop()
}

func (gui *Gui) showWallet(context *qml.Context) (*qml.Window, error) {
	component, err := gui.engine.LoadFile(gui.uiLib.AssetPath("qml/wallet.qml"))
	if err != nil {
		return nil, err
	}

	win := gui.createWindow(component)

	go gui.setInitialBlockChain()
	go gui.readPreviousTransactions()
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

	db := &Debugger{gui.win, make(chan bool)}
	gui.lib.Db = db
	gui.uiLib.Db = db

	return gui.win
}

func (gui *Gui) setInitialBlockChain() {
	// Load previous 10 blocks
	chain := gui.eth.BlockChain().GetChain(gui.eth.BlockChain().CurrentBlock.Hash(), 10)
	for _, block := range chain {
		gui.processBlock(block)
	}

}

func (gui *Gui) readPreviousTransactions() {
	it := gui.txDb.Db().NewIterator(nil, nil)
	for it.Next() {
		tx := ethchain.NewTransactionFromBytes(it.Value())

		gui.win.Root().Call("addTx", ethpub.NewPTx(tx))
	}
	it.Release()
}

func (gui *Gui) processBlock(block *ethchain.Block) {
	gui.win.Root().Call("addBlock", ethpub.NewPBlock(block))
}

// Simple go routine function that updates the list of peers in the GUI
func (gui *Gui) update() {
	txChan := make(chan ethchain.TxMsg, 1)
	gui.eth.TxPool().Subscribe(txChan)

	state := gui.eth.StateManager().TransState()

	unconfirmedFunds := new(big.Int)
	gui.win.Root().Call("setWalletValue", fmt.Sprintf("%v", ethutil.CurrencyToString(state.GetAccount(gui.addr).Amount)))

	for {
		select {
		case txMsg := <-txChan:
			tx := txMsg.Tx

			if txMsg.Type == ethchain.TxPre {
				object := state.GetAccount(gui.addr)

				if bytes.Compare(tx.Sender(), gui.addr) == 0 && object.Nonce <= tx.Nonce {
					gui.win.Root().Call("addTx", ethpub.NewPTx(tx))
					gui.txDb.Put(tx.Hash(), tx.RlpEncode())

					object.Nonce += 1
					state.SetStateObject(object)

					unconfirmedFunds.Sub(unconfirmedFunds, tx.Value)
				} else if bytes.Compare(tx.Recipient, gui.addr) == 0 {
					gui.win.Root().Call("addTx", ethpub.NewPTx(tx))
					gui.txDb.Put(tx.Hash(), tx.RlpEncode())

					unconfirmedFunds.Add(unconfirmedFunds, tx.Value)
				}

				pos := "+"
				if unconfirmedFunds.Cmp(big.NewInt(0)) >= 0 {
					pos = "-"
				}
				val := ethutil.CurrencyToString(new(big.Int).Abs(ethutil.BigCopy(unconfirmedFunds)))
				str := fmt.Sprintf("%v (%s %v)", ethutil.CurrencyToString(object.Amount), pos, val)

				gui.win.Root().Call("setWalletValue", str)
			} else {
				object := state.GetAccount(gui.addr)
				if bytes.Compare(tx.Sender(), gui.addr) == 0 {
					object.SubAmount(tx.Value)
				} else if bytes.Compare(tx.Recipient, gui.addr) == 0 {
					object.AddAmount(tx.Value)
				}

				gui.win.Root().Call("setWalletValue", fmt.Sprintf("%v", ethutil.CurrencyToString(object.Amount)))

				state.SetStateObject(object)
			}
		}
	}
}

// Logging functions that log directly to the GUI interface
func (gui *Gui) Println(v ...interface{}) {
	str := strings.TrimRight(fmt.Sprintln(v...), "\n")
	lines := strings.Split(str, "\n")
	for _, line := range lines {
		gui.win.Root().Call("addLog", line)
	}
}

func (gui *Gui) Printf(format string, v ...interface{}) {
	str := strings.TrimRight(fmt.Sprintf(format, v...), "\n")
	lines := strings.Split(str, "\n")
	for _, line := range lines {
		gui.win.Root().Call("addLog", line)
	}
}

func (gui *Gui) Transact(recipient, value, gas, gasPrice, data string) (*ethpub.PReceipt, error) {
	keyPair := ethutil.GetKeyRing().Get(0)

	return gui.pub.Transact(ethutil.Hex(keyPair.PrivateKey), recipient, value, gas, gasPrice, data)
}

func (gui *Gui) Create(recipient, value, gas, gasPrice, data string) (*ethpub.PReceipt, error) {
	keyPair := ethutil.GetKeyRing().Get(0)

	mainInput, initInput := mutan.PreParse(data)

	return gui.pub.Create(ethutil.Hex(keyPair.PrivateKey), value, gas, gasPrice, initInput, mainInput)
}

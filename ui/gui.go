package ethui

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/niemeyer/qml"
	"strings"
)

// Block interface exposed to QML
type Block struct {
	Number int
	Hash   string
}

type Tx struct {
	Value, Hash, Address string
}

func NewTxFromTransaction(tx *ethchain.Transaction) *Tx {
	hash := hex.EncodeToString(tx.Hash())
	sender := hex.EncodeToString(tx.Recipient)

	return &Tx{Hash: hash, Value: ethutil.CurrencyToString(tx.Value), Address: sender}
}

// Creates a new QML Block from a chain block
func NewBlockFromBlock(block *ethchain.Block) *Block {
	info := block.BlockInfo()
	hash := hex.EncodeToString(block.Hash())

	return &Block{Number: int(info.Number), Hash: hash}
}

type Gui struct {
	// The main application window
	win *qml.Window
	// QML Engine
	engine    *qml.Engine
	component *qml.Common
	// The ethereum interface
	eth *eth.Ethereum

	// The public Ethereum library
	lib *EthLib

	txDb *ethdb.LDBDatabase

	addr []byte
}

// Create GUI, but doesn't start it
func New(ethereum *eth.Ethereum) *Gui {
	lib := &EthLib{blockManager: ethereum.BlockManager, blockChain: ethereum.BlockManager.BlockChain(), txPool: ethereum.TxPool}
	db, err := ethdb.NewLDBDatabase("tx_database")
	if err != nil {
		panic(err)
	}

	data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
	keyRing := ethutil.NewValueFromBytes(data)
	addr := keyRing.Get(1).Bytes()

	ethereum.BlockManager.WatchAddr(addr)

	return &Gui{eth: ethereum, lib: lib, txDb: db, addr: addr}
}

func (ui *Gui) Start() {
	defer ui.txDb.Close()

	// Register ethereum functions
	qml.RegisterTypes("Ethereum", 1, 0, []qml.TypeSpec{{
		Init: func(p *Block, obj qml.Object) { p.Number = 0; p.Hash = "" },
	}, {
		Init: func(p *Tx, obj qml.Object) { p.Value = ""; p.Hash = ""; p.Address = "" },
	}})

	ethutil.Config.Log.Infoln("[GUI] Starting GUI")
	// Create a new QML engine
	ui.engine = qml.NewEngine()
	// Load the main QML interface
	component, err := ui.engine.LoadFile("wallet.qml")
	if err != nil {
		panic(err)
	}
	ui.engine.LoadFile("transactions.qml")

	ui.win = component.CreateWindow(nil)

	context := ui.engine.Context()

	// Expose the eth library and the ui library to QML
	context.SetVar("eth", ui.lib)
	context.SetVar("ui", &UiLib{engine: ui.engine, eth: ui.eth})

	// Register the ui as a block processor
	ui.eth.BlockManager.SecondaryBlockProcessor = ui
	//ui.eth.TxPool.SecondaryProcessor = ui

	// Add the ui as a log system so we can log directly to the UGI
	ethutil.Config.Log.AddLogSystem(ui)

	// Loads previous blocks
	go ui.setInitialBlockChain()
	go ui.readPreviousTransactions()
	go ui.update()

	ui.win.Show()
	ui.win.Wait()

	ui.eth.Stop()
}

func (ui *Gui) setInitialBlockChain() {
	// Load previous 10 blocks
	chain := ui.eth.BlockManager.BlockChain().GetChain(ui.eth.BlockManager.BlockChain().CurrentBlock.Hash(), 10)
	for _, block := range chain {
		ui.ProcessBlock(block)
	}

}

func (ui *Gui) readPreviousTransactions() {
	it := ui.txDb.Db().NewIterator(nil)
	for it.Next() {
		tx := ethchain.NewTransactionFromBytes(it.Value())

		ui.win.Root().Call("addTx", NewTxFromTransaction(tx))
	}
	it.Release()
}

func (ui *Gui) ProcessBlock(block *ethchain.Block) {
	ui.win.Root().Call("addBlock", NewBlockFromBlock(block))
}

func (ui *Gui) ProcessTransaction(tx *ethchain.Transaction) {
	ui.txDb.Put(tx.Hash(), tx.RlpEncode())

	ui.win.Root().Call("addTx", NewTxFromTransaction(tx))

	// TODO replace with general subscribe model
}

// Simple go routine function that updates the list of peers in the GUI
func (ui *Gui) update() {
	txChan := make(chan ethchain.TxMsg)
	ui.eth.TxPool.Subscribe(txChan)

	account := ui.eth.BlockManager.GetAddrState(ui.addr).Account
	ui.win.Root().Call("setWalletValue", fmt.Sprintf("%v", ethutil.CurrencyToString(account.Amount)))
	for {
		select {
		case txMsg := <-txChan:
			tx := txMsg.Tx
			ui.txDb.Put(tx.Hash(), tx.RlpEncode())

			ui.win.Root().Call("addTx", NewTxFromTransaction(tx))
			// TODO FOR THE LOVE OF EVERYTHING GOOD IN THIS WORLD REFACTOR ME
			if txMsg.Type == ethchain.TxPre {
				if bytes.Compare(tx.Sender(), ui.addr) == 0 {
					ui.win.Root().Call("setWalletValue", fmt.Sprintf("%v (- %v)", ethutil.CurrencyToString(account.Amount), ethutil.CurrencyToString(tx.Value)))
					ui.eth.BlockManager.GetAddrState(ui.addr).Nonce += 1
					fmt.Println("Nonce", ui.eth.BlockManager.GetAddrState(ui.addr).Nonce)
				} else if bytes.Compare(tx.Recipient, ui.addr) == 0 {
					ui.win.Root().Call("setWalletValue", fmt.Sprintf("%v (+ %v)", ethutil.CurrencyToString(account.Amount), ethutil.CurrencyToString(tx.Value)))
				}
			} else {
				if bytes.Compare(tx.Sender(), ui.addr) == 0 {
					amount := account.Amount.Sub(account.Amount, tx.Value)
					ui.win.Root().Call("setWalletValue", fmt.Sprintf("%v", ethutil.CurrencyToString(amount)))
				} else if bytes.Compare(tx.Recipient, ui.addr) == 0 {
					amount := account.Amount.Sub(account.Amount, tx.Value)
					ui.win.Root().Call("setWalletValue", fmt.Sprintf("%v", ethutil.CurrencyToString(amount)))
				}
			}
		}

		/*
			accountAmount := ui.eth.BlockManager.GetAddrState(ui.addr).Account.Amount
			ui.win.Root().Call("setWalletValue", fmt.Sprintf("%v", accountAmount))

			ui.win.Root().Call("setPeers", fmt.Sprintf("%d / %d", ui.eth.Peers().Len(), ui.eth.MaxPeers))

			time.Sleep(1 * time.Second)
		*/

	}
}

// Logging functions that log directly to the GUI interface
func (ui *Gui) Println(v ...interface{}) {
	str := strings.TrimRight(fmt.Sprintln(v...), "\n")
	lines := strings.Split(str, "\n")
	for _, line := range lines {
		ui.win.Root().Call("addLog", line)
	}
}

func (ui *Gui) Printf(format string, v ...interface{}) {
	str := strings.TrimRight(fmt.Sprintf(format, v...), "\n")
	lines := strings.Split(str, "\n")
	for _, line := range lines {
		ui.win.Root().Call("addLog", line)
	}
}

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
	"math/big"
	"strings"
)

// Block interface exposed to QML
type Block struct {
	Number int
	Hash   string
}

type Tx struct {
	Value, Hash, Address string
	Contract             bool
}

func NewTxFromTransaction(tx *ethchain.Transaction) *Tx {
	hash := hex.EncodeToString(tx.Hash())
	sender := hex.EncodeToString(tx.Recipient)
	isContract := len(tx.Data) > 0

	return &Tx{Hash: hash, Value: ethutil.CurrencyToString(tx.Value), Address: sender, Contract: isContract}
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
	lib := &EthLib{stateManager: ethereum.StateManager(), blockChain: ethereum.BlockChain(), txPool: ethereum.TxPool()}
	db, err := ethdb.NewLDBDatabase("tx_database")
	if err != nil {
		panic(err)
	}

	data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
	// On first run we won't have any keys yet, so this would crash.
	// Therefor we check if we are ready to actually start this process
	var addr []byte
	if len(data) > 0 {
		key := ethutil.Config.Db.GetKeys()[0]
		addr = key.Address()

		ethereum.StateManager().WatchAddr(addr)
	}

	return &Gui{eth: ethereum, lib: lib, txDb: db, addr: addr}
}

func (ui *Gui) Start(assetPath string) {
	defer ui.txDb.Close()

	// Register ethereum functions
	qml.RegisterTypes("Ethereum", 1, 0, []qml.TypeSpec{{
		Init: func(p *Block, obj qml.Object) { p.Number = 0; p.Hash = "" },
	}, {
		Init: func(p *Tx, obj qml.Object) { p.Value = ""; p.Hash = ""; p.Address = "" },
	}})

	ethutil.Config.SetClientString(fmt.Sprintf("/Ethereal v%s", "0.1"))
	ethutil.Config.Log.Infoln("[GUI] Starting GUI")
	// Create a new QML engine
	ui.engine = qml.NewEngine()
	context := ui.engine.Context()

	// Expose the eth library and the ui library to QML
	context.SetVar("eth", ui.lib)
	uiLib := NewUiLib(ui.engine, ui.eth, assetPath)
	context.SetVar("ui", uiLib)

	// Load the main QML interface
	data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
	var err error
	var component qml.Object
	firstRun := len(data) == 0

	if firstRun {
		component, err = ui.engine.LoadFile(uiLib.AssetPath("qml/first_run.qml"))
	} else {
		component, err = ui.engine.LoadFile(uiLib.AssetPath("qml/wallet.qml"))
	}
	if err != nil {
		ethutil.Config.Log.Infoln("FATAL: asset not found: you can set an alternative asset path on on the command line using option 'asset_path'")
		panic(err)
	}
	ui.engine.LoadFile(uiLib.AssetPath("qml/transactions.qml"))

	ui.win = component.CreateWindow(nil)
	uiLib.win = ui.win

	// Register the ui as a block processor
	//ui.eth.BlockManager.SecondaryBlockProcessor = ui
	//ui.eth.TxPool.SecondaryProcessor = ui

	// Add the ui as a log system so we can log directly to the UGI
	ethutil.Config.Log.AddLogSystem(ui)

	// Loads previous blocks
	if firstRun == false {
		go ui.setInitialBlockChain()
		go ui.readPreviousTransactions()
		go ui.update()
	}

	ui.win.Show()
	ui.win.Wait()

	ui.eth.Stop()
}

func (ui *Gui) setInitialBlockChain() {
	// Load previous 10 blocks
	chain := ui.eth.BlockChain().GetChain(ui.eth.BlockChain().CurrentBlock.Hash(), 10)
	for _, block := range chain {
		ui.ProcessBlock(block)
	}

}

func (ui *Gui) readPreviousTransactions() {
	it := ui.txDb.Db().NewIterator(nil, nil)
	for it.Next() {
		tx := ethchain.NewTransactionFromBytes(it.Value())

		ui.win.Root().Call("addTx", NewTxFromTransaction(tx))
	}
	it.Release()
}

func (ui *Gui) ProcessBlock(block *ethchain.Block) {
	ui.win.Root().Call("addBlock", NewBlockFromBlock(block))
}

// Simple go routine function that updates the list of peers in the GUI
func (ui *Gui) update() {
	txChan := make(chan ethchain.TxMsg, 1)
	ui.eth.TxPool().Subscribe(txChan)

	account := ui.eth.StateManager().GetAddrState(ui.addr).Account
	unconfirmedFunds := new(big.Int)
	ui.win.Root().Call("setWalletValue", fmt.Sprintf("%v", ethutil.CurrencyToString(account.Amount)))
	for {
		select {
		case txMsg := <-txChan:
			tx := txMsg.Tx

			if txMsg.Type == ethchain.TxPre {
				if bytes.Compare(tx.Sender(), ui.addr) == 0 {
					ui.win.Root().Call("addTx", NewTxFromTransaction(tx))
					ui.txDb.Put(tx.Hash(), tx.RlpEncode())

					ui.eth.StateManager().GetAddrState(ui.addr).Nonce += 1
					unconfirmedFunds.Sub(unconfirmedFunds, tx.Value)
				} else if bytes.Compare(tx.Recipient, ui.addr) == 0 {
					ui.win.Root().Call("addTx", NewTxFromTransaction(tx))
					ui.txDb.Put(tx.Hash(), tx.RlpEncode())

					unconfirmedFunds.Add(unconfirmedFunds, tx.Value)
				}

				pos := "+"
				if unconfirmedFunds.Cmp(big.NewInt(0)) >= 0 {
					pos = "-"
				}
				val := ethutil.CurrencyToString(new(big.Int).Abs(ethutil.BigCopy(unconfirmedFunds)))
				str := fmt.Sprintf("%v (%s %v)", ethutil.CurrencyToString(account.Amount), pos, val)

				ui.win.Root().Call("setWalletValue", str)
			} else {
				amount := account.Amount
				if bytes.Compare(tx.Sender(), ui.addr) == 0 {
					amount.Sub(account.Amount, tx.Value)
				} else if bytes.Compare(tx.Recipient, ui.addr) == 0 {
					amount.Add(account.Amount, tx.Value)
				}

				ui.win.Root().Call("setWalletValue", fmt.Sprintf("%v", ethutil.CurrencyToString(amount)))
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

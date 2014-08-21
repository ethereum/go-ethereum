package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethminer"
	"github.com/ethereum/eth-go/ethpipe"
	"github.com/ethereum/eth-go/ethreact"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
	"github.com/ethereum/go-ethereum/utils"
	"gopkg.in/qml.v1"
)

var logger = ethlog.NewLogger("GUI")

type plugin struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Gui struct {
	// The main application window
	win *qml.Window
	// QML Engine
	engine    *qml.Engine
	component *qml.Common
	qmlDone   bool
	// The ethereum interface
	eth *eth.Ethereum

	// The public Ethereum library
	uiLib *UiLib

	txDb *ethdb.LDBDatabase

	logLevel ethlog.LogLevel
	open     bool

	pipe *ethpipe.JSPipe

	Session        string
	clientIdentity *ethwire.SimpleClientIdentity
	config         *ethutil.ConfigManager

	plugins map[string]plugin

	miner *ethminer.Miner
}

// Create GUI, but doesn't start it
func NewWindow(ethereum *eth.Ethereum, config *ethutil.ConfigManager, clientIdentity *ethwire.SimpleClientIdentity, session string, logLevel int) *Gui {
	db, err := ethdb.NewLDBDatabase("tx_database")
	if err != nil {
		panic(err)
	}

	pipe := ethpipe.NewJSPipe(ethereum)
	gui := &Gui{eth: ethereum, txDb: db, pipe: pipe, logLevel: ethlog.LogLevel(logLevel), Session: session, open: false, clientIdentity: clientIdentity, config: config, plugins: make(map[string]plugin)}
	data, err := ethutil.ReadAllFile(ethutil.Config.ExecPath + "/plugins.json")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("plugins:", string(data))

	json.Unmarshal([]byte(data), &gui.plugins)

	return gui
}

func (gui *Gui) Start(assetPath string) {

	defer gui.txDb.Close()

	// Register ethereum functions
	qml.RegisterTypes("Ethereum", 1, 0, []qml.TypeSpec{{
		Init: func(p *ethpipe.JSBlock, obj qml.Object) { p.Number = 0; p.Hash = "" },
	}, {
		Init: func(p *ethpipe.JSTransaction, obj qml.Object) { p.Value = ""; p.Hash = ""; p.Address = "" },
	}, {
		Init: func(p *ethpipe.KeyVal, obj qml.Object) { p.Key = ""; p.Value = "" },
	}})
	// Create a new QML engine
	gui.engine = qml.NewEngine()
	context := gui.engine.Context()
	gui.uiLib = NewUiLib(gui.engine, gui.eth, assetPath)

	// Expose the eth library and the ui library to QML
	context.SetVar("gui", gui)
	context.SetVar("eth", gui.uiLib)

	// Load the main QML interface
	data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))

	var win *qml.Window
	var err error
	var addlog = false
	if len(data) == 0 {
		win, err = gui.showKeyImport(context)
	} else {
		win, err = gui.showWallet(context)
		addlog = true
	}
	if err != nil {
		logger.Errorln("asset not found: you can set an alternative asset path on the command line using option 'asset_path'", err)

		panic(err)
	}

	logger.Infoln("Starting GUI")
	gui.open = true
	win.Show()

	// only add the gui logger after window is shown otherwise slider wont be shown
	if addlog {
		ethlog.AddLogSystem(gui)
	}
	win.Wait()

	// need to silence gui logger after window closed otherwise logsystem hangs (but do not save loglevel)
	gui.logLevel = ethlog.Silence
	gui.open = false
}

func (gui *Gui) Stop() {
	if gui.open {
		gui.logLevel = ethlog.Silence
		gui.open = false
		gui.win.Hide()
	}

	gui.uiLib.jsEngine.Stop()

	logger.Infoln("Stopped")
}

func (gui *Gui) ToggleMining() {
	var txt string
	if gui.eth.Mining {
		utils.StopMining(gui.eth)
		txt = "Start mining"

		gui.getObjectByName("miningLabel").Set("visible", false)
	} else {
		utils.StartMining(gui.eth)
		gui.miner = utils.GetMiner()
		txt = "Stop mining"

		gui.getObjectByName("miningLabel").Set("visible", true)
	}

	gui.win.Root().Set("miningButtonText", txt)
}

func (gui *Gui) showWallet(context *qml.Context) (*qml.Window, error) {
	component, err := gui.engine.LoadFile(gui.uiLib.AssetPath("qml/wallet.qml"))
	if err != nil {
		return nil, err
	}

	gui.win = gui.createWindow(component)

	gui.update()

	return gui.win, nil
}

func (self *Gui) DumpState(hash, path string) {
	var stateDump []byte

	if len(hash) == 0 {
		stateDump = self.eth.StateManager().CurrentState().Dump()
	} else {
		var block *ethchain.Block
		if hash[0] == '#' {
			i, _ := strconv.Atoi(hash[1:])
			block = self.eth.BlockChain().GetBlockByNumber(uint64(i))
		} else {
			block = self.eth.BlockChain().GetBlock(ethutil.Hex2Bytes(hash))
		}

		if block == nil {
			logger.Infof("block err: not found %s\n", hash)
			return
		}

		stateDump = block.State().Dump()
	}

	file, err := os.OpenFile(path[7:], os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		logger.Infoln("dump err: ", err)
		return
	}
	defer file.Close()

	logger.Infof("dumped state (%s) to %s\n", hash, path)

	file.Write(stateDump)
}

// The done handler will be called by QML when all views have been loaded
func (gui *Gui) Done() {
	gui.qmlDone = true

}

func (gui *Gui) ImportKey(filePath string) {
}

func (gui *Gui) showKeyImport(context *qml.Context) (*qml.Window, error) {
	context.SetVar("lib", gui)
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

	return gui.win
}

func (gui *Gui) ImportAndSetPrivKey(secret string) bool {
	err := gui.eth.KeyManager().InitFromString(gui.Session, 0, secret)
	if err != nil {
		logger.Errorln("unable to import: ", err)
		return false
	}
	logger.Errorln("successfully imported: ", err)
	return true
}

func (gui *Gui) CreateAndSetPrivKey() (string, string, string, string) {
	err := gui.eth.KeyManager().Init(gui.Session, 0, true)
	if err != nil {
		logger.Errorln("unable to create key: ", err)
		return "", "", "", ""
	}
	return gui.eth.KeyManager().KeyPair().AsStrings()
}

func (gui *Gui) setInitialBlockChain() {
	sBlk := gui.eth.BlockChain().LastBlockHash
	blk := gui.eth.BlockChain().GetBlock(sBlk)
	for ; blk != nil; blk = gui.eth.BlockChain().GetBlock(sBlk) {
		sBlk = blk.PrevHash
		addr := gui.address()

		// Loop through all transactions to see if we missed any while being offline
		for _, tx := range blk.Transactions() {
			if bytes.Compare(tx.Sender(), addr) == 0 || bytes.Compare(tx.Recipient, addr) == 0 {
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

func (gui *Gui) loadAddressBook() {
	view := gui.getObjectByName("infoView")
	view.Call("clearAddress")

	nameReg := gui.pipe.World().Config().Get("NameReg")
	if nameReg != nil {
		nameReg.EachStorage(func(name string, value *ethutil.Value) {
			if name[0] != 0 {
				value.Decode()

				view.Call("addAddress", struct{ Name, Address string }{name, ethutil.Bytes2Hex(value.Bytes())})
			}
		})
	}
}

func (gui *Gui) insertTransaction(window string, tx *ethchain.Transaction) {
	nameReg := ethpipe.New(gui.eth).World().Config().Get("NameReg")
	addr := gui.address()

	var inout string
	if bytes.Compare(tx.Sender(), addr) == 0 {
		inout = "send"
	} else {
		inout = "recv"
	}

	var (
		ptx  = ethpipe.NewJSTx(tx)
		send = nameReg.Storage(tx.Sender())
		rec  = nameReg.Storage(tx.Recipient)
		s, r string
	)

	if tx.CreatesContract() {
		rec = nameReg.Storage(tx.CreationAddress())
	}

	if send.Len() != 0 {
		s = strings.Trim(send.Str(), "\x00")
	} else {
		s = ethutil.Bytes2Hex(tx.Sender())
	}
	if rec.Len() != 0 {
		r = strings.Trim(rec.Str(), "\x00")
	} else {
		if tx.CreatesContract() {
			r = ethutil.Bytes2Hex(tx.CreationAddress())
		} else {
			r = ethutil.Bytes2Hex(tx.Recipient)
		}
	}
	ptx.Sender = s
	ptx.Address = r

	if window == "post" {
		gui.getObjectByName("transactionView").Call("addTx", ptx, inout)
	} else {
		gui.getObjectByName("pendingTxView").Call("addTx", ptx, inout)
	}
}

func (gui *Gui) readPreviousTransactions() {
	it := gui.txDb.Db().NewIterator(nil, nil)
	for it.Next() {
		tx := ethchain.NewTransactionFromBytes(it.Value())

		gui.insertTransaction("post", tx)

	}
	it.Release()
}

func (gui *Gui) processBlock(block *ethchain.Block, initial bool) {
	name := strings.Trim(gui.pipe.World().Config().Get("NameReg").Storage(block.Coinbase).Str(), "\x00")
	b := ethpipe.NewJSBlock(block)
	b.Name = name

	gui.getObjectByName("chainView").Call("addBlock", b, initial)
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

func (self *Gui) getObjectByName(objectName string) qml.Object {
	return self.win.Root().ObjectByName(objectName)
}

// Simple go routine function that updates the list of peers in the GUI
func (gui *Gui) update() {
	// We have to wait for qml to be done loading all the windows.
	for !gui.qmlDone {
		time.Sleep(500 * time.Millisecond)
	}

	go func() {
		go gui.setInitialBlockChain()
		gui.loadAddressBook()
		gui.setPeerInfo()
		gui.readPreviousTransactions()
	}()

	for _, plugin := range gui.plugins {
		gui.win.Root().Call("addPlugin", plugin.Path, "")
	}

	var (
		blockChan     = make(chan ethreact.Event, 100)
		txChan        = make(chan ethreact.Event, 100)
		objectChan    = make(chan ethreact.Event, 100)
		peerChan      = make(chan ethreact.Event, 100)
		chainSyncChan = make(chan ethreact.Event, 100)
		miningChan    = make(chan ethreact.Event, 100)
	)

	peerUpdateTicker := time.NewTicker(5 * time.Second)
	generalUpdateTicker := time.NewTicker(1 * time.Second)

	state := gui.eth.StateManager().TransState()

	unconfirmedFunds := new(big.Int)
	gui.win.Root().Call("setWalletValue", fmt.Sprintf("%v", ethutil.CurrencyToString(state.GetAccount(gui.address()).Balance)))
	gui.getObjectByName("syncProgressIndicator").Set("visible", !gui.eth.IsUpToDate())

	lastBlockLabel := gui.getObjectByName("lastBlockLabel")
	miningLabel := gui.getObjectByName("miningLabel")

	go func() {
		for {
			select {
			case b := <-blockChan:
				block := b.Resource.(*ethchain.Block)
				gui.processBlock(block, false)
				if bytes.Compare(block.Coinbase, gui.address()) == 0 {
					gui.setWalletValue(gui.eth.StateManager().CurrentState().GetAccount(gui.address()).Balance, nil)
				}
			case txMsg := <-txChan:
				tx := txMsg.Resource.(*ethchain.Transaction)

				if txMsg.Name == "newTx:pre" {
					object := state.GetAccount(gui.address())

					if bytes.Compare(tx.Sender(), gui.address()) == 0 {
						unconfirmedFunds.Sub(unconfirmedFunds, tx.Value)
					} else if bytes.Compare(tx.Recipient, gui.address()) == 0 {
						unconfirmedFunds.Add(unconfirmedFunds, tx.Value)
					}

					gui.setWalletValue(object.Balance, unconfirmedFunds)

					gui.insertTransaction("pre", tx)
				} else {
					object := state.GetAccount(gui.address())
					if bytes.Compare(tx.Sender(), gui.address()) == 0 {
						object.SubAmount(tx.Value)

						gui.getObjectByName("transactionView").Call("addTx", ethpipe.NewJSTx(tx), "send")
						gui.txDb.Put(tx.Hash(), tx.RlpEncode())
					} else if bytes.Compare(tx.Recipient, gui.address()) == 0 {
						object.AddAmount(tx.Value)

						gui.getObjectByName("transactionView").Call("addTx", ethpipe.NewJSTx(tx), "recv")
						gui.txDb.Put(tx.Hash(), tx.RlpEncode())
					}

					gui.setWalletValue(object.Balance, nil)

					state.UpdateStateObject(object)
				}
			case msg := <-chainSyncChan:
				sync := msg.Resource.(bool)
				gui.win.Root().ObjectByName("syncProgressIndicator").Set("visible", sync)

			case <-objectChan:
				gui.loadAddressBook()
			case <-peerChan:
				gui.setPeerInfo()
			case <-peerUpdateTicker.C:
				gui.setPeerInfo()
			case msg := <-miningChan:
				if msg.Name == "miner:start" {
					gui.miner = msg.Resource.(*ethminer.Miner)
				} else {
					gui.miner = nil
				}
			case <-generalUpdateTicker.C:
				statusText := "#" + gui.eth.BlockChain().CurrentBlock.Number.String()
				lastBlockLabel.Set("text", statusText)

				if gui.miner != nil {
					pow := gui.miner.GetPow()
					miningLabel.Set("text", "Mining @ "+strconv.FormatInt(pow.GetHashrate(), 10)+"Khash")
				}
			}
		}
	}()

	reactor := gui.eth.Reactor()

	reactor.Subscribe("newBlock", blockChan)
	reactor.Subscribe("newTx:pre", txChan)
	reactor.Subscribe("newTx:post", txChan)
	reactor.Subscribe("chainSync", chainSyncChan)
	reactor.Subscribe("miner:start", miningChan)
	reactor.Subscribe("miner:stop", miningChan)

	nameReg := gui.pipe.World().Config().Get("NameReg")
	reactor.Subscribe("object:"+string(nameReg.Address()), objectChan)

	reactor.Subscribe("peerList", peerChan)
}

func (gui *Gui) CopyToClipboard(data string) {
	//clipboard.WriteAll("test")
	fmt.Println("COPY currently BUGGED. Here are the contents:\n", data)
}

func (gui *Gui) setPeerInfo() {
	gui.win.Root().Call("setPeers", fmt.Sprintf("%d / %d", gui.eth.PeerCount(), gui.eth.MaxPeers))

	gui.win.Root().Call("resetPeers")
	for _, peer := range gui.pipe.Peers() {
		gui.win.Root().Call("addPeer", peer)
	}
}

func (gui *Gui) privateKey() string {
	return ethutil.Bytes2Hex(gui.eth.KeyManager().PrivateKey())
}

func (gui *Gui) address() []byte {
	return gui.eth.KeyManager().Address()
}

func (gui *Gui) Transact(recipient, value, gas, gasPrice, d string) (*ethpipe.JSReceipt, error) {
	var data string
	if len(recipient) == 0 {
		code, err := ethutil.Compile(d, false)
		if err != nil {
			return nil, err
		}
		data = ethutil.Bytes2Hex(code)
	} else {
		data = ethutil.Bytes2Hex(utils.FormatTransactionData(d))
	}

	return gui.pipe.Transact(gui.privateKey(), recipient, value, gas, gasPrice, data)
}

func (gui *Gui) SetCustomIdentifier(customIdentifier string) {
	gui.clientIdentity.SetCustomIdentifier(customIdentifier)
	gui.config.Save("id", customIdentifier)
}

func (gui *Gui) GetCustomIdentifier() string {
	return gui.clientIdentity.GetCustomIdentifier()
}

func (gui *Gui) ToggleTurboMining() {
	gui.miner.ToggleTurbo()
}

// functions that allow Gui to implement interface ethlog.LogSystem
func (gui *Gui) SetLogLevel(level ethlog.LogLevel) {
	gui.logLevel = level
	gui.config.Save("loglevel", level)
}

func (gui *Gui) GetLogLevel() ethlog.LogLevel {
	return gui.logLevel
}

func (self *Gui) AddPlugin(pluginPath string) {
	self.plugins[pluginPath] = plugin{Name: "SomeName", Path: pluginPath}

	json, _ := json.MarshalIndent(self.plugins, "", "    ")
	ethutil.WriteFile(ethutil.Config.ExecPath+"/plugins.json", json)
}

func (self *Gui) RemovePlugin(pluginPath string) {
	delete(self.plugins, pluginPath)

	json, _ := json.MarshalIndent(self.plugins, "", "    ")
	ethutil.WriteFile(ethutil.Config.ExecPath+"/plugins.json", json)
}

// this extra function needed to give int typecast value to gui widget
// that sets initial loglevel to default
func (gui *Gui) GetLogLevelInt() int {
	return int(gui.logLevel)
}

func (gui *Gui) Println(v ...interface{}) {
	gui.printLog(fmt.Sprintln(v...))
}

func (gui *Gui) Printf(format string, v ...interface{}) {
	gui.printLog(fmt.Sprintf(format, v...))
}

// Print function that logs directly to the GUI
func (gui *Gui) printLog(s string) {
	/*
		str := strings.TrimRight(s, "\n")
		lines := strings.Split(str, "\n")

		view := gui.getObjectByName("infoView")
		for _, line := range lines {
			view.Call("addLog", line)
		}
	*/
}

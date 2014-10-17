package main

import "C"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethminer"
	"github.com/ethereum/eth-go/ethpipe"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
	"gopkg.in/qml.v1"
)

/*
func LoadExtension(path string) (uintptr, error) {
	lib, err := ffi.NewLibrary(path)
	if err != nil {
		return 0, err
	}

	so, err := lib.Fct("sharedObject", ffi.Pointer, nil)
	if err != nil {
		return 0, err
	}

	ptr := so()

		err = lib.Close()
		if err != nil {
			return 0, err
		}

	return ptr.Interface().(uintptr), nil
}
*/

var logger = ethlog.NewLogger("GUI")

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

	miner  *ethminer.Miner
	stdLog ethlog.LogSystem
}

// Create GUI, but doesn't start it
func NewWindow(ethereum *eth.Ethereum, config *ethutil.ConfigManager, clientIdentity *ethwire.SimpleClientIdentity, session string, logLevel int) *Gui {
	db, err := ethdb.NewLDBDatabase("tx_database")
	if err != nil {
		panic(err)
	}

	pipe := ethpipe.NewJSPipe(ethereum)
	gui := &Gui{eth: ethereum, txDb: db, pipe: pipe, logLevel: ethlog.LogLevel(logLevel), Session: session, open: false, clientIdentity: clientIdentity, config: config, plugins: make(map[string]plugin)}
	data, _ := ethutil.ReadAllFile(path.Join(ethutil.Config.ExecPath, "plugins.json"))
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

	/*
		vec, errr := LoadExtension("/Users/jeffrey/Desktop/build-libqmltest-Desktop_Qt_5_2_1_clang_64bit-Debug/liblibqmltest_debug.dylib")
		fmt.Printf("Fetched vec with addr: %#x\n", vec)
		if errr != nil {
			fmt.Println(errr)
		} else {
			context.SetVar("vec", (unsafe.Pointer)(vec))
		}
	*/

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

func (gui *Gui) showWallet(context *qml.Context) (*qml.Window, error) {
	component, err := gui.engine.LoadFile(gui.uiLib.AssetPath("qml/main.qml"))
	if err != nil {
		return nil, err
	}

	gui.win = gui.createWindow(component)

	gui.update()

	return gui.win, nil
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
	pipe := ethpipe.New(gui.eth)
	nameReg := pipe.World().Config().Get("NameReg")
	addr := gui.address()

	var inout string
	if bytes.Compare(tx.Sender(), addr) == 0 {
		inout = "send"
	} else {
		inout = "recv"
	}

	var (
		ptx  = ethpipe.NewJSTx(tx, pipe.World().State())
		send = nameReg.Storage(tx.Sender())
		rec  = nameReg.Storage(tx.Recipient)
		s, r string
	)

	if tx.CreatesContract() {
		rec = nameReg.Storage(tx.CreationAddress(pipe.World().State()))
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
			r = ethutil.Bytes2Hex(tx.CreationAddress(pipe.World().State()))
		} else {
			r = ethutil.Bytes2Hex(tx.Recipient)
		}
	}
	ptx.Sender = s
	ptx.Address = r

	if window == "post" {
		//gui.getObjectByName("transactionView").Call("addTx", ptx, inout)
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
		logger.Infoln("Loading plugin ", plugin.Name)

		gui.win.Root().Call("addPlugin", plugin.Path, "")
	}

	peerUpdateTicker := time.NewTicker(5 * time.Second)
	generalUpdateTicker := time.NewTicker(500 * time.Millisecond)
	statsUpdateTicker := time.NewTicker(5 * time.Second)

	state := gui.eth.StateManager().TransState()

	unconfirmedFunds := new(big.Int)
	gui.win.Root().Call("setWalletValue", fmt.Sprintf("%v", ethutil.CurrencyToString(state.GetAccount(gui.address()).Balance)))

	lastBlockLabel := gui.getObjectByName("lastBlockLabel")
	miningLabel := gui.getObjectByName("miningLabel")

	events := gui.eth.EventMux().Subscribe(
		eth.ChainSyncEvent{},
		eth.PeerListEvent{},
		ethchain.NewBlockEvent{},
		ethchain.TxEvent{},
		ethminer.Event{},
	)

	// nameReg := gui.pipe.World().Config().Get("NameReg")
	// mux.Subscribe("object:"+string(nameReg.Address()), objectChan)

	go func() {
		defer events.Unsubscribe()
		for {
			select {
			case ev, isopen := <-events.Chan():
				if !isopen {
					return
				}
				switch ev := ev.(type) {
				case ethchain.NewBlockEvent:
					gui.processBlock(ev.Block, false)
					if bytes.Compare(ev.Block.Coinbase, gui.address()) == 0 {
						gui.setWalletValue(gui.eth.StateManager().CurrentState().GetAccount(gui.address()).Balance, nil)
					}

				case ethchain.TxEvent:
					tx := ev.Tx
					if ev.Type == ethchain.TxPre {
						object := state.GetAccount(gui.address())

						if bytes.Compare(tx.Sender(), gui.address()) == 0 {
							unconfirmedFunds.Sub(unconfirmedFunds, tx.Value)
						} else if bytes.Compare(tx.Recipient, gui.address()) == 0 {
							unconfirmedFunds.Add(unconfirmedFunds, tx.Value)
						}

						gui.setWalletValue(object.Balance, unconfirmedFunds)

						gui.insertTransaction("pre", tx)

					} else if ev.Type == ethchain.TxPost {
						object := state.GetAccount(gui.address())
						if bytes.Compare(tx.Sender(), gui.address()) == 0 {
							object.SubAmount(tx.Value)

							//gui.getObjectByName("transactionView").Call("addTx", ethpipe.NewJSTx(tx), "send")
							gui.txDb.Put(tx.Hash(), tx.RlpEncode())
						} else if bytes.Compare(tx.Recipient, gui.address()) == 0 {
							object.AddAmount(tx.Value)

							//gui.getObjectByName("transactionView").Call("addTx", ethpipe.NewJSTx(tx), "recv")
							gui.txDb.Put(tx.Hash(), tx.RlpEncode())
						}

						gui.setWalletValue(object.Balance, nil)

						state.UpdateStateObject(object)
					}

				// case object:
				// 	gui.loadAddressBook()

				case eth.PeerListEvent:
					gui.setPeerInfo()

				case ethminer.Event:
					if ev.Type == ethminer.Started {
						gui.miner = ev.Miner
					} else {
						gui.miner = nil
					}
				}

			case <-peerUpdateTicker.C:
				gui.setPeerInfo()
			case <-generalUpdateTicker.C:
				statusText := "#" + gui.eth.BlockChain().CurrentBlock.Number.String()
				lastBlockLabel.Set("text", statusText)

				if gui.miner != nil {
					pow := gui.miner.GetPow()
					miningLabel.Set("text", "Mining @ "+strconv.FormatInt(pow.GetHashrate(), 10)+"Khash")
				}

				blockLength := gui.eth.BlockPool().BlocksProcessed
				chainLength := gui.eth.BlockPool().ChainLength

				var (
					pct      float64 = 1.0 / float64(chainLength) * float64(blockLength)
					dlWidget         = gui.win.Root().ObjectByName("downloadIndicator")
					dlLabel          = gui.win.Root().ObjectByName("downloadLabel")
				)

				dlWidget.Set("value", pct)
				dlLabel.Set("text", fmt.Sprintf("%d / %d", blockLength, chainLength))

			case <-statsUpdateTicker.C:
				gui.setStatsPane()
			}
		}
	}()
}

func (gui *Gui) setStatsPane() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	statsPane := gui.getObjectByName("statsPane")
	statsPane.Set("text", fmt.Sprintf(`###### Mist 0.6.8 (%s) #######

eth %d (p2p = %d)

CPU:        # %d
Goroutines: # %d
CGoCalls:   # %d

Alloc:      %d
Heap Alloc: %d

CGNext:     %x
NumGC:      %d
`, runtime.Version(),
		eth.ProtocolVersion, eth.P2PVersion,
		runtime.NumCPU, runtime.NumGoroutine(), runtime.NumCgoCall(),
		memStats.Alloc, memStats.HeapAlloc,
		memStats.NextGC, memStats.NumGC,
	))
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

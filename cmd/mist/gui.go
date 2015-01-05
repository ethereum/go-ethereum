// Copyright (c) 2013-2014, Jeffrey Wilcke. All rights reserved.
//
// This library is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public
// License as published by the Free Software Foundation; either
// version 2.1 of the License, or (at your option) any later version.
//
// This library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this library; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston,
// MA 02110-1301  USA

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

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/ui/qt/qwhisper"
	"github.com/ethereum/go-ethereum/xeth"
	"gopkg.in/qml.v1"
)

var guilogger = logger.NewLogger("GUI")

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
	uiLib   *UiLib
	whisper *qwhisper.Whisper

	txDb *ethdb.LDBDatabase

	logLevel logger.LogLevel
	open     bool

	xeth *xeth.JSXEth

	Session        string
	clientIdentity *p2p.SimpleClientIdentity
	config         *ethutil.ConfigManager

	plugins map[string]plugin

	miner *miner.Miner
}

// Create GUI, but doesn't start it
func NewWindow(ethereum *eth.Ethereum, config *ethutil.ConfigManager, clientIdentity *p2p.SimpleClientIdentity, session string, logLevel int) *Gui {
	db, err := ethdb.NewLDBDatabase("tx_database")
	if err != nil {
		panic(err)
	}

	xeth := xeth.NewJSXEth(ethereum)
	gui := &Gui{eth: ethereum, txDb: db, xeth: xeth, logLevel: logger.LogLevel(logLevel), Session: session, open: false, clientIdentity: clientIdentity, config: config, plugins: make(map[string]plugin)}
	data, _ := ethutil.ReadAllFile(path.Join(ethutil.Config.ExecPath, "plugins.json"))
	json.Unmarshal([]byte(data), &gui.plugins)

	return gui
}

func (gui *Gui) Start(assetPath string) {
	defer gui.txDb.Close()

	guilogger.Infoln("Starting GUI")

	// Register ethereum functions
	qml.RegisterTypes("Ethereum", 1, 0, []qml.TypeSpec{{
		Init: func(p *xeth.JSBlock, obj qml.Object) { p.Number = 0; p.Hash = "" },
	}, {
		Init: func(p *xeth.JSTransaction, obj qml.Object) { p.Value = ""; p.Hash = ""; p.Address = "" },
	}, {
		Init: func(p *xeth.KeyVal, obj qml.Object) { p.Key = ""; p.Value = "" },
	}})
	// Create a new QML engine
	gui.engine = qml.NewEngine()
	context := gui.engine.Context()
	gui.uiLib = NewUiLib(gui.engine, gui.eth, assetPath)
	gui.whisper = qwhisper.New(gui.eth.Whisper())

	// Expose the eth library and the ui library to QML
	context.SetVar("gui", gui)
	context.SetVar("eth", gui.uiLib)
	context.SetVar("shh", gui.whisper)

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
		guilogger.Errorln("asset not found: you can set an alternative asset path on the command line using option 'asset_path'", err)

		panic(err)
	}

	gui.open = true
	win.Show()

	// only add the gui guilogger after window is shown otherwise slider wont be shown
	if addlog {
		logger.AddLogSystem(gui)
	}
	win.Wait()

	// need to silence gui guilogger after window closed otherwise logsystem hangs (but do not save loglevel)
	gui.logLevel = logger.Silence
	gui.open = false
}

func (gui *Gui) Stop() {
	if gui.open {
		gui.logLevel = logger.Silence
		gui.open = false
		gui.win.Hide()
	}

	gui.uiLib.jsEngine.Stop()

	guilogger.Infoln("Stopped")
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
		guilogger.Errorln("unable to import: ", err)
		return false
	}
	guilogger.Errorln("successfully imported: ", err)
	return true
}

func (gui *Gui) CreateAndSetPrivKey() (string, string, string, string) {
	err := gui.eth.KeyManager().Init(gui.Session, 0, true)
	if err != nil {
		guilogger.Errorln("unable to create key: ", err)
		return "", "", "", ""
	}
	return gui.eth.KeyManager().KeyPair().AsStrings()
}

func (gui *Gui) setInitialChain(ancientBlocks bool) {
	sBlk := gui.eth.ChainManager().LastBlockHash()
	blk := gui.eth.ChainManager().GetBlock(sBlk)
	for ; blk != nil; blk = gui.eth.ChainManager().GetBlock(sBlk) {
		sBlk = blk.ParentHash()

		gui.processBlock(blk, true)
	}
}

func (gui *Gui) loadAddressBook() {
	view := gui.getObjectByName("infoView")
	nameReg := gui.xeth.World().Config().Get("NameReg")
	if nameReg != nil {
		it := nameReg.Trie().Iterator()
		for it.Next() {
			if it.Key[0] != 0 {
				view.Call("addAddress", struct{ Name, Address string }{string(it.Key), ethutil.Bytes2Hex(it.Value)})
			}

		}
	}
}

func (self *Gui) loadMergedMiningOptions() {
	view := self.getObjectByName("mergedMiningModel")

	mergeMining := self.xeth.World().Config().Get("MergeMining")
	if mergeMining != nil {
		i := 0
		it := mergeMining.Trie().Iterator()
		for it.Next() {
			view.Call("addMergedMiningOption", struct {
				Checked       bool
				Name, Address string
				Id, ItemId    int
			}{false, string(it.Key), ethutil.Bytes2Hex(it.Value), 0, i})

			i++

		}
	}
}

func (gui *Gui) insertTransaction(window string, tx *types.Transaction) {
	nameReg := gui.xeth.World().Config().Get("NameReg")
	addr := gui.address()

	var inout string
	if bytes.Compare(tx.From(), addr) == 0 {
		inout = "send"
	} else {
		inout = "recv"
	}

	var (
		ptx  = xeth.NewJSTx(tx, gui.xeth.World().State())
		send = nameReg.Storage(tx.From())
		rec  = nameReg.Storage(tx.To())
		s, r string
	)

	if core.MessageCreatesContract(tx) {
		rec = nameReg.Storage(core.AddressFromMessage(tx))
	}

	if send.Len() != 0 {
		s = strings.Trim(send.Str(), "\x00")
	} else {
		s = ethutil.Bytes2Hex(tx.From())
	}
	if rec.Len() != 0 {
		r = strings.Trim(rec.Str(), "\x00")
	} else {
		if core.MessageCreatesContract(tx) {
			r = ethutil.Bytes2Hex(core.AddressFromMessage(tx))
		} else {
			r = ethutil.Bytes2Hex(tx.To())
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
	it := gui.txDb.NewIterator()
	for it.Next() {
		tx := types.NewTransactionFromBytes(it.Value())

		gui.insertTransaction("post", tx)

	}
	it.Release()
}

func (gui *Gui) processBlock(block *types.Block, initial bool) {
	name := strings.Trim(gui.xeth.World().Config().Get("NameReg").Storage(block.Coinbase()).Str(), "\x00")
	b := xeth.NewJSBlock(block)
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
		time.Sleep(300 * time.Millisecond)
	}

	go func() {
		go gui.setInitialChain(false)
		gui.loadAddressBook()
		gui.loadMergedMiningOptions()
		gui.setPeerInfo()
	}()

	gui.whisper.SetView(gui.win.Root().ObjectByName("whisperView"))

	for _, plugin := range gui.plugins {
		guilogger.Infoln("Loading plugin ", plugin.Name)

		gui.win.Root().Call("addPlugin", plugin.Path, "")
	}

	peerUpdateTicker := time.NewTicker(5 * time.Second)
	generalUpdateTicker := time.NewTicker(500 * time.Millisecond)
	statsUpdateTicker := time.NewTicker(5 * time.Second)

	state := gui.eth.ChainManager().TransState()

	gui.win.Root().Call("setWalletValue", fmt.Sprintf("%v", ethutil.CurrencyToString(state.GetAccount(gui.address()).Balance())))

	lastBlockLabel := gui.getObjectByName("lastBlockLabel")
	miningLabel := gui.getObjectByName("miningLabel")

	events := gui.eth.EventMux().Subscribe(
		//eth.PeerListEvent{},
		core.NewBlockEvent{},
		core.TxPreEvent{},
		core.TxPostEvent{},
	)

	go func() {
		defer events.Unsubscribe()
		for {
			select {
			case ev, isopen := <-events.Chan():
				if !isopen {
					return
				}
				switch ev := ev.(type) {
				case core.NewBlockEvent:
					gui.processBlock(ev.Block, false)
					if bytes.Compare(ev.Block.Coinbase(), gui.address()) == 0 {
						gui.setWalletValue(gui.eth.ChainManager().State().GetBalance(gui.address()), nil)
					}

				case core.TxPreEvent:
					tx := ev.Tx

					tstate := gui.eth.ChainManager().TransState()
					cstate := gui.eth.ChainManager().State()

					taccount := tstate.GetAccount(gui.address())
					caccount := cstate.GetAccount(gui.address())
					unconfirmedFunds := new(big.Int).Sub(taccount.Balance(), caccount.Balance())

					gui.setWalletValue(taccount.Balance(), unconfirmedFunds)
					gui.insertTransaction("pre", tx)

				case core.TxPostEvent:
					tx := ev.Tx
					object := state.GetAccount(gui.address())

					if bytes.Compare(tx.From(), gui.address()) == 0 {
						object.SubAmount(tx.Value())

						gui.txDb.Put(tx.Hash(), tx.RlpEncode())
					} else if bytes.Compare(tx.To(), gui.address()) == 0 {
						object.AddAmount(tx.Value())

						gui.txDb.Put(tx.Hash(), tx.RlpEncode())
					}

					gui.setWalletValue(object.Balance(), nil)
					state.UpdateStateObject(object)
				}

			case <-peerUpdateTicker.C:
				gui.setPeerInfo()
			case <-generalUpdateTicker.C:
				statusText := "#" + gui.eth.ChainManager().CurrentBlock().Number().String()
				lastBlockLabel.Set("text", statusText)
				miningLabel.Set("text", "Mining @ "+strconv.FormatInt(gui.uiLib.miner.GetPow().GetHashrate(), 10)+"Khash")

				/*
					blockLength := gui.eth.BlockPool().BlocksProcessed
					chainLength := gui.eth.BlockPool().ChainLength

					var (
						pct      float64 = 1.0 / float64(chainLength) * float64(blockLength)
						dlWidget         = gui.win.Root().ObjectByName("downloadIndicator")
						dlLabel          = gui.win.Root().ObjectByName("downloadLabel")
					)
					dlWidget.Set("value", pct)
					dlLabel.Set("text", fmt.Sprintf("%d / %d", blockLength, chainLength))
				*/

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
	statsPane.Set("text", fmt.Sprintf(`###### Mist %s (%s) #######

eth %d (p2p = %d)

CPU:        # %d
Goroutines: # %d
CGoCalls:   # %d

Alloc:      %d
Heap Alloc: %d

CGNext:     %x
NumGC:      %d
`, Version, runtime.Version(),
		eth.ProtocolVersion, 2,
		runtime.NumCPU, runtime.NumGoroutine(), runtime.NumCgoCall(),
		memStats.Alloc, memStats.HeapAlloc,
		memStats.NextGC, memStats.NumGC,
	))
}

func (gui *Gui) setPeerInfo() {
	gui.win.Root().Call("setPeers", fmt.Sprintf("%d / %d", gui.eth.PeerCount(), gui.eth.MaxPeers))
	gui.win.Root().Call("resetPeers")
	for _, peer := range gui.xeth.Peers() {
		gui.win.Root().Call("addPeer", peer)
	}
}

func (gui *Gui) privateKey() string {
	return ethutil.Bytes2Hex(gui.eth.KeyManager().PrivateKey())
}

func (gui *Gui) address() []byte {
	return gui.eth.KeyManager().Address()
}

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
/*
	vec, errr := LoadExtension("/Users/jeffrey/Desktop/build-libqmltest-Desktop_Qt_5_2_1_clang_64bit-Debug/liblibqmltest_debug.dylib")
	fmt.Printf("Fetched vec with addr: %#x\n", vec)
	if errr != nil {
		fmt.Println(errr)
	} else {
		context.SetVar("vec", (unsafe.Pointer)(vec))
	}
*/

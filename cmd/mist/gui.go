/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Jeffrey Wilcke <i@jev.io>
 */
package main

import "C"

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/ui/qt/qwhisper"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/obscuren/qml"
)

var guilogger = logger.NewLogger("GUI")

type ServEv byte

const (
	setup ServEv = iota
	update
)

type Gui struct {
	// The main application window
	win *qml.Window
	// QML Engine
	engine    *qml.Engine
	component *qml.Common
	// The ethereum interface
	eth           *eth.Ethereum
	serviceEvents chan ServEv

	// The public Ethereum library
	uiLib   *UiLib
	whisper *qwhisper.Whisper

	txDb *ethdb.LDBDatabase

	open bool

	xeth *xeth.XEth

	Session string

	plugins map[string]plugin
}

// Create GUI, but doesn't start it
func NewWindow(ethereum *eth.Ethereum) *Gui {
	db, err := ethdb.NewLDBDatabase(filepath.Join(ethereum.DataDir, "tx_database"))
	if err != nil {
		panic(err)
	}

	xeth := xeth.New(ethereum, nil)
	gui := &Gui{eth: ethereum,
		txDb:          db,
		xeth:          xeth,
		open:          false,
		plugins:       make(map[string]plugin),
		serviceEvents: make(chan ServEv, 1),
	}
	data, _ := ioutil.ReadFile(filepath.Join(ethereum.DataDir, "plugins.json"))
	json.Unmarshal(data, &gui.plugins)

	return gui
}

func (gui *Gui) Start(assetPath, libPath string) {
	defer gui.txDb.Close()

	guilogger.Infoln("Starting GUI")

	go gui.service()

	// Register ethereum functions
	qml.RegisterTypes("Ethereum", 1, 0, []qml.TypeSpec{{
		Init: func(p *xeth.Block, obj qml.Object) { p.Number = 0; p.Hash = "" },
	}, {
		Init: func(p *xeth.Transaction, obj qml.Object) { p.Value = ""; p.Hash = ""; p.Address = "" },
	}, {
		Init: func(p *xeth.KeyVal, obj qml.Object) { p.Key = ""; p.Value = "" },
	}})
	// Create a new QML engine
	gui.engine = qml.NewEngine()
	context := gui.engine.Context()
	gui.uiLib = NewUiLib(gui.engine, gui.eth, assetPath, libPath)
	gui.whisper = qwhisper.New(gui.eth.Whisper())

	// Expose the eth library and the ui library to QML
	context.SetVar("gui", gui)
	context.SetVar("eth", gui.uiLib)
	context.SetVar("shh", gui.whisper)
	//clipboard.SetQMLClipboard(context)

	win, err := gui.showWallet(context)
	if err != nil {
		guilogger.Errorln("asset not found: you can set an alternative asset path on the command line using option 'asset_path'", err)

		panic(err)
	}

	gui.open = true
	win.Show()

	win.Wait()
	gui.open = false
}

func (gui *Gui) Stop() {
	if gui.open {
		gui.open = false
		gui.win.Hide()
	}

	guilogger.Infoln("Stopped")
}

func (gui *Gui) showWallet(context *qml.Context) (*qml.Window, error) {
	component, err := gui.engine.LoadFile(gui.uiLib.AssetPath("qml/main.qml"))
	if err != nil {
		return nil, err
	}

	gui.createWindow(component)

	return gui.win, nil
}

func (gui *Gui) GenerateKey() {
	_, err := gui.eth.AccountManager().NewAccount("hurr")
	if err != nil {
		// TODO: UI feedback?
	}
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
	gui.win = comp.CreateWindow(nil)
	gui.uiLib.win = gui.win

	return gui.win
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
	/*
		view := gui.getObjectByName("infoView")
		nameReg := gui.xeth.World().Config().Get("NameReg")
		if nameReg != nil {
			it := nameReg.Trie().Iterator()
			for it.Next() {
				if it.Key[0] != 0 {
					view.Call("addAddress", struct{ Name, Address string }{string(it.Key), common.Bytes2Hex(it.Value)})
				}

			}
		}
	*/
}

func (self *Gui) loadMergedMiningOptions() {
	/*
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
				}{false, string(it.Key), common.Bytes2Hex(it.Value), 0, i})

				i++

			}
		}
	*/
}

func (gui *Gui) insertTransaction(window string, tx *types.Transaction) {
	var inout string
	from, _ := tx.From()
	if gui.eth.AccountManager().HasAccount(from) {
		inout = "send"
	} else {
		inout = "recv"
	}

	ptx := xeth.NewTx(tx)
	ptx.Sender = from.Hex()
	if to := tx.To(); to != nil {
		ptx.Address = to.Hex()
	}

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
	name := block.Coinbase().Hex()
	b := xeth.NewBlock(block)
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
		val := common.CurrencyToString(new(big.Int).Abs(common.BigCopy(unconfirmedFunds)))
		str = fmt.Sprintf("%v (%s %v)", common.CurrencyToString(amount), pos, val)
	} else {
		str = fmt.Sprintf("%v", common.CurrencyToString(amount))
	}

	gui.win.Root().Call("setWalletValue", str)
}

func (self *Gui) getObjectByName(objectName string) qml.Object {
	return self.win.Root().ObjectByName(objectName)
}

func (gui *Gui) SendCommand(cmd ServEv) {
	gui.serviceEvents <- cmd
}

func (gui *Gui) service() {
	for ev := range gui.serviceEvents {
		switch ev {
		case setup:
			go gui.setup()
		case update:
			go gui.update()
		}
	}
}

func (gui *Gui) setup() {
	for gui.win == nil {
		time.Sleep(time.Millisecond * 200)
	}

	for _, plugin := range gui.plugins {
		guilogger.Infoln("Loading plugin ", plugin.Name)
		gui.win.Root().Call("addPlugin", plugin.Path, "")
	}

	go func() {
		go gui.setInitialChain(false)
		gui.loadAddressBook()
		gui.loadMergedMiningOptions()
		gui.setPeerInfo()
	}()

	gui.whisper.SetView(gui.getObjectByName("whisperView"))

	gui.SendCommand(update)
}

// Simple go routine function that updates the list of peers in the GUI
func (gui *Gui) update() {
	peerUpdateTicker := time.NewTicker(5 * time.Second)
	generalUpdateTicker := time.NewTicker(500 * time.Millisecond)
	statsUpdateTicker := time.NewTicker(5 * time.Second)

	lastBlockLabel := gui.getObjectByName("lastBlockLabel")
	//miningLabel := gui.getObjectByName("miningLabel")

	events := gui.eth.EventMux().Subscribe(
		core.ChainEvent{},
		core.TxPreEvent{},
		core.TxPostEvent{},
	)

	defer events.Unsubscribe()
	for {
		select {
		case ev, isopen := <-events.Chan():
			if !isopen {
				return
			}
			switch ev := ev.(type) {
			case core.ChainEvent:
				gui.processBlock(ev.Block, false)
			case core.TxPreEvent:
				gui.insertTransaction("pre", ev.Tx)

			case core.TxPostEvent:
				gui.getObjectByName("pendingTxView").Call("removeTx", xeth.NewTx(ev.Tx))
			}

		case <-peerUpdateTicker.C:
			gui.setPeerInfo()

		case <-generalUpdateTicker.C:
			statusText := "#" + gui.eth.ChainManager().CurrentBlock().Number().String()
			lastBlockLabel.Set("text", statusText)
			//miningLabel.Set("text", strconv.FormatInt(gui.uiLib.Miner().HashRate(), 10))
		case <-statsUpdateTicker.C:
			gui.setStatsPane()
		}
	}
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

type qmlpeer struct{ Addr, NodeID, Name, Caps string }

type peersByID []*qmlpeer

func (s peersByID) Len() int           { return len(s) }
func (s peersByID) Less(i, j int) bool { return s[i].NodeID < s[j].NodeID }
func (s peersByID) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (gui *Gui) setPeerInfo() {
	peers := gui.eth.Peers()
	qpeers := make(peersByID, len(peers))
	for i, p := range peers {
		qpeers[i] = &qmlpeer{
			NodeID: p.ID().String(),
			Addr:   p.RemoteAddr().String(),
			Name:   p.Name(),
			Caps:   fmt.Sprint(p.Caps()),
		}
	}
	// we need to sort the peers because they jump around randomly
	// otherwise. order returned by eth.Peers is random because they
	// are taken from a map.
	sort.Sort(qpeers)

	gui.win.Root().Call("setPeerCounters", fmt.Sprintf("%d / %d", len(peers), gui.eth.MaxPeers()))
	gui.win.Root().Call("clearPeers")
	for _, p := range qpeers {
		gui.win.Root().Call("addPeer", p)
	}
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

package ethui

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/niemeyer/qml"
	"strings"
	"time"
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

	return &Tx{Hash: hash[:4], Value: tx.Value.String(), Address: sender}
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
}

// Create GUI, but doesn't start it
func New(ethereum *eth.Ethereum) *Gui {
	lib := &EthLib{blockManager: ethereum.BlockManager, blockChain: ethereum.BlockManager.BlockChain(), txPool: ethereum.TxPool}

	data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
	keyRing := ethutil.NewValueFromBytes(data)
	addr := keyRing.Get(1).Bytes()

	ethereum.BlockManager.WatchAddr(addr)

	return &Gui{eth: ethereum, lib: lib}
}

func (ui *Gui) Start() {
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
	ui.eth.TxPool.SecondaryProcessor = ui

	// Add the ui as a log system so we can log directly to the UGI
	ethutil.Config.Log.AddLogSystem(ui)

	// Loads previous blocks
	go ui.setInitialBlockChain()
	go ui.updatePeers()

	ui.win.Show()
	ui.win.Wait()
}

func (ui *Gui) setInitialBlockChain() {
	// Load previous 10 blocks
	chain := ui.eth.BlockManager.BlockChain().GetChain(ui.eth.BlockManager.BlockChain().CurrentBlock.Hash(), 10)
	for _, block := range chain {
		ui.ProcessBlock(block)
	}

}

func (ui *Gui) ProcessBlock(block *ethchain.Block) {
	ui.win.Root().Call("addBlock", NewBlockFromBlock(block))
}

func (ui *Gui) ProcessTransaction(tx *ethchain.Transaction) {
	ui.win.Root().Call("addTx", NewTxFromTransaction(tx))
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

// Simple go routine function that updates the list of peers in the GUI
func (ui *Gui) updatePeers() {
	for {
		ui.win.Root().Call("setPeers", fmt.Sprintf("%d / %d", ui.eth.Peers().Len(), ui.eth.MaxPeers))
		time.Sleep(1 * time.Second)
	}
}

// UI Library that has some basic functionality exposed
type UiLib struct {
	engine    *qml.Engine
	eth       *eth.Ethereum
	connected bool
}

// Opens a QML file (external application)
func (ui *UiLib) Open(path string) {
	component, err := ui.engine.LoadFile(path[7:])
	if err != nil {
		ethutil.Config.Log.Debugln(err)
	}
	win := component.CreateWindow(nil)

	go func() {
		win.Show()
		win.Wait()
	}()
}

func (ui *UiLib) Connect() {
	if !ui.connected {
		ui.eth.Start()
	}
}

func (ui *UiLib) ConnectToPeer(addr string) {
	ui.eth.ConnectToPeer(addr)
}

type Tester struct {
	root qml.Object
}

func (t *Tester) Compile(area qml.Object) {
	fmt.Println(area)
	ethutil.Config.Log.Infoln("[TESTER] Compiling")

	code := area.String("text")

	scanner := bufio.NewScanner(strings.NewReader(code))
	scanner.Split(bufio.ScanLines)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
}

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

type Gui struct {
	win       *qml.Window
	engine    *qml.Engine
	component *qml.Common
	eth       *eth.Ethereum

	// The Ethereum library
	lib *EthLib
}

func New(ethereum *eth.Ethereum) *Gui {
	lib := &EthLib{blockManager: ethereum.BlockManager, blockChain: ethereum.BlockManager.BlockChain(), txPool: ethereum.TxPool}

	return &Gui{eth: ethereum, lib: lib}
}

type Block struct {
	Number int
	Hash   string
}

func NewBlockFromBlock(block *ethchain.Block) *Block {
	info := block.BlockInfo()
	hash := hex.EncodeToString(block.Hash())

	return &Block{Number: int(info.Number), Hash: hash}
}

func (ui *Gui) Start() {
	qml.RegisterTypes("GoExtensions", 1, 0, []qml.TypeSpec{{
		Init: func(p *Block, obj qml.Object) { p.Number = 0; p.Hash = "" },
	}})

	ethutil.Config.Log.Infoln("[GUI] Starting GUI")
	ui.engine = qml.NewEngine()
	component, err := ui.engine.LoadFile("wallet.qml")
	if err != nil {
		panic(err)
	}

	ui.win = component.CreateWindow(nil)

	context := ui.engine.Context()
	context.SetVar("eth", ui.lib)
	context.SetVar("ui", &UiLib{engine: ui.engine})

	ui.eth.BlockManager.SecondaryBlockProcessor = ui

	go ui.setInitialBlockChain()
	go ui.updatePeers()

	ui.win.Show()
	ui.win.Wait()
}

func (ui *Gui) setInitialBlockChain() {
	chain := ui.eth.BlockManager.BlockChain().GetChain(ui.eth.BlockManager.BlockChain().CurrentBlock.Hash(), 10)
	for _, block := range chain {
		ui.ProcessBlock(block)
	}

	ui.eth.Start()
}

func (ui *Gui) ProcessBlock(block *ethchain.Block) {
	ui.win.Root().Call("addBlock", NewBlockFromBlock(block))
}

func (ui *Gui) updatePeers() {
	for {
		ui.win.Root().Call("setPeers", fmt.Sprintf("%d / %d", ui.eth.Peers().Len(), ui.eth.MaxPeers))
		time.Sleep(1 * time.Second)
	}
}

type UiLib struct {
	engine *qml.Engine
}

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

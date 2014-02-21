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
)

type Gui struct {
	win       *qml.Window
	engine    *qml.Engine
	component *qml.Common
	eth       *eth.Ethereum
}

func New(ethereum *eth.Ethereum) *Gui {
	return &Gui{eth: ethereum}
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
	root := ui.win.Root()

	context := ui.engine.Context()
	context.SetVar("tester", &Tester{root: root})

	ui.eth.BlockManager.SecondaryBlockProcessor = ui
	ui.eth.Start()

	ui.win.Show()
	ui.win.Wait()
}

func (ui *Gui) ProcessBlock(block *ethchain.Block) {
	ui.win.Root().Call("addBlock", NewBlockFromBlock(block))
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

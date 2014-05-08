package ethui

import (
	"bitbucket.org/kardianos/osext"
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/go-ethereum/utils"
	"github.com/go-qml/qml"
	"github.com/obscuren/mutan"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

type memAddr struct {
	Num   string
	Value string
}

// UI Library that has some basic functionality exposed
type UiLib struct {
	engine    *qml.Engine
	eth       *eth.Ethereum
	connected bool
	assetPath string
	// The main application window
	win *qml.Window
	Db  *Debugger
}

func NewUiLib(engine *qml.Engine, eth *eth.Ethereum, assetPath string) *UiLib {
	if assetPath == "" {
		assetPath = DefaultAssetPath()
	}
	return &UiLib{engine: engine, eth: eth, assetPath: assetPath}
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

func (ui *UiLib) OpenHtml(path string) {
	container := NewHtmlApplication(path, ui)
	app := NewExtApplication(container, ui)

	go app.run()
}

func (ui *UiLib) Watch(addr, storageAddr string) {
	if len(storageAddr) == 0 {
		ui.eth.Reactor().Subscribe("storage:"+string(ethutil.FromHex(addr))+":"+string(ethutil.FromHex(storageAddr)), nil)
	} else {
		ui.eth.Reactor().Subscribe("object:"+string(ethutil.FromHex(addr)), nil)
	}
}

func (ui *UiLib) Muted(content string) {
	component, err := ui.engine.LoadFile(ui.AssetPath("qml/muted.qml"))
	if err != nil {
		ethutil.Config.Log.Debugln(err)

		return
	}
	win := component.CreateWindow(nil)
	go func() {
		path := "file://" + ui.AssetPath("muted/index.html")
		win.Set("url", path)
		//debuggerPath := "file://" + ui.AssetPath("muted/debugger.html")
		//win.Set("debugUrl", debuggerPath)

		win.Show()
		win.Wait()
	}()
}

func (ui *UiLib) Connect(button qml.Object) {
	if !ui.connected {
		ui.eth.Start()
		ui.connected = true
		button.Set("enabled", false)
	}
}

func (ui *UiLib) ConnectToPeer(addr string) {
	ui.eth.ConnectToPeer(addr)
}

func (ui *UiLib) AssetPath(p string) string {
	return path.Join(ui.assetPath, p)
}

func DefaultAssetPath() string {
	var base string
	// If the current working directory is the go-ethereum dir
	// assume a debug build and use the source directory as
	// asset directory.
	pwd, _ := os.Getwd()
	if pwd == path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "ethereal") {
		base = path.Join(pwd, "assets")
	} else {
		switch runtime.GOOS {
		case "darwin":
			// Get Binary Directory
			exedir, _ := osext.ExecutableFolder()
			base = filepath.Join(exedir, "../Resources")
		case "linux":
			base = "/usr/share/ethereal"
		case "window":
			fallthrough
		default:
			base = "."
		}
	}

	return base
}

func (ui *UiLib) DebugTx(recipient, valueStr, gasStr, gasPriceStr, data string) {
	state := ui.eth.BlockChain().CurrentBlock.State()

	mainInput, _ := mutan.PreProcess(data)
	callerScript, err := utils.Compile(mainInput)
	if err != nil {
		ethutil.Config.Log.Debugln(err)

		return
	}

	dis := ethchain.Disassemble(callerScript)
	ui.win.Root().Call("clearAsm")

	for _, str := range dis {
		ui.win.Root().Call("setAsm", str)
	}
	callerTx := ethchain.NewContractCreationTx(ethutil.Big(valueStr), ethutil.Big(gasStr), ethutil.Big(gasPriceStr), callerScript, nil)

	// Contract addr as test address
	keyPair := ethutil.Config.Db.GetKeys()[0]
	account := ui.eth.StateManager().GetAddrState(keyPair.Address()).Object
	c := ethchain.MakeContract(callerTx, state)
	callerClosure := ethchain.NewClosure(account, c, c.Script(), state, ethutil.Big(gasStr), ethutil.Big(gasPriceStr))

	block := ui.eth.BlockChain().CurrentBlock
	vm := ethchain.NewVm(state, ui.eth.StateManager(), ethchain.RuntimeVars{
		Origin:      account.Address(),
		BlockNumber: block.BlockInfo().Number,
		PrevHash:    block.PrevHash,
		Coinbase:    block.Coinbase,
		Time:        block.Time,
		Diff:        block.Difficulty,
		TxData:      nil,
	})

	go func() {
		callerClosure.Call(vm, nil, ui.Db.halting)

		state.Reset()
	}()
}

func (ui *UiLib) Next() {
	ui.Db.Next()
}

type Debugger struct {
	win *qml.Window
	N   chan bool
}

func (d *Debugger) halting(pc int, op ethchain.OpCode, mem *ethchain.Memory, stack *ethchain.Stack) {
	d.win.Root().Call("setInstruction", pc)
	d.win.Root().Call("clearMem")
	d.win.Root().Call("clearStack")

	addr := 0
	for i := 0; i+32 <= mem.Len(); i += 32 {
		d.win.Root().Call("setMem", memAddr{fmt.Sprintf("%03d", addr), fmt.Sprintf("% x", mem.Data()[i:i+32])})
		addr++
	}

	for _, val := range stack.Data() {
		d.win.Root().Call("setStack", val.String())
	}

out:
	for {
		select {
		case <-d.N:
			break out
		default:
		}
	}
}

func (d *Debugger) Next() {
	d.N <- true
}

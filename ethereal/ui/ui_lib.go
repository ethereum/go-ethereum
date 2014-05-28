package ethui

import (
	"bitbucket.org/kardianos/osext"
	"encoding/hex"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/go-qml/qml"
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
	win      *qml.Window
	Db       *Debugger
	DbWindow *DebuggerWindow
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

		win.Show()
		win.Wait()
	}()
}

func (ui *UiLib) Connect(button qml.Object) {
	if !ui.connected {
		ui.eth.Start(true)
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
func (self *UiLib) StartDbWithContractAndData(contractHash, data string) {
	dbWindow := NewDebuggerWindow(self)
	object := self.eth.StateManager().CurrentState().GetStateObject(ethutil.FromHex(contractHash))
	if len(object.Script()) > 0 {
		dbWindow.SetCode("0x" + hex.EncodeToString(object.Script()))
	}
	dbWindow.SetData(data)

	dbWindow.Show()
}

func (self *UiLib) StartDbWithCode(code string) {
	dbWindow := NewDebuggerWindow(self)
	dbWindow.SetCode("0x" + code)
	dbWindow.Show()
}

func (self *UiLib) StartDebugger() {
	dbWindow := NewDebuggerWindow(self)
	//self.DbWindow = dbWindow

	dbWindow.Show()
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

	script, err := ethutil.Compile(data)
	if err != nil {
		ethutil.Config.Log.Debugln(err)

		return
	}

	dis := ethchain.Disassemble(script)
	ui.win.Root().Call("clearAsm")

	for _, str := range dis {
		ui.win.Root().Call("setAsm", str)
	}
	// Contract addr as test address
	keyPair := ethutil.GetKeyRing().Get(0)
	callerTx :=
		ethchain.NewContractCreationTx(ethutil.Big(valueStr), ethutil.Big(gasStr), ethutil.Big(gasPriceStr), script)
	callerTx.Sign(keyPair.PrivateKey)

	account := ui.eth.StateManager().TransState().GetStateObject(keyPair.Address())
	contract := ethchain.MakeContract(callerTx, state)
	callerClosure := ethchain.NewClosure(account, contract, contract.Init(), state, ethutil.Big(gasStr), ethutil.Big(gasPriceStr))

	block := ui.eth.BlockChain().CurrentBlock
	vm := ethchain.NewVm(state, ui.eth.StateManager(), ethchain.RuntimeVars{
		Origin:      account.Address(),
		BlockNumber: block.BlockInfo().Number,
		PrevHash:    block.PrevHash,
		Coinbase:    block.Coinbase,
		Time:        block.Time,
		Diff:        block.Difficulty,
	})

	ui.Db.done = false
	go func() {
		callerClosure.Call(vm, contract.Init(), ui.Db.halting)

		state.Reset()

		ui.Db.done = true
	}()
}

func (ui *UiLib) Next() {
	ui.Db.Next()
}

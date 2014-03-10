package ethui

import (
	"bitbucket.org/kardianos/osext"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/niemeyer/qml"
	"path"
	"path/filepath"
	"runtime"
)

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
	return AssetPath(p)
}

func AssetPath(p string) string {
	var base string
	switch runtime.GOOS {
	case "darwin":
		// Get Binary Directory
		exedir, _ := osext.ExecutableFolder()
		base = filepath.Join(exedir, "../Resources")
		base = "/Users/maranhidskes/projects/go/src/github.com/ethereum/go-ethereum"
	case "linux":
		base = "/usr/share/ethereal"
	case "window":
		fallthrough
	default:
		base = "."
	}

	return path.Join(base, p)
}

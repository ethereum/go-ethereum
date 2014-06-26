package main

import (
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/go-ethereum/ethereal/ui"
	"github.com/ethereum/go-ethereum/utils"
	"github.com/go-qml/qml"
	"os"
	"runtime"
)

func main() {
	// Leave QT on top at ALL times. Qt Needs to be initialized from the main thread
	qml.Init(nil)

	runtime.GOMAXPROCS(runtime.NumCPU())

	var interrupted = false
	utils.RegisterInterrupt(func(os.Signal) {
		interrupted = true
	})

	utils.HandleInterrupt()

	// precedence: code-internal flag default < config file < environment variables < command line
	Init() // parsing command line
	utils.InitConfig(ConfigFile, Datadir, Identifier, "ETH")

	utils.InitDataDir(Datadir)

	utils.InitLogging(Datadir, LogFile, LogLevel, DebugFile)

	ethereum := utils.NewEthereum(UseUPnP, OutboundPort, MaxPeer)

	// create, import, export keys
	utils.KeyTasks(GenAddr, ImportKey, ExportKey, NonInteractive)

	if ShowGenesis {
		utils.ShowGenesis(ethereum)
	}

	if StartRpc {
		utils.StartRpc(ethereum, RpcPort)
	}

	gui := ethui.New(ethereum, LogLevel)

	utils.RegisterInterrupt(func(os.Signal) {
		gui.Stop()
	})
	utils.StartEthereum(ethereum, UseSeed)
	// gui blocks the main thread
	gui.Start(AssetPath)
	// we need to run the interrupt callbacks in case gui is closed
	// this skips if we got here by actual interrupt stopping the GUI
	if !interrupted {
		utils.RunInterruptCallbacks(os.Interrupt)
	}
	// this blocks the thread
	ethereum.WaitForShutdown()
	ethlog.Flush()
}

package main

import (
	"os"
	"runtime"

	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/go-ethereum/utils"
	"gopkg.in/qml.v1"
)

const (
	ClientIdentifier = "Ethereal"
	Version          = "0.6.1"
)

var ethereum *eth.Ethereum

func run() error {
	// precedence: code-internal flag default < config file < environment variables < command line
	Init() // parsing command line

	config := utils.InitConfig(ConfigFile, Datadir, "ETH")

	utils.InitDataDir(Datadir)

	utils.InitLogging(Datadir, LogFile, LogLevel, DebugFile)

	db := utils.NewDatabase()

	keyManager := utils.NewKeyManager(KeyStore, Datadir, db)

	// create, import, export keys
	utils.KeyTasks(keyManager, KeyRing, GenAddr, SecretFile, ExportDir, NonInteractive)

	clientIdentity := utils.NewClientIdentity(ClientIdentifier, Version, Identifier)

	ethereum = utils.NewEthereum(db, clientIdentity, keyManager, UseUPnP, OutboundPort, MaxPeer)

	if ShowGenesis {
		utils.ShowGenesis(ethereum)
	}

	if StartRpc {
		utils.StartRpc(ethereum, RpcPort)
	}

	gui := NewWindow(ethereum, config, clientIdentity, KeyRing, LogLevel)

	utils.RegisterInterrupt(func(os.Signal) {
		gui.Stop()
	})
	utils.StartEthereum(ethereum, UseSeed)
	// gui blocks the main thread
	gui.Start(AssetPath)

	return nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// This is a bit of a cheat, but ey!
	os.Setenv("QTWEBKIT_INSPECTOR_SERVER", "127.0.0.1:99999")

	//qml.Init(nil)
	qml.Run(run)

	var interrupted = false
	utils.RegisterInterrupt(func(os.Signal) {
		interrupted = true
	})

	utils.HandleInterrupt()

	// we need to run the interrupt callbacks in case gui is closed
	// this skips if we got here by actual interrupt stopping the GUI
	if !interrupted {
		utils.RunInterruptCallbacks(os.Interrupt)
	}
	// this blocks the thread
	ethereum.WaitForShutdown()
	ethlog.Flush()
}

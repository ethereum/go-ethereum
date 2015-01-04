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

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p"
	"gopkg.in/qml.v1"
)

const (
	ClientIdentifier = "Mist"
	Version          = "0.7.11"
)

var ethereum *eth.Ethereum
var mainlogger = logger.NewLogger("MAIN")

func run() error {
	// precedence: code-internal flag default < config file < environment variables < command line
	Init() // parsing command line

	tstart := time.Now()
	config := utils.InitConfig(VmType, ConfigFile, Datadir, "ETH")

	ethereum, err := eth.New(&eth.Config{
		Name:       ClientIdentifier,
		Version:    Version,
		KeyStore:   KeyStore,
		DataDir:    Datadir,
		LogFile:    LogFile,
		LogLevel:   LogLevel,
		Identifier: Identifier,
		MaxPeers:   MaxPeer,
		Port:       OutboundPort,
		NATType:    PMPGateway,
		PMPGateway: PMPGateway,
		KeyRing:    KeyRing,
	})
	if err != nil {
		mainlogger.Fatalln(err)
	}
	utils.KeyTasks(ethereum.KeyManager(), KeyRing, GenAddr, SecretFile, ExportDir, NonInteractive)

	if StartRpc {
		utils.StartRpc(ethereum, RpcPort)
	}

	if StartWebSockets {
		utils.StartWebSockets(ethereum)
	}

	gui := NewWindow(ethereum, config, ethereum.ClientIdentity().(*p2p.SimpleClientIdentity), KeyRing, LogLevel)

	utils.RegisterInterrupt(func(os.Signal) {
		gui.Stop()
	})
	go utils.StartEthereum(ethereum, UseSeed)

	fmt.Println("ETH stack took", time.Since(tstart))

	// gui blocks the main thread
	gui.Start(AssetPath)

	return nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// This is a bit of a cheat, but ey!
	os.Setenv("QTWEBKIT_INSPECTOR_SERVER", "127.0.0.1:99999")

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
	logger.Flush()
}

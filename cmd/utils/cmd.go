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
 * 	Viktor Tron <viktor@ethdev.com>
 */
package utils

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/rlp"
	rpchttp "github.com/ethereum/go-ethereum/rpc/http"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/xeth"
)

var clilogger = logger.NewLogger("CLI")
var interruptCallbacks = []func(os.Signal){}

// Register interrupt handlers callbacks
func RegisterInterrupt(cb func(os.Signal)) {
	interruptCallbacks = append(interruptCallbacks, cb)
}

// go routine that call interrupt handlers in order of registering
func HandleInterrupt() {
	c := make(chan os.Signal, 1)
	go func() {
		signal.Notify(c, os.Interrupt)
		for sig := range c {
			clilogger.Errorf("Shutting down (%v) ... \n", sig)
			RunInterruptCallbacks(sig)
		}
	}()
}

func RunInterruptCallbacks(sig os.Signal) {
	for _, cb := range interruptCallbacks {
		cb(sig)
	}
}

func openLogFile(Datadir string, filename string) *os.File {
	path := ethutil.AbsolutePath(Datadir, filename)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("error opening log file '%s': %v", filename, err))
	}
	return file
}

func confirm(message string) bool {
	fmt.Println(message, "Are you sure? (y/n)")
	var r string
	fmt.Scanln(&r)
	for ; ; fmt.Scanln(&r) {
		if r == "n" || r == "y" {
			break
		} else {
			fmt.Printf("Yes or no? (%s)", r)
		}
	}
	return r == "y"
}

func initDataDir(Datadir string) {
	_, err := os.Stat(Datadir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Data directory '%s' doesn't exist, creating it\n", Datadir)
			os.Mkdir(Datadir, 0777)
		}
	}
}

func InitConfig(vmType int, ConfigFile string, Datadir string, EnvPrefix string) *ethutil.ConfigManager {
	initDataDir(Datadir)
	cfg := ethutil.ReadConfig(ConfigFile, Datadir, EnvPrefix)
	cfg.VmType = vmType

	return cfg
}

func exit(err error) {
	status := 0
	if err != nil {
		fmt.Fprintln(os.Stderr, "Fatal: ", err)
		status = 1
	}
	logger.Flush()
	os.Exit(status)
}

// Fatalf formats a message to standard output and exits the program.
func Fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Fatal: "+format+"\n", args...)
	logger.Flush()
	os.Exit(1)
}

func StartEthereum(ethereum *eth.Ethereum) {
	clilogger.Infoln("Starting ", ethereum.Name())
	if err := ethereum.Start(); err != nil {
		exit(err)
	}
	RegisterInterrupt(func(sig os.Signal) {
		ethereum.Stop()
		logger.Flush()
	})
}

func StartEthereumForTest(ethereum *eth.Ethereum) {
	clilogger.Infoln("Starting ", ethereum.Name())
	ethereum.StartForTest()
	RegisterInterrupt(func(sig os.Signal) {
		ethereum.Stop()
		logger.Flush()
	})
}

func KeyTasks(keyManager *crypto.KeyManager, KeyRing string, GenAddr bool, SecretFile string, ExportDir string, NonInteractive bool) {
	var err error
	switch {
	case GenAddr:
		if NonInteractive || confirm("This action overwrites your old private key.") {
			err = keyManager.Init(KeyRing, 0, true)
		}
		exit(err)
	case len(SecretFile) > 0:
		SecretFile = ethutil.ExpandHomePath(SecretFile)

		if NonInteractive || confirm("This action overwrites your old private key.") {
			err = keyManager.InitFromSecretsFile(KeyRing, 0, SecretFile)
		}
		exit(err)
	case len(ExportDir) > 0:
		err = keyManager.Init(KeyRing, 0, false)
		if err == nil {
			err = keyManager.Export(ExportDir)
		}
		exit(err)
	default:
		// Creates a keypair if none exists
		err = keyManager.Init(KeyRing, 0, false)
		if err != nil {
			exit(err)
		}
	}
	clilogger.Infof("Main address %x\n", keyManager.Address())
}

func StartRpc(ethereum *eth.Ethereum, RpcListenAddress string, RpcPort int) {
	var err error
	ethereum.RpcServer, err = rpchttp.NewRpcHttpServer(xeth.New(ethereum, nil), RpcListenAddress, RpcPort)
	if err != nil {
		clilogger.Errorf("Could not start RPC interface (port %v): %v", RpcPort, err)
	} else {
		go ethereum.RpcServer.Start()
	}
}

func FormatTransactionData(data string) []byte {
	d := ethutil.StringToByteFunc(data, func(s string) (ret []byte) {
		slice := regexp.MustCompile("\\n|\\s").Split(s, 1000000000)
		for _, dataItem := range slice {
			d := ethutil.FormatData(dataItem)
			ret = append(ret, d...)
		}
		return
	})

	return d
}

// Replay block
func BlockDo(ethereum *eth.Ethereum, hash []byte) error {
	block := ethereum.ChainManager().GetBlock(hash)
	if block == nil {
		return fmt.Errorf("unknown block %x", hash)
	}

	parent := ethereum.ChainManager().GetBlock(block.ParentHash())

	statedb := state.New(parent.Root(), ethereum.StateDb())
	_, err := ethereum.BlockProcessor().TransitionState(statedb, parent, block, true)
	if err != nil {
		return err
	}

	return nil

}

func ImportChain(chain *core.ChainManager, fn string) error {
	fmt.Printf("importing chain '%s'\n", fn)
	fh, err := os.OpenFile(fn, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()

	var blocks types.Blocks
	if err := rlp.Decode(fh, &blocks); err != nil {
		return err
	}

	chain.Reset()
	if err := chain.InsertChain(blocks); err != nil {
		return err
	}
	fmt.Printf("imported %d blocks\n", len(blocks))

	return nil
}

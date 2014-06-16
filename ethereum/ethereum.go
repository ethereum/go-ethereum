package main

import (
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/go-ethereum/utils"
	"github.com/rakyll/globalconf"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
)

const Debug = true

func RegisterInterrupt(cb func(os.Signal)) {
	go func() {
		// Buffered chan of one is enough
		c := make(chan os.Signal, 1)
		// Notify about interrupts for now
		signal.Notify(c, os.Interrupt)

		for sig := range c {
			cb(sig)
		}
	}()
}

func confirm(message string) bool {
	fmt.Println(message, "Are you sure? (y/n)")
	var r string
	fmt.Scanln(&r)
	for ; ; fmt.Scanln(&r) {
		if r == "n" || r == "y" {
			break
		} else {
			fmt.Printf("Yes or no?", r)
		}
	}
	return r == "y"
}

func main() {
	Init()

	runtime.GOMAXPROCS(runtime.NumCPU())

	// set logger
	var logSys *log.Logger
	flags := log.LstdFlags

	var lt ethutil.LoggerType
	if StartJsConsole || len(InputFile) > 0 {
		lt = ethutil.LogFile
	} else {
		lt = ethutil.LogFile | ethutil.LogStd
	}

	g, err := globalconf.NewWithOptions(&globalconf.Options{
		Filename: path.Join(ethutil.ApplicationFolder(Datadir), "conf.ini"),
	})
	if err != nil {
		fmt.Println(err)
	} else {
		g.ParseAll()
	}
	ethutil.ReadConfig(Datadir, lt, g, Identifier)

	logger := ethutil.Config.Log

	if LogFile != "" {
		logfile, err := os.OpenFile(LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(fmt.Sprintf("error opening log file '%s': %v", LogFile, err))
		}
		defer logfile.Close()
		log.SetOutput(logfile)
		logSys = log.New(logfile, "", flags)
		logger.AddLogSystem(logSys)
	} else {
		logSys = log.New(os.Stdout, "", flags)
	}

	// Instantiated a eth stack
	ethereum, err := eth.New(eth.CapDefault, UseUPnP)
	if err != nil {
		log.Println("eth start err:", err)
		return
	}
	ethereum.Port = OutboundPort

	// bookkeeping tasks
	switch {
	case GenAddr:
		if NonInteractive || confirm("This action overwrites your old private key.") {
			utils.CreateKeyPair(true)
		}
		os.Exit(0)
	case len(ImportKey) > 0:
		if NonInteractive || confirm("This action overwrites your old private key.") {
			mnemonic := strings.Split(ImportKey, " ")
			if len(mnemonic) == 24 {
				logSys.Println("Got mnemonic key, importing.")
				key := ethutil.MnemonicDecode(mnemonic)
				utils.ImportPrivateKey(key)
			} else if len(mnemonic) == 1 {
				logSys.Println("Got hex key, importing.")
				utils.ImportPrivateKey(ImportKey)
			} else {
				logSys.Println("Did not recognise format, exiting.")
			}
		}
		os.Exit(0)
	case ExportKey:
		keyPair := ethutil.GetKeyRing().Get(0)
		fmt.Printf(`
Generating new address and keypair.
Please keep your keys somewhere save.

++++++++++++++++ KeyRing +++++++++++++++++++
addr: %x
prvk: %x
pubk: %x
++++++++++++++++++++++++++++++++++++++++++++
save these words so you can restore your account later: %s
`, keyPair.Address(), keyPair.PrivateKey, keyPair.PublicKey)

		os.Exit(0)
	case ShowGenesis:
		logSys.Println(ethereum.BlockChain().Genesis())
		os.Exit(0)
	default:
		// Creates a keypair if non exists
		utils.CreateKeyPair(false)
	}

	// client
	logger.Infoln(fmt.Sprintf("Starting Ethereum v%s", ethutil.Config.Ver))

	// Set the max peers
	ethereum.MaxPeers = MaxPeer

	// Set Mining status
	ethereum.Mining = StartMining

	if StartMining {
		utils.DoMining(ethereum)
	}

	if StartJsConsole {
		repl := NewJSRepl(ethereum)

		go repl.Start()

		RegisterInterrupt(func(os.Signal) {
			repl.Stop()
		})
	} else if len(InputFile) > 0 {
		file, err := os.Open(InputFile)
		if err != nil {
			ethutil.Config.Log.Fatal(err)
		}

		content, err := ioutil.ReadAll(file)
		if err != nil {
			ethutil.Config.Log.Fatal(err)
		}

		re := NewJSRE(ethereum)
		RegisterInterrupt(func(os.Signal) {
			re.Stop()
		})
		re.Run(string(content))
	}

	if StartRpc {
		utils.DoRpc(ethereum, RpcPort)
	}

	RegisterInterrupt(func(sig os.Signal) {
		fmt.Printf("Shutting down (%v) ... \n", sig)
		ethereum.Stop()
	})

	ethereum.Start(UseSeed)

	// Wait for shutdown
	ethereum.WaitForShutdown()
}

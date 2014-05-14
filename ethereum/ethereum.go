package main

import (
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethrpc"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/go-ethereum/utils"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
)

const Debug = true

// Register interrupt handlers so we can stop the ethereum
func RegisterInterrupts(s *eth.Ethereum) {
	// Buffered chan of one is enough
	c := make(chan os.Signal, 1)
	// Notify about interrupts for now
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			fmt.Printf("Shutting down (%v) ... \n", sig)
			s.Stop()
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

	ethutil.ReadConfig(DataDir)
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
	}
	/*else {
		logSys = log.New(os.Stdout, "", flags)
	}*/

	ethchain.InitFees()

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
		key := ethutil.Config.Db.GetKeys()[0]
		logSys.Println(fmt.Sprintf("prvk: %x\n", key.PrivateKey))
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

	if StartConsole {
		err := os.Mkdir(ethutil.Config.ExecPath, os.ModePerm)
		// Error is OK if the error is ErrExist
		if err != nil && !os.IsExist(err) {
			log.Panic("Unable to create EXECPATH:", err)
		}

		console := NewConsole(ethereum)
		go console.Start()
	}
	if StartRpc {
		ethereum.RpcServer, err = ethrpc.NewJsonRpcServer(ethpub.NewPEthereum(ethereum), RpcPort)
		if err != nil {
			logger.Infoln("Could not start RPC interface:", err)
		} else {
			go ethereum.RpcServer.Start()
		}
	}

	RegisterInterrupts(ethereum)

	ethereum.Start(UseSeed)

	// Wait for shutdown
	ethereum.WaitForShutdown()
}

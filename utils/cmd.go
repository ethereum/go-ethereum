package utils

import (
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethminer"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethrpc"
	"github.com/ethereum/eth-go/ethutil"
	"io"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"
	"time"
)

var logger = ethlog.NewLogger("CLI")
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
			logger.Errorf("Shutting down (%v) ... \n", sig)
			RunInterruptCallbacks(sig)
		}
	}()
}

func RunInterruptCallbacks(sig os.Signal) {
	for _, cb := range interruptCallbacks {
		cb(sig)
	}
}

func AbsolutePath(Datadir string, filename string) string {
	if path.IsAbs(filename) {
		return filename
	}
	return path.Join(Datadir, filename)
}

func openLogFile(Datadir string, filename string) *os.File {
	path := AbsolutePath(Datadir, filename)
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
			fmt.Printf("Yes or no?", r)
		}
	}
	return r == "y"
}

func InitDataDir(Datadir string) {
	_, err := os.Stat(Datadir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Debug logging directory '%s' doesn't exist, creating it\n", Datadir)
			os.Mkdir(Datadir, 0777)
		}
	}
}

func InitLogging(Datadir string, LogFile string, LogLevel int, DebugFile string) {
	var writer io.Writer
	if LogFile == "" {
		writer = os.Stdout
	} else {
		writer = openLogFile(Datadir, LogFile)
	}
	ethlog.AddLogSystem(ethlog.NewStdLogSystem(writer, log.LstdFlags, ethlog.LogLevel(LogLevel)))
	if DebugFile != "" {
		writer = openLogFile(Datadir, DebugFile)
		ethlog.AddLogSystem(ethlog.NewStdLogSystem(writer, log.LstdFlags, ethlog.DebugLevel))
	}
}

func InitConfig(ConfigFile string, Datadir string, Identifier string, EnvPrefix string) {
	InitDataDir(Datadir)
	ethutil.ReadConfig(ConfigFile, Datadir, Identifier, EnvPrefix)
	ethutil.Config.Set("rpcport", "700")
}

func exit(status int) {
	ethlog.Flush()
	os.Exit(status)
}

func NewEthereum(UseUPnP bool, OutboundPort string, MaxPeer int) *eth.Ethereum {
	ethereum, err := eth.New(eth.CapDefault, UseUPnP)
	if err != nil {
		logger.Fatalln("eth start err:", err)
	}
	ethereum.Port = OutboundPort
	ethereum.MaxPeers = MaxPeer
	return ethereum
}

func StartEthereum(ethereum *eth.Ethereum, UseSeed bool) {
	logger.Infof("Starting Ethereum v%s", ethutil.Config.Ver)
	ethereum.Start(UseSeed)
	RegisterInterrupt(func(sig os.Signal) {
		ethereum.Stop()
		ethlog.Flush()
	})
}

func ShowGenesis(ethereum *eth.Ethereum) {
	logger.Infoln(ethereum.BlockChain().Genesis())
	exit(0)
}

func KeyTasks(GenAddr bool, ImportKey string, ExportKey bool, NonInteractive bool) {
	switch {
	case GenAddr:
		if NonInteractive || confirm("This action overwrites your old private key.") {
			CreateKeyPair(true)
		}
		exit(0)
	case len(ImportKey) > 0:
		if NonInteractive || confirm("This action overwrites your old private key.") {
			// import should be from file
			mnemonic := strings.Split(ImportKey, " ")
			if len(mnemonic) == 24 {
				logger.Infoln("Got mnemonic key, importing.")
				key := ethutil.MnemonicDecode(mnemonic)
				ImportPrivateKey(key)
			} else if len(mnemonic) == 1 {
				logger.Infoln("Got hex key, importing.")
				ImportPrivateKey(ImportKey)
			} else {
				logger.Errorln("Did not recognise format, exiting.")
			}
		}
		exit(0)
	case ExportKey: // this should be exporting to a filename
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

		exit(0)
	default:
		// Creates a keypair if none exists
		CreateKeyPair(false)
	}
}

func StartRpc(ethereum *eth.Ethereum, RpcPort int) {
	var err error
	ethereum.RpcServer, err = ethrpc.NewJsonRpcServer(ethpub.NewPEthereum(ethereum), RpcPort)
	if err != nil {
		logger.Errorf("Could not start RPC interface (port %v): %v", RpcPort, err)
	} else {
		go ethereum.RpcServer.Start()
	}
}

var miner ethminer.Miner

func StartMining(ethereum *eth.Ethereum) bool {
	if !ethereum.Mining {
		ethereum.Mining = true

		if ethutil.GetKeyRing().Len() == 0 {
			logger.Errorln("No address found, can't start mining")
			ethereum.Mining = false
			return true //????
		}
		keyPair := ethutil.GetKeyRing().Get(0)
		addr := keyPair.Address()

		go func() {
			miner = ethminer.NewDefaultMiner(addr, ethereum)
			// Give it some time to connect with peers
			time.Sleep(3 * time.Second)
			for !ethereum.IsUpToDate() {
				time.Sleep(5 * time.Second)
			}

			logger.Infoln("Miner started")
			miner := ethminer.NewDefaultMiner(addr, ethereum)
			miner.Start()
		}()
		RegisterInterrupt(func(os.Signal) {
			StopMining(ethereum)
		})
		return true
	}
	return false
}

func StopMining(ethereum *eth.Ethereum) bool {
	if ethereum.Mining {
		miner.Stop()
		logger.Infoln("Miner stopped")
		ethereum.Mining = false
		return true
	}
	return false
}

// Replay block
func BlockDo(ethereum *eth.Ethereum, hash []byte) error {
	block := ethereum.BlockChain().GetBlock(hash)
	if block == nil {
		return fmt.Errorf("unknown block %x", hash)
	}

	parent := ethereum.BlockChain().GetBlock(block.PrevHash)

	_, err := ethereum.StateManager().ApplyDiff(parent.State(), parent, block)
	if err != nil {
		return err
	}

	return nil

}

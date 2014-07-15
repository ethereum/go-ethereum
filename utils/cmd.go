package utils

import (
	"bitbucket.org/kardianos/osext"
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethminer"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethrpc"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
	"io"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
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
			fmt.Printf("Data directory '%s' doesn't exist, creating it\n", Datadir)
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

func InitConfig(ConfigFile string, Datadir string, EnvPrefix string) *ethutil.ConfigManager {
	InitDataDir(Datadir)
	return ethutil.ReadConfig(ConfigFile, Datadir, EnvPrefix)
}

func exit(err error) {
	status := 0
	if err != nil {
		fmt.Println(err)
		logger.Errorln("Fatal: ", err)
		status = 1
	}
	ethlog.Flush()
	os.Exit(status)
}

func NewDatabase() ethutil.Database {
	db, err := ethdb.NewLDBDatabase("database")
	if err != nil {
		exit(err)
	}
	return db
}

func NewClientIdentity(clientIdentifier, version, customIdentifier string) *ethwire.SimpleClientIdentity {
	return ethwire.NewSimpleClientIdentity(clientIdentifier, version, customIdentifier)
}

func NewEthereum(db ethutil.Database, clientIdentity ethwire.ClientIdentity, keyManager *ethcrypto.KeyManager, usePnp bool, OutboundPort string, MaxPeer int) *eth.Ethereum {
	ethereum, err := eth.New(db, clientIdentity, keyManager, eth.CapDefault, usePnp)
	if err != nil {
		logger.Fatalln("eth start err:", err)
	}
	ethereum.Port = OutboundPort
	ethereum.MaxPeers = MaxPeer
	return ethereum
}

func StartEthereum(ethereum *eth.Ethereum, UseSeed bool) {
	logger.Infof("Starting %s", ethereum.ClientIdentity())
	ethereum.Start(UseSeed)
	RegisterInterrupt(func(sig os.Signal) {
		ethereum.Stop()
		ethlog.Flush()
	})
}

func ShowGenesis(ethereum *eth.Ethereum) {
	logger.Infoln(ethereum.BlockChain().Genesis())
	exit(nil)
}

func NewKeyManager(KeyStore string, Datadir string, db ethutil.Database) *ethcrypto.KeyManager {
	var keyManager *ethcrypto.KeyManager
	switch {
	case KeyStore == "db":
		keyManager = ethcrypto.NewDBKeyManager(db)
	case KeyStore == "file":
		keyManager = ethcrypto.NewFileKeyManager(Datadir)
	default:
		exit(fmt.Errorf("unknown keystore type: %s", KeyStore))
	}
	return keyManager
}

func DefaultAssetPath() string {
	var assetPath string
	// If the current working directory is the go-ethereum dir
	// assume a debug build and use the source directory as
	// asset directory.
	pwd, _ := os.Getwd()
	if pwd == path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "ethereal") {
		assetPath = path.Join(pwd, "assets")
	} else {
		switch runtime.GOOS {
		case "darwin":
			// Get Binary Directory
			exedir, _ := osext.ExecutableFolder()
			assetPath = filepath.Join(exedir, "../Resources")
		case "linux":
			assetPath = "/usr/share/ethereal"
		case "windows":
			assetPath = "./assets"
		default:
			assetPath = "."
		}
	}
	return assetPath
}
func KeyTasks(keyManager *ethcrypto.KeyManager, KeyRing string, GenAddr bool, SecretFile string, ExportDir string, NonInteractive bool) {
	ethcrypto.InitWords(DefaultAssetPath()) // Init mnemonic word list

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

		addr := ethereum.KeyManager().Address()

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

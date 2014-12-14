package utils

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"runtime"

	"bitbucket.org/kardianos/osext"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
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

func DBSanityCheck(db ethutil.Database) error {
	d, _ := db.Get([]byte("ProtocolVersion"))
	protov := ethutil.NewValue(d).Uint()
	if protov != eth.ProtocolVersion && protov != 0 {
		return fmt.Errorf("Database version mismatch. Protocol(%d / %d). `rm -rf %s`", protov, eth.ProtocolVersion, ethutil.Config.ExecPath+"/database")
	}

	return nil
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

func InitLogging(Datadir string, LogFile string, LogLevel int, DebugFile string) logger.LogSystem {
	var writer io.Writer
	if LogFile == "" {
		writer = os.Stdout
	} else {
		writer = openLogFile(Datadir, LogFile)
	}

	sys := logger.NewStdLogSystem(writer, log.LstdFlags, logger.LogLevel(LogLevel))
	logger.AddLogSystem(sys)
	if DebugFile != "" {
		writer = openLogFile(Datadir, DebugFile)
		logger.AddLogSystem(logger.NewStdLogSystem(writer, log.LstdFlags, logger.DebugLevel))
	}

	return sys
}

func InitConfig(vmType int, ConfigFile string, Datadir string, EnvPrefix string) *ethutil.ConfigManager {
	InitDataDir(Datadir)
	cfg := ethutil.ReadConfig(ConfigFile, Datadir, EnvPrefix)
	cfg.VmType = vmType

	return cfg
}

func exit(err error) {
	status := 0
	if err != nil {
		clilogger.Errorln("Fatal: ", err)
		status = 1
	}
	logger.Flush()
	os.Exit(status)
}

func NewDatabase() ethutil.Database {
	db, err := ethdb.NewLDBDatabase("database")
	if err != nil {
		exit(err)
	}
	return db
}

func NewClientIdentity(clientIdentifier, version, customIdentifier string, pubkey string) *p2p.SimpleClientIdentity {
	return p2p.NewSimpleClientIdentity(clientIdentifier, version, customIdentifier, pubkey)
}

func NatType(natType string, gateway string) (nat p2p.NAT) {
	switch natType {
	case "UPNP":
		nat = p2p.UPNP()
	case "PMP":
		ip := net.ParseIP(gateway)
<<<<<<< HEAD
		if ip != nil {
			clilogger.Fatalf("bad PMP gateway '%s'", gateway)
=======
		if ip == nil {
			clilogger.Fatalln("cannot resolve PMP gateway IP %s", gateway)
>>>>>>> adapt cmd/cli to new backend
		}
		nat = p2p.PMP(ip)
	case "":
	default:
<<<<<<< HEAD
		clilogger.Fatalf("unrecognised NAT type '%s'", natType)
=======
		clilogger.Fatalln("unrecognised NAT type %s", natType)
>>>>>>> adapt cmd/cli to new backend
	}
	return
}

func NewEthereum(db ethutil.Database, clientIdentity p2p.ClientIdentity, keyManager *crypto.KeyManager, nat p2p.NAT, OutboundPort string, MaxPeer int) *eth.Ethereum {
	ethereum, err := eth.New(db, clientIdentity, keyManager, nat, OutboundPort, MaxPeer)
	if err != nil {
		clilogger.Fatalln("eth start err:", err)
	}
	return ethereum
}

func StartEthereum(ethereum *eth.Ethereum, UseSeed bool) {
	clilogger.Infof("Starting %s", ethereum.ClientIdentity())
	ethereum.Start(UseSeed)
	RegisterInterrupt(func(sig os.Signal) {
		ethereum.Stop()
		logger.Flush()
	})
}

func ShowGenesis(ethereum *eth.Ethereum) {
	clilogger.Infoln(ethereum.ChainManager().Genesis())
	exit(nil)
}

func NewKeyManager(KeyStore string, Datadir string, db ethutil.Database) *crypto.KeyManager {
	var keyManager *crypto.KeyManager
	switch {
	case KeyStore == "db":
		keyManager = crypto.NewDBKeyManager(db)
	case KeyStore == "file":
		keyManager = crypto.NewFileKeyManager(Datadir)
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
	if pwd == path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "cmd", "mist") {
		assetPath = path.Join(pwd, "assets")
	} else {
		switch runtime.GOOS {
		case "darwin":
			// Get Binary Directory
			exedir, _ := osext.ExecutableFolder()
			assetPath = filepath.Join(exedir, "../Resources")
		case "linux":
			assetPath = "/usr/share/mist"
		case "windows":
			assetPath = "./assets"
		default:
			assetPath = "."
		}
	}
	return assetPath
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

func StartRpc(ethereum *eth.Ethereum, RpcPort int) {
	var err error
	ethereum.RpcServer, err = rpc.NewJsonRpcServer(xeth.NewJSXEth(ethereum), RpcPort)
	if err != nil {
		clilogger.Errorf("Could not start RPC interface (port %v): %v", RpcPort, err)
	} else {
		go ethereum.RpcServer.Start()
	}
}

var gminer *miner.Miner

func GetMiner() *miner.Miner {
	return gminer
}

func StartMining(ethereum *eth.Ethereum) bool {
	if !ethereum.Mining {
		ethereum.Mining = true
		addr := ethereum.KeyManager().Address()

		go func() {
			clilogger.Infoln("Start mining")
			if gminer == nil {
				gminer = miner.New(addr, ethereum)
			}
			gminer.Start()
		}()
		RegisterInterrupt(func(os.Signal) {
			StopMining(ethereum)
		})
		return true
	}
	return false
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

func StopMining(ethereum *eth.Ethereum) bool {
	if ethereum.Mining && gminer != nil {
		gminer.Stop()
		clilogger.Infoln("Stopped mining")
		ethereum.Mining = false

		return true
	}

	return false
}

// Replay block
func BlockDo(ethereum *eth.Ethereum, hash []byte) error {
	block := ethereum.ChainManager().GetBlock(hash)
	if block == nil {
		return fmt.Errorf("unknown block %x", hash)
	}

	parent := ethereum.ChainManager().GetBlock(block.PrevHash)

	_, err := ethereum.BlockManager().TransitionState(parent.State(), parent, block)
	if err != nil {
		return err
	}

	return nil

}

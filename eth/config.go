package eth

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
)

// Config is the configuration object which holds information about the various
// sub system and ethereum's environment and settings.
type Config struct {
	DevMode bool // Developer mode
	TestNet bool // Testnet mode

	Name         string       // Name of the instance (visible through p2p)
	NetworkId    int          // The network id used for p2p
	GenesisFile  string       // genesis file for initialising the genesis block
	GenesisBlock *types.Block // used by block tests
	FastSync     bool         // enble fast sync
	Olympic      bool         // enable olympic settings

	BlockChainVersion  int  // version of the block chain in database
	SkipBcVersionCheck bool // e.g. blockchain export
	DatabaseCache      int  // Max cache for leveldb

	DataDir   string // datadir containing leveldb, node settings, etc.
	LogFile   string // file to which logs are written
	Verbosity int    // the level of verbosity
	VmDebug   bool   // log debug output during vm execution
	NatSpec   bool   // enable natspec
	DocRoot   string // documentation root for natspec
	AutoDAG   bool   // pre-generate dags
	PowTest   bool   // enabled pow test
	ExtraData []byte // default extra data to be used for the miner

	MaxPeers        int    // maximum amount of peers
	MaxPendingPeers int    // maximum amount of pending peers
	Discovery       bool   // enable discovery
	Port            string // port to be used for p2p

	BootNodes string // Space-separated list of discovery node URLs

	// This key is used to identify the node on the network.
	// If nil, an ephemeral key is used.
	NodeKey *ecdsa.PrivateKey

	NAT    nat.Interface // NAT interface
	Shh    bool          // enable shh
	NoDial bool          // disable outgoing dials
	NoConn bool          // disable peers

	AccountManager *accounts.Manager // account manager
	LightKDF       bool              // enables light KDF

	Etherbase    common.Address // default coinbase
	GasPrice     *big.Int       // minimum acceptable gas price (tx relay, mining)
	MinerThreads int            // amount of default miner threads to be used during mining
	SolcPath     string         // path to solidity executable

	GpoMinGasPrice          *big.Int // GPO minimum gas price
	GpoMaxGasPrice          *big.Int // GPO maximum gas price
	GpoFullBlockRatio       int
	GpobaseStepDown         int
	GpobaseStepUp           int
	GpobaseCorrectionFactor int

	// NewDB is used to create databases.
	// If nil, the default is to create leveldb databases on disk.
	NewDB func(path string) (ethdb.Database, error)
}

// MakeConfig sets missing default values
func MakeConfig(cfg *Config) {
	if cfg.BlockChainVersion < 3 {
		cfg.BlockChainVersion = 3
	}
	if len(cfg.Name) == 0 {
		cfg.Name = "Custom-Ethereum-Client"
	}
	if len(cfg.DataDir) == 0 {
		cfg.DataDir = common.DefaultDataDir()
	}
	if cfg.MaxPeers == 0 && !cfg.NoConn {
		cfg.MaxPeers = 25
	}
	if len(cfg.Port) == 0 {
		cfg.Port = "0" // auto
	}
	if cfg.TestNet && cfg.NetworkId == 0 {
		cfg.NetworkId = 2
	}
	if cfg.NAT == nil {
		cfg.NAT, _ = nat.Parse("any")
	}
	if cfg.GasPrice == nil {
		cfg.GasPrice = new(big.Int).Mul(big.NewInt(10), common.Szabo)
	}
	if cfg.GpoMinGasPrice == nil {
		cfg.GpoMinGasPrice = new(big.Int).Mul(big.NewInt(50), common.Shannon)
	}
	if cfg.GpoMaxGasPrice == nil {
		cfg.GpoMaxGasPrice = new(big.Int).Mul(big.NewInt(500), common.Shannon)
	}
	if cfg.GpoFullBlockRatio == 0 && cfg.GpobaseStepDown == 0 && cfg.GpobaseStepUp == 0 && cfg.GpobaseCorrectionFactor == 0 {
		cfg.GpoFullBlockRatio = 80
		cfg.GpobaseStepDown = 10
		cfg.GpobaseStepUp = 100
		cfg.GpobaseCorrectionFactor = 110
	}
	if cfg.AccountManager == nil {
		scryptN := crypto.StandardScryptN
		scryptP := crypto.StandardScryptP
		if cfg.LightKDF {
			scryptN = crypto.LightScryptN
			scryptP = crypto.LightScryptP
		}
		cfg.AccountManager = accounts.NewManager(crypto.NewKeyStorePassphrase(filepath.Join(cfg.DataDir, "keystore"), scryptN, scryptP))
	}
	glog.SetV(cfg.Verbosity)
	glog.CopyStandardLogTo("INFO")
	glog.SetToStderr(true)
}

func (cfg *Config) parseBootNodes() []*discover.Node {
	if cfg.BootNodes == "" {
		if cfg.TestNet {
			return defaultTestNetBootNodes
		}

		return defaultBootNodes
	}
	var ns []*discover.Node
	for _, url := range strings.Split(cfg.BootNodes, " ") {
		if url == "" {
			continue
		}
		n, err := discover.ParseNode(url)
		if err != nil {
			glog.V(logger.Error).Infof("Bootstrap URL %s: %v\n", url, err)
			continue
		}
		ns = append(ns, n)
	}
	return ns
}

// parseNodes parses a list of discovery node URLs loaded from a .json file.
func (cfg *Config) parseNodes(file string) []*discover.Node {
	// Short circuit if no node config is present
	path := filepath.Join(cfg.DataDir, file)
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	// Load the nodes from the config file
	blob, err := ioutil.ReadFile(path)
	if err != nil {
		glog.V(logger.Error).Infof("Failed to access nodes: %v", err)
		return nil
	}
	nodelist := []string{}
	if err := json.Unmarshal(blob, &nodelist); err != nil {
		glog.V(logger.Error).Infof("Failed to load nodes: %v", err)
		return nil
	}
	// Interpret the list as a discovery node array
	var nodes []*discover.Node
	for _, url := range nodelist {
		if url == "" {
			continue
		}
		node, err := discover.ParseNode(url)
		if err != nil {
			glog.V(logger.Error).Infof("Node URL %s: %v\n", url, err)
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func (cfg *Config) nodeKey() (*ecdsa.PrivateKey, error) {
	// use explicit key from command line args if set
	if cfg.NodeKey != nil {
		return cfg.NodeKey, nil
	}
	// use persistent key if present
	keyfile := filepath.Join(cfg.DataDir, "nodekey")
	key, err := crypto.LoadECDSA(keyfile)
	if err == nil {
		return key, nil
	}
	// no persistent key, generate and store a new one
	if key, err = crypto.GenerateKey(); err != nil {
		return nil, fmt.Errorf("could not generate server key: %v", err)
	}
	if err := crypto.SaveECDSA(keyfile, key); err != nil {
		glog.V(logger.Error).Infoln("could not persist nodekey: ", err)
	}
	return key, nil
}

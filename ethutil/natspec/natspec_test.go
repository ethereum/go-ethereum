package natspec

import (
	"flag"
	//	"crypto/rand"
	//	"io/ioutil"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/eth"
	"testing"
)

const (
	ClientIdentifier = "Ethereum(G)"
	Version          = "0.8.1"
)

var (
	Identifier      string
	KeyRing         string
	DiffTool        bool
	DiffType        string
	KeyStore        string
	StartRpc        bool
	StartWebSockets bool
	RpcPort         int
	NatType         string
	PMPGateway      string
	OutboundPort    string
	ShowGenesis     bool
	AddPeer         string
	MaxPeer         int
	GenAddr         bool
	UseSeed         bool
	SecretFile      string
	ExportDir       string
	NonInteractive  bool
	Datadir         string
	LogFile         string
	ConfigFile      string
	DebugFile       string
	LogLevel        int
	Dump            bool
	DumpHash        string
	DumpNumber      int
	VmType          int
	ImportChain     string
	SHH             bool
	Dial            bool
	PrintVersion    bool
)

func Init() {
	/*	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s [options] [filename]:\noptions precedence: default < config file < environment variables < command line\n", os.Args[0])
		flag.PrintDefaults()
	}*/

	flag.IntVar(&VmType, "vm", 0, "Virtual Machine type: 0-1: standard, debug")
	flag.StringVar(&Identifier, "id", "", "Custom client identifier")
	flag.StringVar(&KeyRing, "keyring", "", "identifier for keyring to use")
	flag.StringVar(&KeyStore, "keystore", "db", "system to store keyrings: db|file (db)")
	flag.StringVar(&OutboundPort, "port", "30303", "listening port")
	flag.StringVar(&NatType, "nat", "", "NAT support (UPNP|PMP) (none)")
	flag.StringVar(&PMPGateway, "pmp", "", "Gateway IP for PMP")
	flag.IntVar(&MaxPeer, "maxpeer", 30, "maximum desired peers")
	flag.IntVar(&RpcPort, "rpcport", 8080, "port to start json-rpc server on")
	flag.BoolVar(&StartRpc, "rpc", false, "start rpc server")
	flag.BoolVar(&StartWebSockets, "ws", false, "start websocket server")
	flag.BoolVar(&NonInteractive, "y", false, "non-interactive mode (say yes to confirmations)")
	flag.BoolVar(&UseSeed, "seed", true, "seed peers")
	flag.BoolVar(&SHH, "shh", true, "whisper protocol (on)")
	flag.BoolVar(&Dial, "dial", true, "dial out connections (on)")
	flag.BoolVar(&GenAddr, "genaddr", false, "create a new priv/pub key")
	flag.StringVar(&SecretFile, "import", "", "imports the file given (hex or mnemonic formats)")
	flag.StringVar(&ExportDir, "export", "", "exports the session keyring to files in the directory given")
	flag.StringVar(&LogFile, "logfile", "", "log file (defaults to standard output)")
	flag.StringVar(&Datadir, "datadir", "", "specifies the datadir to use")
	flag.StringVar(&ConfigFile, "conf", "", "config file")
	flag.StringVar(&DebugFile, "debug", "", "debug file (no debugging if not set)")
	flag.IntVar(&LogLevel, "loglevel", 0, "loglevel: 0-5: silent,error,warn,info,debug,debug detail)")
	flag.BoolVar(&DiffTool, "difftool", false, "creates output for diff'ing. Sets LogLevel=0")
	flag.StringVar(&DiffType, "diff", "all", "sets the level of diff output [vm, all]. Has no effect if difftool=false")
	flag.BoolVar(&ShowGenesis, "genesis", false, "Dump the genesis block")
	flag.StringVar(&ImportChain, "chain", "", "Imports given chain")

	flag.BoolVar(&Dump, "dump", false, "output the ethereum state in JSON format. Sub args [number, hash]")
	flag.StringVar(&DumpHash, "hash", "", "specify arg in hex")
	flag.IntVar(&DumpNumber, "number", -1, "specify arg in number")

	/*	flag.BoolVar(&StartMining, "mine", false, "start dagger mining")
		flag.BoolVar(&StartJsConsole, "js", false, "launches javascript console")
		flag.BoolVar(&PrintVersion, "version", false, "prints version number")*/

	flag.Parse()

}

func TestNotice(t *testing.T) {

	Init()

	utils.InitConfig(VmType, ConfigFile, Datadir, "ETH")

	ethereum, _ := eth.New(&eth.Config{
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
		Shh:        SHH,
		Dial:       Dial,
	})

	ns, err := NewNATSpec(ethereum, `
	{
            "jsonrpc": "2.0",
            "method": "eth_call",
            "params": [{
                "to": "0x8521742d3f456bd237e312d6e30724960f72517a",
                "data": "0xc6888fa1000000000000000000000000000000000000000000000000000000000000007a"
            }],
            "id": 6
        }
	`)

	if err != nil {
		t.Errorf("NewNATSpec error %v", err)
	}

	ns.SetABI(`
	[{
            "name": "multiply",
            "constant": false,
            "type": "function",
            "inputs": [{
                "name": "a",
                "type": "uint256"
            }],
            "outputs": [{
                "name": "d",
                "type": "uint256"
            }]
        }]
	`)
	ns.SetDescription("Will multiply `a` by 7 and return `a * 7`.")
	ns.SetMethod("multiply")

	notice := ns.Parse()

	expected := "Will multiply 122 by 7 and return 854."
	if notice != expected {
		t.Errorf("incorrect notice. expected %v, got %v", expected, notice)
	} else {
		t.Logf("returned notice \"%v\"", notice)
	}
}

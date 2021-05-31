package main

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/naoina/toml"
)

// defaultMainnetConfig - default config for csc mainnet
const defaultMainnetConfig = `[Eth]
NetworkId = 52
SyncMode = "fast"
NoPruning = false
NoPrefetch = false
LightPeers = 100
UltraLightFraction = 75
DatabaseCache = 512
DatabaseFreezer = ""
TrieCleanCache = 256
TrieDirtyCache = 256
TrieTimeout = 100000000000
EnablePreimageRecording = false
EWASMInterpreter = ""
EVMInterpreter = ""

[Eth.Miner]
GasFloor = 30000000
GasCeil = 42000000
GasPrice = 100000000000
Recommit = 10000000000
Noverify = false

[Eth.TxPool]
Locals = []
NoLocals = true
Journal = "transactions.rlp"
Rejournal = 3600000000000
PriceLimit = 100000000000
PriceBump = 10
AccountSlots = 16
GlobalSlots = 4096
AccountQueue = 64
GlobalQueue = 1024
Lifetime = 10800000000000

[Eth.GPO]
Blocks = 20
Percentile = 60

[Node]
IPCPath = "cetd.ipc"
HTTPHost = "localhost"
NoUSB = true
InsecureUnlockAllowed = false
HTTPPort = 8545
HTTPVirtualHosts = ["localhost"]
HTTPModules = ["eth", "net", "web3", "txpool", "senatus"]
WSPort = 8546
WSModules = ["net", "web3", "eth"]

[Node.P2P]
MaxPeers = 200
NoDiscovery = false
StaticNodes = []
TrustedNodes = []
ListenAddr = ":36652"
EnableMsgEvents = false

[Node.HTTPTimeouts]
ReadTimeout = 30000000000
WriteTimeout = 30000000000
IdleTimeout = 120000000000`

// defaultTestnetConfig - default config for csc testnet
const defaultTestnetConfig = `[Eth]
NetworkId = 53
SyncMode = "fast"
NoPruning = false
NoPrefetch = false
LightPeers = 100
UltraLightFraction = 75
DatabaseCache = 512
DatabaseFreezer = ""
TrieCleanCache = 256
TrieDirtyCache = 256
TrieTimeout = 100000000000
EnablePreimageRecording = false
EWASMInterpreter = ""
EVMInterpreter = ""

[Eth.Miner]
GasFloor = 30000000
GasCeil = 42000000
GasPrice = 100000000000
Recommit = 10000000000
Noverify = false

[Eth.TxPool]
Locals = []
NoLocals = true
Journal = "transactions.rlp"
Rejournal = 3600000000000
PriceLimit = 100000000000
PriceBump = 10
AccountSlots = 16
GlobalSlots = 4096
AccountQueue = 64
GlobalQueue = 1024
Lifetime = 10800000000000

[Eth.GPO]
Blocks = 20
Percentile = 60

[Node]
IPCPath = "cetd.ipc"
HTTPHost = "0.0.0.0"
NoUSB = true
InsecureUnlockAllowed = false
HTTPPort = 8545
HTTPVirtualHosts = ["localhost"]
HTTPModules = ["eth", "net", "web3", "txpool", "senatus"]
WSPort = 8546
WSModules = ["net", "web3", "eth"]

[Node.P2P]
MaxPeers = 200
NoDiscovery = false
StaticNodes = ["enode://6d97c62365495e706739822bf231bc4b13ad66ca0a5664965d437e40087c6c76f2cedf1286fffbcec2fc1500aa2634c70a26b2c7408c85081578ab85069b919f@47.242.178.212:36653", "enode://b858216d3c626dcc83ce6c9169d243cd8ebadd0dcdb67cdba5d63c4b6d6989c0a8fdf2278d5b68e20cc8eeefa8eb58cf4d5bb0c3dda3cbfae3e42586eb6897bb@47.242.181.109:36653"]
TrustedNodes = []
ListenAddr = ":36653"
EnableMsgEvents = false

[Node.HTTPTimeouts]
ReadTimeout = 30000000000
WriteTimeout = 30000000000
IdleTimeout = 120000000000`

// loadDefaultConfig - load default config for csc
func loadDefaultConfig(cfg *gethConfig, isTestnet bool) error {
	if isTestnet {
		log.Trace("load testnet default config")
		return toml.Unmarshal([]byte(defaultTestnetConfig), cfg)
	}
	log.Trace("load mainnet default config")
	return toml.Unmarshal([]byte(defaultMainnetConfig), cfg)
}

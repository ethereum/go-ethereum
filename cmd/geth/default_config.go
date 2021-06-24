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
GasPrice = 500000000000
Recommit = 10000000000
Noverify = false

[Eth.TxPool]
Locals = []
NoLocals = true
Journal = "transactions.rlp"
Rejournal = 3600000000000
PriceLimit = 500000000000
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
WSModules = ["eth", "net", "web3", "txpool", "senatus"]

[Node.P2P]
MaxPeers = 200
NoDiscovery = false
StaticNodes = ["enode://7fbf4f0f14a808aab87d8cab90707e008fb3664da36c46904f822b365c9a59b13d153b20d574d5d3a3a7ab8f4a1fa42c8c83eb0cbf628acde04b1e05fa749a47@47.243.93.185:36652", "enode://2121b7393c1273acfc614243b0e249a116115398831e29fe662efa3f3eaa21a072c060bf44fb6626682a21558c9abac1e628d6dbc1396ceec33868b822e20cbe@47.253.82.152:36652", "enode://c09c2f0e01c251871a65ed39f7892c675bf032ba4d0472f78bddd64dfb048a9b2be4ffc0520573abc42055fd7c0d9f8b00b9981d323663a4d7c63e3e567603c7@8.209.70.23:36652"]
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
GasPrice = 500000000000
Recommit = 10000000000
Noverify = false

[Eth.TxPool]
Locals = []
NoLocals = true
Journal = "transactions.rlp"
Rejournal = 3600000000000
PriceLimit = 500000000000
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
WSModules = ["eth", "net", "web3", "txpool", "senatus"]

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

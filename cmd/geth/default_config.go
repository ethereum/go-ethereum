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
StaticNodes = ["enode://f565ad6606390a5e265797e3422d3c53b5a177b57e36d76d2d5f68b2231fef9c2f1c7bd7db6a73012cdfd7b58d0b19c26dc390b766e9489192492398482cfcf1@47.243.103.171:36652", "enode://f624c9deeff1c851966943aaab540a54453302d712cc74c042aa9cef7e108c223a47c147f3dba6d7abe0eac7a7a326c2d947b7e680e70d0e76f80e462048e023@47.242.201.219:36652", "enode://7f431a631eeb26ab603a3d23aca306b0cc9eadd9f58f6850010fd94fa0665bb89b2be45d85ce5d5f2dc5c3547ee1ecaacb68c92a3bef22c0cf56bbf395af5879@8.210.97.29:36652", "enode://c09c2f0e01c251871a65ed39f7892c675bf032ba4d0472f78bddd64dfb048a9b2be4ffc0520573abc42055fd7c0d9f8b00b9981d323663a4d7c63e3e567603c7@47.243.95.151:36652"]
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

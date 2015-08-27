package rpc

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/rpc/comms"
)

type GenApi struct {
	Admin    *Admin
	Db       *Db
	Debug    *Debug
	Eth      *Eth
	Miner    *Miner
	Net      *Net
	Personal *Personal
	Shh      *Shh
	Txpool   *Txpool
	Web3     *Web3
}

func NewGenApi(client comms.EthereumClient) *GenApi {
	xeth := NewXeth(client)

	return &GenApi{
		Admin:    &Admin{xeth},
		Db:       &Db{xeth},
		Debug:    &Debug{xeth},
		Eth:      &Eth{xeth},
		Miner:    &Miner{xeth},
		Net:      &Net{xeth},
		Personal: &Personal{xeth},
		Shh:      &Shh{xeth},
		Txpool:   &Txpool{xeth},
		Web3:     &Web3{xeth},
	}
}

type Admin struct {
	xeth *Xeth
}

func (self *Admin) AddPeer(url string) (interface{}, error) {
	return self.xeth.Call("admin_addPeer", []interface{}{url})
}
func (self *Admin) ChainSyncStatus() (interface{}, error) {
	return self.xeth.Call("admin_chainSyncStatus", nil)
}
func (self *Admin) Datadir() (interface{}, error) {
	return self.xeth.Call("admin_datadir", nil)
}
func (self *Admin) EnableUserAgent() (interface{}, error) {
	return self.xeth.Call("admin_enableUserAgent", nil)
}
func (self *Admin) ExportChain() (interface{}, error) {
	return self.xeth.Call("admin_exportChain", nil)
}
func (self *Admin) GetContractInfo(contract string) (interface{}, error) {
	return self.xeth.Call("admin_getContractInfo", []interface{}{contract})
}
func (self *Admin) HttpGet(uri string, path string) (interface{}, error) {
	return self.xeth.Call("admin_httpGet", []interface{}{uri, path})
}
func (self *Admin) ImportChain() (interface{}, error) {
	return self.xeth.Call("admin_importChain", nil)
}
func (self *Admin) NodeInfo() (interface{}, error) {
	return self.xeth.Call("admin_nodeInfo", nil)
}
func (self *Admin) Peers() (interface{}, error) {
	return self.xeth.Call("admin_peers", nil)
}
func (self *Admin) Register(sender string, address string, contentHashHex string) (interface{}, error) {
	return self.xeth.Call("admin_register", []interface{}{sender, address, contentHashHex})
}
func (self *Admin) RegisterUrl(sender string, contentHash string, url string) (interface{}, error) {
	return self.xeth.Call("admin_registerUrl", []interface{}{sender, contentHash, url})
}
func (self *Admin) SaveInfo(contractInfo compiler.ContractInfo, filename string) (interface{}, error) {
	return self.xeth.Call("admin_saveInfo", []interface{}{contractInfo, filename})
}
func (self *Admin) SetGlobalRegistrar(nameReg string, contractAddress string) (interface{}, error) {
	return self.xeth.Call("admin_setGlobalRegistrar", []interface{}{nameReg, contractAddress})
}
func (self *Admin) SetHashReg(hashReg string, sender string) (interface{}, error) {
	return self.xeth.Call("admin_setHashReg", []interface{}{hashReg, sender})
}
func (self *Admin) SetSolc(path string) (interface{}, error) {
	return self.xeth.Call("admin_setSolc", []interface{}{path})
}
func (self *Admin) SetUrlHint(urlHint string, sender string) (interface{}, error) {
	return self.xeth.Call("admin_setUrlHint", []interface{}{urlHint, sender})
}
func (self *Admin) Sleep(s int) (interface{}, error) {
	return self.xeth.Call("admin_sleep", []interface{}{s})
}
func (self *Admin) SleepBlocks(n int64, timeout int64) (interface{}, error) {
	return self.xeth.Call("admin_sleepBlocks", []interface{}{n, timeout})
}
func (self *Admin) StartNatSpec() (interface{}, error) {
	return self.xeth.Call("admin_startNatSpec", nil)
}
func (self *Admin) StartRPC(listenAddress string, listenPort uint, corsDomain string, apis string) (interface{}, error) {
	return self.xeth.Call("admin_startRPC", []interface{}{listenAddress, listenPort, corsDomain, apis})
}
func (self *Admin) StopNatSpec() (interface{}, error) {
	return self.xeth.Call("admin_stopNatSpec", nil)
}
func (self *Admin) StopRPC() (interface{}, error) {
	return self.xeth.Call("admin_stopRPC", nil)
}
func (self *Admin) Verbosity(level int) (interface{}, error) {
	return self.xeth.Call("admin_verbosity", []interface{}{level})
}

type Db struct {
	xeth *Xeth
}

func (self *Db) GetHex() (interface{}, error) {
	return self.xeth.Call("db_getHex", nil)
}
func (self *Db) GetString() (interface{}, error) {
	return self.xeth.Call("db_getString", nil)
}
func (self *Db) PutHex() (interface{}, error) {
	return self.xeth.Call("db_putHex", nil)
}
func (self *Db) PutString() (interface{}, error) {
	return self.xeth.Call("db_putString", nil)
}

type Debug struct {
	xeth *Xeth
}

func (self *Debug) DumpBlock() (interface{}, error) {
	return self.xeth.Call("debug_dumpBlock", nil)
}
func (self *Debug) GetBlockRlp() (interface{}, error) {
	return self.xeth.Call("debug_getBlockRlp", nil)
}
func (self *Debug) Metrics(raw bool) (interface{}, error) {
	return self.xeth.Call("debug_metrics", []interface{}{raw})
}
func (self *Debug) PrintBlock() (interface{}, error) {
	return self.xeth.Call("debug_printBlock", nil)
}
func (self *Debug) ProcessBlock() (interface{}, error) {
	return self.xeth.Call("debug_processBlock", nil)
}
func (self *Debug) SeedHash() (interface{}, error) {
	return self.xeth.Call("debug_seedHash", nil)
}
func (self *Debug) SetHead() (interface{}, error) {
	return self.xeth.Call("debug_setHead", nil)
}

type Eth struct {
	xeth *Xeth
}

func (self *Eth) Accounts() (interface{}, error) {
	return self.xeth.Call("eth_accounts", nil)
}
func (self *Eth) BlockNumber() (interface{}, error) {
	return self.xeth.Call("eth_blockNumber", nil)
}
func (self *Eth) Call(from string, to string, value *big.Int, gas *big.Int, gasPrice *big.Int, data string, blockNumber int64) (interface{}, error) {
	return self.xeth.Call("eth_call", []interface{}{from, to, value, gas, gasPrice, data, blockNumber})
}
func (self *Eth) Coinbase() (interface{}, error) {
	return self.xeth.Call("eth_coinbase", nil)
}
func (self *Eth) CompileSolidity() (interface{}, error) {
	return self.xeth.Call("eth_compileSolidity", nil)
}
func (self *Eth) EstimateGas() (interface{}, error) {
	return self.xeth.Call("eth_estimateGas", nil)
}
func (self *Eth) Flush() (interface{}, error) {
	return self.xeth.Call("eth_flush", nil)
}
func (self *Eth) GasPrice(price string) (interface{}, error) {
	return self.xeth.Call("eth_gasPrice", []interface{}{price})
}
func (self *Eth) GetBalance(address string, blockNumber int64) (interface{}, error) {
	return self.xeth.Call("eth_getBalance", []interface{}{address, blockNumber})
}
func (self *Eth) GetBlockByHash(blockHash string, includeTxs bool) (interface{}, error) {
	return self.xeth.Call("eth_getBlockByHash", []interface{}{blockHash, includeTxs})
}
func (self *Eth) GetBlockByNumber(blockNumber int64, includeTxs bool) (interface{}, error) {
	return self.xeth.Call("eth_getBlockByNumber", []interface{}{blockNumber, includeTxs})
}
func (self *Eth) GetBlockTransactionCountByHash() (interface{}, error) {
	return self.xeth.Call("eth_getBlockTransactionCountByHash", nil)
}
func (self *Eth) GetBlockTransactionCountByNumber() (interface{}, error) {
	return self.xeth.Call("eth_getBlockTransactionCountByNumber", nil)
}
func (self *Eth) GetCode(address string, blockNumber int64) (interface{}, error) {
	return self.xeth.Call("eth_getCode", []interface{}{address, blockNumber})
}
func (self *Eth) GetCompilers() (interface{}, error) {
	return self.xeth.Call("eth_getCompilers", nil)
}
func (self *Eth) GetData(address string, blockNumber int64) (interface{}, error) {
	return self.xeth.Call("eth_getData", []interface{}{address, blockNumber})
}
func (self *Eth) GetFilterChanges() (interface{}, error) {
	return self.xeth.Call("eth_getFilterChanges", nil)
}
func (self *Eth) GetFilterLogs() (interface{}, error) {
	return self.xeth.Call("eth_getFilterLogs", nil)
}
func (self *Eth) GetLogs() (interface{}, error) {
	return self.xeth.Call("eth_getLogs", nil)
}
func (self *Eth) GetStorage(address string, blockNumber int64) (interface{}, error) {
	return self.xeth.Call("eth_getStorage", []interface{}{address, blockNumber})
}
func (self *Eth) GetStorageAt(address string, blockNumber int64, key string) (interface{}, error) {
	return self.xeth.Call("eth_getStorageAt", []interface{}{address, blockNumber, key})
}
func (self *Eth) GetTransactionByBlockHashAndIndex() (interface{}, error) {
	return self.xeth.Call("eth_getTransactionByBlockHashAndIndex", nil)
}
func (self *Eth) GetTransactionByBlockNumberAndIndex() (interface{}, error) {
	return self.xeth.Call("eth_getTransactionByBlockNumberAndIndex", nil)
}
func (self *Eth) GetTransactionByHash() (interface{}, error) {
	return self.xeth.Call("eth_getTransactionByHash", nil)
}
func (self *Eth) GetTransactionCount() (interface{}, error) {
	return self.xeth.Call("eth_getTransactionCount", nil)
}
func (self *Eth) GetTransactionReceipt() (interface{}, error) {
	return self.xeth.Call("eth_getTransactionReceipt", nil)
}
func (self *Eth) GetUncleByBlockHashAndIndex() (interface{}, error) {
	return self.xeth.Call("eth_getUncleByBlockHashAndIndex", nil)
}
func (self *Eth) GetUncleByBlockNumberAndIndex() (interface{}, error) {
	return self.xeth.Call("eth_getUncleByBlockNumberAndIndex", nil)
}
func (self *Eth) GetUncleCountByBlockHash() (interface{}, error) {
	return self.xeth.Call("eth_getUncleCountByBlockHash", nil)
}
func (self *Eth) GetUncleCountByBlockNumber() (interface{}, error) {
	return self.xeth.Call("eth_getUncleCountByBlockNumber", nil)
}
func (self *Eth) GetWork() (interface{}, error) {
	return self.xeth.Call("eth_getWork", nil)
}
func (self *Eth) Hashrate() (interface{}, error) {
	return self.xeth.Call("eth_hashrate", nil)
}
func (self *Eth) Mining() (interface{}, error) {
	return self.xeth.Call("eth_mining", nil)
}
func (self *Eth) NewBlockFilter() (interface{}, error) {
	return self.xeth.Call("eth_newBlockFilter", nil)
}
func (self *Eth) NewFilter() (interface{}, error) {
	return self.xeth.Call("eth_newFilter", nil)
}
func (self *Eth) NewPendingTransactionFilter() (interface{}, error) {
	return self.xeth.Call("eth_newPendingTransactionFilter", nil)
}
func (self *Eth) PendingTransactions() (interface{}, error) {
	return self.xeth.Call("eth_pendingTransactions", nil)
}
func (self *Eth) ProtocolVersion() (interface{}, error) {
	return self.xeth.Call("eth_protocolVersion", nil)
}
func (self *Eth) SendRawTransaction() (interface{}, error) {
	return self.xeth.Call("eth_sendRawTransaction", nil)
}
func (self *Eth) SendTransaction() (interface{}, error) {
	return self.xeth.Call("eth_sendTransaction", nil)
}
func (self *Eth) Sign() (interface{}, error) {
	return self.xeth.Call("eth_sign", nil)
}
func (self *Eth) StorageAt(address string, blockNumber int64) (interface{}, error) {
	return self.xeth.Call("eth_storageAt", []interface{}{address, blockNumber})
}
func (self *Eth) SubmitHashrate() (interface{}, error) {
	return self.xeth.Call("eth_submitHashrate", nil)
}
func (self *Eth) SubmitWork(nonce uint64, header string, digest string) (interface{}, error) {
	return self.xeth.Call("eth_submitWork", []interface{}{nonce, header, digest})
}
func (self *Eth) Transact() (interface{}, error) {
	return self.xeth.Call("eth_transact", nil)
}
func (self *Eth) UninstallFilter() (interface{}, error) {
	return self.xeth.Call("eth_uninstallFilter", nil)
}

type Miner struct {
	xeth *Xeth
}

func (self *Miner) Hashrate() (interface{}, error) {
	return self.xeth.Call("miner_hashrate", nil)
}
func (self *Miner) MakeDAG(blockNumber int64) (interface{}, error) {
	return self.xeth.Call("miner_makeDAG", []interface{}{blockNumber})
}
func (self *Miner) SetEtherbase(etherbase common.Address) (interface{}, error) {
	return self.xeth.Call("miner_setEtherbase", []interface{}{etherbase})
}
func (self *Miner) SetExtra(data string) (interface{}, error) {
	return self.xeth.Call("miner_setExtra", []interface{}{data})
}
func (self *Miner) SetGasPrice() (interface{}, error) {
	return self.xeth.Call("miner_setGasPrice", nil)
}
func (self *Miner) Start(threads int) (interface{}, error) {
	return self.xeth.Call("miner_start", []interface{}{threads})
}
func (self *Miner) StartAutoDAG() (interface{}, error) {
	return self.xeth.Call("miner_startAutoDAG", nil)
}
func (self *Miner) Stop() (interface{}, error) {
	return self.xeth.Call("miner_stop", nil)
}
func (self *Miner) StopAutoDAG() (interface{}, error) {
	return self.xeth.Call("miner_stopAutoDAG", nil)
}

type Net struct {
	xeth *Xeth
}

func (self *Net) Listening() (interface{}, error) {
	return self.xeth.Call("net_listening", nil)
}
func (self *Net) PeerCount() (interface{}, error) {
	return self.xeth.Call("net_peerCount", nil)
}
func (self *Net) Version() (interface{}, error) {
	return self.xeth.Call("net_version", nil)
}

type Personal struct {
	xeth *Xeth
}

func (self *Personal) ListAccounts() (interface{}, error) {
	return self.xeth.Call("personal_listAccounts", nil)
}
func (self *Personal) NewAccount(passphrase string) (interface{}, error) {
	return self.xeth.Call("personal_newAccount", []interface{}{passphrase})
}
func (self *Personal) UnlockAccount(address string, passphrase string, duration int) (interface{}, error) {
	return self.xeth.Call("personal_unlockAccount", []interface{}{address, passphrase, duration})
}

type Shh struct {
	xeth *Xeth
}

func (self *Shh) GetFilterChanges() (interface{}, error) {
	return self.xeth.Call("shh_getFilterChanges", nil)
}
func (self *Shh) GetMessages() (interface{}, error) {
	return self.xeth.Call("shh_getMessages", nil)
}
func (self *Shh) HasIdentity() (interface{}, error) {
	return self.xeth.Call("shh_hasIdentity", nil)
}
func (self *Shh) NewFilter() (interface{}, error) {
	return self.xeth.Call("shh_newFilter", nil)
}
func (self *Shh) NewIdentity() (interface{}, error) {
	return self.xeth.Call("shh_newIdentity", nil)
}
func (self *Shh) Post() (interface{}, error) {
	return self.xeth.Call("shh_post", nil)
}
func (self *Shh) UninstallFilter() (interface{}, error) {
	return self.xeth.Call("shh_uninstallFilter", nil)
}
func (self *Shh) Version() (interface{}, error) {
	return self.xeth.Call("shh_version", nil)
}

type Txpool struct {
	xeth *Xeth
}

func (self *Txpool) Status() (interface{}, error) {
	return self.xeth.Call("txpool_status", nil)
}

type Web3 struct {
	xeth *Xeth
}

func (self *Web3) ClientVersion() (interface{}, error) {
	return self.xeth.Call("web3_clientVersion", nil)
}
func (self *Web3) Sha3(data string) (interface{}, error) {
	return self.xeth.Call("web3_sha3", []interface{}{data})
}

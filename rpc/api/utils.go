package api

import (
	"strings"

	"fmt"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
)

var (
	// Mapping between the different methods each api supports
	AutoCompletion = map[string][]string{
		"admin": []string{
			"addPeer",
			"peers",
			"nodeInfo",
			"exportChain",
			"importChain",
			"verbosity",
			"chainSyncStatus",
			"setSolc",
			"datadir",
			"startRPC",
			"stopRPC",
		},
		"db": []string{
			"getString",
			"putString",
			"getHex",
			"putHex",
		},
		"debug": []string{
			"dumpBlock",
			"getBlockRlp",
			"printBlock",
			"processBlock",
			"seedHash",
			"setHead",
		},
		"eth": []string{
			"accounts",
			"blockNumber",
			"getBalance",
			"protocolVersion",
			"coinbase",
			"mining",
			"gasPrice",
			"getStorage",
			"storageAt",
			"getStorageAt",
			"getTransactionCount",
			"getBlockTransactionCountByHash",
			"getBlockTransactionCountByNumber",
			"getUncleCountByBlockHash",
			"getUncleCountByBlockNumber",
			"getData",
			"getCode",
			"sign",
			"sendRawTransaction",
			"sendTransaction",
			"transact",
			"estimateGas",
			"call",
			"flush",
			"getBlockByHash",
			"getBlockByNumber",
			"getTransactionByHash",
			"getTransactionByBlockHashAndIndex",
			"getUncleByBlockHashAndIndex",
			"getUncleByBlockNumberAndIndex",
			"getCompilers",
			"compileSolidity",
			"newFilter",
			"newBlockFilter",
			"newPendingTransactionFilter",
			"uninstallFilter",
			"getFilterChanges",
			"getFilterLogs",
			"getLogs",
			"hashrate",
			"getWork",
			"submitWork",
			"pendingTransactions",
			"resend",
		},
		"miner": []string{
			"hashrate",
			"makeDAG",
			"setExtra",
			"setGasPrice",
			"startAutoDAG",
			"start",
			"stopAutoDAG",
			"stop",
		},
		"net": []string{
			"peerCount",
			"listening",
		},
		"personal": []string{
			"listAccounts",
			"newAccount",
			"deleteAccount",
			"unlockAccount",
		},
		"shh": []string{
			"version",
			"post",
			"hasIdentity",
			"newIdentity",
			"newFilter",
			"uninstallFilter",
			"getFilterChanges",
		},
		"txpool": []string{
			"status",
		},
		"web3": []string{
			"sha3",
			"version",
			"fromWei",
			"toWei",
			"toHex",
			"toAscii",
			"fromAscii",
			"toBigNumber",
			"isAddress",
		},
	}
)

// Parse a comma separated API string to individual api's
func ParseApiString(apistr string, codec codec.Codec, xeth *xeth.XEth, eth *eth.Ethereum) ([]shared.EthereumApi, error) {
	if len(strings.TrimSpace(apistr)) == 0 {
		return nil, fmt.Errorf("Empty apistr provided")
	}

	names := strings.Split(apistr, ",")
	apis := make([]shared.EthereumApi, len(names))

	for i, name := range names {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case shared.AdminApiName:
			apis[i] = NewAdminApi(xeth, eth, codec)
		case shared.DebugApiName:
			apis[i] = NewDebugApi(xeth, eth, codec)
		case shared.DbApiName:
			apis[i] = NewDbApi(xeth, eth, codec)
		case shared.EthApiName:
			apis[i] = NewEthApi(xeth, eth, codec)
		case shared.MinerApiName:
			apis[i] = NewMinerApi(eth, codec)
		case shared.NetApiName:
			apis[i] = NewNetApi(xeth, eth, codec)
		case shared.ShhApiName:
			apis[i] = NewShhApi(xeth, eth, codec)
		case shared.TxPoolApiName:
			apis[i] = NewTxPoolApi(xeth, eth, codec)
		case shared.PersonalApiName:
			apis[i] = NewPersonalApi(xeth, eth, codec)
		case shared.Web3ApiName:
			apis[i] = NewWeb3Api(xeth, codec)
		default:
			return nil, fmt.Errorf("Unknown API '%s'", name)
		}
	}

	return apis, nil
}

func Javascript(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case shared.AdminApiName:
		return Admin_JS
	case shared.DebugApiName:
		return Debug_JS
	case shared.DbApiName:
		return Db_JS
	case shared.EthApiName:
		return Eth_JS
	case shared.MinerApiName:
		return Miner_JS
	case shared.NetApiName:
		return Net_JS
	case shared.ShhApiName:
		return Shh_JS
	case shared.TxPoolApiName:
		return TxPool_JS
	case shared.PersonalApiName:
		return Personal_JS
	}

	return ""
}

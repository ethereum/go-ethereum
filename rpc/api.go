package rpc

import (
	"encoding/json"
	"math/big"
	"path"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/xeth"
)

type EthereumApi struct {
	eth    *xeth.XEth
	xethMu sync.RWMutex
	db     common.Database

	// Miner agent
	agent *Agent
}

func NewEthereumApi(eth *xeth.XEth, dataDir string) *EthereumApi {
	// What about when dataDir is empty?
	db, _ := ethdb.NewLDBDatabase(path.Join(dataDir, "dapps"))
	api := &EthereumApi{
		eth:   eth,
		db:    db,
		agent: NewAgent(),
	}
	eth.Backend().Miner().Register(api.agent)

	return api
}

func (self *EthereumApi) xeth() *xeth.XEth {
	self.xethMu.RLock()
	defer self.xethMu.RUnlock()

	return self.eth
}

func (self *EthereumApi) xethAtStateNum(num int64) *xeth.XEth {
	return self.xeth().AtStateNum(num)
}

func (p *EthereumApi) GetRequestReply(req *RpcRequest, reply *interface{}) error {
	// Spec at https://github.com/ethereum/wiki/wiki/Generic-JSON-RPC
	rpclogger.Debugf("%s %s", req.Method, req.Params)

	switch req.Method {
	case "web3_sha3":
		args := new(Sha3Args)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = common.ToHex(crypto.Sha3(common.FromHex(args.Data)))
	case "web3_clientVersion":
		*reply = p.xeth().Backend().Version()
	case "net_version":
		return NewNotImplementedError(req.Method)
	case "net_listening":
		*reply = p.xeth().IsListening()
	case "net_peerCount":
		v := p.xeth().PeerCount()
		*reply = common.ToHex(big.NewInt(int64(v)).Bytes())
	case "eth_coinbase":
		// TODO handling of empty coinbase due to lack of accounts
		res := p.xeth().Coinbase()
		if res == "0x" || res == "0x0" {
			*reply = nil
		} else {
			*reply = res
		}
	case "eth_mining":
		*reply = p.xeth().IsMining()
	case "eth_gasPrice":
		v := p.xeth().DefaultGas()
		*reply = common.ToHex(v.Bytes())
	case "eth_accounts":
		*reply = p.xeth().Accounts()
	case "eth_blockNumber":
		v := p.xeth().Backend().ChainManager().CurrentBlock().Number()
		*reply = common.ToHex(v.Bytes())
	case "eth_getBalance":
		args := new(GetBalanceArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		if err := args.requirements(); err != nil {
			return err
		}

		v := p.xethAtStateNum(args.BlockNumber).State().SafeGet(args.Address).Balance()
		*reply = common.ToHex(v.Bytes())
	case "eth_getStorage", "eth_storageAt":
		args := new(GetStorageArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		if err := args.requirements(); err != nil {
			return err
		}

		*reply = p.xethAtStateNum(args.BlockNumber).State().SafeGet(args.Address).Storage()
	case "eth_getStorageAt":
		args := new(GetStorageAtArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		if err := args.requirements(); err != nil {
			return err
		}

		state := p.xethAtStateNum(args.BlockNumber).State().SafeGet(args.Address)
		value := state.StorageString(args.Key)

		*reply = common.Bytes2Hex(value.Bytes())
	case "eth_getTransactionCount":
		args := new(GetTxCountArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		err := args.requirements()
		if err != nil {
			return err
		}

		*reply = p.xethAtStateNum(args.BlockNumber).TxCountAt(args.Address)
	case "eth_getBlockTransactionCountByHash":
		args := new(GetBlockByHashArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := NewBlockRes(p.xeth().EthBlockByHash(args.BlockHash))
		*reply = common.ToHex(big.NewInt(int64(len(block.Transactions))).Bytes())
	case "eth_getBlockTransactionCountByNumber":
		args := new(GetBlockByNumberArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := NewBlockRes(p.xeth().EthBlockByNumber(args.BlockNumber))
		*reply = common.ToHex(big.NewInt(int64(len(block.Transactions))).Bytes())
	case "eth_getUncleCountByBlockHash":
		args := new(GetBlockByHashArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := p.xeth().EthBlockByHash(args.BlockHash)
		br := NewBlockRes(block)
		*reply = common.ToHex(big.NewInt(int64(len(br.Uncles))).Bytes())
	case "eth_getUncleCountByBlockNumber":
		args := new(GetBlockByNumberArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := p.xeth().EthBlockByNumber(args.BlockNumber)
		br := NewBlockRes(block)
		*reply = common.ToHex(big.NewInt(int64(len(br.Uncles))).Bytes())
	case "eth_getData", "eth_getCode":
		args := new(GetDataArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		if err := args.requirements(); err != nil {
			return err
		}
		*reply = p.xethAtStateNum(args.BlockNumber).CodeAt(args.Address)
	case "eth_sendTransaction", "eth_transact":
		args := new(NewTxArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		if err := args.requirements(); err != nil {
			return err
		}

		v, err := p.xeth().Transact(args.From, args.To, args.Value.String(), args.Gas.String(), args.GasPrice.String(), args.Data)
		if err != nil {
			return err
		}
		*reply = v
	case "eth_call":
		args := new(NewTxArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		v, err := p.xethAtStateNum(args.BlockNumber).Call(args.From, args.To, args.Value.String(), args.Gas.String(), args.GasPrice.String(), args.Data)
		if err != nil {
			return err
		}

		*reply = v
	case "eth_flush":
		return NewNotImplementedError(req.Method)
	case "eth_getBlockByHash":
		args := new(GetBlockByHashArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := p.xeth().EthBlockByHash(args.BlockHash)
		br := NewBlockRes(block)
		br.fullTx = args.IncludeTxs

		*reply = br
	case "eth_getBlockByNumber":
		args := new(GetBlockByNumberArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := p.xeth().EthBlockByNumber(args.BlockNumber)
		br := NewBlockRes(block)
		br.fullTx = args.IncludeTxs

		*reply = br
	case "eth_getTransactionByHash":
		// HashIndexArgs used, but only the "Hash" part we need.
		args := new(HashIndexArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
		}
		tx := p.xeth().EthTransactionByHash(args.Hash)
		if tx != nil {
			*reply = NewTransactionRes(tx)
		}
	case "eth_getTransactionByBlockHashAndIndex":
		args := new(HashIndexArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := p.xeth().EthBlockByHash(args.Hash)
		br := NewBlockRes(block)
		br.fullTx = true

		if args.Index > int64(len(br.Transactions)) || args.Index < 0 {
			return NewValidationError("Index", "does not exist")
		}
		*reply = br.Transactions[args.Index]
	case "eth_getTransactionByBlockNumberAndIndex":
		args := new(BlockNumIndexArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := p.xeth().EthBlockByNumber(args.BlockNumber)
		v := NewBlockRes(block)
		v.fullTx = true

		if args.Index > int64(len(v.Transactions)) || args.Index < 0 {
			return NewValidationError("Index", "does not exist")
		}
		*reply = v.Transactions[args.Index]
	case "eth_getUncleByBlockHashAndIndex":
		args := new(HashIndexArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		br := NewBlockRes(p.xeth().EthBlockByHash(args.Hash))

		if args.Index > int64(len(br.Uncles)) || args.Index < 0 {
			return NewValidationError("Index", "does not exist")
		}

		uhash := br.Uncles[args.Index].Hex()
		uncle := NewBlockRes(p.xeth().EthBlockByHash(uhash))

		*reply = uncle
	case "eth_getUncleByBlockNumberAndIndex":
		args := new(BlockNumIndexArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := p.xeth().EthBlockByNumber(args.BlockNumber)
		v := NewBlockRes(block)
		v.fullTx = true

		if args.Index > int64(len(v.Uncles)) || args.Index < 0 {
			return NewValidationError("Index", "does not exist")
		}

		uhash := v.Uncles[args.Index].Hex()
		uncle := NewBlockRes(p.xeth().EthBlockByHash(uhash))

		*reply = uncle
	case "eth_getCompilers":
		c := []string{""}
		*reply = c
	case "eth_compileSolidity", "eth_compileLLL", "eth_compileSerpent":
		return NewNotImplementedError(req.Method)
	case "eth_newFilter":
		args := new(BlockFilterArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		opts := toFilterOptions(args)
		id := p.xeth().RegisterFilter(opts)
		*reply = common.ToHex(big.NewInt(int64(id)).Bytes())
	case "eth_newBlockFilter":
		args := new(FilterStringArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		if err := args.requirements(); err != nil {
			return err
		}

		id := p.xeth().NewFilterString(args.Word)
		*reply = common.ToHex(big.NewInt(int64(id)).Bytes())
	case "eth_uninstallFilter":
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = p.xeth().UninstallFilter(args.Id)
	case "eth_getFilterChanges":
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = NewLogsRes(p.xeth().FilterChanged(args.Id))
	case "eth_getFilterLogs":
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = NewLogsRes(p.xeth().Logs(args.Id))
	case "eth_getLogs":
		args := new(BlockFilterArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		opts := toFilterOptions(args)
		*reply = NewLogsRes(p.xeth().AllLogs(opts))
	case "eth_getWork":
		*reply = p.getWork()
	case "eth_submitWork":
		// TODO what is the reply here?
		// TODO what are the arguments?
		p.agent.SetResult(0, common.Hash{}, common.Hash{})

		return NewNotImplementedError(req.Method)
	case "db_putString":
		args := new(DbArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		if err := args.requirements(); err != nil {
			return err
		}

		p.db.Put([]byte(args.Database+args.Key), []byte(args.Value))
		*reply = true
	case "db_getString":
		args := new(DbArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		if err := args.requirements(); err != nil {
			return err
		}

		res, _ := p.db.Get([]byte(args.Database + args.Key))
		*reply = string(res)
	case "db_putHex", "db_getHex":
		return NewNotImplementedError(req.Method)
	case "shh_post":
		args := new(WhisperMessageArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		err := p.xeth().Whisper().Post(args.Payload, args.To, args.From, args.Topics, args.Priority, args.Ttl)
		if err != nil {
			return err
		}

		*reply = true
	case "shh_newIdentity":
		*reply = p.xeth().Whisper().NewIdentity()
	// case "shh_removeIdentity":
	// 	args := new(WhisperIdentityArgs)
	// 	if err := json.Unmarshal(req.Params, &args); err != nil {
	// 		return err
	// 	}
	// 	*reply = p.xeth().Whisper().RemoveIdentity(args.Identity)
	case "shh_hasIdentity":
		args := new(WhisperIdentityArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = p.xeth().Whisper().HasIdentity(args.Identity)
	case "shh_newGroup", "shh_addToGroup":
		return NewNotImplementedError(req.Method)
	case "shh_newFilter":
		args := new(WhisperFilterArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		opts := new(xeth.Options)
		opts.From = args.From
		opts.To = args.To
		opts.Topics = args.Topics
		id := p.xeth().NewWhisperFilter(opts)
		*reply = common.ToHex(big.NewInt(int64(id)).Bytes())
	case "shh_uninstallFilter":
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = p.xeth().UninstallWhisperFilter(args.Id)
	case "shh_getFilterChanges":
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = p.xeth().MessagesChanged(args.Id)
	case "shh_getMessages":
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = p.xeth().Whisper().Messages(args.Id)

	// case "eth_register":
	// 	// Placeholder for actual type
	// 	args := new(HashIndexArgs)
	// 	if err := json.Unmarshal(req.Params, &args); err != nil {
	// 		return err
	// 	}
	// 	*reply = p.xeth().Register(args.Hash)
	// case "eth_unregister":
	// 	args := new(HashIndexArgs)
	// 	if err := json.Unmarshal(req.Params, &args); err != nil {
	// 		return err
	// 	}
	// 	*reply = p.xeth().Unregister(args.Hash)
	// case "eth_watchTx":
	// 	args := new(HashIndexArgs)
	// 	if err := json.Unmarshal(req.Params, &args); err != nil {
	// 		return err
	// 	}
	// 	*reply = p.xeth().PullWatchTx(args.Hash)
	default:
		return NewNotImplementedError(req.Method)
	}

	rpclogger.DebugDetailf("Reply: %T %s", reply, reply)
	return nil
}

func (p *EthereumApi) getWork() string {
	p.xeth().SetMining(true)
	return p.agent.GetWork().Hex()
}

func toFilterOptions(options *BlockFilterArgs) *core.FilterOptions {
	var opts core.FilterOptions

	// Convert optional address slice/string to byte slice
	if str, ok := options.Address.(string); ok {
		opts.Address = []common.Address{common.HexToAddress(str)}
	} else if slice, ok := options.Address.([]interface{}); ok {
		bslice := make([]common.Address, len(slice))
		for i, addr := range slice {
			if saddr, ok := addr.(string); ok {
				bslice[i] = common.HexToAddress(saddr)
			}
		}
		opts.Address = bslice
	}

	opts.Earliest = options.Earliest
	opts.Latest = options.Latest

	topics := make([][]common.Hash, len(options.Topics))
	for i, topicDat := range options.Topics {
		if slice, ok := topicDat.([]interface{}); ok {
			topics[i] = make([]common.Hash, len(slice))
			for j, topic := range slice {
				topics[i][j] = common.HexToHash(topic.(string))
			}
		} else if str, ok := topicDat.(string); ok {
			topics[i] = []common.Hash{common.HexToHash(str)}
		}
	}
	opts.Topics = topics

	return &opts
}

/*
	Work() chan<- *types.Block
	SetWorkCh(chan<- Work)
	Stop()
	Start()
	Rate() uint64
*/

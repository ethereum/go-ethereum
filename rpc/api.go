package rpc

import (
	"encoding/json"
	"fmt"
	"math/big"
	"path"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/xeth"
)

var (
	defaultGasPrice = big.NewInt(150000000000)
	defaultGas      = big.NewInt(500000)
)

type EthereumApi struct {
	eth    *xeth.XEth
	xethMu sync.RWMutex
	mux    *event.TypeMux

	// // Register keeps a list of accounts and transaction data
	// regmut   sync.Mutex
	// register map[string][]*NewTxArgs

	db common.Database
}

func NewEthereumApi(eth *xeth.XEth, dataDir string) *EthereumApi {
	db, _ := ethdb.NewLDBDatabase(path.Join(dataDir, "dapps"))
	api := &EthereumApi{
		eth: eth,
		mux: eth.Backend().EventMux(),
		db:  db,
	}

	return api
}

func (self *EthereumApi) xethWithStateNum(num int64) *xeth.XEth {
	chain := self.xeth().Backend().ChainManager()
	var block *types.Block

	if num < 0 {
		num = chain.CurrentBlock().Number().Int64() + num + 1
	}
	block = chain.GetBlockByNumber(uint64(num))

	var st *state.StateDB
	if block != nil {
		st = state.New(block.Root(), self.xeth().Backend().StateDb())
	} else {
		st = chain.State()
	}
	return self.xeth().WithState(st)
}

func (self *EthereumApi) getStateWithNum(num int64) *xeth.State {
	return self.xethWithStateNum(num).State()
}

// func (self *EthereumApi) Register(args string, reply *interface{}) error {
// 	self.regmut.Lock()
// 	defer self.regmut.Unlock()

// 	if _, ok := self.register[args]; ok {
// 		self.register[args] = nil // register with empty
// 	}
// 	return nil
// }

// func (self *EthereumApi) Unregister(args string, reply *interface{}) error {
// 	self.regmut.Lock()
// 	defer self.regmut.Unlock()

// 	delete(self.register, args)

// 	return nil
// }

// func (self *EthereumApi) WatchTx(args string, reply *interface{}) error {
// 	self.regmut.Lock()
// 	defer self.regmut.Unlock()

// 	txs := self.register[args]
// 	self.register[args] = nil

// 	*reply = txs
// 	return nil
// }

func (p *EthereumApi) Transact(args *NewTxArgs, reply *interface{}) (err error) {
	// TODO if no_private_key then
	//if _, exists := p.register[args.From]; exists {
	//	p.register[args.From] = append(p.register[args.From], args)
	//} else {
	/*
		account := accounts.Get(common.FromHex(args.From))
		if account != nil {
			if account.Unlocked() {
				if !unlockAccount(account) {
					return
				}
			}

			result, _ := account.Transact(common.FromHex(args.To), common.FromHex(args.Value), common.FromHex(args.Gas), common.FromHex(args.GasPrice), common.FromHex(args.Data))
			if len(result) > 0 {
				*reply = common.ToHex(result)
			}
		} else if _, exists := p.register[args.From]; exists {
			p.register[ags.From] = append(p.register[args.From], args)
		}
	*/

	if err := args.requirements(); err != nil {
		return err
	}

	// TODO: align default values to have the same type, e.g. not depend on
	// common.Value conversions later on
	if args.Gas.Cmp(big.NewInt(0)) == 0 {
		args.Gas = defaultGas
	}

	if args.GasPrice.Cmp(big.NewInt(0)) == 0 {
		args.GasPrice = defaultGasPrice
	}

	*reply, err = p.xeth().Transact(args.From, args.To, args.Value.String(), args.Gas.String(), args.GasPrice.String(), args.Data)
	if err != nil {
		fmt.Println("err:", err)
		return err
	}

	return nil
}

func (p *EthereumApi) Call(args *NewTxArgs, reply *interface{}) error {
	result, err := p.xethWithStateNum(args.BlockNumber).Call(args.From, args.To, args.Value.String(), args.Gas.String(), args.GasPrice.String(), args.Data)
	if err != nil {
		return err
	}

	*reply = result
	return nil
}

func (p *EthereumApi) GetStorageAt(args *GetStorageAtArgs, reply *interface{}) error {
	if err := args.requirements(); err != nil {
		return err
	}

	state := p.xethWithStateNum(args.BlockNumber).State().SafeGet(args.Address)
	value := state.StorageString(args.Key)

	var hx string
	if strings.Index(args.Key, "0x") == 0 {
		hx = string([]byte(args.Key)[2:])
	} else {
		// Convert the incoming string (which is a bigint) into hex
		i, _ := new(big.Int).SetString(args.Key, 10)
		hx = common.Bytes2Hex(i.Bytes())
	}
	rpclogger.Debugf("GetStateAt(%s, %s)\n", args.Address, hx)
	*reply = map[string]string{args.Key: value.Str()}
	return nil
}

func (self *EthereumApi) MessagesChanged(id int, reply *interface{}) error {
	*reply = self.xeth().MessagesChanged(id)
	return nil
}

func (p *EthereumApi) GetBlockByHash(blockhash string, includetx bool) (*BlockRes, error) {
	block := p.xeth().EthBlockByHash(blockhash)
	br := NewBlockRes(block)
	br.fullTx = includetx
	return br, nil
}

func (p *EthereumApi) GetBlockByNumber(blocknum int64, includetx bool) (*BlockRes, error) {
	block := p.xeth().EthBlockByNumber(blocknum)
	br := NewBlockRes(block)
	br.fullTx = includetx
	return br, nil
}

func (p *EthereumApi) GetRequestReply(req *RpcRequest, reply *interface{}) error {
	// Spec at https://github.com/ethereum/wiki/wiki/Generic-JSON-RPC
	rpclogger.Infof("%s %s", req.Method, req.Params)
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
		*reply = common.ToHex(big.NewInt(int64(p.xeth().PeerCount())).Bytes())
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
		*reply = common.ToHex(defaultGasPrice.Bytes())
	case "eth_accounts":
		*reply = p.xeth().Accounts()
	case "eth_blockNumber":
		*reply = common.ToHex(p.xeth().Backend().ChainManager().CurrentBlock().Number().Bytes())
	case "eth_getBalance":
		args := new(GetBalanceArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		if err := args.requirements(); err != nil {
			return err
		}

		*reply = common.ToHex(p.xethWithStateNum(args.BlockNumber).State().SafeGet(args.Address).Balance().Bytes())
	case "eth_getStorage", "eth_storageAt":
		args := new(GetStorageArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		if err := args.requirements(); err != nil {
			return err
		}

		*reply = p.xethWithStateNum(args.BlockNumber).State().SafeGet(args.Address).Storage()
	case "eth_getStorageAt":
		args := new(GetStorageAtArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.GetStorageAt(args, reply)
	case "eth_getTransactionCount":
		args := new(GetTxCountArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		err := args.requirements()
		if err != nil {
			return err
		}

		*reply = p.xethWithStateNum(args.BlockNumber).TxCountAt(args.Address)
	case "eth_getBlockTransactionCountByHash":
		args := new(GetBlockByHashArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := p.xeth().EthBlockByHash(args.BlockHash)
		br := NewBlockRes(block)
		*reply = common.ToHex(big.NewInt(int64(len(br.Transactions))).Bytes())
	case "eth_getBlockTransactionCountByNumber":
		args := new(GetBlockByNumberArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := p.xeth().EthBlockByNumber(args.BlockNumber)
		br := NewBlockRes(block)
		*reply = common.ToHex(big.NewInt(int64(len(br.Transactions))).Bytes())
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
		*reply = p.xethWithStateNum(args.BlockNumber).CodeAt(args.Address)
	case "eth_sendTransaction", "eth_transact":
		args := new(NewTxArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.Transact(args, reply)
	case "eth_call":
		args := new(NewTxArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.Call(args, reply)
	case "eth_flush":
		return NewNotImplementedError(req.Method)
	case "eth_getBlockByHash":
		args := new(GetBlockByHashArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		v, err := p.GetBlockByHash(args.BlockHash, args.Transactions)
		if err != nil {
			return err
		}
		*reply = v
	case "eth_getBlockByNumber":
		args := new(GetBlockByNumberArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		v, err := p.GetBlockByNumber(args.BlockNumber, args.Transactions)
		if err != nil {
			return err
		}
		*reply = v
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

		v, err := p.GetBlockByHash(args.Hash, true)
		if err != nil {
			return err
		}
		if args.Index > int64(len(v.Transactions)) || args.Index < 0 {
			return NewValidationError("Index", "does not exist")
		}
		*reply = v.Transactions[args.Index]
	case "eth_getTransactionByBlockNumberAndIndex":
		args := new(BlockNumIndexArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		v, err := p.GetBlockByNumber(args.BlockNumber, true)
		if err != nil {
			return err
		}
		if args.Index > int64(len(v.Transactions)) || args.Index < 0 {
			return NewValidationError("Index", "does not exist")
		}
		*reply = v.Transactions[args.Index]
	case "eth_getUncleByBlockHashAndIndex":
		args := new(HashIndexArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		v, err := p.GetBlockByHash(args.Hash, false)
		if err != nil {
			return err
		}
		if args.Index > int64(len(v.Uncles)) || args.Index < 0 {
			return NewValidationError("Index", "does not exist")
		}

		uncle, err := p.GetBlockByHash(common.ToHex(v.Uncles[args.Index]), false)
		if err != nil {
			return err
		}
		*reply = uncle
	case "eth_getUncleByBlockNumberAndIndex":
		args := new(BlockNumIndexArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		v, err := p.GetBlockByNumber(args.BlockNumber, true)
		if err != nil {
			return err
		}
		if args.Index > int64(len(v.Uncles)) || args.Index < 0 {
			return NewValidationError("Index", "does not exist")
		}

		uncle, err := p.GetBlockByHash(common.ToHex(v.Uncles[args.Index]), false)
		if err != nil {
			return err
		}
		*reply = uncle
	case "eth_getCompilers":
		c := []string{""}
		*reply = c
	case "eth_compileSolidity", "eth_compileLLL", "eth_compileSerpent":
		return NewNotImplementedError(req.Method)
	case "eth_newFilter":
		args := new(FilterOptions)
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
		args := new(FilterOptions)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		opts := toFilterOptions(args)
		*reply = NewLogsRes(p.xeth().AllLogs(opts))
	case "eth_getWork", "eth_submitWork":
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
		return p.MessagesChanged(args.Id, reply)
	case "shh_getMessages":
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = p.xeth().Whisper().Messages(args.Id)
	// case "eth_register":
	// 	args, err := req.ToRegisterArgs()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return p.Register(args, reply)
	// case "eth_unregister":
	// 	args, err := req.ToRegisterArgs()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return p.Unregister(args, reply)
	// case "eth_watchTx":
	// 	args, err := req.ToWatchTxArgs()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return p.WatchTx(args, reply)
	default:
		return NewNotImplementedError(req.Method)
	}

	rpclogger.DebugDetailf("Reply: %T %s", reply, reply)
	return nil
}

func (self *EthereumApi) xeth() *xeth.XEth {
	self.xethMu.RLock()
	defer self.xethMu.RUnlock()

	return self.eth
}

func toFilterOptions(options *FilterOptions) *core.FilterOptions {
	var opts core.FilterOptions

	// Convert optional address slice/string to byte slice
	if str, ok := options.Address.(string); ok {
		opts.Address = [][]byte{common.FromHex(str)}
	} else if slice, ok := options.Address.([]interface{}); ok {
		bslice := make([][]byte, len(slice))
		for i, addr := range slice {
			if saddr, ok := addr.(string); ok {
				bslice[i] = common.FromHex(saddr)
			}
		}
		opts.Address = bslice
	}

	opts.Earliest = options.Earliest
	opts.Latest = options.Latest

	topics := make([][][]byte, len(options.Topics))
	for i, topicDat := range options.Topics {
		if slice, ok := topicDat.([]interface{}); ok {
			topics[i] = make([][]byte, len(slice))
			for j, topic := range slice {
				topics[i][j] = common.FromHex(topic.(string))
			}
		} else if str, ok := topicDat.(string); ok {
			topics[i] = make([][]byte, 1)
			topics[i][0] = common.FromHex(str)
		}
	}
	opts.Topics = topics

	return &opts
}

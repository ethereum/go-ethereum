package rpc

import (
	"encoding/json"
	"fmt"
	"math/big"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/event/filter"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/xeth"
)

var (
	defaultGasPrice  = big.NewInt(150000000000)
	defaultGas       = big.NewInt(500000)
	filterTickerTime = 15 * time.Second
)

type EthereumApi struct {
	eth    *xeth.XEth
	xethMu sync.RWMutex
	mux    *event.TypeMux

	quit          chan struct{}
	filterManager *filter.FilterManager

	logMut sync.RWMutex
	logs   map[int]*logFilter

	messagesMut sync.RWMutex
	messages    map[int]*whisperFilter
	// Register keeps a list of accounts and transaction data
	regmut   sync.Mutex
	register map[string][]*NewTxArgs

	db ethutil.Database
}

func NewEthereumApi(eth *xeth.XEth, dataDir string) *EthereumApi {
	db, _ := ethdb.NewLDBDatabase(path.Join(dataDir, "dapps"))
	api := &EthereumApi{
		eth:           eth,
		mux:           eth.Backend().EventMux(),
		quit:          make(chan struct{}),
		filterManager: filter.NewFilterManager(eth.Backend().EventMux()),
		logs:          make(map[int]*logFilter),
		messages:      make(map[int]*whisperFilter),
		db:            db,
	}
	go api.filterManager.Start()
	go api.start()

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

func (self *EthereumApi) start() {
	timer := time.NewTicker(filterTickerTime)
done:
	for {
		select {
		case <-timer.C:
			self.logMut.Lock()
			self.messagesMut.Lock()
			for id, filter := range self.logs {
				if time.Since(filter.timeout) > 20*time.Second {
					self.filterManager.UninstallFilter(id)
					delete(self.logs, id)
				}
			}

			for id, filter := range self.messages {
				if time.Since(filter.timeout) > 20*time.Second {
					self.xeth().Whisper().Unwatch(id)
					delete(self.messages, id)
				}
			}
			self.logMut.Unlock()
			self.messagesMut.Unlock()
		case <-self.quit:
			break done
		}
	}
}

func (self *EthereumApi) stop() {
	close(self.quit)
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

func (self *EthereumApi) NewFilter(args *FilterOptions, reply *interface{}) error {
	var id int
	filter := core.NewFilter(self.xeth().Backend())
	filter.SetOptions(toFilterOptions(args))
	filter.LogsCallback = func(logs state.Logs) {
		self.logMut.Lock()
		defer self.logMut.Unlock()

		self.logs[id].add(logs...)
	}
	id = self.filterManager.InstallFilter(filter)
	self.logs[id] = &logFilter{timeout: time.Now()}

	*reply = i2hex(id)

	return nil
}

func (self *EthereumApi) UninstallFilter(id int, reply *interface{}) error {
	if _, ok := self.logs[id]; ok {
		delete(self.logs, id)
	}

	self.filterManager.UninstallFilter(id)
	*reply = true
	return nil
}

func (self *EthereumApi) NewFilterString(args *FilterStringArgs, reply *interface{}) error {
	var id int
	filter := core.NewFilter(self.xeth().Backend())

	callback := func(block *types.Block) {
		self.logMut.Lock()
		defer self.logMut.Unlock()

		self.logs[id].add(&state.StateLog{})
	}

	switch args.Word {
	case "pending":
		filter.PendingCallback = callback
	case "latest":
		filter.BlockCallback = callback
	default:
		return NewValidationError("Word", "Must be `latest` or `pending`")
	}

	id = self.filterManager.InstallFilter(filter)
	self.logs[id] = &logFilter{timeout: time.Now()}
	*reply = i2hex(id)

	return nil
}

func (self *EthereumApi) FilterChanged(id int, reply *interface{}) error {
	self.logMut.Lock()
	defer self.logMut.Unlock()

	if self.logs[id] != nil {
		*reply = toLogs(self.logs[id].get())
	}

	return nil
}

func (self *EthereumApi) Logs(id int, reply *interface{}) error {
	self.logMut.Lock()
	defer self.logMut.Unlock()

	filter := self.filterManager.GetFilter(id)
	if filter != nil {
		*reply = toLogs(filter.Find())
	}

	return nil
}

func (self *EthereumApi) AllLogs(args *FilterOptions, reply *interface{}) error {
	filter := core.NewFilter(self.xeth().Backend())
	filter.SetOptions(toFilterOptions(args))

	*reply = toLogs(filter.Find())

	return nil
}

func (p *EthereumApi) Transact(args *NewTxArgs, reply *interface{}) (err error) {
	// TODO if no_private_key then
	//if _, exists := p.register[args.From]; exists {
	//	p.register[args.From] = append(p.register[args.From], args)
	//} else {
	/*
		account := accounts.Get(ethutil.FromHex(args.From))
		if account != nil {
			if account.Unlocked() {
				if !unlockAccount(account) {
					return
				}
			}

			result, _ := account.Transact(ethutil.FromHex(args.To), ethutil.FromHex(args.Value), ethutil.FromHex(args.Gas), ethutil.FromHex(args.GasPrice), ethutil.FromHex(args.Data))
			if len(result) > 0 {
				*reply = toHex(result)
			}
		} else if _, exists := p.register[args.From]; exists {
			p.register[ags.From] = append(p.register[args.From], args)
		}
	*/
	// TODO: align default values to have the same type, e.g. not depend on
	// ethutil.Value conversions later on
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

func (p *EthereumApi) GetBalance(args *GetBalanceArgs, reply *interface{}) error {
	if err := args.requirements(); err != nil {
		return err
	}
	state := p.getStateWithNum(args.BlockNumber).SafeGet(args.Address)
	*reply = toHex(state.Balance().Bytes())
	return nil
}

func (p *EthereumApi) GetStorage(args *GetStorageArgs, reply *interface{}) error {
	if err := args.requirements(); err != nil {
		return err
	}
	*reply = p.getStateWithNum(args.BlockNumber).SafeGet(args.Address).Storage()
	return nil
}

func (p *EthereumApi) GetStorageAt(args *GetStorageAtArgs, reply *interface{}) error {
	if err := args.requirements(); err != nil {
		return err
	}
	state := p.getStateWithNum(args.BlockNumber).SafeGet(args.Address)

	value := state.StorageString(args.Key)
	var hx string
	if strings.Index(args.Key, "0x") == 0 {
		hx = string([]byte(args.Key)[2:])
	} else {
		// Convert the incoming string (which is a bigint) into hex
		i, _ := new(big.Int).SetString(args.Key, 10)
		hx = ethutil.Bytes2Hex(i.Bytes())
	}
	rpclogger.Debugf("GetStateAt(%s, %s)\n", args.Address, hx)
	*reply = map[string]string{args.Key: value.Str()}
	return nil
}

func (p *EthereumApi) GetTxCountAt(args *GetTxCountArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}
	*reply = p.xethWithStateNum(args.BlockNumber).TxCountAt(args.Address)
	return nil
}

func (p *EthereumApi) GetData(args *GetDataArgs, reply *interface{}) error {
	if err := args.requirements(); err != nil {
		return err
	}
	*reply = p.xethWithStateNum(args.BlockNumber).CodeAt(args.Address)
	return nil
}

func (p *EthereumApi) GetCompilers(reply *interface{}) error {
	c := []string{""}
	*reply = c
	return nil
}

func (p *EthereumApi) DbPut(args *DbArgs, reply *interface{}) error {
	if err := args.requirements(); err != nil {
		return err
	}

	p.db.Put([]byte(args.Database+args.Key), []byte(args.Value))
	*reply = true
	return nil
}

func (p *EthereumApi) DbGet(args *DbArgs, reply *interface{}) error {
	if err := args.requirements(); err != nil {
		return err
	}

	res, _ := p.db.Get([]byte(args.Database + args.Key))
	*reply = string(res)
	return nil
}

func (p *EthereumApi) NewWhisperIdentity(reply *interface{}) error {
	*reply = p.xeth().Whisper().NewIdentity()
	return nil
}

func (p *EthereumApi) NewWhisperFilter(args *WhisperFilterArgs, reply *interface{}) error {
	var id int
	opts := new(xeth.Options)
	opts.From = args.From
	opts.To = args.To
	opts.Topics = args.Topics
	opts.Fn = func(msg xeth.WhisperMessage) {
		p.messagesMut.Lock()
		defer p.messagesMut.Unlock()
		p.messages[id].add(msg) // = append(p.messages[id], msg)
	}
	id = p.xeth().Whisper().Watch(opts)
	p.messages[id] = &whisperFilter{timeout: time.Now()}
	*reply = toHex(big.NewInt(int64(id)).Bytes())
	return nil
}

func (p *EthereumApi) UninstallWhisperFilter(id int, reply *interface{}) error {
	delete(p.messages, id)
	*reply = true
	return nil
}

func (self *EthereumApi) MessagesChanged(id int, reply *interface{}) error {
	self.messagesMut.Lock()
	defer self.messagesMut.Unlock()

	if self.messages[id] != nil {
		*reply = self.messages[id].get()
	}

	return nil
}

func (p *EthereumApi) WhisperPost(args *WhisperMessageArgs, reply *interface{}) error {
	err := p.xeth().Whisper().Post(args.Payload, args.To, args.From, args.Topics, args.Priority, args.Ttl)
	if err != nil {
		return err
	}

	*reply = true
	return nil
}

func (p *EthereumApi) HasWhisperIdentity(args string, reply *interface{}) error {
	*reply = p.xeth().Whisper().HasIdentity(args)
	return nil
}

func (p *EthereumApi) WhisperMessages(id int, reply *interface{}) error {
	*reply = p.xeth().Whisper().Messages(id)
	return nil
}

func (p *EthereumApi) GetTransactionByHash(hash string, reply *interface{}) error {
	tx := p.xeth().EthTransactionByHash(hash)
	if tx != nil {
		*reply = NewTransactionRes(tx)
	}
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

func (p *EthereumApi) GetBlockTransactionCountByHash(blockhash string) (int64, error) {
	block := p.xeth().EthBlockByHash(blockhash)
	br := NewBlockRes(block)
	return int64(len(br.Transactions)), nil
}

func (p *EthereumApi) GetBlockTransactionCountByNumber(blocknum int64) (int64, error) {
	block := p.xeth().EthBlockByNumber(blocknum)
	br := NewBlockRes(block)
	return int64(len(br.Transactions)), nil
}

func (p *EthereumApi) GetBlockUncleCountByHash(blockhash string) (int64, error) {
	block := p.xeth().EthBlockByHash(blockhash)
	br := NewBlockRes(block)
	return int64(len(br.Uncles)), nil
}

func (p *EthereumApi) GetBlockUncleCountByNumber(blocknum int64) (int64, error) {
	block := p.xeth().EthBlockByNumber(blocknum)
	br := NewBlockRes(block)
	return int64(len(br.Uncles)), nil
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
		*reply = toHex(crypto.Sha3(ethutil.FromHex(args.Data)))
	case "web3_clientVersion":
		*reply = p.xeth().Backend().Version()
	case "net_version":
		return NewNotImplementedError(req.Method)
	case "net_listening":
		*reply = p.xeth().IsListening()
	case "net_peerCount":
		*reply = toHex(big.NewInt(int64(p.xeth().PeerCount())).Bytes())
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
		*reply = toHex(defaultGasPrice.Bytes())
	case "eth_accounts":
		*reply = p.xeth().Accounts()
	case "eth_blockNumber":
		*reply = toHex(p.xeth().Backend().ChainManager().CurrentBlock().Number().Bytes())
	case "eth_getBalance":
		args := new(GetBalanceArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.GetBalance(args, reply)
	case "eth_getStorage", "eth_storageAt":
		args := new(GetStorageArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.GetStorage(args, reply)
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
		return p.GetTxCountAt(args, reply)
	case "eth_getBlockTransactionCountByHash":
		args := new(GetBlockByHashArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		v, err := p.GetBlockTransactionCountByHash(args.BlockHash)
		if err != nil {
			return err
		}
		*reply = toHex(big.NewInt(v).Bytes())
	case "eth_getBlockTransactionCountByNumber":
		args := new(GetBlockByNumberArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		v, err := p.GetBlockTransactionCountByNumber(args.BlockNumber)
		if err != nil {
			return err
		}
		*reply = toHex(big.NewInt(v).Bytes())
	case "eth_getUncleCountByBlockHash":
		args := new(GetBlockByHashArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		v, err := p.GetBlockUncleCountByHash(args.BlockHash)
		if err != nil {
			return err
		}
		*reply = toHex(big.NewInt(v).Bytes())
	case "eth_getUncleCountByBlockNumber":
		args := new(GetBlockByNumberArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		v, err := p.GetBlockUncleCountByNumber(args.BlockNumber)
		if err != nil {
			return err
		}
		*reply = toHex(big.NewInt(v).Bytes())
	case "eth_getData":
		args := new(GetDataArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.GetData(args, reply)
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
		return p.GetTransactionByHash(args.Hash, reply)
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

		uncle, err := p.GetBlockByHash(toHex(v.Uncles[args.Index]), false)
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

		uncle, err := p.GetBlockByHash(toHex(v.Uncles[args.Index]), false)
		if err != nil {
			return err
		}
		*reply = uncle
	case "eth_getCompilers":
		return p.GetCompilers(reply)
	case "eth_compileSolidity", "eth_compileLLL", "eth_compileSerpent":
		return NewNotImplementedError(req.Method)
	case "eth_newFilter":
		args := new(FilterOptions)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.NewFilter(args, reply)
	case "eth_newBlockFilter":
		args := new(FilterStringArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.NewFilterString(args, reply)
	case "eth_uninstallFilter":
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.UninstallFilter(args.Id, reply)
	case "eth_getFilterChanges":
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.FilterChanged(args.Id, reply)
	case "eth_getFilterLogs":
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.Logs(args.Id, reply)
	case "eth_getLogs":
		args := new(FilterOptions)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.AllLogs(args, reply)
	case "eth_getWork", "eth_submitWork":
		return NewNotImplementedError(req.Method)
	case "db_putString":
		args := new(DbArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.DbPut(args, reply)
	case "db_getString":
		args := new(DbArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.DbGet(args, reply)
	case "db_putHex", "db_getHex":
		return NewNotImplementedError(req.Method)
	case "shh_post":
		args := new(WhisperMessageArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.WhisperPost(args, reply)
	case "shh_newIdentity":
		return p.NewWhisperIdentity(reply)
	case "shh_hasIdentity":
		args := new(WhisperIdentityArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.HasWhisperIdentity(args.Identity, reply)
	case "shh_newGroup", "shh_addToGroup":
		return NewNotImplementedError(req.Method)
	case "shh_newFilter":
		args := new(WhisperFilterArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.NewWhisperFilter(args, reply)
	case "shh_uninstallFilter":
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.UninstallWhisperFilter(args.Id, reply)
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
		return p.WhisperMessages(args.Id, reply)
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

func toFilterOptions(options *FilterOptions) core.FilterOptions {
	var opts core.FilterOptions

	// Convert optional address slice/string to byte slice
	if str, ok := options.Address.(string); ok {
		opts.Address = [][]byte{ethutil.FromHex(str)}
	} else if slice, ok := options.Address.([]interface{}); ok {
		bslice := make([][]byte, len(slice))
		for i, addr := range slice {
			if saddr, ok := addr.(string); ok {
				bslice[i] = ethutil.FromHex(saddr)
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
				topics[i][j] = ethutil.FromHex(topic.(string))
			}
		} else if str, ok := topicDat.(string); ok {
			topics[i] = make([][]byte, 1)
			topics[i][0] = ethutil.FromHex(str)
		}
	}
	opts.Topics = topics

	return opts
}

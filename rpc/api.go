package rpc

import (
	"encoding/json"
	"fmt"
	"math/big"
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
	"github.com/ethereum/go-ethereum/ui"
	"github.com/ethereum/go-ethereum/xeth"
)

var (
	defaultGasPrice  = big.NewInt(10000000000000)
	defaultGas       = big.NewInt(10000)
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

	// defaultBlockAge int64
}

func NewEthereumApi(eth *xeth.XEth) *EthereumApi {
	db, _ := ethdb.NewLDBDatabase("dapps")
	api := &EthereumApi{
		eth:           eth,
		mux:           eth.Backend().EventMux(),
		quit:          make(chan struct{}),
		filterManager: filter.NewFilterManager(eth.Backend().EventMux()),
		logs:          make(map[int]*logFilter),
		messages:      make(map[int]*whisperFilter),
		db:            db,
		// defaultBlockAge: -1,
	}
	go api.filterManager.Start()
	go api.start()

	return api
}

// func (self *EthereumApi) setStateByBlockNumber(num int64) {
// 	chain := self.xeth().Backend().ChainManager()
// 	var block *types.Block

// 	if self.defaultBlockAge < 0 {
// 		num = chain.CurrentBlock().Number().Int64() + num + 1
// 	}
// 	block = chain.GetBlockByNumber(uint64(num))

// 	if block != nil {
// 		self.useState(state.New(block.Root(), self.xeth().Backend().Db()))
// 	} else {
// 		self.useState(chain.State())
// 	}
// }

func (self *EthereumApi) start() {
	timer := time.NewTicker(filterTickerTime)
	// events := self.mux.Subscribe(core.ChainEvent{})

done:
	for {
		select {
		// case ev := <-events.Chan():
		// 	switch ev.(type) {
		// 	case core.ChainEvent:
		// 		if self.defaultBlockAge < 0 {
		// 			self.setStateByBlockNumber(self.defaultBlockAge)
		// 		}
		// 	}
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

func (self *EthereumApi) Register(args string, reply *interface{}) error {
	self.regmut.Lock()
	defer self.regmut.Unlock()

	if _, ok := self.register[args]; ok {
		self.register[args] = nil // register with empty
	}
	return nil
}

func (self *EthereumApi) Unregister(args string, reply *interface{}) error {
	self.regmut.Lock()
	defer self.regmut.Unlock()

	delete(self.register, args)

	return nil
}

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

	*reply = id

	return nil
}

func (self *EthereumApi) UninstallFilter(id int, reply *interface{}) error {
	delete(self.logs, id)
	self.filterManager.UninstallFilter(id)
	*reply = true
	return nil
}

func (self *EthereumApi) NewFilterString(args string, reply *interface{}) error {
	var id int
	filter := core.NewFilter(self.xeth().Backend())

	callback := func(block *types.Block) {
		self.logMut.Lock()
		defer self.logMut.Unlock()

		self.logs[id].add(&state.StateLog{})
	}
	if args == "pending" {
		filter.PendingCallback = callback
	} else if args == "chain" {
		filter.BlockCallback = callback
	}

	id = self.filterManager.InstallFilter(filter)
	self.logs[id] = &logFilter{timeout: time.Now()}
	*reply = id

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

func (p *EthereumApi) Transact(args *NewTxArgs, reply *interface{}) error {
	if args.Gas == ethutil.Big0 {
		args.Gas = defaultGas
	}

	if args.GasPrice == ethutil.Big0 {
		args.GasPrice = defaultGasPrice
	}

	// TODO if no_private_key then
	//if _, exists := p.register[args.From]; exists {
	//	p.register[args.From] = append(p.register[args.From], args)
	//} else {
	/*
		account := accounts.Get(fromHex(args.From))
		if account != nil {
			if account.Unlocked() {
				if !unlockAccount(account) {
					return
				}
			}

			result, _ := account.Transact(fromHex(args.To), fromHex(args.Value), fromHex(args.Gas), fromHex(args.GasPrice), fromHex(args.Data))
			if len(result) > 0 {
				*reply = toHex(result)
			}
		} else if _, exists := p.register[args.From]; exists {
			p.register[ags.From] = append(p.register[args.From], args)
		}
	*/
	result, err := p.xeth().Transact( /* TODO specify account */ args.To, args.Value.String(), args.Gas.String(), args.GasPrice.String(), args.Data)
	if err != nil {
		return err
	}
	*reply = result
	//}

	return nil
}

func (p *EthereumApi) Call(args *NewTxArgs, reply *interface{}) error {
	result, err := p.xeth().Call( /* TODO specify account */ args.To, args.Value.String(), args.Gas.String(), args.GasPrice.String(), args.Data)
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
	state := p.xeth().State().SafeGet(args.Address)
	*reply = toHex(state.Balance().Bytes())
	return nil
}

func (p *EthereumApi) GetStorage(args *GetStorageArgs, reply *interface{}) error {
	if err := args.requirements(); err != nil {
		return err
	}
	*reply = p.xeth().State().SafeGet(args.Address).Storage()
	return nil
}

func (p *EthereumApi) GetStorageAt(args *GetStorageAtArgs, reply *interface{}) error {
	if err := args.requirements(); err != nil {
		return err
	}
	state := p.xeth().State().SafeGet(args.Address)

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
	*reply = p.xeth().TxCountAt(args.Address)
	return nil
}

func (p *EthereumApi) GetData(args *GetDataArgs, reply *interface{}) error {
	if err := args.requirements(); err != nil {
		return err
	}
	*reply = p.xeth().CodeAt(args.Address)
	return nil
}

func (p *EthereumApi) GetCompilers(reply *interface{}) error {
	c := []string{"serpent"}
	*reply = c
	return nil
}

func (p *EthereumApi) CompileSerpent(args *CompileArgs, reply *interface{}) error {
	res, err := ethutil.Compile(args.Source, false)
	if err != nil {
		return err
	}
	*reply = res
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
	*reply = id
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
	err := p.xeth().Whisper().Post(args.Payload, args.To, args.From, args.Topic, args.Priority, args.Ttl)
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

func (p *EthereumApi) GetRequestReply(req *RpcRequest, reply *interface{}) error {
	// Spec at https://github.com/ethereum/wiki/wiki/Generic-JSON-RPC
	rpclogger.DebugDetailf("%T %s", req.Params, req.Params)
	switch req.Method {
	case "web3_sha3":
		args := new(Sha3Args)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = toHex(crypto.Sha3(fromHex(args.Data)))
	case "net_listening":
		*reply = p.xeth().IsListening()
	case "net_peerCount":
		*reply = toHex(big.NewInt(int64(p.xeth().PeerCount())).Bytes())
	case "eth_coinbase":
		*reply = p.xeth().Coinbase()
	case "eth_mining":
		*reply = p.xeth().IsMining()
	case "eth_gasPrice":
		*reply = toHex(defaultGasPrice.Bytes())
	case "eth_accounts":
		*reply = p.xeth().Accounts()
	case "eth_blockNumber":
		*reply = toHex(p.xeth().Backend().ChainManager().CurrentBlock().Number().Bytes())
	case "eth_getBalance":
		// TODO handle BlockNumber
		args := new(GetBalanceArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.GetBalance(args, reply)
	case "eth_getStorage":
		// TODO handle BlockNumber
		args := new(GetStorageArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.GetStorage(args, reply)
	case "eth_getStorageAt":
		// TODO handle BlockNumber
		args := new(GetStorageAtArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.GetStorageAt(args, reply)
	case "eth_getTransactionCount":
		// TODO handle BlockNumber
		args := new(GetTxCountArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.GetTxCountAt(args, reply)
	case "eth_getBlockTransactionCountByHash":
	case "eth_getBlockTransactionCountByNumber":
	case "eth_getUncleCountByBlockHash":
	case "eth_getUncleCountByBlockNumber":
		return errNotImplemented
	case "eth_getData":
		// TODO handle BlockNumber
		args := new(GetDataArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.GetData(args, reply)
	case "eth_sendTransaction":
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
		return errNotImplemented
	case "eth_getBlockByHash":
		// TODO handle second param for "include transaction objects"
		args := new(GetBlockByHashArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = p.xeth().BlockByHash(args.BlockHash)
	case "eth_getBlockByNumber":
		// TODO handle second param for "include transaction objects"
		args := new(GetBlockByNumberArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = p.xeth().BlockByNumber(args.BlockNumber)
	case "eth_getTransactionByHash":
	case "eth_getTransactionByBlockHashAndIndex":
	case "eth_getTransactionByBlockNumberAndIndex":
	case "eth_getUncleByBlockHashAndIndex":
	case "eth_getUncleByBlockNumberAndIndex":
		return errNotImplemented
	case "eth_getCompilers":
		return p.GetCompilers(reply)
	case "eth_compileSolidity":
	case "eth_compileLLL":
		return errNotImplemented
	case "eth_compileSerpent":
		args := new(CompileArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.CompileSerpent(args, reply)
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
		return p.NewFilterString(args.Word, reply)
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
	case "eth_getWork":
	case "eth_submitWork":
		return errNotImplemented
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
	case "db_put":
		args := new(DbArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.DbPut(args, reply)
	case "db_get":
		args := new(DbArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.DbGet(args, reply)
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
	case "shh_newGroup":
	case "shh_addToGroup":
		return errNotImplemented
	case "shh_newFilter":
		args := new(WhisperFilterArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		return p.NewWhisperFilter(args, reply)
	case "shh_uninstallFilter":
		return errNotImplemented
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
	case "client_version":
		*reply = p.eth.GetClientVersion()
	default:
		return NewErrorWithMessage(errNotImplemented, req.Method)
	}

	rpclogger.DebugDetailf("Reply: %T %s", reply, reply)
	return nil
}

func (self *EthereumApi) xeth() *xeth.XEth {
	self.xethMu.RLock()
	defer self.xethMu.RUnlock()

	return self.eth
}

func (self *EthereumApi) useState(statedb *state.StateDB) {
	self.xethMu.Lock()
	defer self.xethMu.Unlock()

	self.eth = self.eth.UseState(statedb)
}

func t(f ui.Frontend) {
	// Call the password dialog
	ret, err := f.Call("PasswordDialog")
	if err != nil {
		fmt.Println(err)
	}
	// Get the first argument
	t, _ := ret.Get(0)
	fmt.Println("return:", t)
}

/*
For each request type, define the following:

1. RpcRequest "To" method [message.go], which does basic validation and conversion to "Args" type via json.Decoder()
2. json.Decoder() calls "UnmarshalON" defined on each "Args" struct
3. EthereumApi method, taking the "Args" type and replying with an interface to be marshalled to ON

*/
package rpc

import (
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

func (self *EthereumApi) WatchTx(args string, reply *interface{}) error {
	self.regmut.Lock()
	defer self.regmut.Unlock()

	txs := self.register[args]
	self.register[args] = nil

	*reply = txs
	return nil
}

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

func (p *EthereumApi) GetBlock(args *GetBlockArgs, reply *interface{}) error {
	// This seems a bit precarious Maybe worth splitting to discrete functions
	if len(args.Hash) > 0 {
		*reply = p.xeth().BlockByHash(args.Hash)
	} else {
		*reply = p.xeth().BlockByNumber(args.BlockNumber)
	}
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

func (p *EthereumApi) PushTx(args *PushTxArgs, reply *interface{}) error {
	err := args.requirementsPushTx()
	if err != nil {
		return err
	}
	result, _ := p.xeth().PushTx(args.Tx)
	*reply = result
	return nil
}

func (p *EthereumApi) GetStateAt(args *GetStateArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
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

func (p *EthereumApi) GetStorageAt(args *GetStorageArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}

	*reply = p.xeth().State().SafeGet(args.Address).Storage()
	return nil
}

func (p *EthereumApi) GetPeerCount(reply *interface{}) error {
	c := p.xeth().PeerCount()
	*reply = toHex(big.NewInt(int64(c)).Bytes())
	return nil
}

func (p *EthereumApi) GetIsListening(reply *interface{}) error {
	*reply = p.xeth().IsListening()
	return nil
}

func (p *EthereumApi) GetCoinbase(reply *interface{}) error {
	*reply = p.xeth().Coinbase()
	return nil
}

func (p *EthereumApi) Accounts(reply *interface{}) error {
	*reply = p.xeth().Accounts()
	return nil
}

func (p *EthereumApi) GetIsMining(reply *interface{}) error {
	*reply = p.xeth().IsMining()
	return nil
}

func (p *EthereumApi) BlockNumber(reply *interface{}) error {
	*reply = toHex(p.xeth().Backend().ChainManager().CurrentBlock().Number().Bytes())
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

func (p *EthereumApi) GetBalanceAt(args *GetBalanceArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}
	state := p.xeth().State().SafeGet(args.Address)
	*reply = toHex(state.Balance().Bytes())
	return nil
}

func (p *EthereumApi) GetCodeAt(args *GetCodeAtArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
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

func (p *EthereumApi) CompileSerpent(script string, reply *interface{}) error {
	res, err := ethutil.Compile(script, false)
	if err != nil {
		return err
	}
	*reply = res
	return nil
}

func (p *EthereumApi) Sha3(args *Sha3Args, reply *interface{}) error {
	*reply = toHex(crypto.Sha3(fromHex(args.Data)))
	return nil
}

func (p *EthereumApi) DbPut(args *DbArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}

	p.db.Put([]byte(args.Database+args.Key), []byte(args.Value))
	*reply = true
	return nil
}

func (p *EthereumApi) DbGet(args *DbArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
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

func (p *EthereumApi) NewWhisperFilter(args *xeth.Options, reply *interface{}) error {
	var id int
	args.Fn = func(msg xeth.WhisperMessage) {
		p.messagesMut.Lock()
		defer p.messagesMut.Unlock()
		p.messages[id].add(msg) // = append(p.messages[id], msg)
	}
	id = p.xeth().Whisper().Watch(args)
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
		args, err := req.ToSha3Args()
		if err != nil {
			return err
		}
		return p.Sha3(args, reply)
	case "net_listening":
		return p.GetIsListening(reply)
	case "net_peerCount":
		return p.GetPeerCount(reply)
	case "eth_coinbase":
		return p.GetCoinbase(reply)
	case "eth_mining":
		return p.GetIsMining(reply)
	case "eth_gasPrice":
		*reply = toHex(defaultGasPrice.Bytes())
		return nil
	case "eth_accounts":
		return p.Accounts(reply)
	case "eth_blockNumber":
		return p.BlockNumber(reply)
	case "eth_getBalance":
		// TODO handle defaultBlock
		args, err := req.ToGetBalanceArgs()
		if err != nil {
			return err
		}
		return p.GetBalanceAt(args, reply)
	case "eth_getStorage":
		// TODO handle defaultBlock
		args, err := req.ToGetStateArgs()
		if err != nil {
			return err
		}
		return p.GetStateAt(args, reply)
	case "eth_getStorageAt":
		// TODO handle defaultBlock
		args, err := req.ToStorageAtArgs()
		if err != nil {
			return err
		}
		return p.GetStorageAt(args, reply)
	case "eth_getTransactionCount":
		// TODO handle defaultBlock
		args, err := req.ToGetTxCountArgs()
		if err != nil {
			return err
		}
		return p.GetTxCountAt(args, reply)
	case "eth_getBlockTransactionCountByHash":
	case "eth_getBlockTransactionCountByNumber":
	case "eth_getUncleCountByBlockHash":
	case "eth_getUncleCountByBlockNumber":
		return errNotImplemented
	case "eth_getData":
		// TODO handle defaultBlock
		args, err := req.ToGetCodeAtArgs()
		if err != nil {
			return err
		}
		return p.GetCodeAt(args, reply)
	case "eth_sendTransaction":
		args, err := req.ToNewTxArgs()
		if err != nil {
			return err
		}
		return p.Transact(args, reply)
	case "eth_call":
		args, err := req.ToNewTxArgs()
		if err != nil {
			return err
		}
		return p.Call(args, reply)
	case "eth_flush":
		return errNotImplemented
	case "eth_getBlockByNumber":
	case "eth_getBlockByHash":
		// TODO handle second param for "include transaction objects"
		args, err := req.ToGetBlockArgs()
		if err != nil {
			return err
		}
		return p.GetBlock(args, reply)
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
		args, err := req.ToCompileArgs()
		if err != nil {
			return err
		}
		return p.CompileSerpent(args, reply)
	case "eth_newFilter":
		args, err := req.ToFilterArgs()
		if err != nil {
			return err
		}
		return p.NewFilter(args, reply)
	case "eth_newBlockFilter":
		args, err := req.ToFilterStringArgs()
		if err != nil {
			return err
		}
		return p.NewFilterString(args, reply)
	case "eth_uninstallFilter":
		args, err := req.ToUninstallFilterArgs()
		if err != nil {
			return err
		}
		return p.UninstallFilter(args, reply)
	case "eth_getFilterChanges":
		args, err := req.ToIdArgs()
		if err != nil {
			return err
		}
		return p.FilterChanged(args, reply)
	case "eth_getFilterLogs":
		args, err := req.ToIdArgs()
		if err != nil {
			return err
		}
		return p.Logs(args, reply)
	case "eth_getLogs":
		args, err := req.ToFilterArgs()
		if err != nil {
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
		args, err := req.ToDbPutArgs()
		if err != nil {
			return err
		}
		return p.DbPut(args, reply)
	case "db_get":
		args, err := req.ToDbGetArgs()
		if err != nil {
			return err
		}
		return p.DbGet(args, reply)
	case "shh_post":
		args, err := req.ToWhisperPostArgs()
		if err != nil {
			return err
		}
		return p.WhisperPost(args, reply)
	case "shh_newIdentity":
		return p.NewWhisperIdentity(reply)
	case "shh_hasIdentity":
		args, err := req.ToWhisperHasIdentityArgs()
		if err != nil {
			return err
		}
		return p.HasWhisperIdentity(args, reply)
	case "shh_newGroup":
	case "shh_addToGroup":
		return errNotImplemented
	case "shh_newFilter":
		args, err := req.ToWhisperFilterArgs()
		if err != nil {
			return err
		}
		return p.NewWhisperFilter(args, reply)
	case "shh_uninstallFilter":
		return errNotImplemented
	case "shh_getFilterChanges":
		args, err := req.ToIdArgs()
		if err != nil {
			return err
		}
		return p.MessagesChanged(args, reply)
	case "shh_getMessages":
		args, err := req.ToIdArgs()
		if err != nil {
			return err
		}
		return p.WhisperMessages(args, reply)
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

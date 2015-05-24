package rpc

import (
	"bytes"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/xeth"
)

type EthereumApi struct {
	eth *xeth.XEth
}

func NewEthereumApi(xeth *xeth.XEth) *EthereumApi {
	api := &EthereumApi{
		eth: xeth,
	}

	return api
}

func (api *EthereumApi) xeth() *xeth.XEth {
	return api.eth
}

func (api *EthereumApi) xethAtStateNum(num int64) *xeth.XEth {
	return api.xeth().AtStateNum(num)
}

func (api *EthereumApi) GetRequestReply(req *RpcRequest, reply *interface{}) error {
	// Spec at https://github.com/ethereum/wiki/wiki/JSON-RPC
	glog.V(logger.Debug).Infof("%s %s", req.Method, req.Params)

	switch req.Method {
	case "web3_sha3":
		args := new(Sha3Args)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = common.ToHex(crypto.Sha3(common.FromHex(args.Data)))
	case "web3_clientVersion":
		*reply = api.xeth().ClientVersion()
	case "net_version":
		*reply = api.xeth().NetworkVersion()
	case "net_listening":
		*reply = api.xeth().IsListening()
	case "net_peerCount":
		*reply = newHexNum(api.xeth().PeerCount())
	case "eth_protocolVersion":
		*reply = api.xeth().EthVersion()
	case "eth_coinbase":
		*reply = newHexData(api.xeth().Coinbase())
	case "eth_mining":
		*reply = api.xeth().IsMining()
	case "eth_gasPrice":
		v := xeth.DefaultGasPrice()
		*reply = newHexNum(v.Bytes())
	case "eth_accounts":
		*reply = api.xeth().Accounts()
	case "eth_blockNumber":
		v := api.xeth().CurrentBlock().Number()
		*reply = newHexNum(v.Bytes())
	case "eth_getBalance":
		args := new(GetBalanceArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		*reply = api.xethAtStateNum(args.BlockNumber).BalanceAt(args.Address)
		//v := api.xethAtStateNum(args.BlockNumber).State().SafeGet(args.Address).Balance()
		//*reply = common.ToHex(v.Bytes())
	case "eth_getStorage", "eth_storageAt":
		args := new(GetStorageArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		*reply = api.xethAtStateNum(args.BlockNumber).State().SafeGet(args.Address).Storage()
	case "eth_getStorageAt":
		args := new(GetStorageAtArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		*reply = api.xethAtStateNum(args.BlockNumber).StorageAt(args.Address, args.Key)
	case "eth_getTransactionCount":
		args := new(GetTxCountArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		count := api.xethAtStateNum(args.BlockNumber).TxCountAt(args.Address)
		*reply = newHexNum(big.NewInt(int64(count)).Bytes())
	case "eth_getBlockTransactionCountByHash":
		args := new(HashArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := NewBlockRes(api.xeth().EthBlockByHash(args.Hash), false)
		if block == nil {
			*reply = nil
		} else {
			*reply = newHexNum(big.NewInt(int64(len(block.Transactions))).Bytes())
		}
	case "eth_getBlockTransactionCountByNumber":
		args := new(BlockNumArg)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := NewBlockRes(api.xeth().EthBlockByNumber(args.BlockNumber), false)
		if block == nil {
			*reply = nil
			break
		}

		*reply = newHexNum(big.NewInt(int64(len(block.Transactions))).Bytes())
	case "eth_getUncleCountByBlockHash":
		args := new(HashArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := api.xeth().EthBlockByHash(args.Hash)
		br := NewBlockRes(block, false)
		if br == nil {
			*reply = nil
			break
		}

		*reply = newHexNum(big.NewInt(int64(len(br.Uncles))).Bytes())
	case "eth_getUncleCountByBlockNumber":
		args := new(BlockNumArg)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := api.xeth().EthBlockByNumber(args.BlockNumber)
		br := NewBlockRes(block, false)
		if br == nil {
			*reply = nil
			break
		}

		*reply = newHexNum(big.NewInt(int64(len(br.Uncles))).Bytes())

	case "eth_getData", "eth_getCode":
		args := new(GetDataArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		v := api.xethAtStateNum(args.BlockNumber).CodeAtBytes(args.Address)
		*reply = newHexData(v)

	case "eth_sign":
		args := new(NewSigArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		v, err := api.xeth().Sign(args.From, args.Data, false)
		if err != nil {
			return err
		}
		*reply = v

	case "eth_sendTransaction", "eth_transact":
		args := new(NewTxArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		// nonce may be nil ("guess" mode)
		var nonce string
		if args.Nonce != nil {
			nonce = args.Nonce.String()
		}

		v, err := api.xeth().Transact(args.From, args.To, nonce, args.Value.String(), args.Gas.String(), args.GasPrice.String(), args.Data)
		if err != nil {
			return err
		}
		*reply = v
	case "eth_estimateGas":
		_, gas, err := api.doCall(req.Params)
		if err != nil {
			return err
		}

		// TODO unwrap the parent method's ToHex call
		if len(gas) == 0 {
			*reply = newHexNum(0)
		} else {
			*reply = newHexNum(gas)
		}
	case "eth_call":
		v, _, err := api.doCall(req.Params)
		if err != nil {
			return err
		}

		// TODO unwrap the parent method's ToHex call
		if v == "0x0" {
			*reply = newHexData([]byte{})
		} else {
			*reply = newHexData(common.FromHex(v))
		}
	case "eth_flush":
		return NewNotImplementedError(req.Method)
	case "eth_getBlockByHash":
		args := new(GetBlockByHashArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := api.xeth().EthBlockByHash(args.BlockHash)
		br := NewBlockRes(block, args.IncludeTxs)

		*reply = br
	case "eth_getBlockByNumber":
		args := new(GetBlockByNumberArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := api.xeth().EthBlockByNumber(args.BlockNumber)
		br := NewBlockRes(block, args.IncludeTxs)
		// If request was for "pending", nil nonsensical fields
		if args.BlockNumber == -2 {
			br.BlockHash = nil
			br.BlockNumber = nil
			br.Miner = nil
			br.Nonce = nil
			br.LogsBloom = nil
		}
		*reply = br
	case "eth_getTransactionByHash":
		args := new(HashArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		tx, bhash, bnum, txi := api.xeth().EthTransactionByHash(args.Hash)
		if tx != nil {
			v := NewTransactionRes(tx)
			// if the blockhash is 0, assume this is a pending transaction
			if bytes.Compare(bhash.Bytes(), bytes.Repeat([]byte{0}, 32)) != 0 {
				v.BlockHash = newHexData(bhash)
				v.BlockNumber = newHexNum(bnum)
				v.TxIndex = newHexNum(txi)
			}
			*reply = v
		}
	case "eth_getTransactionByBlockHashAndIndex":
		args := new(HashIndexArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := api.xeth().EthBlockByHash(args.Hash)
		br := NewBlockRes(block, true)
		if br == nil {
			*reply = nil
			break
		}

		if args.Index >= int64(len(br.Transactions)) || args.Index < 0 {
			// return NewValidationError("Index", "does not exist")
			*reply = nil
		} else {
			*reply = br.Transactions[args.Index]
		}
	case "eth_getTransactionByBlockNumberAndIndex":
		args := new(BlockNumIndexArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := api.xeth().EthBlockByNumber(args.BlockNumber)
		v := NewBlockRes(block, true)
		if v == nil {
			*reply = nil
			break
		}

		if args.Index >= int64(len(v.Transactions)) || args.Index < 0 {
			// return NewValidationError("Index", "does not exist")
			*reply = nil
		} else {
			*reply = v.Transactions[args.Index]
		}
	case "eth_getUncleByBlockHashAndIndex":
		args := new(HashIndexArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		br := NewBlockRes(api.xeth().EthBlockByHash(args.Hash), false)
		if br == nil {
			*reply = nil
			return nil
		}

		if args.Index >= int64(len(br.Uncles)) || args.Index < 0 {
			// return NewValidationError("Index", "does not exist")
			*reply = nil
		} else {
			*reply = br.Uncles[args.Index]
		}
	case "eth_getUncleByBlockNumberAndIndex":
		args := new(BlockNumIndexArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		block := api.xeth().EthBlockByNumber(args.BlockNumber)
		v := NewBlockRes(block, true)

		if v == nil {
			*reply = nil
			return nil
		}

		if args.Index >= int64(len(v.Uncles)) || args.Index < 0 {
			// return NewValidationError("Index", "does not exist")
			*reply = nil
		} else {
			*reply = v.Uncles[args.Index]
		}

	case "eth_getCompilers":
		var lang string
		if solc, _ := api.xeth().Solc(); solc != nil {
			lang = "Solidity"
		}
		c := []string{lang}
		*reply = c

	case "eth_compileLLL", "eth_compileSerpent":
		return NewNotImplementedError(req.Method)

	case "eth_compileSolidity":
		solc, _ := api.xeth().Solc()
		if solc == nil {
			return NewNotAvailableError(req.Method, "solc (solidity compiler) not found")
		}

		args := new(SourceArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		contracts, err := solc.Compile(args.Source)
		if err != nil {
			return err
		}
		*reply = contracts

	case "eth_newFilter":
		args := new(BlockFilterArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		id := api.xeth().NewLogFilter(args.Earliest, args.Latest, args.Skip, args.Max, args.Address, args.Topics)
		*reply = newHexNum(big.NewInt(int64(id)).Bytes())

	case "eth_newBlockFilter":
		*reply = newHexNum(api.xeth().NewBlockFilter())
	case "eth_newPendingTransactionFilter":
		*reply = newHexNum(api.xeth().NewTransactionFilter())
	case "eth_uninstallFilter":
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = api.xeth().UninstallFilter(args.Id)
	case "eth_getFilterChanges":
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		switch api.xeth().GetFilterType(args.Id) {
		case xeth.BlockFilterTy:
			*reply = NewHashesRes(api.xeth().BlockFilterChanged(args.Id))
		case xeth.TransactionFilterTy:
			*reply = NewHashesRes(api.xeth().TransactionFilterChanged(args.Id))
		case xeth.LogFilterTy:
			*reply = NewLogsRes(api.xeth().LogFilterChanged(args.Id))
		default:
			*reply = []string{} // reply empty string slice
		}
	case "eth_getFilterLogs":
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = NewLogsRes(api.xeth().Logs(args.Id))
	case "eth_getLogs":
		args := new(BlockFilterArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = NewLogsRes(api.xeth().AllLogs(args.Earliest, args.Latest, args.Skip, args.Max, args.Address, args.Topics))
	case "eth_getWork":
		api.xeth().SetMining(true, 0)
		*reply = api.xeth().RemoteMining().GetWork()
	case "eth_submitWork":
		args := new(SubmitWorkArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = api.xeth().RemoteMining().SubmitWork(args.Nonce, common.HexToHash(args.Digest), common.HexToHash(args.Header))
	case "db_putString":
		args := new(DbArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		if err := args.requirements(); err != nil {
			return err
		}

		api.xeth().DbPut([]byte(args.Database+args.Key), args.Value)

		*reply = true
	case "db_getString":
		args := new(DbArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		if err := args.requirements(); err != nil {
			return err
		}

		res, _ := api.xeth().DbGet([]byte(args.Database + args.Key))
		*reply = string(res)
	case "db_putHex":
		args := new(DbHexArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		if err := args.requirements(); err != nil {
			return err
		}

		api.xeth().DbPut([]byte(args.Database+args.Key), args.Value)
		*reply = true
	case "db_getHex":
		args := new(DbHexArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		if err := args.requirements(); err != nil {
			return err
		}

		res, _ := api.xeth().DbGet([]byte(args.Database + args.Key))
		*reply = newHexData(res)

	case "shh_version":
		// Short circuit if whisper is not running
		if api.xeth().Whisper() == nil {
			return NewNotAvailableError(req.Method, "whisper offline")
		}
		// Retrieves the currently running whisper protocol version
		*reply = api.xeth().WhisperVersion()

	case "shh_post":
		// Short circuit if whisper is not running
		if api.xeth().Whisper() == nil {
			return NewNotAvailableError(req.Method, "whisper offline")
		}
		// Injects a new message into the whisper network
		args := new(WhisperMessageArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		err := api.xeth().Whisper().Post(args.Payload, args.To, args.From, args.Topics, args.Priority, args.Ttl)
		if err != nil {
			return err
		}
		*reply = true

	case "shh_newIdentity":
		// Short circuit if whisper is not running
		if api.xeth().Whisper() == nil {
			return NewNotAvailableError(req.Method, "whisper offline")
		}
		// Creates a new whisper identity to use for sending/receiving messages
		*reply = api.xeth().Whisper().NewIdentity()

	case "shh_hasIdentity":
		// Short circuit if whisper is not running
		if api.xeth().Whisper() == nil {
			return NewNotAvailableError(req.Method, "whisper offline")
		}
		// Checks if an identity if owned or not
		args := new(WhisperIdentityArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = api.xeth().Whisper().HasIdentity(args.Identity)

	case "shh_newFilter":
		// Short circuit if whisper is not running
		if api.xeth().Whisper() == nil {
			return NewNotAvailableError(req.Method, "whisper offline")
		}
		// Create a new filter to watch and match messages with
		args := new(WhisperFilterArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		id := api.xeth().NewWhisperFilter(args.To, args.From, args.Topics)
		*reply = newHexNum(big.NewInt(int64(id)).Bytes())

	case "shh_uninstallFilter":
		// Short circuit if whisper is not running
		if api.xeth().Whisper() == nil {
			return NewNotAvailableError(req.Method, "whisper offline")
		}
		// Remove an existing filter watching messages
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = api.xeth().UninstallWhisperFilter(args.Id)

	case "shh_getFilterChanges":
		// Short circuit if whisper is not running
		if api.xeth().Whisper() == nil {
			return NewNotAvailableError(req.Method, "whisper offline")
		}
		// Retrieve all the new messages arrived since the last request
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = api.xeth().WhisperMessagesChanged(args.Id)

	case "shh_getMessages":
		// Short circuit if whisper is not running
		if api.xeth().Whisper() == nil {
			return NewNotAvailableError(req.Method, "whisper offline")
		}
		// Retrieve all the cached messages matching a specific, existing filter
		args := new(FilterIdArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}
		*reply = api.xeth().WhisperMessages(args.Id)

	case "eth_hashrate":
		*reply = newHexNum(api.xeth().HashRate())
	case "ext_disasm":
		args := new(SourceArgs)
		if err := json.Unmarshal(req.Params, &args); err != nil {
			return err
		}

		*reply = vm.Disasm(common.FromHex(args.Source))

	// case "eth_register":
	// 	// Placeholder for actual type
	// 	args := new(HashIndexArgs)
	// 	if err := json.Unmarshal(req.Params, &args); err != nil {
	// 		return err
	// 	}
	// 	*reply = api.xeth().Register(args.Hash)
	// case "eth_unregister":
	// 	args := new(HashIndexArgs)
	// 	if err := json.Unmarshal(req.Params, &args); err != nil {
	// 		return err
	// 	}
	// 	*reply = api.xeth().Unregister(args.Hash)
	// case "eth_watchTx":
	// 	args := new(HashIndexArgs)
	// 	if err := json.Unmarshal(req.Params, &args); err != nil {
	// 		return err
	// 	}
	// 	*reply = api.xeth().PullWatchTx(args.Hash)
	default:
		return NewNotImplementedError(req.Method)
	}

	// glog.V(logger.Detail).Infof("Reply: %v\n", reply)
	return nil
}

func (api *EthereumApi) doCall(params json.RawMessage) (string, string, error) {
	args := new(CallArgs)
	if err := json.Unmarshal(params, &args); err != nil {
		return "", "", err
	}

	return api.xethAtStateNum(args.BlockNumber).Call(args.From, args.To, args.Value.String(), args.Gas.String(), args.GasPrice.String(), args.Data)
}

package api

import (
	"bytes"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
)

const (
	EthApiVersion = "1.0"
)

// eth api provider
// See https://github.com/ethereum/wiki/wiki/JSON-RPC
type ethApi struct {
	xeth    *xeth.XEth
	methods map[string]ethhandler
	codec   codec.ApiCoder
}

// eth callback handler
type ethhandler func(*ethApi, *shared.Request) (interface{}, error)

var (
	ethMapping = map[string]ethhandler{
		"eth_accounts":                          (*ethApi).Accounts,
		"eth_blockNumber":                       (*ethApi).BlockNumber,
		"eth_getBalance":                        (*ethApi).GetBalance,
		"eth_protocolVersion":                   (*ethApi).ProtocolVersion,
		"eth_coinbase":                          (*ethApi).Coinbase,
		"eth_mining":                            (*ethApi).IsMining,
		"eth_gasPrice":                          (*ethApi).GasPrice,
		"eth_getStorage":                        (*ethApi).GetStorage,
		"eth_storageAt":                         (*ethApi).GetStorage,
		"eth_getStorageAt":                      (*ethApi).GetStorageAt,
		"eth_getTransactionCount":               (*ethApi).GetTransactionCount,
		"eth_getBlockTransactionCountByHash":    (*ethApi).GetBlockTransactionCountByHash,
		"eth_getBlockTransactionCountByNumber":  (*ethApi).GetBlockTransactionCountByNumber,
		"eth_getUncleCountByBlockHash":          (*ethApi).GetUncleCountByBlockHash,
		"eth_getUncleCountByBlockNumber":        (*ethApi).GetUncleCountByBlockNumber,
		"eth_getData":                           (*ethApi).GetData,
		"eth_getCode":                           (*ethApi).GetData,
		"eth_sign":                              (*ethApi).Sign,
		"eth_sendRawTransaction":                (*ethApi).PushTx,
		"eth_sendTransaction":                   (*ethApi).SendTransaction,
		"eth_transact":                          (*ethApi).SendTransaction,
		"eth_estimateGas":                       (*ethApi).EstimateGas,
		"eth_call":                              (*ethApi).Call,
		"eth_flush":                             (*ethApi).Flush,
		"eth_getBlockByHash":                    (*ethApi).GetBlockByHash,
		"eth_getBlockByNumber":                  (*ethApi).GetBlockByNumber,
		"eth_getTransactionByHash":              (*ethApi).GetTransactionByHash,
		"eth_getTransactionByBlockHashAndIndex": (*ethApi).GetTransactionByBlockHashAndIndex,
		"eth_getUncleByBlockHashAndIndex":       (*ethApi).GetUncleByBlockHashAndIndex,
		"eth_getUncleByBlockNumberAndIndex":     (*ethApi).GetUncleByBlockNumberAndIndex,
		"eth_getCompilers":                      (*ethApi).GetCompilers,
		"eth_compileSolidity":                   (*ethApi).CompileSolidity,
		"eth_newFilter":                         (*ethApi).NewFilter,
		"eth_newBlockFilter":                    (*ethApi).NewBlockFilter,
		"eth_newPendingTransactionFilter":       (*ethApi).NewPendingTransactionFilter,
		"eth_uninstallFilter":                   (*ethApi).UninstallFilter,
		"eth_getFilterChanges":                  (*ethApi).GetFilterChanges,
		"eth_getFilterLogs":                     (*ethApi).GetFilterLogs,
		"eth_getLogs":                           (*ethApi).GetLogs,
		"eth_hashrate":                          (*ethApi).Hashrate,
		"eth_getWork":                           (*ethApi).GetWork,
		"eth_submitWork":                        (*ethApi).SubmitWork,
	}
)

// create new ethApi instance
func NewEthApi(xeth *xeth.XEth, codec codec.Codec) *ethApi {
	return &ethApi{xeth, ethMapping, codec.New(nil)}
}

// collection with supported methods
func (self *ethApi) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

// Execute given request
func (self *ethApi) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := self.methods[req.Method]; ok {
		return callback(self, req)
	}

	return nil, shared.NewNotImplementedError(req.Method)
}

func (self *ethApi) Name() string {
	return EthApiName
}

func (self *ethApi) ApiVersion() string {
	return EthApiVersion
}

func (self *ethApi) Accounts(req *shared.Request) (interface{}, error) {
	return self.xeth.Accounts(), nil
}

func (self *ethApi) Hashrate(req *shared.Request) (interface{}, error) {
	return newHexNum(self.xeth.HashRate()), nil
}

func (self *ethApi) BlockNumber(req *shared.Request) (interface{}, error) {
	return self.xeth.CurrentBlock().Number(), nil
}

func (self *ethApi) GetBalance(req *shared.Request) (interface{}, error) {
	args := new(GetBalanceArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return self.xeth.AtStateNum(args.BlockNumber).BalanceAt(args.Address), nil
}

func (self *ethApi) ProtocolVersion(req *shared.Request) (interface{}, error) {
	return self.xeth.EthVersion(), nil
}

func (self *ethApi) Coinbase(req *shared.Request) (interface{}, error) {
	return newHexData(self.xeth.Coinbase()), nil
}

func (self *ethApi) IsMining(req *shared.Request) (interface{}, error) {
	return self.xeth.IsMining(), nil
}

func (self *ethApi) GasPrice(req *shared.Request) (interface{}, error) {
	return newHexNum(self.xeth.DefaultGasPrice().Bytes()), nil
}

func (self *ethApi) GetStorage(req *shared.Request) (interface{}, error) {
	args := new(GetStorageArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return self.xeth.AtStateNum(args.BlockNumber).State().SafeGet(args.Address).Storage(), nil
}

func (self *ethApi) GetStorageAt(req *shared.Request) (interface{}, error) {
	args := new(GetStorageAtArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return self.xeth.AtStateNum(args.BlockNumber).StorageAt(args.Address, args.Key), nil
}

func (self *ethApi) GetTransactionCount(req *shared.Request) (interface{}, error) {
	args := new(GetTxCountArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	count := self.xeth.AtStateNum(args.BlockNumber).TxCountAt(args.Address)
	return newHexNum(big.NewInt(int64(count)).Bytes()), nil
}

func (self *ethApi) GetBlockTransactionCountByHash(req *shared.Request) (interface{}, error) {
	args := new(HashArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := NewBlockRes(self.xeth.EthBlockByHash(args.Hash), false)
	if block == nil {
		return nil, nil
	} else {
		return newHexNum(big.NewInt(int64(len(block.Transactions))).Bytes()), nil
	}
}

func (self *ethApi) GetBlockTransactionCountByNumber(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := NewBlockRes(self.xeth.EthBlockByNumber(args.BlockNumber), false)
	if block == nil {
		return nil, nil
	} else {
		return newHexNum(big.NewInt(int64(len(block.Transactions))).Bytes()), nil
	}
}

func (self *ethApi) GetUncleCountByBlockHash(req *shared.Request) (interface{}, error) {
	args := new(HashArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByHash(args.Hash)
	br := NewBlockRes(block, false)
	if br == nil {
		return nil, nil
	}
	return newHexNum(big.NewInt(int64(len(br.Uncles))).Bytes()), nil
}

func (self *ethApi) GetUncleCountByBlockNumber(req *shared.Request) (interface{}, error) {
	args := new(BlockNumArg)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	br := NewBlockRes(block, false)
	if br == nil {
		return nil, nil
	}
	return newHexNum(big.NewInt(int64(len(br.Uncles))).Bytes()), nil
}

func (self *ethApi) GetData(req *shared.Request) (interface{}, error) {
	args := new(GetDataArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	v := self.xeth.AtStateNum(args.BlockNumber).CodeAtBytes(args.Address)
	return newHexData(v), nil
}

func (self *ethApi) Sign(req *shared.Request) (interface{}, error) {
	args := new(NewSigArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	v, err := self.xeth.Sign(args.From, args.Data, false)
	if err != nil {
		return nil, err
	}
	return v, nil
}


func (self *ethApi) PushTx(req *shared.Request) (interface{}, error) {
	args := new(NewDataArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	v, err := self.xeth.PushTx(args.Data)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (self *ethApi) SendTransaction(req *shared.Request) (interface{}, error) {
	args := new(NewTxArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	// nonce may be nil ("guess" mode)
	var nonce string
	if args.Nonce != nil {
		nonce = args.Nonce.String()
	}

	var gas, price string
	if args.Gas != nil {
		gas = args.Gas.String()
	}
	if args.GasPrice != nil {
		price = args.GasPrice.String()
	}
	v, err := self.xeth.Transact(args.From, args.To, nonce, args.Value.String(), gas, price, args.Data)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (self *ethApi) EstimateGas(req *shared.Request) (interface{}, error) {
	_, gas, err := self.doCall(req.Params)
	if err != nil {
		return nil, err
	}

	// TODO unwrap the parent method's ToHex call
	if len(gas) == 0 {
		return newHexNum(0), nil
	} else {
		return newHexNum(gas), nil
	}
}

func (self *ethApi) Call(req *shared.Request) (interface{}, error) {
	v, _, err := self.doCall(req.Params)
	if err != nil {
		return nil, err
	}

	// TODO unwrap the parent method's ToHex call
	if v == "0x0" {
		return newHexData([]byte{}), nil
	} else {
		return newHexData(common.FromHex(v)), nil
	}
}

func (self *ethApi) Flush(req *shared.Request) (interface{}, error) {
	return nil, shared.NewNotImplementedError(req.Method)
}

func (self *ethApi) doCall(params json.RawMessage) (string, string, error) {
	args := new(CallArgs)
	if err := self.codec.Decode(params, &args); err != nil {
		return "", "", err
	}

	return self.xeth.AtStateNum(args.BlockNumber).Call(args.From, args.To, args.Value.String(), args.Gas.String(), args.GasPrice.String(), args.Data)
}

func (self *ethApi) GetBlockByHash(req *shared.Request) (interface{}, error) {
	args := new(GetBlockByHashArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByHash(args.BlockHash)
	return NewBlockRes(block, args.IncludeTxs), nil
}

func (self *ethApi) GetBlockByNumber(req *shared.Request) (interface{}, error) {
	args := new(GetBlockByNumberArgs)
	if err := json.Unmarshal(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	br := NewBlockRes(block, args.IncludeTxs)
	// If request was for "pending", nil nonsensical fields
	if args.BlockNumber == -2 {
		br.BlockHash = nil
		br.BlockNumber = nil
		br.Miner = nil
		br.Nonce = nil
		br.LogsBloom = nil
	}
	return br, nil
}

func (self *ethApi) GetTransactionByHash(req *shared.Request) (interface{}, error) {
	args := new(HashArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	tx, bhash, bnum, txi := self.xeth.EthTransactionByHash(args.Hash)
	if tx != nil {
		v := NewTransactionRes(tx)
		// if the blockhash is 0, assume this is a pending transaction
		if bytes.Compare(bhash.Bytes(), bytes.Repeat([]byte{0}, 32)) != 0 {
			v.BlockHash = newHexData(bhash)
			v.BlockNumber = newHexNum(bnum)
			v.TxIndex = newHexNum(txi)
		}
		return v, nil
	}
	return nil, nil
}

func (self *ethApi) GetTransactionByBlockHashAndIndex(req *shared.Request) (interface{}, error) {
	args := new(HashIndexArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByHash(args.Hash)
	br := NewBlockRes(block, true)
	if br == nil {
		return nil, nil
	}

	if args.Index >= int64(len(br.Transactions)) || args.Index < 0 {
		return nil, nil
	} else {
		return br.Transactions[args.Index], nil
	}
}

func (self *ethApi) GetTransactionByBlockNumberAndIndex(req *shared.Request) (interface{}, error) {
	args := new(BlockNumIndexArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	v := NewBlockRes(block, true)
	if v == nil {
		return nil, nil
	}

	if args.Index >= int64(len(v.Transactions)) || args.Index < 0 {
		// return NewValidationError("Index", "does not exist")
		return nil, nil
	}
	return v.Transactions[args.Index], nil
}

func (self *ethApi) GetUncleByBlockHashAndIndex(req *shared.Request) (interface{}, error) {
	args := new(HashIndexArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	br := NewBlockRes(self.xeth.EthBlockByHash(args.Hash), false)
	if br == nil {
		return nil, nil
	}

	if args.Index >= int64(len(br.Uncles)) || args.Index < 0 {
		// return NewValidationError("Index", "does not exist")
		return nil, nil
	}

	return br.Uncles[args.Index], nil
}

func (self *ethApi) GetUncleByBlockNumberAndIndex(req *shared.Request) (interface{}, error) {
	args := new(BlockNumIndexArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByNumber(args.BlockNumber)
	v := NewBlockRes(block, true)

	if v == nil {
		return nil, nil
	}

	if args.Index >= int64(len(v.Uncles)) || args.Index < 0 {
		return nil, nil
	} else {
		return v.Uncles[args.Index], nil
	}
}

func (self *ethApi) GetCompilers(req *shared.Request) (interface{}, error) {
	var lang string
	if solc, _ := self.xeth.Solc(); solc != nil {
		lang = "Solidity"
	}
	c := []string{lang}
	return c, nil
}

func (self *ethApi) CompileSolidity(req *shared.Request) (interface{}, error) {
	solc, _ := self.xeth.Solc()
	if solc == nil {
		return nil, shared.NewNotAvailableError(req.Method, "solc (solidity compiler) not found")
	}

	args := new(SourceArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	contracts, err := solc.Compile(args.Source)
	if err != nil {
		return nil, err
	}
	return contracts, nil
}

func (self *ethApi) NewFilter(req *shared.Request) (interface{}, error) {
	args := new(BlockFilterArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	id := self.xeth.NewLogFilter(args.Earliest, args.Latest, args.Skip, args.Max, args.Address, args.Topics)
	return newHexNum(big.NewInt(int64(id)).Bytes()), nil
}

func (self *ethApi) NewBlockFilter(req *shared.Request) (interface{}, error) {
	return newHexNum(self.xeth.NewBlockFilter()), nil
}

func (self *ethApi) NewPendingTransactionFilter(req *shared.Request) (interface{}, error) {
	return newHexNum(self.xeth.NewTransactionFilter()), nil
}

func (self *ethApi) UninstallFilter(req *shared.Request) (interface{}, error) {
	args := new(FilterIdArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	return self.xeth.UninstallFilter(args.Id), nil
}

func (self *ethApi) GetFilterChanges(req *shared.Request) (interface{}, error) {
	args := new(FilterIdArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	switch self.xeth.GetFilterType(args.Id) {
	case xeth.BlockFilterTy:
		return NewHashesRes(self.xeth.BlockFilterChanged(args.Id)), nil
	case xeth.TransactionFilterTy:
		return NewHashesRes(self.xeth.TransactionFilterChanged(args.Id)), nil
	case xeth.LogFilterTy:
		return NewLogsRes(self.xeth.LogFilterChanged(args.Id)), nil
	default:
		return []string{}, nil // reply empty string slice
	}
}

func (self *ethApi) GetFilterLogs(req *shared.Request) (interface{}, error) {
	args := new(FilterIdArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return NewLogsRes(self.xeth.Logs(args.Id)), nil
}

func (self *ethApi) GetLogs(req *shared.Request) (interface{}, error) {
	args := new(BlockFilterArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	return NewLogsRes(self.xeth.AllLogs(args.Earliest, args.Latest, args.Skip, args.Max, args.Address, args.Topics)), nil
}

func (self *ethApi) GetWork(req *shared.Request) (interface{}, error) {
	self.xeth.SetMining(true, 0)
	return self.xeth.RemoteMining().GetWork(), nil
}

func (self *ethApi) SubmitWork(req *shared.Request) (interface{}, error) {
	args := new(SubmitWorkArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	return self.xeth.RemoteMining().SubmitWork(args.Nonce, common.HexToHash(args.Digest), common.HexToHash(args.Header)), nil
}

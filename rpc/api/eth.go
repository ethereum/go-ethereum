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

// eth api provider
// See https://github.com/ethereum/wiki/wiki/JSON-RPC
type EthApi struct {
	xeth    *xeth.XEth
	methods map[string]ethhandler
	codec   codec.ApiCoder
}

// eth callback handler
type ethhandler func(*EthApi, *shared.Request) (interface{}, error)

var (
	ethMapping = map[string]ethhandler{
		"eth_accounts":                          (*EthApi).Accounts,
		"eth_blockNumber":                       (*EthApi).BlockNumber,
		"eth_getBalance":                        (*EthApi).GetBalance,
		"eth_protocolVersion":                   (*EthApi).ProtocolVersion,
		"eth_coinbase":                          (*EthApi).Coinbase,
		"eth_mining":                            (*EthApi).IsMining,
		"eth_gasPrice":                          (*EthApi).GasPrice,
		"eth_getStorage":                        (*EthApi).GetStorage,
		"eth_storageAt":                         (*EthApi).GetStorage,
		"eth_getStorageAt":                      (*EthApi).GetStorageAt,
		"eth_getTransactionCount":               (*EthApi).GetTransactionCount,
		"eth_getBlockTransactionCountByHash":    (*EthApi).GetBlockTransactionCountByHash,
		"eth_getBlockTransactionCountByNumber":  (*EthApi).GetBlockTransactionCountByNumber,
		"eth_getUncleCountByBlockHash":          (*EthApi).GetUncleCountByBlockHash,
		"eth_getUncleCountByBlockNumber":        (*EthApi).GetUncleCountByBlockNumber,
		"eth_getData":                           (*EthApi).GetData,
		"eth_getCode":                           (*EthApi).GetData,
		"eth_sign":                              (*EthApi).Sign,
		"eth_sendTransaction":                   (*EthApi).SendTransaction,
		"eth_transact":                          (*EthApi).SendTransaction,
		"eth_estimateGas":                       (*EthApi).EstimateGas,
		"eth_call":                              (*EthApi).Call,
		"eth_flush":                             (*EthApi).Flush,
		"eth_getBlockByHash":                    (*EthApi).GetBlockByHash,
		"eth_getBlockByNumber":                  (*EthApi).GetBlockByNumber,
		"eth_getTransactionByHash":              (*EthApi).GetTransactionByHash,
		"eth_getTransactionByBlockHashAndIndex": (*EthApi).GetTransactionByBlockHashAndIndex,
		"eth_getUncleByBlockHashAndIndex":       (*EthApi).GetUncleByBlockHashAndIndex,
		"eth_getUncleByBlockNumberAndIndex":     (*EthApi).GetUncleByBlockNumberAndIndex,
		"eth_getCompilers":                      (*EthApi).GetCompilers,
		"eth_compileSolidity":                   (*EthApi).CompileSolidity,
		"eth_newFilter":                         (*EthApi).NewFilter,
		"eth_newBlockFilter":                    (*EthApi).NewBlockFilter,
		"eth_newPendingTransactionFilter":       (*EthApi).NewPendingTransactionFilter,
		"eth_uninstallFilter":                   (*EthApi).UninstallFilter,
		"eth_getFilterChanges":                  (*EthApi).GetFilterChanges,
		"eth_getFilterLogs":                     (*EthApi).GetFilterLogs,
		"eth_getLogs":                           (*EthApi).GetLogs,
		"eth_hashrate":                          (*EthApi).Hashrate,
		"eth_getWork":                           (*EthApi).GetWork,
		"eth_submitWork":                        (*EthApi).SubmitWork,
	}
)

// create new EthApi instance
func NewEthApi(xeth *xeth.XEth, codec codec.Codec) *EthApi {
	return &EthApi{xeth, ethMapping, codec.New(nil)}
}

// collection with supported methods
func (self *EthApi) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

// Execute given request
func (self *EthApi) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := self.methods[req.Method]; ok {
		return callback(self, req)
	}

	return nil, shared.NewNotImplementedError(req.Method)
}

func (self *EthApi) Accounts(req *shared.Request) (interface{}, error) {
	return self.xeth.Accounts(), nil
}

func (self *EthApi) Hashrate(req *shared.Request) (interface{}, error) {
	return newHexNum(self.xeth.HashRate()), nil
}

func (self *EthApi) BlockNumber(req *shared.Request) (interface{}, error) {
	return self.xeth.CurrentBlock().Number(), nil
}

func (self *EthApi) GetBalance(req *shared.Request) (interface{}, error) {
	args := new(GetBalanceArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return self.xeth.AtStateNum(args.BlockNumber).BalanceAt(args.Address), nil
}

func (self *EthApi) ProtocolVersion(req *shared.Request) (interface{}, error) {
	return self.xeth.EthVersion(), nil
}

func (self *EthApi) Coinbase(req *shared.Request) (interface{}, error) {
	return newHexData(self.xeth.Coinbase()), nil
}

func (self *EthApi) IsMining(req *shared.Request) (interface{}, error) {
	return self.xeth.IsMining(), nil
}

func (self *EthApi) GasPrice(req *shared.Request) (interface{}, error) {
	return newHexNum(xeth.DefaultGasPrice().Bytes()), nil
}

func (self *EthApi) GetStorage(req *shared.Request) (interface{}, error) {
	args := new(GetStorageArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return self.xeth.AtStateNum(args.BlockNumber).State().SafeGet(args.Address).Storage(), nil
}

func (self *EthApi) GetStorageAt(req *shared.Request) (interface{}, error) {
	args := new(GetStorageAtArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return self.xeth.AtStateNum(args.BlockNumber).StorageAt(args.Address, args.Key), nil
}

func (self *EthApi) GetTransactionCount(req *shared.Request) (interface{}, error) {
	args := new(GetTxCountArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	count := self.xeth.AtStateNum(args.BlockNumber).TxCountAt(args.Address)
	return newHexNum(big.NewInt(int64(count)).Bytes()), nil
}

func (self *EthApi) GetBlockTransactionCountByHash(req *shared.Request) (interface{}, error) {
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

func (self *EthApi) GetBlockTransactionCountByNumber(req *shared.Request) (interface{}, error) {
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

func (self *EthApi) GetUncleCountByBlockHash(req *shared.Request) (interface{}, error) {
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

func (self *EthApi) GetUncleCountByBlockNumber(req *shared.Request) (interface{}, error) {
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

func (self *EthApi) GetData(req *shared.Request) (interface{}, error) {
	args := new(GetDataArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	v := self.xeth.AtStateNum(args.BlockNumber).CodeAtBytes(args.Address)
	return newHexData(v), nil
}

func (self *EthApi) Sign(req *shared.Request) (interface{}, error) {
	args := new(NewSignArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	v, err := self.xeth.Sign(args.From, args.Data, false)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (self *EthApi) SendTransaction(req *shared.Request) (interface{}, error) {
	args := new(NewTxArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	// nonce may be nil ("guess" mode)
	var nonce string
	if args.Nonce != nil {
		nonce = args.Nonce.String()
	}

	v, err := self.xeth.Transact(args.From, args.To, nonce, args.Value.String(), args.Gas.String(), args.GasPrice.String(), args.Data)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (self *EthApi) EstimateGas(req *shared.Request) (interface{}, error) {
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

func (self *EthApi) Call(req *shared.Request) (interface{}, error) {
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

func (self *EthApi) Flush(req *shared.Request) (interface{}, error) {
	return nil, shared.NewNotImplementedError(req.Method)
}

func (self *EthApi) doCall(params json.RawMessage) (string, string, error) {
	args := new(CallArgs)
	if err := self.codec.Decode(params, &args); err != nil {
		return "", "", err
	}

	return self.xeth.AtStateNum(args.BlockNumber).Call(args.From, args.To, args.Value.String(), args.Gas.String(), args.GasPrice.String(), args.Data)
}

func (self *EthApi) GetBlockByHash(req *shared.Request) (interface{}, error) {
	args := new(GetBlockByHashArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByHash(args.BlockHash)
	return NewBlockRes(block, args.IncludeTxs), nil
}

func (self *EthApi) GetBlockByNumber(req *shared.Request) (interface{}, error) {
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

func (self *EthApi) GetTransactionByHash(req *shared.Request) (interface{}, error) {
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

func (self *EthApi) GetTransactionByBlockHashAndIndex(req *shared.Request) (interface{}, error) {
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

func (self *EthApi) GetTransactionByBlockNumberAndIndex(req *shared.Request) (interface{}, error) {
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

func (self *EthApi) GetUncleByBlockHashAndIndex(req *shared.Request) (interface{}, error) {
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

func (self *EthApi) GetUncleByBlockNumberAndIndex(req *shared.Request) (interface{}, error) {
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

func (self *EthApi) GetCompilers(req *shared.Request) (interface{}, error) {
	var lang string
	if solc, _ := self.xeth.Solc(); solc != nil {
		lang = "Solidity"
	}
	c := []string{lang}
	return c, nil
}

func (self *EthApi) CompileSolidity(req *shared.Request) (interface{}, error) {
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

func (self *EthApi) NewFilter(req *shared.Request) (interface{}, error) {
	args := new(BlockFilterArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	id := self.xeth.NewLogFilter(args.Earliest, args.Latest, args.Skip, args.Max, args.Address, args.Topics)
	return newHexNum(big.NewInt(int64(id)).Bytes()), nil
}

func (self *EthApi) NewBlockFilter(req *shared.Request) (interface{}, error) {
	return newHexNum(self.xeth.NewBlockFilter()), nil
}

func (self *EthApi) NewPendingTransactionFilter(req *shared.Request) (interface{}, error) {
	return newHexNum(self.xeth.NewTransactionFilter()), nil
}

func (self *EthApi) UninstallFilter(req *shared.Request) (interface{}, error) {
	args := new(FilterIdArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	return self.xeth.UninstallFilter(args.Id), nil
}

func (self *EthApi) GetFilterChanges(req *shared.Request) (interface{}, error) {
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

func (self *EthApi) GetFilterLogs(req *shared.Request) (interface{}, error) {
	args := new(FilterIdArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return NewLogsRes(self.xeth.Logs(args.Id)), nil
}

func (self *EthApi) GetLogs(req *shared.Request) (interface{}, error) {
	args := new(BlockFilterArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	return NewLogsRes(self.xeth.AllLogs(args.Earliest, args.Latest, args.Skip, args.Max, args.Address, args.Topics)), nil
}

func (self *EthApi) GetWork(req *shared.Request) (interface{}, error) {
	self.xeth.SetMining(true, 0)
	return self.xeth.RemoteMining().GetWork(), nil
}

func (self *EthApi) SubmitWork(req *shared.Request) (interface{}, error) {
	args := new(SubmitWorkArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	return self.xeth.RemoteMining().SubmitWork(args.Nonce, common.HexToHash(args.Digest), common.HexToHash(args.Header)), nil
}

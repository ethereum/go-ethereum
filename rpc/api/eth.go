package api

import (
	"encoding/json"
	"math/big"

	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
)

const (
	ethVersion = "1.0.0"
)

var (
	// mapping between methods and handlers
	ethMapping = map[string]ethhandler{
		"eth_accounts":                          (*eth).Accounts,
		"eth_blockNumber":                       (*eth).BlockNumber,
		"eth_getBalance":                        (*eth).GetBalance,
		"eth_protocolVersion":                   (*eth).ProtocolVersion,
		"eth_coinbase":                          (*eth).Coinbase,
		"eth_mining":                            (*eth).IsMining,
		"eth_gasPrice":                          (*eth).GasPrice,
		"eth_getStorage":                        (*eth).GetStorage,
		"eth_storageAt":                         (*eth).GetStorage,
		"eth_getStorageAt":                      (*eth).GetStorageAt,
		"eth_getTransactionCount":               (*eth).GetTransactionCount,
		"eth_getBlockTransactionCountByHash":    (*eth).GetBlockTransactionCountByHash,
		"eth_getBlockTransactionCountByNumber":  (*eth).GetBlockTransactionCountByNumber,
		"eth_getUncleCountByBlockHash":          (*eth).GetUncleCountByBlockHash,
		"eth_getUncleCountByBlockNumber":        (*eth).GetUncleCountByBlockNumber,
		"eth_getData":                           (*eth).GetData,
		"eth_getCode":                           (*eth).GetData,
		"eth_sign":                              (*eth).Sign,
		"eth_sendTransaction":                   (*eth).SendTransaction,
		"eth_transact":                          (*eth).SendTransaction,
		"eth_estimateGas":                       (*eth).EstimateGas,
		"eth_call":                              (*eth).Call,
		"eth_flush":                             (*eth).Flush,
		"eth_getBlockByHash":                    (*eth).GetBlockByHash,
		"eth_getBlockByNumber":                  (*eth).GetBlockByNumber,
		"eth_getTransactionByHash":              (*eth).GetTransactionByHash,
		"eth_getTransactionByBlockHashAndIndex": (*eth).GetTransactionByBlockHashAndIndex,
		"eth_getUncleByBlockHashAndIndex":       (*eth).GetUncleByBlockHashAndIndex,
		"eth_getUncleByBlockNumberAndIndex":     (*eth).GetUncleByBlockNumberAndIndex,
		"eth_getCompilers":                      (*eth).GetCompilers,
		"eth_compileSolidity":                   (*eth).CompileSolidity,
		"eth_newFilter":                         (*eth).NewFilter,
		"eth_newBlockFilter":                    (*eth).NewBlockFilter,
		"eth_newPendingTransactionFilter":       (*eth).NewPendingTransactionFilter,
		"eth_uninstallFilter":                   (*eth).UninstallFilter,
		"eth_getFilterChanges":                  (*eth).GetFilterChanges,
		"eth_getFilterLogs":                     (*eth).GetFilterLogs,
		"eth_getLogs":                           (*eth).GetLogs,
		"eth_getWork":                           (*eth).GetWork,
		"eth_submitWork":                        (*eth).SubmitWork,
	}
)

// eth callback handler
type ethhandler func(*eth, *shared.Request) (interface{}, error)

// eth api provider
type eth struct {
	xeth    *xeth.XEth
	methods map[string]ethhandler
	codec   codec.ApiCoder
}

// create a new eth api instance
func NewEth(xeth *xeth.XEth, coder codec.Codec) *eth {
	return &eth{
		xeth:    xeth,
		methods: ethMapping,
		codec:   coder.New(nil),
	}
}

// collection with supported methods
func (self *eth) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

// Execute given request
func (self *eth) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := self.methods[req.Method]; ok {
		return callback(self, req)
	}

	return nil, shared.NewNotImplementedError(req.Method)
}

// Version of the API this instance provides
func (self *eth) Version() string {
	return ethVersion
}

func (self *eth) Accounts(req *shared.Request) (interface{}, error) {
	return self.xeth.Accounts(), nil
}

func (self *eth) BlockNumber(req *shared.Request) (interface{}, error) {
	return self.xeth.CurrentBlock().Number(), nil
}

func (self *eth) GetBalance(req *shared.Request) (interface{}, error) {
	args := new(GetBalanceArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return self.xeth.AtStateNum(args.BlockNumber).BalanceAt(args.Address), nil
}

func (self *eth) ProtocolVersion(req *shared.Request) (interface{}, error) {
	return self.xeth.EthVersion(), nil
}

func (self *eth) Coinbase(req *shared.Request) (interface{}, error) {
	return newHexData(self.xeth.Coinbase()), nil
}

func (self *eth) IsMining(req *shared.Request) (interface{}, error) {
	return self.xeth.IsMining(), nil
}

func (self *eth) GasPrice(req *shared.Request) (interface{}, error) {
	return newHexNum(xeth.DefaultGasPrice().Bytes()), nil
}

func (self *eth) GetStorage(req *shared.Request) (interface{}, error) {
	args := new(GetStorageArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return self.xeth.AtStateNum(args.BlockNumber).State().SafeGet(args.Address).Storage(), nil
}

func (self *eth) GetStorageAt(req *shared.Request) (interface{}, error) {
	args := new(GetStorageAtArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return self.xeth.AtStateNum(args.BlockNumber).StorageAt(args.Address, args.Key), nil
}

func (self *eth) GetTransactionCount(req *shared.Request) (interface{}, error) {
	args := new(GetTxCountArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	count := self.xeth.AtStateNum(args.BlockNumber).TxCountAt(args.Address)
	return newHexNum(big.NewInt(int64(count)).Bytes()), nil
}

func (self *eth) GetBlockTransactionCountByHash(req *shared.Request) (interface{}, error) {
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

func (self *eth) GetBlockTransactionCountByNumber(req *shared.Request) (interface{}, error) {
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

func (self *eth) GetUncleCountByBlockHash(req *shared.Request) (interface{}, error) {
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

func (self *eth) GetUncleCountByBlockNumber(req *shared.Request) (interface{}, error) {
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

func (self *eth) GetData(req *shared.Request) (interface{}, error) {
	args := new(GetDataArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	v := self.xeth.AtStateNum(args.BlockNumber).CodeAtBytes(args.Address)
	return newHexData(v), nil
}

func (self *eth) Sign(req *shared.Request) (interface{}, error) {
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

func (self *eth) SendTransaction(req *shared.Request) (interface{}, error) {
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

func (self *eth) EstimateGas(req *shared.Request) (interface{}, error) {
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

func (self *eth) Call(req *shared.Request) (interface{}, error) {
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

func (self *eth) Flush(req *shared.Request) (interface{}, error) {
	return nil, shared.NewNotImplementedError(req.Method)
}

func (self *eth) doCall(params json.RawMessage) (string, string, error) {
	args := new(CallArgs)
	if err := self.codec.Decode(params, &args); err != nil {
		return "", "", err
	}

	return self.xeth.AtStateNum(args.BlockNumber).Call(args.From, args.To, args.Value.String(), args.Gas.String(), args.GasPrice.String(), args.Data)
}

func (self *eth) GetBlockByHash(req *shared.Request) (interface{}, error) {
	args := new(GetBlockByHashArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	block := self.xeth.EthBlockByHash(args.BlockHash)
	return NewBlockRes(block, args.IncludeTxs), nil
}

func (self *eth) GetBlockByNumber(req *shared.Request) (interface{}, error) {
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

func (self *eth) GetTransactionByHash(req *shared.Request) (interface{}, error) {
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

func (self *eth) GetTransactionByBlockHashAndIndex(req *shared.Request) (interface{}, error) {
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

func (self *eth) GetTransactionByBlockNumberAndIndex(req *shared.Request) (interface{}, error) {
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

func (self *eth) GetUncleByBlockHashAndIndex(req *shared.Request) (interface{}, error) {
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

func (self *eth) GetUncleByBlockNumberAndIndex(req *shared.Request) (interface{}, error) {
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

func (self *eth) GetCompilers(req *shared.Request) (interface{}, error) {
	var lang string
	if solc, _ := self.xeth.Solc(); solc != nil {
		lang = "Solidity"
	}
	c := []string{lang}
	return c, nil
}

func (self *eth) CompileSolidity(req *shared.Request) (interface{}, error) {
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

func (self *eth) NewFilter(req *shared.Request) (interface{}, error) {
	args := new(BlockFilterArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	id := self.xeth.NewLogFilter(args.Earliest, args.Latest, args.Skip, args.Max, args.Address, args.Topics)
	return newHexNum(big.NewInt(int64(id)).Bytes()), nil
}

func (self *eth) NewBlockFilter(req *shared.Request) (interface{}, error) {
	return newHexNum(self.xeth.NewBlockFilter()), nil
}

func (self *eth) NewPendingTransactionFilter(req *shared.Request) (interface{}, error) {
	return newHexNum(self.xeth.NewTransactionFilter()), nil
}

func (self *eth) UninstallFilter(req *shared.Request) (interface{}, error) {
	args := new(FilterIdArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	return self.xeth.UninstallFilter(args.Id), nil
}

func (self *eth) GetFilterChanges(req *shared.Request) (interface{}, error) {
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

func (self *eth) GetFilterLogs(req *shared.Request) (interface{}, error) {
	args := new(FilterIdArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return NewLogsRes(self.xeth.Logs(args.Id)), nil
}

func (self *eth) GetLogs(req *shared.Request) (interface{}, error) {
	args := new(BlockFilterArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	return NewLogsRes(self.xeth.AllLogs(args.Earliest, args.Latest, args.Skip, args.Max, args.Address, args.Topics)), nil
}

func (self *eth) GetWork(req *shared.Request) (interface{}, error) {
	self.xeth.SetMining(true, 0)
	return self.xeth.RemoteMining().GetWork(), nil
}

func (self *eth) SubmitWork(req *shared.Request) (interface{}, error) {
	args := new(SubmitWorkArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}
	return self.xeth.RemoteMining().SubmitWork(args.Nonce, common.HexToHash(args.Digest), common.HexToHash(args.Header)), nil
}

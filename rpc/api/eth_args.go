// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

const (
	defaultLogLimit  = 100
	defaultLogOffset = 0
)

type GetBalanceArgs struct {
	Address     string
	BlockNumber int64
}

func (args *GetBalanceArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("address", "not a string")
	}
	args.Address = addstr

	if len(obj) > 1 {
		if err := blockHeight(obj[1], &args.BlockNumber); err != nil {
			return err
		}
	} else {
		args.BlockNumber = -1
	}

	return nil
}

type GetStorageArgs struct {
	Address     string
	BlockNumber int64
}

func (args *GetStorageArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("address", "not a string")
	}
	args.Address = addstr

	if len(obj) > 1 {
		if err := blockHeight(obj[1], &args.BlockNumber); err != nil {
			return err
		}
	} else {
		args.BlockNumber = -1
	}

	return nil
}

type GetStorageAtArgs struct {
	Address     string
	BlockNumber int64
	Key         string
}

func (args *GetStorageAtArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 2)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("address", "not a string")
	}
	args.Address = addstr

	keystr, ok := obj[1].(string)
	if !ok {
		return shared.NewInvalidTypeError("key", "not a string")
	}
	args.Key = keystr

	if len(obj) > 2 {
		if err := blockHeight(obj[2], &args.BlockNumber); err != nil {
			return err
		}
	} else {
		args.BlockNumber = -1
	}

	return nil
}

type GetTxCountArgs struct {
	Address     string
	BlockNumber int64
}

func (args *GetTxCountArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("address", "not a string")
	}
	args.Address = addstr

	if len(obj) > 1 {
		if err := blockHeight(obj[1], &args.BlockNumber); err != nil {
			return err
		}
	} else {
		args.BlockNumber = -1
	}

	return nil
}

type SubmitHashRateArgs struct {
	Id   string
	Rate uint64
}

func (args *SubmitHashRateArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 2)
	}

	arg0, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("hash", "not a string")
	}
	args.Id = arg0

	arg1, ok := obj[1].(string)
	if !ok {
		return shared.NewInvalidTypeError("rate", "not a string")
	}

	args.Rate = common.String2Big(arg1).Uint64()

	return nil
}

type HashArgs struct {
	Hash string
}

func (args *HashArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	arg0, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("hash", "not a string")
	}
	args.Hash = arg0

	return nil
}

type BlockNumArg struct {
	BlockNumber int64
}

func (args *BlockNumArg) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	if err := blockHeight(obj[0], &args.BlockNumber); err != nil {
		return err
	}

	return nil
}

type GetDataArgs struct {
	Address     string
	BlockNumber int64
}

func (args *GetDataArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("address", "not a string")
	}
	args.Address = addstr

	if len(obj) > 1 {
		if err := blockHeight(obj[1], &args.BlockNumber); err != nil {
			return err
		}
	} else {
		args.BlockNumber = -1
	}

	return nil
}

type NewDataArgs struct {
	Data string
}

func (args *NewDataArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}

	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	// Check for sufficient params
	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	data, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("data", "not a string")
	}
	args.Data = data

	if len(args.Data) == 0 {
		return shared.NewValidationError("data", "is required")
	}

	return nil
}

type NewSigArgs struct {
	From string
	Data string
}

func (args *NewSigArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}

	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	// Check for sufficient params
	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	from, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("from", "not a string")
	}
	args.From = from

	if len(args.From) == 0 {
		return shared.NewValidationError("from", "is required")
	}

	data, ok := obj[1].(string)
	if !ok {
		return shared.NewInvalidTypeError("data", "not a string")
	}
	args.Data = data

	if len(args.Data) == 0 {
		return shared.NewValidationError("data", "is required")
	}

	return nil
}

type NewTxArgs struct {
	From     string
	To       string
	Nonce    *big.Int
	Value    *big.Int
	Gas      *big.Int
	GasPrice *big.Int
	Data     string

	BlockNumber int64
}

func (args *NewTxArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []json.RawMessage
	var ext struct {
		From     string
		To       string
		Nonce    interface{}
		Value    interface{}
		Gas      interface{}
		GasPrice interface{}
		Data     string
	}

	// Decode byte slice to array of RawMessages
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	// Check for sufficient params
	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	// Decode 0th RawMessage to temporary struct
	if err := json.Unmarshal(obj[0], &ext); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(ext.From) == 0 {
		return shared.NewValidationError("from", "is required")
	}

	args.From = ext.From
	args.To = ext.To
	args.Data = ext.Data

	var num *big.Int
	if ext.Nonce != nil {
		num, err = numString(ext.Nonce)
		if err != nil {
			return err
		}
	}
	args.Nonce = num

	if ext.Value == nil {
		num = big.NewInt(0)
	} else {
		num, err = numString(ext.Value)
		if err != nil {
			return err
		}
	}
	args.Value = num

	num = nil
	if ext.Gas != nil {
		if num, err = numString(ext.Gas); err != nil {
			return err
		}
	}
	args.Gas = num

	num = nil
	if ext.GasPrice != nil {
		if num, err = numString(ext.GasPrice); err != nil {
			return err
		}
	}
	args.GasPrice = num

	// Check for optional BlockNumber param
	if len(obj) > 1 {
		if err := blockHeightFromJson(obj[1], &args.BlockNumber); err != nil {
			return err
		}
	} else {
		args.BlockNumber = -1
	}

	return nil
}

type SourceArgs struct {
	Source string
}

func (args *SourceArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	arg0, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("source code", "not a string")
	}
	args.Source = arg0

	return nil
}

type CallArgs struct {
	From     string
	To       string
	Value    *big.Int
	Gas      *big.Int
	GasPrice *big.Int
	Data     string

	BlockNumber int64
}

func (args *CallArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []json.RawMessage
	var ext struct {
		From     string
		To       string
		Value    interface{}
		Gas      interface{}
		GasPrice interface{}
		Data     string
	}

	// Decode byte slice to array of RawMessages
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	// Check for sufficient params
	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	// Decode 0th RawMessage to temporary struct
	if err := json.Unmarshal(obj[0], &ext); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	args.From = ext.From
	args.To = ext.To

	var num *big.Int
	if ext.Value == nil {
		num = big.NewInt(0)
	} else {
		if num, err = numString(ext.Value); err != nil {
			return err
		}
	}
	args.Value = num

	if ext.Gas != nil {
		if num, err = numString(ext.Gas); err != nil {
			return err
		}
	} else {
		num = nil
	}
	args.Gas = num

	if ext.GasPrice != nil {
		if num, err = numString(ext.GasPrice); err != nil {
			return err
		}
	} else {
		num = nil
	}
	args.GasPrice = num

	args.Data = ext.Data

	// Check for optional BlockNumber param
	if len(obj) > 1 {
		if err := blockHeightFromJson(obj[1], &args.BlockNumber); err != nil {
			return err
		}
	} else {
		args.BlockNumber = -1
	}

	return nil
}

type HashIndexArgs struct {
	Hash  string
	Index int64
}

func (args *HashIndexArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 2)
	}

	arg0, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("hash", "not a string")
	}
	args.Hash = arg0

	arg1, ok := obj[1].(string)
	if !ok {
		return shared.NewInvalidTypeError("index", "not a string")
	}
	args.Index = common.Big(arg1).Int64()

	return nil
}

type BlockNumIndexArgs struct {
	BlockNumber int64
	Index       int64
}

func (args *BlockNumIndexArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 2)
	}

	if err := blockHeight(obj[0], &args.BlockNumber); err != nil {
		return err
	}

	var arg1 *big.Int
	if arg1, err = numString(obj[1]); err != nil {
		return err
	}
	args.Index = arg1.Int64()

	return nil
}

type GetBlockByHashArgs struct {
	BlockHash  string
	IncludeTxs bool
}

func (args *GetBlockByHashArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}

	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 2)
	}

	argstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("blockHash", "not a string")
	}
	args.BlockHash = argstr

	args.IncludeTxs = obj[1].(bool)

	return nil
}

type GetBlockByNumberArgs struct {
	BlockNumber int64
	IncludeTxs  bool
}

func (args *GetBlockByNumberArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 2)
	}

	if err := blockHeight(obj[0], &args.BlockNumber); err != nil {
		return err
	}

	args.IncludeTxs = obj[1].(bool)

	return nil
}

type BlockFilterArgs struct {
	Earliest int64
	Latest   int64
	Address  []string
	Topics   [][]string
	Skip     int
	Max      int
}

func (args *BlockFilterArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []struct {
		FromBlock interface{} `json:"fromBlock"`
		ToBlock   interface{} `json:"toBlock"`
		Limit     interface{} `json:"limit"`
		Offset    interface{} `json:"offset"`
		Address   interface{} `json:"address"`
		Topics    interface{} `json:"topics"`
	}

	if err = json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	// args.Earliest, err = toNumber(obj[0].ToBlock)
	// if err != nil {
	// 	return shared.NewDecodeParamError(fmt.Sprintf("FromBlock %v", err))
	// }
	// args.Latest, err = toNumber(obj[0].FromBlock)
	// if err != nil {
	// 	return shared.NewDecodeParamError(fmt.Sprintf("ToBlock %v", err))

	var num int64
	var numBig *big.Int

	// if blank then latest
	if obj[0].FromBlock == nil {
		num = -1
	} else {
		if err := blockHeight(obj[0].FromBlock, &num); err != nil {
			return err
		}
	}
	// if -2 or other "silly" number, use latest
	if num < 0 {
		args.Earliest = -1 //latest block
	} else {
		args.Earliest = num
	}

	// if blank than latest
	if obj[0].ToBlock == nil {
		num = -1
	} else {
		if err := blockHeight(obj[0].ToBlock, &num); err != nil {
			return err
		}
	}

	if num == -2 {
		return fmt.Errorf("\"pending\" is unsupported")
	} else if num < -2 {
		return fmt.Errorf("Invalid to block number")
	}

	args.Latest = num

	if obj[0].Limit == nil {
		numBig = big.NewInt(defaultLogLimit)
	} else {
		if numBig, err = numString(obj[0].Limit); err != nil {
			return err
		}
	}
	args.Max = int(numBig.Int64())

	if obj[0].Offset == nil {
		numBig = big.NewInt(defaultLogOffset)
	} else {
		if numBig, err = numString(obj[0].Offset); err != nil {
			return err
		}
	}
	args.Skip = int(numBig.Int64())

	if obj[0].Address != nil {
		marg, ok := obj[0].Address.([]interface{})
		if ok {
			v := make([]string, len(marg))
			for i, arg := range marg {
				argstr, ok := arg.(string)
				if !ok {
					return shared.NewInvalidTypeError(fmt.Sprintf("address[%d]", i), "is not a string")
				}
				v[i] = argstr
			}
			args.Address = v
		} else {
			argstr, ok := obj[0].Address.(string)
			if ok {
				v := make([]string, 1)
				v[0] = argstr
				args.Address = v
			} else {
				return shared.NewInvalidTypeError("address", "is not a string or array")
			}
		}
	}

	if obj[0].Topics != nil {
		other, ok := obj[0].Topics.([]interface{})
		if ok {
			topicdbl := make([][]string, len(other))
			for i, iv := range other {
				if argstr, ok := iv.(string); ok {
					// Found a string, push into first element of array
					topicsgl := make([]string, 1)
					topicsgl[0] = argstr
					topicdbl[i] = topicsgl
				} else if argarray, ok := iv.([]interface{}); ok {
					// Found an array of other
					topicdbl[i] = make([]string, len(argarray))
					for j, jv := range argarray {
						if v, ok := jv.(string); ok {
							topicdbl[i][j] = v
						} else if jv == nil {
							topicdbl[i][j] = ""
						} else {
							return shared.NewInvalidTypeError(fmt.Sprintf("topic[%d][%d]", i, j), "is not a string")
						}
					}
				} else if iv == nil {
					topicdbl[i] = []string{""}
				} else {
					return shared.NewInvalidTypeError(fmt.Sprintf("topic[%d]", i), "not a string or array")
				}
			}
			args.Topics = topicdbl
			return nil
		} else {
			return shared.NewInvalidTypeError("topic", "is not a string or array")
		}
	}

	return nil
}

type FilterIdArgs struct {
	Id int
}

func (args *FilterIdArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	var num *big.Int
	if num, err = numString(obj[0]); err != nil {
		return err
	}
	args.Id = int(num.Int64())

	return nil
}

type LogRes struct {
	Address          *hexdata   `json:"address"`
	Topics           []*hexdata `json:"topics"`
	Data             *hexdata   `json:"data"`
	BlockNumber      *hexnum    `json:"blockNumber"`
	LogIndex         *hexnum    `json:"logIndex"`
	BlockHash        *hexdata   `json:"blockHash"`
	TransactionHash  *hexdata   `json:"transactionHash"`
	TransactionIndex *hexnum    `json:"transactionIndex"`
}

func NewLogRes(log *vm.Log) LogRes {
	var l LogRes
	l.Topics = make([]*hexdata, len(log.Topics))
	for j, topic := range log.Topics {
		l.Topics[j] = newHexData(topic)
	}
	l.Address = newHexData(log.Address)
	l.Data = newHexData(log.Data)
	l.BlockNumber = newHexNum(log.BlockNumber)
	l.LogIndex = newHexNum(log.Index)
	l.TransactionHash = newHexData(log.TxHash)
	l.TransactionIndex = newHexNum(log.TxIndex)
	l.BlockHash = newHexData(log.BlockHash)

	return l
}

func NewLogsRes(logs vm.Logs) (ls []LogRes) {
	ls = make([]LogRes, len(logs))

	for i, log := range logs {
		ls[i] = NewLogRes(log)
	}

	return
}

func NewHashesRes(hs []common.Hash) []string {
	hashes := make([]string, len(hs))

	for i, hash := range hs {
		hashes[i] = hash.Hex()
	}

	return hashes
}

type SubmitWorkArgs struct {
	Nonce  uint64
	Header string
	Digest string
}

func (args *SubmitWorkArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err = json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 3 {
		return shared.NewInsufficientParamsError(len(obj), 3)
	}

	var objstr string
	var ok bool
	if objstr, ok = obj[0].(string); !ok {
		return shared.NewInvalidTypeError("nonce", "not a string")
	}

	args.Nonce = common.String2Big(objstr).Uint64()
	if objstr, ok = obj[1].(string); !ok {
		return shared.NewInvalidTypeError("header", "not a string")
	}

	args.Header = objstr

	if objstr, ok = obj[2].(string); !ok {
		return shared.NewInvalidTypeError("digest", "not a string")
	}

	args.Digest = objstr

	return nil
}

type tx struct {
	tx *types.Transaction

	To       string `json:"to"`
	From     string `json:"from"`
	Nonce    string `json:"nonce"`
	Value    string `json:"value"`
	Data     string `json:"data"`
	GasLimit string `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Hash     string `json:"hash"`
}

func newTx(t *types.Transaction) *tx {
	from, _ := t.From()
	var to string
	if t := t.To(); t != nil {
		to = t.Hex()
	}

	return &tx{
		tx:       t,
		To:       to,
		From:     from.Hex(),
		Value:    t.Value().String(),
		Nonce:    strconv.Itoa(int(t.Nonce())),
		Data:     "0x" + common.Bytes2Hex(t.Data()),
		GasLimit: t.Gas().String(),
		GasPrice: t.GasPrice().String(),
		Hash:     t.Hash().Hex(),
	}
}

type ResendArgs struct {
	Tx       *tx
	GasPrice string
	GasLimit string
}

func (tx *tx) UnmarshalJSON(b []byte) (err error) {
	var fields map[string]interface{}
	if err := json.Unmarshal(b, &fields); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	var (
		nonce            uint64
		to               common.Address
		amount           = new(big.Int).Set(common.Big0)
		gasLimit         = new(big.Int).Set(common.Big0)
		gasPrice         = new(big.Int).Set(common.Big0)
		data             []byte
		contractCreation = true
	)

	if val, found := fields["Hash"]; found {
		if hashVal, ok := val.(string); ok {
			tx.Hash = hashVal
		}
	}

	if val, found := fields["To"]; found {
		if strVal, ok := val.(string); ok && len(strVal) > 0 {
			tx.To = strVal
			to = common.HexToAddress(strVal)
			contractCreation = false
		}
	}

	if val, found := fields["From"]; found {
		if strVal, ok := val.(string); ok {
			tx.From = strVal
		}
	}

	if val, found := fields["Nonce"]; found {
		if strVal, ok := val.(string); ok {
			tx.Nonce = strVal
			if nonce, err = strconv.ParseUint(strVal, 10, 64); err != nil {
				return shared.NewDecodeParamError(fmt.Sprintf("Unable to decode tx.Nonce - %v", err))
			}
		}
	} else {
		return shared.NewDecodeParamError("tx.Nonce not found")
	}

	var parseOk bool
	if val, found := fields["Value"]; found {
		if strVal, ok := val.(string); ok {
			tx.Value = strVal
			if _, parseOk = amount.SetString(strVal, 0); !parseOk {
				return shared.NewDecodeParamError(fmt.Sprintf("Unable to decode tx.Amount - %v", err))
			}
		}
	}

	if val, found := fields["Data"]; found {
		if strVal, ok := val.(string); ok {
			tx.Data = strVal
			if strings.HasPrefix(strVal, "0x") {
				data = common.Hex2Bytes(strVal[2:])
			} else {
				data = common.Hex2Bytes(strVal)
			}
		}
	}

	if val, found := fields["GasLimit"]; found {
		if strVal, ok := val.(string); ok {
			tx.GasLimit = strVal
			if _, parseOk = gasLimit.SetString(strVal, 0); !parseOk {
				return shared.NewDecodeParamError(fmt.Sprintf("Unable to decode tx.GasLimit - %v", err))
			}
		}
	}

	if val, found := fields["GasPrice"]; found {
		if strVal, ok := val.(string); ok {
			tx.GasPrice = strVal
			if _, parseOk = gasPrice.SetString(strVal, 0); !parseOk {
				return shared.NewDecodeParamError(fmt.Sprintf("Unable to decode tx.GasPrice - %v", err))
			}
		}
	}

	if contractCreation {
		tx.tx = types.NewContractCreation(nonce, amount, gasLimit, gasPrice, data)
	} else {
		tx.tx = types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, data)
	}

	return nil
}

func (args *ResendArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err = json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	data, err := json.Marshal(obj[0])
	if err != nil {
		return shared.NewDecodeParamError("Unable to parse transaction object")
	}

	trans := new(tx)
	err = json.Unmarshal(data, trans)
	if err != nil {
		return shared.NewDecodeParamError("Unable to parse transaction object")
	}

	if trans == nil || trans.tx == nil {
		return shared.NewDecodeParamError("Unable to parse transaction object")
	}

	gasLimit, gasPrice := trans.GasLimit, trans.GasPrice

	if len(obj) > 1 && obj[1] != nil {
		if gp, ok := obj[1].(string); ok {
			gasPrice = gp
		} else {
			return shared.NewInvalidTypeError("gasPrice", "not a string")
		}
	}
	if len(obj) > 2 && obj[2] != nil {
		if gl, ok := obj[2].(string); ok {
			gasLimit = gl
		} else {
			return shared.NewInvalidTypeError("gasLimit", "not a string")
		}
	}

	args.Tx = trans
	args.GasPrice = gasPrice
	args.GasLimit = gasLimit

	return nil
}

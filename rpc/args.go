package rpc

import (
	"bytes"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func blockNumber(raw json.RawMessage, number *int64) (err error) {
	var str string
	if err = json.Unmarshal(raw, &str); err != nil {
		return NewDecodeParamError(err.Error())
	}

	switch str {
	case "latest":
		*number = -1
	case "pending":
		*number = 0
	default:
		*number = common.String2Big(str).Int64()
	}
	return nil
}

type GetBlockByHashArgs struct {
	BlockHash    string
	Transactions bool
}

func (args *GetBlockByHashArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	r := bytes.NewReader(b)
	if err := json.NewDecoder(r).Decode(&obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	argstr, ok := obj[0].(string)
	if !ok {
		return NewDecodeParamError("BlockHash not a string")
	}
	args.BlockHash = argstr

	if len(obj) > 1 {
		args.Transactions = obj[1].(bool)
	}

	return nil
}

type GetBlockByNumberArgs struct {
	BlockNumber  int64
	Transactions bool
}

func (args *GetBlockByNumberArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	r := bytes.NewReader(b)
	if err := json.NewDecoder(r).Decode(&obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	if v, ok := obj[0].(float64); ok {
		args.BlockNumber = int64(v)
	} else {
		args.BlockNumber = common.Big(obj[0].(string)).Int64()
	}

	if len(obj) > 1 {
		args.Transactions = obj[1].(bool)
	}

	return nil
}

type NewTxArgs struct {
	From     string
	To       string
	Value    *big.Int
	Gas      *big.Int
	GasPrice *big.Int
	Data     string

	BlockNumber int64
}

func (args *NewTxArgs) UnmarshalJSON(b []byte) (err error) {
	var obj struct{ From, To, Value, Gas, GasPrice, Data string }
	if err = UnmarshalRawMessages(b, &obj, &args.BlockNumber); err != nil {
		return err
	}

	args.From = obj.From
	args.To = obj.To
	args.Value = common.Big(obj.Value)
	args.Gas = common.Big(obj.Gas)
	args.GasPrice = common.Big(obj.GasPrice)
	args.Data = obj.Data

	return nil
}

type GetStorageArgs struct {
	Address     string
	BlockNumber int64
}

func (args *GetStorageArgs) UnmarshalJSON(b []byte) (err error) {
	if err = UnmarshalRawMessages(b, &args.Address, &args.BlockNumber); err != nil {
		return NewDecodeParamError(err.Error())
	}

	return nil
}

func (args *GetStorageArgs) requirements() error {
	if len(args.Address) == 0 {
		return NewValidationError("Address", "cannot be blank")
	}
	return nil
}

type GetStorageAtArgs struct {
	Address     string
	Key         string
	BlockNumber int64
}

func (args *GetStorageAtArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []string
	if err = UnmarshalRawMessages(b, &obj, &args.BlockNumber); err != nil {
		return NewDecodeParamError(err.Error())
	}
	if len(obj) < 2 {
		return NewInsufficientParamsError(len(obj), 2)
	}

	args.Address = obj[0]
	args.Key = obj[1]

	return nil
}

func (args *GetStorageAtArgs) requirements() error {
	if len(args.Address) == 0 {
		return NewValidationError("Address", "cannot be blank")
	}

	if len(args.Key) == 0 {
		return NewValidationError("Key", "cannot be blank")
	}
	return nil
}

type GetTxCountArgs struct {
	Address     string
	BlockNumber int64
}

func (args *GetTxCountArgs) UnmarshalJSON(b []byte) (err error) {
	if err = UnmarshalRawMessages(b, &args.Address, &args.BlockNumber); err != nil {
		return NewDecodeParamError(err.Error())
	}

	return nil
}

func (args *GetTxCountArgs) requirements() error {
	if len(args.Address) == 0 {
		return NewValidationError("Address", "cannot be blank")
	}
	return nil
}

type GetBalanceArgs struct {
	Address     string
	BlockNumber int64
}

func (args *GetBalanceArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	r := bytes.NewReader(b)
	if err := json.NewDecoder(r).Decode(&obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return NewDecodeParamError("Address is not a string")
	}
	args.Address = addstr

	if len(obj) > 1 {
		if obj[1].(string) == "latest" {
			args.BlockNumber = -1
		} else {
			args.BlockNumber = common.Big(obj[1].(string)).Int64()
		}
	}

	// if err = UnmarshalRawMessages(b, &args.Address, &args.BlockNumber); err != nil {
	// 	return NewDecodeParamError(err.Error())
	// }

	return nil
}

func (args *GetBalanceArgs) requirements() error {
	if len(args.Address) == 0 {
		return NewValidationError("Address", "cannot be blank")
	}
	return nil
}

type GetDataArgs struct {
	Address     string
	BlockNumber int64
}

func (args *GetDataArgs) UnmarshalJSON(b []byte) (err error) {
	if err = UnmarshalRawMessages(b, &args.Address, &args.BlockNumber); err != nil {
		return NewDecodeParamError(err.Error())
	}

	return nil
}

func (args *GetDataArgs) requirements() error {
	if len(args.Address) == 0 {
		return NewValidationError("Address", "cannot be blank")
	}
	return nil
}

type BlockNumIndexArgs struct {
	BlockNumber int64
	Index       int64
}

func (args *BlockNumIndexArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	r := bytes.NewReader(b)
	if err := json.NewDecoder(r).Decode(&obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	arg0, ok := obj[0].(string)
	if !ok {
		return NewDecodeParamError("BlockNumber is not string")
	}
	args.BlockNumber = common.Big(arg0).Int64()

	if len(obj) > 1 {
		arg1, ok := obj[1].(string)
		if !ok {
			return NewDecodeParamError("Index not a string")
		}
		args.Index = common.Big(arg1).Int64()
	}

	return nil
}

type HashIndexArgs struct {
	Hash  string
	Index int64
}

func (args *HashIndexArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	r := bytes.NewReader(b)
	if err := json.NewDecoder(r).Decode(&obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	arg0, ok := obj[0].(string)
	if !ok {
		return NewDecodeParamError("Hash not a string")
	}
	args.Hash = arg0

	if len(obj) > 1 {
		arg1, ok := obj[1].(string)
		if !ok {
			return NewDecodeParamError("Index not a string")
		}
		args.Index = common.Big(arg1).Int64()
	}

	return nil
}

type Sha3Args struct {
	Data string
}

func (args *Sha3Args) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	r := bytes.NewReader(b)
	if err := json.NewDecoder(r).Decode(&obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}
	args.Data = obj[0].(string)

	return nil
}

type FilterOptions struct {
	Earliest int64
	Latest   int64
	Address  interface{}
	Topics   []interface{}
	Skip     int
	Max      int
}

func (args *FilterOptions) UnmarshalJSON(b []byte) (err error) {
	var obj []struct {
		FromBlock interface{}   `json:"fromBlock"`
		ToBlock   interface{}   `json:"toBlock"`
		Limit     string        `json:"limit"`
		Offset    string        `json:"offset"`
		Address   string        `json:"address"`
		Topics    []interface{} `json:"topics"`
	}

	if err = json.Unmarshal(b, &obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	fromstr, ok := obj[0].FromBlock.(string)
	if !ok {
		return NewDecodeParamError("FromBlock is not a string")
	}

	switch fromstr {
	case "latest":
		args.Earliest = 0
	default:
		args.Earliest = int64(common.Big(obj[0].FromBlock.(string)).Int64())
	}

	tostr, ok := obj[0].ToBlock.(string)
	if !ok {
		return NewDecodeParamError("ToBlock is not a string")
	}

	switch tostr {
	case "latest":
		args.Latest = 0
	case "pending":
		args.Latest = -1
	default:
		args.Latest = int64(common.Big(obj[0].ToBlock.(string)).Int64())
	}

	args.Max = int(common.Big(obj[0].Limit).Int64())
	args.Skip = int(common.Big(obj[0].Offset).Int64())
	args.Address = obj[0].Address
	args.Topics = obj[0].Topics

	return nil
}

// type FilterChangedArgs struct {
// 	n int
// }

type DbArgs struct {
	Database string
	Key      string
	Value    string
}

func (args *DbArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	r := bytes.NewReader(b)
	if err := json.NewDecoder(r).Decode(&obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return NewInsufficientParamsError(len(obj), 2)
	}
	args.Database = obj[0].(string)
	args.Key = obj[1].(string)

	if len(obj) > 2 {
		args.Value = obj[2].(string)
	}

	return nil
}

func (a *DbArgs) requirements() error {
	if len(a.Database) == 0 {
		return NewValidationError("Database", "cannot be blank")
	}
	if len(a.Key) == 0 {
		return NewValidationError("Key", "cannot be blank")
	}
	return nil
}

type WhisperMessageArgs struct {
	Payload  string
	To       string
	From     string
	Topics   []string
	Priority uint32
	Ttl      uint32
}

func (args *WhisperMessageArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []struct {
		Payload  string
		To       string
		From     string
		Topics   []string
		Priority string
		Ttl      string
	}

	if err = json.Unmarshal(b, &obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}
	args.Payload = obj[0].Payload
	args.To = obj[0].To
	args.From = obj[0].From
	args.Topics = obj[0].Topics
	args.Priority = uint32(common.Big(obj[0].Priority).Int64())
	args.Ttl = uint32(common.Big(obj[0].Ttl).Int64())

	return nil
}

type CompileArgs struct {
	Source string
}

func (args *CompileArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	r := bytes.NewReader(b)
	if err := json.NewDecoder(r).Decode(&obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) > 0 {
		args.Source = obj[0].(string)
	}

	return nil
}

type FilterStringArgs struct {
	Word string
}

func (args *FilterStringArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	r := bytes.NewReader(b)
	if err := json.NewDecoder(r).Decode(&obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	var argstr string
	argstr, ok := obj[0].(string)
	if !ok {
		return NewDecodeParamError("Filter is not a string")
	}
	args.Word = argstr

	return nil
}

type FilterIdArgs struct {
	Id int
}

func (args *FilterIdArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []string
	r := bytes.NewReader(b)
	if err := json.NewDecoder(r).Decode(&obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	args.Id = int(common.Big(obj[0]).Int64())

	return nil
}

type WhisperIdentityArgs struct {
	Identity string
}

func (args *WhisperIdentityArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []string
	r := bytes.NewReader(b)
	if err := json.NewDecoder(r).Decode(&obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	args.Identity = obj[0]

	return nil
}

type WhisperFilterArgs struct {
	To     string `json:"to"`
	From   string
	Topics []string
}

func (args *WhisperFilterArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []struct {
		To     string
		From   string
		Topics []string
	}

	if err = json.Unmarshal(b, &obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	args.To = obj[0].To
	args.From = obj[0].From
	args.Topics = obj[0].Topics

	return nil
}

// func (req *RpcRequest) ToRegisterArgs() (string, error) {
// 	if len(req.Params) < 1 {
// 		return "", errArguments
// 	}

// 	var args string
// 	err := json.Unmarshal(req.Params, &args)
// 	if err != nil {
// 		return "", err
// 	}

// 	return args, nil
// }

// func (req *RpcRequest) ToWatchTxArgs() (string, error) {
// 	if len(req.Params) < 1 {
// 		return "", errArguments
// 	}

// 	var args string
// 	err := json.Unmarshal(req.Params, &args)
// 	if err != nil {
// 		return "", err
// 	}

// 	return args, nil
// }

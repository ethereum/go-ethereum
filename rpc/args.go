package rpc

import (
	"bytes"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func blockHeightFromJson(msg json.RawMessage, number *int64) error {
	var raw interface{}
	if err := json.Unmarshal(msg, &raw); err != nil {
		return NewDecodeParamError(err.Error())
	}
	return blockHeight(raw, number)
}

func blockHeight(raw interface{}, number *int64) error {
	// Parse as integer
	num, ok := raw.(float64)
	if ok {
		*number = int64(num)
		return nil
	}

	// Parse as string/hexstring
	str, ok := raw.(string)
	if !ok {
		return NewInvalidTypeError("blockNumber", "not a number or string")
	}

	switch str {
	case "latest":
		*number = -1
	case "pending":
		*number = -2
	default:
		*number = common.String2Big(str).Int64()
	}

	return nil
}

type GetBlockByHashArgs struct {
	BlockHash  common.Hash
	IncludeTxs bool
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
		return NewInvalidTypeError("blockHash", "not a string")
	}
	args.BlockHash = common.HexToHash(argstr)

	if len(obj) > 1 {
		args.IncludeTxs = obj[1].(bool)
	}

	return nil
}

type GetBlockByNumberArgs struct {
	BlockNumber int64
	IncludeTxs  bool
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
	} else if v, ok := obj[0].(string); ok {
		args.BlockNumber = common.Big(v).Int64()
	} else {
		return NewInvalidTypeError("blockNumber", "not a number or string")
	}

	if len(obj) > 1 {
		args.IncludeTxs = obj[1].(bool)
	}

	return nil
}

type NewTxArgs struct {
	From     common.Address
	To       common.Address
	Value    *big.Int
	Gas      *big.Int
	GasPrice *big.Int
	Data     string

	BlockNumber int64
}

func (args *NewTxArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []json.RawMessage
	var ext struct{ From, To, Value, Gas, GasPrice, Data string }

	// Decode byte slice to array of RawMessages
	if err := json.Unmarshal(b, &obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	// Check for sufficient params
	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	// Decode 0th RawMessage to temporary struct
	if err := json.Unmarshal(obj[0], &ext); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(ext.From) == 0 {
		return NewValidationError("from", "is required")
	}

	args.From = common.HexToAddress(ext.From)
	args.To = common.HexToAddress(ext.To)
	args.Value = common.String2Big(ext.Value)
	args.Gas = common.String2Big(ext.Gas)
	args.GasPrice = common.String2Big(ext.GasPrice)
	args.Data = ext.Data

	// Check for optional BlockNumber param
	if len(obj) > 1 {
		if err := blockHeightFromJson(obj[1], &args.BlockNumber); err != nil {
			return err
		}
	}

	return nil
}

type GetStorageArgs struct {
	Address     common.Address
	BlockNumber int64
}

func (args *GetStorageArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return NewInvalidTypeError("address", "not a string")
	}
	args.Address = common.HexToAddress(addstr)

	if len(obj) > 1 {
		if err := blockHeight(obj[1], &args.BlockNumber); err != nil {
			return err
		}
	}

	return nil
}

type GetStorageAtArgs struct {
	Address     common.Address
	Key         common.Hash
	BlockNumber int64
}

func (args *GetStorageAtArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return NewInsufficientParamsError(len(obj), 2)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return NewInvalidTypeError("address", "not a string")
	}
	args.Address = common.HexToAddress(addstr)

	keystr, ok := obj[1].(string)
	if !ok {
		return NewInvalidTypeError("key", "not a string")
	}
	args.Key = common.HexToHash(keystr)

	if len(obj) > 2 {
		if err := blockHeight(obj[2], &args.BlockNumber); err != nil {
			return err
		}
	}

	return nil
}

type GetTxCountArgs struct {
	Address     common.Address
	BlockNumber int64
}

func (args *GetTxCountArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return NewInvalidTypeError("address", "not a string")
	}
	args.Address = common.HexToAddress(addstr)

	if len(obj) > 1 {
		if err := blockHeight(obj[1], &args.BlockNumber); err != nil {
			return err
		}
	}

	return nil
}

type GetBalanceArgs struct {
	Address     common.Address
	BlockNumber int64
}

func (args *GetBalanceArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return NewInvalidTypeError("address", "not a string")
	}
	args.Address = common.HexToAddress(addstr)

	if len(obj) > 1 {
		if err := blockHeight(obj[1], &args.BlockNumber); err != nil {
			return err
		}
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
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return NewInvalidTypeError("address", "not a string")
	}
	args.Address = addstr

	if len(obj) > 1 {
		if err := blockHeight(obj[1], &args.BlockNumber); err != nil {
			return err
		}
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
		return NewInvalidTypeError("blockNumber", "not a string")
	}
	args.BlockNumber = common.Big(arg0).Int64()

	if len(obj) > 1 {
		arg1, ok := obj[1].(string)
		if !ok {
			return NewInvalidTypeError("index", "not a string")
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
		return NewInvalidTypeError("hash", "not a string")
	}
	args.Hash = arg0

	if len(obj) > 1 {
		arg1, ok := obj[1].(string)
		if !ok {
			return NewInvalidTypeError("index", "not a string")
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

type BlockFilterArgs struct {
	Earliest int64
	Latest   int64
	Address  interface{}
	Topics   []interface{}
	Skip     int
	Max      int
}

func (args *BlockFilterArgs) UnmarshalJSON(b []byte) (err error) {
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
		return NewInvalidTypeError("fromBlock", "is not a string")
	}

	switch fromstr {
	case "latest":
		args.Earliest = -1
	default:
		args.Earliest = int64(common.Big(obj[0].FromBlock.(string)).Int64())
	}

	tostr, ok := obj[0].ToBlock.(string)
	if !ok {
		return NewInvalidTypeError("toBlock", "not a string")
	}

	switch tostr {
	case "latest":
		args.Latest = -1
	case "pending":
		args.Latest = -2
	default:
		args.Latest = int64(common.Big(obj[0].ToBlock.(string)).Int64())
	}

	args.Max = int(common.Big(obj[0].Limit).Int64())
	args.Skip = int(common.Big(obj[0].Offset).Int64())
	args.Address = obj[0].Address
	args.Topics = obj[0].Topics

	return nil
}

type DbArgs struct {
	Database string
	Key      string
	Value    []byte
}

func (args *DbArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return NewInsufficientParamsError(len(obj), 2)
	}

	var objstr string
	var ok bool

	if objstr, ok = obj[0].(string); !ok {
		return NewInvalidTypeError("database", "not a string")
	}
	args.Database = objstr

	if objstr, ok = obj[1].(string); !ok {
		return NewInvalidTypeError("key", "not a string")
	}
	args.Key = objstr

	if len(obj) > 2 {
		objstr, ok = obj[2].(string)
		if !ok {
			return NewInvalidTypeError("value", "not a string")
		}

		args.Value = []byte(objstr)
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

type DbHexArgs struct {
	Database string
	Key      string
	Value    []byte
}

func (args *DbHexArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return NewInsufficientParamsError(len(obj), 2)
	}

	var objstr string
	var ok bool

	if objstr, ok = obj[0].(string); !ok {
		return NewInvalidTypeError("database", "not a string")
	}
	args.Database = objstr

	if objstr, ok = obj[1].(string); !ok {
		return NewInvalidTypeError("key", "not a string")
	}
	args.Key = objstr

	if len(obj) > 2 {
		objstr, ok = obj[2].(string)
		if !ok {
			return NewInvalidTypeError("value", "not a string")
		}

		args.Value = common.FromHex(objstr)
	}

	return nil
}

func (a *DbHexArgs) requirements() error {
	if len(a.Database) == 0 {
		return NewInvalidTypeError("Database", "cannot be blank")
	}
	if len(a.Key) == 0 {
		return NewInvalidTypeError("Key", "cannot be blank")
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
		return NewInvalidTypeError("filter", "not a string")
	}
	args.Word = argstr

	return nil
}

func (args *FilterStringArgs) requirements() error {
	switch args.Word {
	case "latest", "pending":
		break
	default:
		return NewValidationError("Word", "Must be `latest` or `pending`")
	}
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

type SubmitWorkArgs struct {
	Nonce  uint64
	Header common.Hash
	Digest common.Hash
}

func (args *SubmitWorkArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err = json.Unmarshal(b, &obj); err != nil {
		return NewDecodeParamError(err.Error())
	}

	if len(obj) < 3 {
		return NewInsufficientParamsError(len(obj), 3)
	}

	var objstr string
	var ok bool
	if objstr, ok = obj[0].(string); !ok {
		return NewInvalidTypeError("nonce", "not a string")
	}

	args.Nonce = common.String2Big(objstr).Uint64()
	if objstr, ok = obj[1].(string); !ok {
		return NewInvalidTypeError("header", "not a string")
	}

	args.Header = common.HexToHash(objstr)

	if objstr, ok = obj[2].(string); !ok {
		return NewInvalidTypeError("digest", "not a string")
	}

	args.Digest = common.HexToHash(objstr)

	return nil
}

package rpc

import "encoding/json"

import "github.com/ethereum/go-ethereum/core"

type GetBlockArgs struct {
	BlockNumber int32
	Hash        string
}

func (obj *GetBlockArgs) UnmarshalJSON(b []byte) (err error) {
	argint, argstr := int32(0), ""
	if err = json.Unmarshal(b, &argint); err == nil {
		obj.BlockNumber = argint
		return
	}
	if err = json.Unmarshal(b, &argstr); err == nil {
		obj.Hash = argstr
		return
	}
	return errDecodeArgs
}

type NewTxArgs struct {
	From     string `json:"from"`
	Pass     string `json:"pass"`
	To       string `json:"to"`
	Value    string `json:"value"`
	Gas      string `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Data     string `json:"data"`
}

func (obj *NewTxArgs) UnmarshalJSON(b []byte) (err error) {
	// Data can be either specified as "data" or "code" :-/
	var ext struct {
		From     string
		To       string
		Value    string
		Gas      string
		GasPrice string
		Data     string
		Code     string
	}

	if err = json.Unmarshal(b, &ext); err == nil {
		if len(ext.Data) == 0 {
			ext.Data = ext.Code
		}
		obj.From = ext.From
		obj.To = ext.To
		obj.Value = ext.Value
		obj.Gas = ext.Gas
		obj.GasPrice = ext.GasPrice
		obj.Data = ext.Data

		return
	}

	return errDecodeArgs
}

type PushTxArgs struct {
	Tx string `json:"tx"`
}

func (obj *PushTxArgs) UnmarshalJSON(b []byte) (err error) {
	arg0 := ""
	if err = json.Unmarshal(b, &arg0); err == nil {
		obj.Tx = arg0
		return
	}
	return errDecodeArgs
}

func (a *PushTxArgs) requirementsPushTx() error {
	if a.Tx == "" {
		return NewErrorWithMessage(errArguments, "PushTx requires a 'tx' as argument")
	}
	return nil
}

type GetStorageArgs struct {
	Address string
}

func (obj *GetStorageArgs) UnmarshalJSON(b []byte) (err error) {
	if err = json.Unmarshal(b, &obj.Address); err != nil {
		return errDecodeArgs
	}
	return
}

func (a *GetStorageArgs) requirements() error {
	if len(a.Address) == 0 {
		return NewErrorWithMessage(errArguments, "GetStorageAt requires an 'address' value as argument")
	}
	return nil
}

type GetStateArgs struct {
	Address string
	Key     string
}

func (obj *GetStateArgs) UnmarshalJSON(b []byte) (err error) {
	arg0 := ""
	if err = json.Unmarshal(b, &arg0); err == nil {
		obj.Address = arg0
		return
	}
	return errDecodeArgs
}

func (a *GetStateArgs) requirements() error {
	if a.Address == "" {
		return NewErrorWithMessage(errArguments, "GetStorageAt requires an 'address' value as argument")
	}
	if a.Key == "" {
		return NewErrorWithMessage(errArguments, "GetStorageAt requires an 'key' value as argument")
	}
	return nil
}

type GetTxCountArgs struct {
	Address string `json:"address"`
}

func (obj *GetTxCountArgs) UnmarshalJSON(b []byte) (err error) {
	arg0 := ""
	if err = json.Unmarshal(b, &arg0); err == nil {
		obj.Address = arg0
		return
	}
	return errDecodeArgs
}

func (a *GetTxCountArgs) requirements() error {
	if a.Address == "" {
		return NewErrorWithMessage(errArguments, "GetTxCountAt requires an 'address' value as argument")
	}
	return nil
}

type GetBalanceArgs struct {
	Address string
}

func (obj *GetBalanceArgs) UnmarshalJSON(b []byte) (err error) {
	arg0 := ""
	if err = json.Unmarshal(b, &arg0); err == nil {
		obj.Address = arg0
		return
	}
	return errDecodeArgs
}

func (a *GetBalanceArgs) requirements() error {
	if a.Address == "" {
		return NewErrorWithMessage(errArguments, "GetBalanceAt requires an 'address' value as argument")
	}
	return nil
}

type GetCodeAtArgs struct {
	Address string
}

func (obj *GetCodeAtArgs) UnmarshalJSON(b []byte) (err error) {
	arg0 := ""
	if err = json.Unmarshal(b, &arg0); err == nil {
		obj.Address = arg0
		return
	}
	return errDecodeArgs
}

func (a *GetCodeAtArgs) requirements() error {
	if a.Address == "" {
		return NewErrorWithMessage(errArguments, "GetCodeAt requires an 'address' value as argument")
	}
	return nil
}

type Sha3Args struct {
	Data string
}

func (obj *Sha3Args) UnmarshalJSON(b []byte) (err error) {
	if err = json.Unmarshal(b, &obj.Data); err != nil {
		return errDecodeArgs
	}
	return
}

type FilterOptions struct {
	Earliest int64
	Latest   int64
	Address  interface{}
	Topic    []interface{}
	Skip     int
	Max      int
}

func toFilterOptions(options *FilterOptions) core.FilterOptions {
	var opts core.FilterOptions

	// Convert optional address slice/string to byte slice
	if str, ok := options.Address.(string); ok {
		opts.Address = [][]byte{fromHex(str)}
	} else if slice, ok := options.Address.([]interface{}); ok {
		bslice := make([][]byte, len(slice))
		for i, addr := range slice {
			if saddr, ok := addr.(string); ok {
				bslice[i] = fromHex(saddr)
			}
		}
		opts.Address = bslice
	}

	opts.Earliest = options.Earliest
	opts.Latest = options.Latest

	topics := make([][][]byte, len(options.Topic))
	for i, topicDat := range options.Topic {
		if slice, ok := topicDat.([]interface{}); ok {
			topics[i] = make([][]byte, len(slice))
			for j, topic := range slice {
				topics[i][j] = fromHex(topic.(string))
			}
		} else if str, ok := topicDat.(string); ok {
			topics[i] = make([][]byte, 1)
			topics[i][0] = fromHex(str)
		}
	}
	opts.Topics = topics

	return opts
}

type FilterChangedArgs struct {
	n int
}

type DbArgs struct {
	Database string
	Key      string
	Value    string
}

func (a *DbArgs) requirements() error {
	if len(a.Database) == 0 {
		return NewErrorWithMessage(errArguments, "DbPutArgs requires an 'Database' value as argument")
	}
	if len(a.Key) == 0 {
		return NewErrorWithMessage(errArguments, "DbPutArgs requires an 'Key' value as argument")
	}
	return nil
}

type WhisperMessageArgs struct {
	Payload  string
	To       string
	From     string
	Topic    []string
	Priority uint32
	Ttl      uint32
}

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
	return NewErrorResponse(ErrorDecodeArgs)
}

func (obj *GetBlockArgs) requirements() error {
	if obj.BlockNumber == 0 && obj.Hash == "" {
		return NewErrorResponse("GetBlock requires either a block 'number' or a block 'hash' as argument")
	}
	return nil
}

type NewTxArgs struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Value    string `json:"value"`
	Gas      string `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Data     string `json:"data"`
}

func (a *NewTxArgs) requirements() error {
	if a.Gas == "" {
		return NewErrorResponse("Transact requires a 'gas' value as argument")
	}
	if a.GasPrice == "" {
		return NewErrorResponse("Transact requires a 'gasprice' value as argument")
	}
	return nil
}

type PushTxArgs struct {
	Tx string `json:"tx"`
}

func (obj *PushTxArgs) UnmarshalJSON(b []byte) (err error) {
	arg0 := ""
	if err = json.Unmarshal(b, arg0); err == nil {
		obj.Tx = arg0
		return
	}
	return NewErrorResponse(ErrorDecodeArgs)
}

func (a *PushTxArgs) requirementsPushTx() error {
	if a.Tx == "" {
		return NewErrorResponse("PushTx requires a 'tx' as argument")
	}
	return nil
}

type GetStorageArgs struct {
	Address string
}

func (obj *GetStorageArgs) UnmarshalJSON(b []byte) (err error) {
	if err = json.Unmarshal(b, &obj.Address); err != nil {
		return NewErrorResponse(ErrorDecodeArgs)
	}
	return
}

func (a *GetStorageArgs) requirements() error {
	if len(a.Address) == 0 {
		return NewErrorResponse("GetStorageAt requires an 'address' value as argument")
	}
	return nil
}

type GetStateArgs struct {
	Address string
	Key     string
}

func (obj *GetStateArgs) UnmarshalJSON(b []byte) (err error) {
	arg0 := ""
	if err = json.Unmarshal(b, arg0); err == nil {
		obj.Address = arg0
		return
	}
	return NewErrorResponse(ErrorDecodeArgs)
}

func (a *GetStateArgs) requirements() error {
	if a.Address == "" {
		return NewErrorResponse("GetStorageAt requires an 'address' value as argument")
	}
	if a.Key == "" {
		return NewErrorResponse("GetStorageAt requires an 'key' value as argument")
	}
	return nil
}

type GetStorageAtRes struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type GetTxCountArgs struct {
	Address string `json:"address"`
}

// type GetTxCountRes struct {
//  Nonce int `json:"nonce"`
// }

func (obj *GetTxCountArgs) UnmarshalJSON(b []byte) (err error) {
	arg0 := ""
	if err = json.Unmarshal(b, arg0); err == nil {
		obj.Address = arg0
		return
	}
	return NewErrorResponse("Could not determine JSON parameters")
}

func (a *GetTxCountArgs) requirements() error {
	if a.Address == "" {
		return NewErrorResponse("GetTxCountAt requires an 'address' value as argument")
	}
	return nil
}

// type GetPeerCountRes struct {
//  PeerCount int `json:"peerCount"`
// }

// type GetListeningRes struct {
//  IsListening bool `json:"isListening"`
// }

// type GetCoinbaseRes struct {
//  Coinbase string `json:"coinbase"`
// }

// type GetMiningRes struct {
//  IsMining bool `json:"isMining"`
// }

type GetBalanceArgs struct {
	Address string
}

func (obj *GetBalanceArgs) UnmarshalJSON(b []byte) (err error) {
	arg0 := ""
	if err = json.Unmarshal(b, &arg0); err == nil {
		obj.Address = arg0
		return
	}
	return NewErrorResponse("Could not determine JSON parameters")
}

func (a *GetBalanceArgs) requirements() error {
	if a.Address == "" {
		return NewErrorResponse("GetBalanceAt requires an 'address' value as argument")
	}
	return nil
}

type BalanceRes struct {
	Balance string `json:"balance"`
	Address string `json:"address"`
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
	return NewErrorResponse(ErrorDecodeArgs)
}

func (a *GetCodeAtArgs) requirements() error {
	if a.Address == "" {
		return NewErrorResponse("GetCodeAt requires an 'address' value as argument")
	}
	return nil
}

type Sha3Args struct {
	Data string
}

func (obj *Sha3Args) UnmarshalJSON(b []byte) (err error) {
	if err = json.Unmarshal(b, &obj.Data); err != nil {
		return NewErrorResponse(ErrorDecodeArgs)
	}
	return
}

type FilterOptions struct {
	Earliest int64
	Latest   int64
	Address  string
	Topics   []string
	Skip     int
	Max      int
}

func toFilterOptions(options *FilterOptions) core.FilterOptions {
	var opts core.FilterOptions
	opts.Earliest = options.Earliest
	opts.Latest = options.Latest
	opts.Address = fromHex(options.Address)
	opts.Topics = make([][]byte, len(options.Topics))
	for i, topic := range options.Topics {
		opts.Topics[i] = fromHex(topic)
	}

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
		return NewErrorResponse("DbPutArgs requires an 'Database' value as argument")
	}
	if len(a.Key) == 0 {
		return NewErrorResponse("DbPutArgs requires an 'Key' value as argument")
	}
	return nil
}

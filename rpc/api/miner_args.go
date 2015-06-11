package api

import (
	"encoding/json"

	"math/big"

	"github.com/ethereum/go-ethereum/rpc/shared"
)

type StartMinerArgs struct {
	Threads int
}

func (args *StartMinerArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) == 0 || obj[0] == nil {
		args.Threads = -1
		return nil
	}

	var num *big.Int
	if num, err = numString(obj[0]); err != nil {
		return err
	}
	args.Threads = int(num.Int64())
	return nil
}

type SetExtraArgs struct {
	Data string
}

func (args *SetExtraArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	extrastr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("Price", "not a string")
	}
	args.Data = extrastr

	return nil
}

type GasPriceArgs struct {
	Price string
}

func (args *GasPriceArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	if pricestr, ok := obj[0].(string); ok {
		args.Price = pricestr
		return nil
	}

	return shared.NewInvalidTypeError("Price", "not a string")
}

type MakeDAGArgs struct {
	BlockNumber int64
}

func (args *MakeDAGArgs) UnmarshalJSON(b []byte) (err error) {
	args.BlockNumber = -1
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

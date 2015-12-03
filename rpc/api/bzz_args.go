// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common/chequebook"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

type BzzDepositArgs struct {
	Amount *big.Int
}

func (args *BzzDepositArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	amount, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("Amount", "not a string")
	}
	args.Amount, ok = new(big.Int).SetString(amount, 10)
	if !ok {
		return shared.NewInvalidTypeError("Amount", "not a number")
	}

	return nil
}

type BzzCashArgs struct {
	Cheque *chequebook.Cheque
}

func (args *BzzCashArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	chequestr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("Cheque", "not a string")
	}
	var cheque chequebook.Cheque
	err = json.Unmarshal([]byte(chequestr), &cheque)
	if err != nil {
		return shared.NewDecodeParamError(err.Error())
	}
	args.Cheque = &cheque

	return nil
}

type BzzIssueArgs struct {
	Beneficiary string
	Amount      *big.Int
}

func (args *BzzIssueArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 2)
	}

	beneficiary, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("Amount", "not a string")
	}
	args.Beneficiary = beneficiary

	amount, ok := obj[1].(string)
	if !ok {
		return shared.NewInvalidTypeError("Amount", "not a string")
	}
	args.Amount, ok = new(big.Int).SetString(amount, 10)
	if !ok {
		return shared.NewInvalidTypeError("Amount", "not a number")
	}

	return nil
}

type BzzRegisterArgs struct {
	Address, ContentHash, Domain string
}

func (args *BzzRegisterArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 3 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("Address", "not a string")
	}
	args.Address = addstr

	addstr, ok = obj[1].(string)
	if !ok {
		return shared.NewInvalidTypeError("Domain", "not a string")
	}
	args.Domain = addstr

	addstr, ok = obj[2].(string)
	if !ok {
		return shared.NewInvalidTypeError("ContentHash", "not a string")
	}
	args.ContentHash = addstr

	return nil
}

type BzzResolveArgs struct {
	Domain string
}

func (args *BzzResolveArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("Domain", "not a string")
	}
	args.Domain = addstr

	return nil
}

type BzzDownloadArgs struct {
	BzzPath, LocalPath string
}

func (args *BzzDownloadArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("BzzPath", "not a string")
	}
	args.BzzPath = addstr

	addstr, ok = obj[1].(string)
	if !ok {
		return shared.NewInvalidTypeError("LocalPath", "not a string")
	}
	args.LocalPath = addstr

	return nil
}

type BzzUploadArgs struct {
	LocalPath, Index string
}

func (args *BzzUploadArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("LocalPath", "not a string")
	}
	args.LocalPath = addstr

	if len(obj) > 1 {
		addstr, ok := obj[1].(string)
		if ok {
			args.Index = addstr
		}
	}

	return nil
}

type BzzGetArgs struct {
	Path string
}

func (args *BzzGetArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("Path", "not a string")
	}
	args.Path = addstr

	return nil
}

type BzzPutArgs struct {
	Content, ContenType string
}

func (args *BzzPutArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("Content", "not a string")
	}
	args.Content = addstr

	addstr, ok = obj[1].(string)
	if !ok {
		return shared.NewInvalidTypeError("ContenType", "not a string")
	}
	args.ContenType = addstr

	return nil
}

type BzzModifyArgs struct {
	RootHash, Path, ContentHash, ContentType string
}

func (args *BzzModifyArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	addstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("RootHash", "not a string")
	}
	args.RootHash = addstr

	addstr, ok = obj[1].(string)
	if !ok {
		return shared.NewInvalidTypeError("Path", "not a string")
	}
	args.Path = addstr

	if len(obj) >= 4 {
		addstr, ok = obj[2].(string)
		if ok {
			args.ContentHash = addstr
		}

		addstr, ok = obj[3].(string)
		if ok {
			args.ContentType = addstr
		}
	}

	return nil
}

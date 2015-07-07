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

	"github.com/ethereum/go-ethereum/rpc/shared"
)

type NewAccountArgs struct {
	Passphrase string
}

func (args *NewAccountArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	if passhrase, ok := obj[0].(string); ok {
		args.Passphrase = passhrase
		return nil
	}

	return shared.NewInvalidTypeError("passhrase", "not a string")
}

type DeleteAccountArgs struct {
	Address    string
	Passphrase string
}

func (args *DeleteAccountArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 2)
	}

	if addr, ok := obj[0].(string); ok {
		args.Address = addr
	} else {
		return shared.NewInvalidTypeError("address", "not a string")
	}

	if passhrase, ok := obj[1].(string); ok {
		args.Passphrase = passhrase
	} else {
		return shared.NewInvalidTypeError("passhrase", "not a string")
	}

	return nil
}

type UnlockAccountArgs struct {
	Address    string
	Passphrase string
	Duration   int
}

func (args *UnlockAccountArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	args.Duration = -1

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 2)
	}

	if addrstr, ok := obj[0].(string); ok {
		args.Address = addrstr
	} else {
		return shared.NewInvalidTypeError("address", "not a string")
	}

	if passphrasestr, ok := obj[1].(string); ok {
		args.Passphrase = passphrasestr
	} else {
		return shared.NewInvalidTypeError("passphrase", "not a string")
	}

	return nil
}

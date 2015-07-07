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

type CompileArgs struct {
	Source string
}

func (args *CompileArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}
	argstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("arg0", "is not a string")
	}
	args.Source = argstr

	return nil
}

type FilterStringArgs struct {
	Word string
}

func (args *FilterStringArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	var argstr string
	argstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("filter", "not a string")
	}
	switch argstr {
	case "latest", "pending":
		break
	default:
		return shared.NewValidationError("Word", "Must be `latest` or `pending`")
	}
	args.Word = argstr
	return nil
}

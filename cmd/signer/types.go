// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"strings"
)

type Accounts []Account

func (as Accounts) String() string {
	var output []string
	for _, a := range as {
		output = append(output, a.String())
	}
	return strings.Join(output, "\n")
}

type Account struct {
	Typ     string         `json:"type"`
	URL     accounts.URL   `json:"url"`
	Address common.Address `json:"address"`
}

func (a Account) String() string {
	s, err := json.Marshal(a)
	if err == nil {
		return string(s)
	}
	return err.Error()
}

// TransactionArg represents a transaction for the signer.
type TransactionArg struct {
	To       *common.Address `json:"to"`
	Gas      *hexutil.Big    `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Data     hexutil.Bytes   `json:"data"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
}

func (t TransactionArg) String() string {
	s, err := json.Marshal(t)
	if err == nil {
		return string(s)
	}
	return err.Error()
}

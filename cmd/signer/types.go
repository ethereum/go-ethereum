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

package signer

import (
	"encoding/json"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"github.com/ethereum/go-ethereum/core/types"
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
/*
// TransactionArg represents a Transaction for the signer.
type TransactionArg struct {
	To       *common.MixedcaseAddress `json:"to"`
	Gas      *hexutil.Big             `json:"gas"`
	GasPrice *hexutil.Big             `json:"gasPrice"`
	Value    *hexutil.Big             `json:"value"`
	Data     hexutil.Bytes            `json:"data"`
	Nonce    *hexutil.Uint64          `json:"nonce"`
}
*/

// SendTxArgs represents the arguments to submit a transaction
type SendTxArgs struct {
	From     common.MixedcaseAddress  `json:"from"`
	To       *common.MixedcaseAddress `json:"to"`
	Gas      *hexutil.Big    `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	// We accept "data" and "input" for backwards-compatibility reasons.
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input"`
}

func (t SendTxArgs) String() string {
	s, err := json.Marshal(t)
	if err == nil {
		return string(s)
	}
	return err.Error()
}

func (args *SendTxArgs) toTransaction() *types.Transaction {
	var input []byte
	if args.Data != nil {
		input = *args.Data
	} else if args.Input != nil {
		input = *args.Input
	}
	if args.To == nil {
		return types.NewContractCreation(uint64(*args.Nonce), (*big.Int)(args.Value), (*big.Int)(args.Gas), (*big.Int)(args.GasPrice), input)
	}
	return types.NewTransaction(uint64(*args.Nonce), (*args.To).Address(), (*big.Int)(args.Value), (*big.Int)(args.Gas), (*big.Int)(args.GasPrice), input)
}

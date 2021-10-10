// Copyright 2021 The go-ethereum Authors
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

package t8ntool

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/tests"
	"gopkg.in/urfave/cli.v1"
)

type result struct {
	Error   error
	Address common.Address
	Hash    common.Hash
}

// MarshalJSON marshals as JSON with a hash.
func (r *result) MarshalJSON() ([]byte, error) {
	type xx struct {
		Error   string          `json:"error,omitempty"`
		Address *common.Address `json:"address,omitempty"`
		Hash    *common.Hash    `json:"hash,omitempty"`
	}
	var out xx
	if r.Error != nil {
		out.Error = r.Error.Error()
	}
	if r.Address != (common.Address{}) {
		out.Address = &r.Address
	}
	if r.Hash != (common.Hash{}) {
		out.Hash = &r.Hash
	}
	return json.Marshal(out)
}

func Transaction(ctx *cli.Context) error {
	// Configure the go-ethereum logger
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(ctx.Int(VerbosityFlag.Name)))
	log.Root().SetHandler(glogger)

	var (
		err error
	)
	// We need to load the transactions. May be either in stdin input or in files.
	// Check if anything needs to be read from stdin
	var (
		txStr       = ctx.String(InputTxsFlag.Name)
		inputData   = &input{}
		chainConfig *params.ChainConfig
	)
	// Construct the chainconfig
	if cConf, _, err := tests.GetChainConfig(ctx.String(ForknameFlag.Name)); err != nil {
		return NewError(ErrorVMConfig, fmt.Errorf("failed constructing chain configuration: %v", err))
	} else {
		chainConfig = cConf
	}
	// Set the chain id
	chainConfig.ChainID = big.NewInt(ctx.Int64(ChainIDFlag.Name))
	var body hexutil.Bytes
	if txStr == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(inputData); err != nil {
			return NewError(ErrorJson, fmt.Errorf("failed unmarshaling stdin: %v", err))
		}
		// Decode the body of already signed transactions
		body = common.FromHex(inputData.TxRlp)
	} else {
		// Read input from file
		inFile, err := os.Open(txStr)
		if err != nil {
			return NewError(ErrorIO, fmt.Errorf("failed reading txs file: %v", err))
		}
		defer inFile.Close()
		decoder := json.NewDecoder(inFile)
		if strings.HasSuffix(txStr, ".rlp") {
			if err := decoder.Decode(&body); err != nil {
				return err
			}
		} else {
			return NewError(ErrorIO, errors.New("only rlp supported"))
		}
	}
	signer := types.MakeSigner(chainConfig, new(big.Int))
	// We now have the transactions in 'body', which is supposed to be an
	// rlp list of transactions
	it, err := rlp.NewListIterator([]byte(body))
	if err != nil {
		return err
	}
	var results []result
	for it.Next() {
		var tx types.Transaction
		err := rlp.DecodeBytes(it.Value(), &tx)
		if err != nil {
			results = append(results, result{Error: err})
			continue
		}
		sender, err := types.Sender(signer, &tx)
		if err != nil {
			results = append(results, result{Error: err})
			continue
		}
		results = append(results, result{Address: sender, Hash: tx.Hash()})
	}
	out, err := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(out))
	return err
}

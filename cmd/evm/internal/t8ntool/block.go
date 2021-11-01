// Copyright 2020 The go-ethereum Authors
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
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"gopkg.in/urfave/cli.v1"
)

type blockInput struct {
	Header *types.Header   `json:"header,omitempty"`
	Uncles []*types.Header `json:"uncles,omitempty"`
	TxRlp  string          `json:"txsRlp,omitempty"`
	Txs    []*types.Transaction
}

func BuildBlock(ctx *cli.Context) error {
	// Configure the go-ethereum logger
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(ctx.Int(VerbosityFlag.Name)))
	log.Root().SetHandler(glogger)

	baseDir, err := createBasedir(ctx)
	if err != nil {
		return NewError(ErrorIO, fmt.Errorf("failed creating output basedir: %v", err))
	}

	inputData, err := readInput(ctx)
	if err != nil {
		fmt.Println("bad error")
		return err
	}

	block := types.NewBlockWithHeader(inputData.Header)
	block.WithBody(inputData.Txs, inputData.Uncles)

	return dispatchBlock(ctx, baseDir, block)
}

func readInput(ctx *cli.Context) (*blockInput, error) {
	var (
		headerStr = ctx.String(InputHeaderFlag.Name)
		unclesStr = ctx.String(InputUnclesFlag.Name)
		txsStr    = ctx.String(InputTxsRlpFlag.Name)
		inputData = &blockInput{}
	)

	if headerStr == stdinSelector || unclesStr == stdinSelector || txsStr == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(inputData); err != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("failed unmarshaling stdin: %v", err))
		}
	}

	if headerStr != stdinSelector {
		header, err := readHeader(headerStr)
		if err != nil {
			return nil, err
		}
		inputData.Header = header
	}

	if unclesStr != stdinSelector {
		uncles, err := readUncles(unclesStr)
		if err != nil {
			return nil, err
		}
		inputData.Uncles = uncles
	}

	if txsStr != stdinSelector {
		txs, err := readTxsRlp(txsStr)
		if err != nil {
			return nil, err
		}
		inputData.TxRlp = txs
	}

	txs := []*types.Transaction{}
	raw := common.FromHex(inputData.TxRlp)
	err := rlp.DecodeBytes(raw, &txs)
	if err != nil {
		return nil, NewError(ErrorJson, fmt.Errorf("unable to decode transaction from rlp data: %v", err))
	}

	return inputData, nil
}

func readHeader(path string) (*types.Header, error) {
	header := &types.Header{}

	if path == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(header); err != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("failed unmarshaling header from stdin: %v", err))
		}
	} else {
		inFile, err := os.Open(path)
		if err != nil {
			return nil, NewError(ErrorIO, fmt.Errorf("failed reading header file: %v", err))
		}
		defer inFile.Close()
		decoder := json.NewDecoder(inFile)
		if err := decoder.Decode(&header); err != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("failed unmarshaling header file: %v", err))
		}
	}

	return header, nil
}

func readUncles(path string) ([]*types.Header, error) {
	uncles := []*types.Header{}

	if path == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(uncles); err != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("failed unmarshaling uncles from stdin: %v", err))
		}
	} else {
		inFile, err := os.Open(path)
		if err != nil {
			return nil, NewError(ErrorIO, fmt.Errorf("failed reading uncles file: %v", err))
		}
		defer inFile.Close()
		decoder := json.NewDecoder(inFile)
		if err := decoder.Decode(&uncles); err != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("failed unmarshaling uncles file: %v", err))
		}
	}

	return uncles, nil
}

func readTxsRlp(path string) (string, error) {
	txsRlp := ""

	if path == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(txsRlp); err != nil {
			return "", NewError(ErrorJson, fmt.Errorf("failed unmarshaling txs from stdin: %v", err))
		}
	} else {
		inFile, err := os.Open(path)
		if err != nil {
			return "", NewError(ErrorIO, fmt.Errorf("failed reading txs file: %v", err))
		}
		defer inFile.Close()
		decoder := json.NewDecoder(inFile)
		if err := decoder.Decode(&txsRlp); err != nil {
			return "", NewError(ErrorJson, fmt.Errorf("failed unmarshaling txs file: %v", err))
		}
	}

	return txsRlp, nil
}

// dispatchOutput writes the output data to either stderr or stdout, or to the specified
// files
func dispatchBlock(ctx *cli.Context, baseDir string, block *types.Block) error {
	raw, _ := rlp.EncodeToBytes(block)

	type BlockInfo struct {
		Rlp  hexutil.Bytes `json:"rlp"`
		Hash common.Hash   `json:"hash"`
	}

	var enc BlockInfo
	enc.Rlp = raw
	enc.Hash = block.Hash()

	b, err := json.MarshalIndent(enc, "", "  ")
	if err != nil {
		return NewError(ErrorJson, fmt.Errorf("failed marshalling output: %v", err))
	}

	dest := ctx.String(OutputBlockFlag.Name)
	if dest == "stdout" {
		os.Stdout.Write(b)
		os.Stdout.WriteString("\n")
	} else if dest == "stderr" {
		os.Stderr.Write(b)
		os.Stderr.WriteString("\n")
	} else {
		if err := saveFile(baseDir, dest, enc); err != nil {
			return err
		}
	}

	return nil
}

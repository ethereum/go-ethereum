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
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"gopkg.in/urfave/cli.v1"
)

//go:generate gencodec -type bbEnv -field-override bbEnvMarshaling -out gen_bbenv.go
type bbEnv struct {
	ParentHash  common.Hash      `json:"parentHash"`
	UncleHash   common.Hash      `json:"sha3Uncles"`
	Coinbase    common.Address   `json:"miner"            gencode:"required"`
	Root        common.Hash      `json:"stateRoot"        gencodec:"required"`
	TxHash      common.Hash      `json:"transactionsRoot"`
	ReceiptHash common.Hash      `json:"receiptsRoot"`
	Bloom       types.Bloom      `json:"logsBloom"`
	Difficulty  *big.Int         `json:"difficulty"`
	Number      *big.Int         `json:"number"           gencodec:"required"`
	GasLimit    uint64           `json:"gasLimit"         gencodec:"required"`
	GasUsed     uint64           `json:"gasUsed"`
	Time        uint64           `json:"timestamp"        gencodec:"required"`
	Extra       []byte           `json:"extraData"`
	MixDigest   common.Hash      `json:"mixHash"`
	Nonce       types.BlockNonce `json:"nonce"`
	BaseFee     *big.Int         `json:"baseFeePerGas" rlp:"optional"`
}

type bbEnvMarshaling struct {
	Difficulty *math.HexOrDecimal256
	Number     *math.HexOrDecimal256
	GasLimit   math.HexOrDecimal64
	GasUsed    math.HexOrDecimal64
	Time       math.HexOrDecimal64
	Extra      hexutil.Bytes
	BaseFee    *math.HexOrDecimal256
}

type blockInput struct {
	Env       *bbEnv   `json:"header,omitempty"`
	UnclesRlp []string `json:"uncles,omitempty"`
	TxRlp     string   `json:"txsRlp,omitempty"`

	Uncles []*types.Header
	Txs    []*types.Transaction
}

func (i *blockInput) toBlock() *types.Block {
	header := &types.Header{
		ParentHash:  i.Env.ParentHash,
		UncleHash:   i.Env.UncleHash,
		Coinbase:    i.Env.Coinbase,
		Root:        i.Env.Root,
		TxHash:      i.Env.TxHash,
		ReceiptHash: i.Env.ReceiptHash,
		Bloom:       i.Env.Bloom,
		Difficulty:  i.Env.Difficulty,
		Number:      i.Env.Number,
		GasLimit:    i.Env.GasLimit,
		GasUsed:     i.Env.GasUsed,
		Time:        i.Env.Time,
		Extra:       i.Env.Extra,
		MixDigest:   i.Env.MixDigest,
		Nonce:       i.Env.Nonce,
		BaseFee:     i.Env.BaseFee,
	}

	none := common.Hash{}
	if header.UncleHash == none {
		header.UncleHash = types.EmptyUncleHash
	}
	if header.TxHash == none {
		header.TxHash = types.EmptyRootHash
	}
	if header.ReceiptHash == none {
		header.ReceiptHash = types.EmptyRootHash
	}
	if header.Difficulty == nil {
		header.Difficulty = common.Big0
	}

	block := types.NewBlockWithHeader(header)
	block.WithBody(i.Txs, i.Uncles)

	return block
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
		return err
	}

	return dispatchBlock(ctx, baseDir, inputData.toBlock())
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
		env, err := readEnv(headerStr)
		if err != nil {
			return nil, err
		}
		inputData.Env = env
	}

	if unclesStr != stdinSelector {
		uncles, err := readUncles(unclesStr)
		if err != nil {
			return nil, err
		}
		inputData.UnclesRlp = uncles
	}

	uncles := []*types.Header{}
	for _, str := range inputData.UnclesRlp {
		var uncle *types.Header
		raw := common.FromHex(str)
		err := rlp.DecodeBytes(raw, &uncle)
		if err != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("unable to decode uncle from rlp data: %v", err))
		}
		uncles = append(uncles, uncle)
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

func readEnv(path string) (*bbEnv, error) {
	env := &bbEnv{}

	if path == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(env); err != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("failed unmarshaling env from stdin: %v", err))
		}
	} else {
		inFile, err := os.Open(path)
		if err != nil {
			return nil, NewError(ErrorIO, fmt.Errorf("failed reading header file: %v", err))
		}
		defer inFile.Close()
		decoder := json.NewDecoder(inFile)
		if err := decoder.Decode(&env); err != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("failed unmarshaling header file: %v", err))
		}
	}

	return env, nil
}

func readUncles(path string) ([]string, error) {
	uncles := []string{}

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

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
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"gopkg.in/urfave/cli.v1"
)

//go:generate gencodec -type bbEnv -field-override bbEnvMarshaling -out gen_bbenv.go
type bbEnv struct {
	ParentHash  common.Hash       `json:"parentHash"`
	UncleHash   *common.Hash      `json:"sha3Uncles"`
	Coinbase    *common.Address   `json:"miner"`
	Root        common.Hash       `json:"stateRoot"        gencodec:"required"`
	TxHash      *common.Hash      `json:"transactionsRoot"`
	ReceiptHash *common.Hash      `json:"receiptsRoot"`
	Bloom       types.Bloom       `json:"logsBloom"`
	Difficulty  *big.Int          `json:"difficulty"`
	Number      *big.Int          `json:"number"           gencodec:"required"`
	GasLimit    uint64            `json:"gasLimit"         gencodec:"required"`
	GasUsed     uint64            `json:"gasUsed"`
	Time        uint64            `json:"timestamp"        gencodec:"required"`
	Extra       []byte            `json:"extraData"`
	MixDigest   common.Hash       `json:"mixHash"`
	Nonce       *types.BlockNonce `json:"nonce"`
	BaseFee     *big.Int          `json:"baseFeePerGas" rlp:"optional"`
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
	Env       *bbEnv       `json:"header,omitempty"`
	UnclesRlp []string     `json:"uncles,omitempty"`
	TxRlp     string       `json:"txsRlp,omitempty"`
	Clique    *cliqueInput `json:"clique,omitempty"`

	EthashDir string
	PowMode   ethash.Mode
	Txs       []*types.Transaction
	Uncles    []*types.Header
}

type cliqueInput struct {
	Key        *ecdsa.PrivateKey
	Voted      *common.Address
	Authorized bool
	Vanity     common.Hash
}

func (c *cliqueInput) UnmarshalJSON(input []byte) error {
	// Read the secretKey, if present
	type sKey struct {
		Key *common.Hash `json:"secretKey"`
	}
	var key sKey
	if err := json.Unmarshal(input, &key); err != nil {
		return err
	}
	if key.Key == nil {
		return errors.New("missing required field 'secretKey' for cliqueInput")
	}
	k := key.Key.Hex()[2:]
	if ecdsaKey, err := crypto.HexToECDSA(k); err != nil {
		return err
	} else {
		c.Key = ecdsaKey
	}

	// Now, read the rest of object
	type others struct {
		Voted      *common.Address `json:"voted"`
		Authorized bool            `json:"authorized"`
		Vanity     common.Hash     `json:"vanity"`
	}
	var x others
	if err := json.Unmarshal(input, &x); err != nil {
		return err
	}
	c.Voted = x.Voted
	c.Authorized = x.Authorized
	c.Vanity = x.Vanity

	return nil
}

func (i *blockInput) toBlock() (*types.Block, error) {
	header := &types.Header{
		ParentHash:  i.Env.ParentHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.Address{},
		Root:        i.Env.Root,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       i.Env.Bloom,
		Difficulty:  common.Big0,
		Number:      i.Env.Number,
		GasLimit:    i.Env.GasLimit,
		GasUsed:     i.Env.GasUsed,
		Time:        i.Env.Time,
		Extra:       i.Env.Extra,
		MixDigest:   i.Env.MixDigest,
		BaseFee:     i.Env.BaseFee,
	}

	// Set optional values to specified values.
	if i.Env.UncleHash != nil {
		header.UncleHash = *i.Env.UncleHash
	}
	if i.Env.Coinbase != nil {
		header.Coinbase = *i.Env.Coinbase
	}
	if i.Env.TxHash != nil {
		header.TxHash = *i.Env.TxHash
	}
	if i.Env.ReceiptHash != nil {
		header.ReceiptHash = *i.Env.ReceiptHash
	}
	if i.Env.Nonce != nil {
		header.Nonce = *i.Env.Nonce
	}
	if header.Difficulty != nil {
		header.Difficulty = i.Env.Difficulty
	}

	block := types.NewBlockWithHeader(header)
	block.WithBody(i.Txs, i.Uncles)

	if i.EthashDir != "" {
		if i.Env.Nonce != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("Sealing with ethash will overwrite specified nonce"))
		}
		ethashConfig := ethash.Config{
			PowMode:        i.PowMode,
			DatasetDir:     i.EthashDir,
			CacheDir:       i.EthashDir,
			DatasetsInMem:  1,
			DatasetsOnDisk: 2,
			CachesInMem:    2,
			CachesOnDisk:   3,
		}
		engine := ethash.New(ethashConfig, nil, false)
		defer engine.Close()

		results := make(chan *types.Block)
		if err := engine.Seal(nil, block, results, nil); err != nil {
			panic(fmt.Sprintf("failed to seal block: %v", err))
		}
		select {
		case found := <-results:
			block.WithSeal(found.Header())
		}
	} else if i.Clique != nil {
		header = block.Header()

		if i.Env.Extra != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("Sealing with clique will overwrite specified extra data"))
		}
		if i.Clique.Voted != nil {
			if i.Env.Coinbase != nil {
				return nil, NewError(ErrorJson, fmt.Errorf("Sealing with clique and voting will overwrite specified coinbase"))
			}
			if i.Env.Nonce != nil {
				return nil, NewError(ErrorJson, fmt.Errorf("Sealing with clique and voting will overwrite specified nonce"))
			}

			header.Coinbase = *i.Clique.Voted
			if i.Clique.Authorized {
				header.Nonce = [8]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
			} else {
				header.Nonce = [8]byte{}
			}
		}

		header.Extra = make([]byte, 97)
		copy(header.Extra[0:32], i.Clique.Vanity.Bytes()[:])
		h := clique.SealHash(header)

		sighash, err := crypto.Sign(h[:], i.Clique.Key)
		if err != nil {
			return nil, err
		}
		copy(header.Extra[32:], sighash)
		block = block.WithSeal(header)
	}

	return block, nil
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
	block, err := inputData.toBlock()
	if err != nil {
		return err
	}

	return dispatchBlock(ctx, baseDir, block)
}

func readInput(ctx *cli.Context) (*blockInput, error) {
	var (
		headerStr  = ctx.String(InputHeaderFlag.Name)
		unclesStr  = ctx.String(InputUnclesFlag.Name)
		txsStr     = ctx.String(InputTxsRlpFlag.Name)
		cliqueStr  = ctx.String(SealerCliqueFlag.Name)
		ethashOn   = ctx.Bool(SealerEthashFlag.Name)
		ethashDir  = ctx.String(SealerEthashDirFlag.Name)
		ethashMode = ctx.String(SealerEthashModeFlag.Name)
		inputData  = &blockInput{}
	)

	if ethashOn && cliqueStr != "" {
		return nil, NewError(ErrorJson, fmt.Errorf("both ethash and clique sealing specified, only one may be chosen"))
	}

	if ethashOn {
		inputData.EthashDir = ethashDir
		switch ethashMode {
		case "normal":
			inputData.PowMode = ethash.ModeNormal
		case "test":
			inputData.PowMode = ethash.ModeTest
		default:
			return nil, NewError(ErrorJson, fmt.Errorf("unknown pow mode: %s", ethashMode))

		}
	}

	if headerStr == stdinSelector || unclesStr == stdinSelector || txsStr == stdinSelector || cliqueStr == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(inputData); err != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("failed unmarshaling stdin: %v", err))
		}
	}

	if cliqueStr != stdinSelector && cliqueStr != "" {
		clique, err := readClique(cliqueStr)
		if err != nil {
			return nil, err
		}
		inputData.Clique = clique
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

func readClique(path string) (*cliqueInput, error) {
	clique := &cliqueInput{}

	if path == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(clique); err != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("failed unmarshaling env from stdin: %v", err))
		}
	} else {
		inFile, err := os.Open(path)
		if err != nil {
			return nil, NewError(ErrorIO, fmt.Errorf("failed reading clique file: %v", err))
		}
		defer inFile.Close()
		decoder := json.NewDecoder(inFile)
		if err := decoder.Decode(&clique); err != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("failed unmarshaling clique file: %v", err))
		}
	}

	return clique, nil
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

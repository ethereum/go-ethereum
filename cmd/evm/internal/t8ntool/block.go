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
	"golang.org/x/crypto/sha3"
	"gopkg.in/urfave/cli.v1"
)

//go:generate gencodec -type header -field-override headerMarshaling -out gen_header.go
type header struct {
	ParentHash  common.Hash       `json:"parentHash"`
	OmmerHash   *common.Hash      `json:"sha3Ommers"`
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

type headerMarshaling struct {
	Difficulty *math.HexOrDecimal256
	Number     *math.HexOrDecimal256
	GasLimit   math.HexOrDecimal64
	GasUsed    math.HexOrDecimal64
	Time       math.HexOrDecimal64
	Extra      hexutil.Bytes
	BaseFee    *math.HexOrDecimal256
}

type bbInput struct {
	Header    *header      `json:"header,omitempty"`
	OmmersRlp []string     `json:"ommers,omitempty"`
	TxRlp     string       `json:"txsRlp,omitempty"`
	Clique    *cliqueInput `json:"clique,omitempty"`

	Ethash    bool
	EthashDir string
	PowMode   ethash.Mode
	Txs       []*types.Transaction
	Ommers    []*types.Header
}

type cliqueInput struct {
	Key        *ecdsa.PrivateKey
	Voted      *common.Address
	Authorized *bool
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
		Authorized *bool           `json:"authorized"`
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

func (i *bbInput) ToBlock() *types.Block {
	header := &types.Header{
		ParentHash:  i.Header.ParentHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.Address{},
		Root:        i.Header.Root,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       i.Header.Bloom,
		Difficulty:  common.Big0,
		Number:      i.Header.Number,
		GasLimit:    i.Header.GasLimit,
		GasUsed:     i.Header.GasUsed,
		Time:        i.Header.Time,
		Extra:       i.Header.Extra,
		MixDigest:   i.Header.MixDigest,
		BaseFee:     i.Header.BaseFee,
	}

	// Fill optional values.
	if i.Header.OmmerHash != nil {
		header.UncleHash = *i.Header.OmmerHash
	} else if len(i.Ommers) != 0 {
		// Calculate the ommer hash if none is provided and there are ommers to hash
		sha := sha3.NewLegacyKeccak256().(crypto.KeccakState)
		rlp.Encode(sha, i.Ommers)
		h := make([]byte, 32)
		sha.Read(h[:])
		header.UncleHash = common.BytesToHash(h)
	}
	if i.Header.Coinbase != nil {
		header.Coinbase = *i.Header.Coinbase
	}
	if i.Header.TxHash != nil {
		header.TxHash = *i.Header.TxHash
	}
	if i.Header.ReceiptHash != nil {
		header.ReceiptHash = *i.Header.ReceiptHash
	}
	if i.Header.Nonce != nil {
		header.Nonce = *i.Header.Nonce
	}
	if header.Difficulty != nil {
		header.Difficulty = i.Header.Difficulty
	}

	block := types.NewBlockWithHeader(header)
	block = block.WithBody(i.Txs, i.Ommers)

	return block
}

func (i *bbInput) SealBlock(block *types.Block) (*types.Block, error) {
	if i.Ethash {
		if i.Header.Nonce != nil {
			return nil, NewError(ErrorConfig, fmt.Errorf("sealing with ethash will overwrite provided nonce"))
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
		found := <-results
		block.WithSeal(found.Header())
	} else if i.Clique != nil {
		// If any clique value overwrites an explicit header value, fail
		// to avoid silently building a block with unexpected values.
		if i.Header.Extra != nil {
			return nil, NewError(ErrorConfig, fmt.Errorf("sealing with clique will overwrite provided extra data"))
		}
		header := block.Header()
		if i.Clique.Voted != nil {
			if i.Header.Coinbase != nil {
				return nil, NewError(ErrorConfig, fmt.Errorf("sealing with clique and voting will overwrite provided coinbase"))
			}
			header.Coinbase = *i.Clique.Voted
		}
		if i.Clique.Authorized != nil {
			if i.Header.Nonce != nil {
				return nil, NewError(ErrorConfig, fmt.Errorf("sealing with clique and voting will overwrite provided nonce"))
			}
			if *i.Clique.Authorized {
				header.Nonce = [8]byte{}
			} else {
				header.Nonce = [8]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
			}
		}
		// Extra is fixed 32 byte vanity and 65 byte signature
		header.Extra = make([]byte, 32+65)
		copy(header.Extra[0:32], i.Clique.Vanity.Bytes()[:])

		// Sign the seal hash and fill in the rest of the extra data
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

	block := inputData.ToBlock()
	block, err = inputData.SealBlock(block)
	if err != nil {
		return err
	}

	return dispatchBlock(ctx, baseDir, block)
}

func readInput(ctx *cli.Context) (*bbInput, error) {
	var (
		headerStr  = ctx.String(InputHeaderFlag.Name)
		ommersStr  = ctx.String(InputOmmersFlag.Name)
		txsStr     = ctx.String(InputTxsRlpFlag.Name)
		cliqueStr  = ctx.String(SealCliqueFlag.Name)
		ethashOn   = ctx.Bool(SealEthashFlag.Name)
		ethashDir  = ctx.String(SealEthashDirFlag.Name)
		ethashMode = ctx.String(SealEthashModeFlag.Name)
		inputData  = &bbInput{}
	)

	if ethashOn && cliqueStr != "" {
		return nil, NewError(ErrorConfig, fmt.Errorf("both ethash and clique sealing specified, only one may be chosen"))
	}

	if ethashOn {
		inputData.Ethash = ethashOn
		inputData.EthashDir = ethashDir
		switch ethashMode {
		case "normal":
			inputData.PowMode = ethash.ModeNormal
		case "test":
			inputData.PowMode = ethash.ModeTest
		default:
			return nil, NewError(ErrorConfig, fmt.Errorf("unknown pow mode: %s", ethashMode))

		}
	}

	if headerStr == stdinSelector || ommersStr == stdinSelector || txsStr == stdinSelector || cliqueStr == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(inputData); err != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("failed unmarshaling stdin: %v", err))
		}
	}
	if cliqueStr != stdinSelector && cliqueStr != "" {
		var clique cliqueInput
		err := readFile(cliqueStr, "clique", &clique)
		if err != nil {
			return nil, err
		}
		inputData.Clique = &clique
	}
	if headerStr != stdinSelector {
		var env header
		err := readFile(headerStr, "header", &env)
		if err != nil {
			return nil, err
		}
		inputData.Header = &env
	}
	if ommersStr != stdinSelector && ommersStr != "" {
		var ommers []string
		err := readFile(ommersStr, "ommers", &ommers)
		if err != nil {
			return nil, err
		}
		inputData.OmmersRlp = ommers
	}
	if txsStr != stdinSelector {
		var txs string
		err := readFile(txsStr, "txs", &txs)
		if err != nil {
			return nil, err
		}
		inputData.TxRlp = txs
	}

	// Deserialize rlp txs and ommers
	var (
		ommers = []*types.Header{}
		txs    = []*types.Transaction{}
	)

	if err := rlp.DecodeBytes(common.FromHex(inputData.TxRlp), &txs); err != nil {
		return nil, NewError(ErrorRlp, fmt.Errorf("unable to decode transaction from rlp data: %v", err))
	}
	inputData.Txs = txs

	for _, str := range inputData.OmmersRlp {
		type extblock struct {
			Header *types.Header
			Txs    []*types.Transaction
			Ommers []*types.Header
		}
		var ommer *extblock
		if err := rlp.DecodeBytes(common.FromHex(str), &ommer); err != nil {
			return nil, NewError(ErrorRlp, fmt.Errorf("unable to decode ommer from rlp data: %v", err))
		}
		ommers = append(ommers, ommer.Header)
	}
	inputData.Ommers = ommers

	return inputData, nil
}

func readFile(path, desc string, dest interface{}) error {
	inFile, err := os.Open(path)
	if err != nil {
		return NewError(ErrorIO, fmt.Errorf("failed reading %s file: %v", desc, err))
	}
	defer inFile.Close()

	decoder := json.NewDecoder(inFile)
	if err := decoder.Decode(dest); err != nil {
		return NewError(ErrorJson, fmt.Errorf("failed unmarshaling %s file: %v", desc, err))
	}

	return nil
}

// dispatchOutput writes the output data to either stderr or stdout, or to the specified
// files
func dispatchBlock(ctx *cli.Context, baseDir string, block *types.Block) error {
	raw, _ := rlp.EncodeToBytes(block)

	type blockInfo struct {
		Rlp  hexutil.Bytes `json:"rlp"`
		Hash common.Hash   `json:"hash"`
	}

	var enc blockInfo
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

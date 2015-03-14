/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU Lesser General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU Lesser General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Gustav Simonsson <gustav.simonsson@gmail.com>
 * @date 2015
 *
 */

package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/core"
	types "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/rlp"
)

type Account struct {
	Balance string
	Code    string
	Nonce   string
	Storage map[string]string
}

type BlockHeader struct {
	Bloom            string
	Coinbase         string
	Difficulty       string
	ExtraData        string
	GasLimit         string
	GasUsed          string
	MixHash          string
	Nonce            string
	Number           string
	ParentHash       string
	ReceiptTrie      string
	SeedHash         string
	StateRoot        string
	Timestamp        string
	TransactionsTrie string
	UncleHash        string
}

type Tx struct {
	Data     string
	GasLimit string
	GasPrice string
	Nonce    string
	R        string
	S        string
	To       string
	V        string
	Value    string
}

type Block struct {
	BlockHeader  BlockHeader
	Rlp          string
	Transactions []Tx
	UncleHeaders []string
}

type Test struct {
	Blocks             []Block
	GenesisBlockHeader BlockHeader
	Pre                map[string]Account
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s <testfile>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stderr, log.LstdFlags, logger.DebugDetailLevel))
	defer func() { logger.Flush() }()

	if len(os.Args) < 2 {
		utils.Fatalf("Please specify a test file as the first argument.")
	}
	blocks, err := loadBlocksFromTestFile(os.Args[1])
	if err != nil {
		utils.Fatalf("Could not load blocks: %v", err)
	}

	chain := memchain()
	chain.ResetWithGenesisBlock(blocks[0])
	if err = chain.InsertChain(types.Blocks{blocks[1]}); err != nil {
		utils.Fatalf("Error: %v", err)
	} else {
		fmt.Println("PASS")
	}
}

func memchain() *core.ChainManager {
	blockdb, err := ethdb.NewMemDatabase()
	if err != nil {
		utils.Fatalf("Could not create in-memory database: %v", err)
	}
	statedb, err := ethdb.NewMemDatabase()
	if err != nil {
		utils.Fatalf("Could not create in-memory database: %v", err)
	}
	return core.NewChainManager(blockdb, statedb, new(event.TypeMux))
}

func loadBlocksFromTestFile(filePath string) (blocks types.Blocks, err error) {
	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		return
	}
	bt := make(map[string]Test)
	if err = json.Unmarshal(fileContent, &bt); err != nil {
		return
	}

	// TODO: support multiple blocks; loop over all blocks
	gbh := new(types.Header)

	// Let's use slighlty different namings for the same things, because that's awesome.
	gbh.ParentHash, err = hex_decode(bt["SimpleTx"].GenesisBlockHeader.ParentHash)
	gbh.UncleHash, err = hex_decode(bt["SimpleTx"].GenesisBlockHeader.UncleHash)
	gbh.Coinbase, err = hex_decode(bt["SimpleTx"].GenesisBlockHeader.Coinbase)
	gbh.Root, err = hex_decode(bt["SimpleTx"].GenesisBlockHeader.StateRoot)
	gbh.TxHash, err = hex_decode(bt["SimpleTx"].GenesisBlockHeader.TransactionsTrie)
	gbh.ReceiptHash, err = hex_decode(bt["SimpleTx"].GenesisBlockHeader.ReceiptTrie)
	gbh.Bloom, err = hex_decode(bt["SimpleTx"].GenesisBlockHeader.Bloom)

	gbh.MixDigest, err = hex_decode(bt["SimpleTx"].GenesisBlockHeader.MixHash)
	//gbh.SeedHash, err = hex_decode(bt["SimpleTx"].GenesisBlockHeader.SeedHash)

	d, _ := new(big.Int).SetString(bt["SimpleTx"].GenesisBlockHeader.Difficulty, 10)
	gbh.Difficulty = d

	n, _ := new(big.Int).SetString(bt["SimpleTx"].GenesisBlockHeader.Number, 10)
	gbh.Number = n

	gl, _ := new(big.Int).SetString(bt["SimpleTx"].GenesisBlockHeader.GasLimit, 10)
	gbh.GasLimit = gl

	gu, _ := new(big.Int).SetString(bt["SimpleTx"].GenesisBlockHeader.GasUsed, 10)
	gbh.GasUsed = gu

	ts, _ := new(big.Int).SetString(bt["SimpleTx"].GenesisBlockHeader.Timestamp, 0)
	gbh.Time = ts.Uint64()

	extra, err := hex_decode(bt["SimpleTx"].GenesisBlockHeader.ExtraData)
	gbh.Extra = string(extra) // TODO: change ExtraData to byte array

	nonce, _ := hex_decode(bt["SimpleTx"].GenesisBlockHeader.Nonce)
	gbh.Nonce = nonce

	if err != nil {
		return
	}

	gb := types.NewBlockWithHeader(gbh)
	//gb.uncles = *new([]*types.Header)
	//gb.transactions = *new(types.Transactions)
	gb.Td = new(big.Int)
	gb.Reward = new(big.Int)

	testBlock := new(types.Block)

	rlpBytes, err := hex_decode(bt["SimpleTx"].Blocks[0].Rlp)
	err = rlp.Decode(bytes.NewReader(rlpBytes), &testBlock)
	if err != nil {
		return
	}

	blocks = types.Blocks{
		gb,
		testBlock,
	}

	return
}

func hex_decode(s string) (res []byte, err error) {
	return hex.DecodeString(strings.TrimPrefix(s, "0x"))
}

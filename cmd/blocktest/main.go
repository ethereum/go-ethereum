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
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	types "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	ClientIdentifier = "Ethereum(G)"
	Version          = "0.8.6"
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

var (
	Identifier       string
	KeyRing          string
	DiffTool         bool
	DiffType         string
	KeyStore         string
	StartRpc         bool
	StartWebSockets  bool
	RpcListenAddress string
	RpcPort          int
	WsPort           int
	OutboundPort     string
	ShowGenesis      bool
	AddPeer          string
	MaxPeer          int
	GenAddr          bool
	BootNodes        string
	NodeKey          *ecdsa.PrivateKey
	NAT              nat.Interface
	SecretFile       string
	ExportDir        string
	NonInteractive   bool
	Datadir          string
	LogFile          string
	ConfigFile       string
	DebugFile        string
	LogLevel         int
	LogFormat        string
	Dump             bool
	DumpHash         string
	DumpNumber       int
	VmType           int
	ImportChain      string
	SHH              bool
	Dial             bool
	PrintVersion     bool
	MinerThreads     int
)

// flags specific to cli client
var (
	StartMining    bool
	StartJsConsole bool
	InputFile      string
)

func main() {
	init_vars()

	Init()

	if len(TestFile) < 1 {
		log.Fatal("Please specify test file")
	}
	blocks, err := loadBlocksFromTestFile(TestFile)
	if err != nil {
		panic(err)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	defer func() {
		logger.Flush()
	}()

	//utils.HandleInterrupt()

	utils.InitConfig(VmType, ConfigFile, Datadir, "ethblocktest")

	ethereum, err := eth.New(&eth.Config{
		Name:         p2p.MakeName(ClientIdentifier, Version),
		KeyStore:     KeyStore,
		DataDir:      Datadir,
		LogFile:      LogFile,
		LogLevel:     LogLevel,
		LogFormat:    LogFormat,
		MaxPeers:     MaxPeer,
		Port:         OutboundPort,
		NAT:          NAT,
		KeyRing:      KeyRing,
		Shh:          true,
		Dial:         Dial,
		BootNodes:    BootNodes,
		NodeKey:      NodeKey,
		MinerThreads: MinerThreads,
	})

	utils.StartEthereumForTest(ethereum)
	utils.StartRpc(ethereum, RpcListenAddress, RpcPort)

	ethereum.ChainManager().ResetWithGenesisBlock(blocks[0])
	// bph := ethereum.ChainManager().GetBlock(blocks[1].Header().ParentHash)
	// fmt.Println("bph: ", bph)

	//fmt.Println("b0: ", hex.EncodeToString(ethutil.Encode(blocks[0].RlpData())))
	//fmt.Println("b0: ", hex.EncodeToString(blocks[0].Hash()))
	//fmt.Println("b1: ", hex.EncodeToString(ethutil.Encode(blocks[1].RlpData())))
	//fmt.Println("b1: ", hex.EncodeToString(blocks[1].Hash()))

	go ethereum.ChainManager().InsertChain(types.Blocks{blocks[1]})
	fmt.Println("OK! ")
	ethereum.WaitForShutdown()
}

func loadBlocksFromTestFile(filePath string) (blocks types.Blocks, err error) {
	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		return
	}
	bt := *new(map[string]Test)
	err = json.Unmarshal(fileContent, &bt)
	if err != nil {
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
	gbh.SeedHash, err = hex_decode(bt["SimpleTx"].GenesisBlockHeader.SeedHash)

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

func init_vars() {
	VmType = 0
	Identifier = ""
	KeyRing = ""
	KeyStore = "db"
	RpcListenAddress = "127.0.0.1"
	RpcPort = 8545
	WsPort = 40404
	StartRpc = true
	StartWebSockets = false
	NonInteractive = false
	GenAddr = false
	SecretFile = ""
	ExportDir = ""
	LogFile = ""

	timeStr := strconv.FormatInt(time.Now().UnixNano(), 10)

	Datadir = path.Join(ethutil.DefaultDataDir(), timeStr)
	ConfigFile = path.Join(ethutil.DefaultDataDir(), timeStr, "conf.ini")

	DebugFile = ""
	LogLevel = 5
	LogFormat = "std"
	DiffTool = false
	DiffType = "all"
	ShowGenesis = false
	ImportChain = ""
	Dump = false
	DumpHash = ""
	DumpNumber = -1
	StartMining = false
	StartJsConsole = false
	PrintVersion = false
	MinerThreads = runtime.NumCPU()

	Dial = false
	OutboundPort = "30303"
	BootNodes = ""
	MaxPeer = 1

}

func hex_decode(s string) (res []byte, err error) {
	return hex.DecodeString(strings.TrimPrefix(s, "0x"))
}

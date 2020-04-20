// Copyright 2019 The go-ethereum Authors
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
	"bytes"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/params"
	"io/ioutil"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"

	cli "gopkg.in/urfave/cli.v1"
)

var (
	rpcPortFlag = cli.IntFlag{
		Name:  "rpcport",
		Usage: "HTTP-RPC server listening port",
		Value: node.DefaultHTTPPort,
	}
	retestethCommand = cli.Command{
		Action:      utils.MigrateFlags(retesteth),
		Name:        "retesteth",
		Usage:       "Launches geth in retesteth mode",
		ArgsUsage:   "",
		Flags:       []cli.Flag{rpcPortFlag},
		Category:    "MISCELLANEOUS COMMANDS",
		Description: `Launches geth in retesteth mode (no database, no network, only retesteth RPC interface)`,
	}
)

// Main test api
type RetestethTestAPI interface {
	SetChainParams(ctx context.Context, chainParams ChainParams) (bool, error)
	MineBlocks(ctx context.Context, number uint64) (bool, error)
	ModifyTimestamp(ctx context.Context, interval uint64) (bool, error)    			   //Subject to remove
	ImportRawBlock(ctx context.Context, rawBlock hexutil.Bytes) (common.Hash, error)
	RewindToBlock(ctx context.Context, number uint64) (bool, error)
	GetLogHash(ctx context.Context, txHash common.Hash) (common.Hash, error)
}

type RetestethDebugAPI interface {
	StorageRangeAt(ctx context.Context,
		blockHashOrNumber *math.HexOrDecimal256, txIndex uint64,
		address common.Address,
		begin *math.HexOrDecimal256, maxResults uint64,
	) (StorageRangeResult, error)
}

type StorageRangeResult struct {
	Complete bool                   `json:"complete"`
	Storage  map[common.Hash]SRItem `json:"storage"`
}

type SRItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RetestethAPI struct {
	testclient	  *tester
	rpchandler    *rpc.Server

	author        common.Address    // Block coinbase for test_mineBlocks
	extraData     []byte			// Block extradata for test_mineBlocks
	blockInterval uint64
}

// tester is a list of classes that are being tested
type tester struct {
	workspace string		// Temp directory
	stack     *node.Node
	ethereum  *eth.Ethereum
}

// test_setChainParams json struct
type ChainParams struct {
	SealEngine string                            `json:"sealEngine"`
	Genesis    core.Genesis	                     `json:"genesis"`
}

// Restart the client with new genesis. Reset txPool and blockchain info
func (api *RetestethAPI) SetChainParams(ctx context.Context, chainParams ChainParams) (bool, error) {
	if api.testclient != nil {
		api.testclient.Close()
	}

	api.testclient = newTester(&chainParams.Genesis)
	//api.testclient.ethereum.SetEtherbase(chainParams.Genesis.Coinbase)
	//api.testclient.ethereum.Miner().SetExtra(chainParams.Genesis.ExtraData)
	api.extraData = chainParams.Genesis.ExtraData
	api.author = chainParams.Genesis.Coinbase

	nonceLock := new(ethapi.AddrLocker) // !!!new
	api.rpchandler.RegisterName("eth", ethapi.NewPublicBlockChainAPI(api.testclient.ethereum.APIBackend))
	api.rpchandler.RegisterName("eth", ethapi.NewPublicTransactionPoolAPI(api.testclient.ethereum.APIBackend, nonceLock))
	api.rpchandler.RegisterName("debug", eth.NewPrivateDebugAPI(api.testclient.ethereum))
	api.rpchandler.RegisterName("web3", node.NewPublicWeb3API(api.testclient.stack))
	return true, nil
}

// Seal transactions from txPool, imported by eth_sendRawTransaction, into a block
func (api *RetestethAPI) MineBlocks(ctx context.Context, number uint64) (bool, error) {
	currentNumber := api.testclient.ethereum.BlockChain().CurrentBlock().Number().Uint64()
	startBlock := currentNumber
	upToBlock := currentNumber + number
	fakeBlockRemoved := false  // Geth adds empty block immediately with StartMining. need to remove

	// NOT THREAD SAFE!!!
	api.testclient.ethereum.StartMining(1)
	times := 0

	for currentNumber < upToBlock {
		currentNumber = api.testclient.ethereum.BlockChain().CurrentBlock().Number().Uint64()
		times += 1
		if currentNumber == startBlock + 1 && fakeBlockRemoved == false {
			if api.testclient.ethereum.BlockChain().CurrentBlock().Transactions().Len() == 0 {
				api.testclient.ethereum.BlockChain().Reset()
				currentNumber = startBlock
				fakeBlockRemoved = true
			}
		}
	}

	api.testclient.ethereum.StopMining()
	api.testclient.ethereum.BlockChain().SetHead(upToBlock)
	time.Sleep(time.Second * 1)
	return true, nil
}

func (api *RetestethAPI) mineBlock() error {
	number := api.testclient.ethereum.BlockChain().CurrentBlock().Number().Uint64()
	parentHash := rawdb.ReadCanonicalHash(api.testclient.ethereum.ChainDb(), number)
	parent := rawdb.ReadBlock(api.testclient.ethereum.ChainDb(), parentHash, number)
	var timestamp uint64
	if api.blockInterval == 0 {
		timestamp = uint64(time.Now().Unix())
	} else {
		timestamp = parent.Time() + api.blockInterval
	}
	gasLimit := core.CalcGasLimit(parent, 9223372036854775807, 9223372036854775807)
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     big.NewInt(int64(number + 1)),
		GasLimit:   gasLimit,
		Extra:      api.extraData,
		Time:       timestamp,
	}
	header.Coinbase = api.author

	api.testclient.ethereum.Engine().Prepare(api.testclient.ethereum.BlockChain(), header)

	// If we are care about TheDAO hard-fork check whether to override the extra-data or not
	if daoBlock := api.testclient.ethereum.BlockChain().Config().DAOForkBlock; daoBlock != nil {
		// Check whether the block is among the fork extra-override range
		limit := new(big.Int).Add(daoBlock, params.DAOForkExtraRange)
		if header.Number.Cmp(daoBlock) >= 0 && header.Number.Cmp(limit) < 0 {
			// Depending whether we support or oppose the fork, override differently
			if api.testclient.ethereum.BlockChain().Config().DAOForkSupport {
				header.Extra = common.CopyBytes(params.DAOForkBlockExtra)
			} else if bytes.Equal(header.Extra, params.DAOForkBlockExtra) {
				header.Extra = []byte{} // If miner opposes, don't let it use the reserved extra-data
			}
		}
	}
	statedb, err := api.testclient.ethereum.BlockChain().StateAt(parent.Root())
	if err != nil {
		return err
	}
	if api.testclient.ethereum.BlockChain().Config().DAOForkSupport && api.testclient.ethereum.BlockChain().Config().DAOForkBlock != nil && api.testclient.ethereum.BlockChain().Config().DAOForkBlock.Cmp(header.Number) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	gasPool := new(core.GasPool).AddGas(header.GasLimit)
	txCount := 0
	var txs []*types.Transaction
	var receipts []*types.Receipt
	var blockFull = gasPool.Gas() < params.TxGas
	var txPending,_ = api.testclient.ethereum.TxPool().Pending()
	for address := range txPending {
		if blockFull {
			break
		}

		var m = txPending[address]

		for nonce := statedb.GetNonce(address); ; nonce++ {
			if tx := m[0]; true {
				// Try to apply transactions to the state
				statedb.Prepare(tx.Hash(), common.Hash{}, txCount)
				snap := statedb.Snapshot()

				receipt, err := core.ApplyTransaction(
					api.testclient.ethereum.BlockChain().Config(),
					api.testclient.ethereum.BlockChain(),
					&api.author,
					gasPool,
					statedb,
					header, tx, &header.GasUsed, *api.testclient.ethereum.BlockChain().GetVMConfig(),
				)
				if err != nil {
					statedb.RevertToSnapshot(snap)
					break
				}
				txs = append(txs, tx)
				receipts = append(receipts, receipt)
				//delete(m, nonce)
				if len(m) == 0 {
					// Last tx for the sender
					//delete(api.txMap, address)
					//delete(api.txSenders, address)
				}
				txCount++
				if gasPool.Gas() < params.TxGas {
					blockFull = true
					break
				}
			} else {
				break // Gap in the nonces
			}
		}
	}
	block, err := api.testclient.ethereum.Engine().FinalizeAndAssemble(api.testclient.ethereum.BlockChain(), header, statedb, txs, []*types.Header{}, receipts)
	if err != nil {
		return err
	}
	return api.importBlock(block)
}

// Original storage range implementation. Geth StorageRangeAt has no consensus
func (api *RetestethAPI) StorageRangeAt2(ctx context.Context,
	blockHashOrNumber *math.HexOrDecimal256, txIndex uint64,
	address common.Address,
	begin *math.HexOrDecimal256, maxResults uint64,
) (StorageRangeResult, error) {
	var (
		header *types.Header
		block  *types.Block
	)
	if (*big.Int)(blockHashOrNumber).Cmp(big.NewInt(math.MaxInt64)) > 0 {
		blockHash := common.BigToHash((*big.Int)(blockHashOrNumber))
		header = api.testclient.ethereum.BlockChain().GetHeaderByHash(blockHash)
		block = api.testclient.ethereum.BlockChain().GetBlockByHash(blockHash)
	} else {
		blockNumber := (*big.Int)(blockHashOrNumber).Uint64()
		header = api.testclient.ethereum.BlockChain().GetHeaderByNumber(blockNumber)
		block = api.testclient.ethereum.BlockChain().GetBlockByNumber(blockNumber)
	}
	parentHeader := api.testclient.ethereum.BlockChain().GetHeaderByHash(header.ParentHash)
	var root common.Hash
	var statedb *state.StateDB
	var err error
	if parentHeader == nil || int(txIndex) >= len(block.Transactions()) {
		root = header.Root
		statedb, err = api.testclient.ethereum.BlockChain().StateAt(root)
		if err != nil {
			return StorageRangeResult{}, err
		}
	} else {
		root = parentHeader.Root
		statedb, err = api.testclient.ethereum.BlockChain().StateAt(root)
		if err != nil {
			return StorageRangeResult{}, err
		}
		// Recompute transactions up to the target index.
		signer := types.MakeSigner(api.testclient.ethereum.BlockChain().Config(), block.Number())
		for idx, tx := range block.Transactions() {
			// Assemble the transaction call message and return if the requested offset
			msg, _ := tx.AsMessage(signer)
			context := core.NewEVMContext(msg, block.Header(), api.testclient.ethereum.BlockChain(), nil)
			// Not yet the searched for transaction, execute on top of the current state
			vmenv := vm.NewEVM(context, statedb, api.testclient.ethereum.BlockChain().Config(), vm.Config{})
			if _, _, _, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(tx.Gas())); err != nil {
				return StorageRangeResult{}, fmt.Errorf("transaction %#x failed: %v", tx.Hash(), err)
			}
			// Ensure any modifications are committed to the state
			// Only delete empty objects if EIP158/161 (a.k.a Spurious Dragon) is in effect
			_ = statedb.IntermediateRoot(vmenv.ChainConfig().IsEIP158(block.Number()))
			if idx == int(txIndex) {
				// This is to make sure root can be opened by OpenTrie
				_, err = statedb.Commit(vmenv.ChainConfig().IsEIP158(block.Number()))
				if err != nil {
					return StorageRangeResult{}, err
				}
			}
		}
	}
	storageTrie := statedb.StorageTrie(address)
	it := trie.NewIterator(storageTrie.NodeIterator(common.BigToHash((*big.Int)(begin)).Bytes()))
	result := StorageRangeResult{Storage: make(map[common.Hash]SRItem)}
	for i := 0; /*i < int(maxResults) && */ it.Next(); i++ {
		if preimage := storageTrie.GetKey(it.Key); preimage != nil {
			key := (*math.HexOrDecimal256)(big.NewInt(0).SetBytes(preimage))
			v, _, err := rlp.SplitString(it.Value)
			if err != nil {
				return StorageRangeResult{}, err
			}
			value := (*math.HexOrDecimal256)(big.NewInt(0).SetBytes(v))
			ks, _ := key.MarshalText()
			vs, _ := value.MarshalText()
			if len(ks)%2 != 0 {
				ks = append(append(append([]byte{}, ks[:2]...), byte('0')), ks[2:]...)
			}
			if len(vs)%2 != 0 {
				vs = append(append(append([]byte{}, vs[:2]...), byte('0')), vs[2:]...)
			}
			result.Storage[common.BytesToHash(it.Key)] = SRItem{
				Key:   string(ks),
				Value: string(vs),
			}
		}
	}
	if it.Next() {
		result.Complete = false
	} else {
		result.Complete = true
	}
	return result, nil
}

// Subject to remove
func (api *RetestethAPI) ModifyTimestamp(ctx context.Context, interval uint64) (bool, error) {
	api.blockInterval = interval
	return true, nil
}

func (api *RetestethAPI) ImportRawBlock(ctx context.Context, rawBlock hexutil.Bytes) (common.Hash, error) {
	block := new(types.Block)
	if err := rlp.DecodeBytes(rawBlock, block); err != nil {
		return common.Hash{}, err
	}
	fmt.Printf("Importing block %d with parent hash: %x, genesisHash: %x\n", block.NumberU64(), block.ParentHash(), api.testclient.ethereum.BlockChain().Genesis().Hash())
	if err := api.importBlock(block); err != nil {
		return common.Hash{}, err
	}
	return block.Hash(), nil
}

func (api *RetestethAPI) importBlock(block *types.Block) error {
	if _, err := api.testclient.ethereum.BlockChain().InsertChain([]*types.Block{block}); err != nil {
		return err
	}
	fmt.Printf("Imported block %d,  head is %d\n", block.NumberU64(), api.testclient.ethereum.BlockChain().CurrentBlock().Number())
	return nil
}

func (api *RetestethAPI) RewindToBlock(ctx context.Context, newHead uint64) (bool, error) {
	if err := api.testclient.ethereum.BlockChain().SetHead(newHead); err != nil {
		return false, err
	}
	return true, nil
}

var emptyListHash common.Hash = common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347")

func (api *RetestethAPI) GetLogHash(ctx context.Context, txHash common.Hash) (common.Hash, error) {
	receipt, _, _, _ := rawdb.ReadReceipt(api.testclient.ethereum.ChainDb(), txHash, api.testclient.ethereum.BlockChain().Config())
	if receipt == nil {
		return emptyListHash, nil
	} else {
		if logListRlp, err := rlp.EncodeToBytes(receipt.Logs); err != nil {
			return common.Hash{}, err
		} else {
			return common.BytesToHash(crypto.Keccak256(logListRlp)), nil
		}
	}
}

// splitAndTrim splits input separated by a comma
// and trims excessive white space from the substrings.
func splitAndTrim(input string) []string {
	result := strings.Split(input, ",")
	for i, r := range result {
		result[i] = strings.TrimSpace(r)
	}
	return result
}

// newTester creates a test environment based on which the console can operate.
// Please ensure you call Close() on the returned tester to avoid leaks.
func newTester(genesisConfig *core.Genesis) *tester {
	// Create a temporary storage for the node keys and initialize it
	workspace, err := ioutil.TempDir("", "console-tester-")
	if err != nil {
		utils.Fatalf("failed to create temporary keystore: %v", err)
	}

	// Create a networkless protocol stack and start an Ethereum service within
	stack, err := node.New(&node.Config{DataDir: workspace, UseLightweightKDF: true, Name: "retesteth"})
	if err != nil {
		utils.Fatalf("failed to create node: %v", err)
	}
	ethConf := &eth.Config{
		Genesis: genesisConfig,
		Miner: miner.Config{
			Etherbase: genesisConfig.Coinbase,
			ExtraData: genesisConfig.ExtraData,
			GasFloor: 0,
			GasCeil: genesisConfig.GasLimit,
			GasPrice: big.NewInt(0),
			Recommit: 1 * time.Second,
			Noverify: true,
		},
		Ethash: ethash.Config{
			PowMode: ethash.ModeFake,
		},
		TxPool: core.TxPoolConfig{
			NoLocals: false,
			PriceLimit: 0,
		},
	}
	if err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) { return eth.New(ctx, ethConf) }); err != nil {
		utils.Fatalf("failed to register Ethereum protocol: %v", err)
	}
	// Start the node and assemble the JavaScript console around it
	if err = stack.Start(); err != nil {
		utils.Fatalf("failed to start test stack: %v", err)
	}

	// Create the final tester and return
	var ethereum *eth.Ethereum
	stack.Service(&ethereum)

	return &tester{
		workspace: workspace,
		stack:     stack,
		ethereum:  ethereum,
	}
}

// Close cleans up any temporary data folders and held resources.
func (env *tester) Close() {
	if err := env.stack.Close(); err != nil {
		utils.Fatalf("failed to tear down embedded node: %v", err)
	}
	env.ethereum.TxPool().Stop()
	os.RemoveAll(env.workspace)
}

func retesteth(ctx *cli.Context) error {
	apiImpl := &RetestethAPI{}
	log.Info("Welcome to retesteth!")

	// register signer API with server
	var (
		extapiURL string
	)
	rpcAPI := []rpc.API{
		{
			Namespace: "test",
			Public:    true,
			Service:   apiImpl,
			Version:   "1.0",
		},
		{
			Namespace: "debug",
			Public:    true,
			Service:   apiImpl,
			Version:   "1.0",
		},
	}

	vhosts := splitAndTrim(ctx.GlobalString(utils.RPCVirtualHostsFlag.Name))
	cors := splitAndTrim(ctx.GlobalString(utils.RPCCORSDomainFlag.Name))

	// start http server
	var RetestethHTTPTimeouts = rpc.HTTPTimeouts{
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	httpEndpoint := fmt.Sprintf("%s:%d", ctx.GlobalString(utils.RPCListenAddrFlag.Name), ctx.Int(rpcPortFlag.Name))
	listener, rpcHandler, err := rpc.StartHTTPEndpoint(httpEndpoint, rpcAPI, []string{"test", "debug"}, cors, vhosts, RetestethHTTPTimeouts)
	if err != nil {
		utils.Fatalf("Could not start RPC api: %v", err)
	}
	apiImpl.rpchandler = rpcHandler

	extapiURL = fmt.Sprintf("http://%s", httpEndpoint)
	log.Info("HTTP endpoint opened", "url", extapiURL)

	defer func() {
		listener.Close()
		log.Info("HTTP endpoint closed", "url", httpEndpoint)
	}()

	abortChan := make(chan os.Signal, 11)
	signal.Notify(abortChan, os.Interrupt)

	sig := <-abortChan
	log.Info("Exiting...", "signal", sig)
	return nil
}

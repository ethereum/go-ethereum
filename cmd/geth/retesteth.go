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
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
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
	"github.com/ethereum/go-ethereum/params"
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

type RetestethTestAPI interface {
	SetChainParams(ctx context.Context, chainParams ChainParams) (bool, error)
	MineBlocks(ctx context.Context, number uint64) (bool, error)
	ModifyTimestamp(ctx context.Context, interval uint64) (bool, error)
	ImportRawBlock(ctx context.Context, rawBlock hexutil.Bytes) (common.Hash, error)
	RewindToBlock(ctx context.Context, number uint64) (bool, error)
	GetLogHash(ctx context.Context, txHash common.Hash) (common.Hash, error)
}

type RetestethAPI struct {
	testclient	  *tester
	rpchandler    *rpc.Server

	// Block Mining Info
	extraData     []byte
	engine        *NoRewardEngine
	blockInterval uint64

	txMap         map[common.Address]map[uint64]*types.Transaction // Sender -> Nonce -> Transaction
	txSenders     map[common.Address]struct{}                      // Set of transaction senders
}

// tester is a list of classes that are being tested
type tester struct {
	workspace string
	stack     *node.Node
	ethereum  *eth.Ethereum
}

type ChainParams struct {
	SealEngine string                            `json:"sealEngine"`
	Genesis    core.Genesis	                     `json:"genesis"`
}

type NoRewardEngine struct {
	inner     consensus.Engine
	rewardsOn bool
}

func (e *NoRewardEngine) Author(header *types.Header) (common.Address, error) {
	return e.inner.Author(header)
}

func (e *NoRewardEngine) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return e.inner.VerifyHeader(chain, header, seal)
}

func (e *NoRewardEngine) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	return e.inner.VerifyHeaders(chain, headers, seals)
}

func (e *NoRewardEngine) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	return e.inner.VerifyUncles(chain, block)
}

func (e *NoRewardEngine) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	return e.inner.VerifySeal(chain, header)
}

func (e *NoRewardEngine) Prepare(chain consensus.ChainReader, header *types.Header) error {
	return e.inner.Prepare(chain, header)
}

func (e *NoRewardEngine) accumulateRewards(config *params.ChainConfig, state *state.StateDB, header *types.Header, uncles []*types.Header) {
	// Simply touch miner and uncle coinbase accounts
	reward := big.NewInt(0)
	for _, uncle := range uncles {
		state.AddBalance(uncle.Coinbase, reward)
	}
	state.AddBalance(header.Coinbase, reward)
}

func (e *NoRewardEngine) Finalize(chain consensus.ChainReader, header *types.Header, statedb *state.StateDB, txs []*types.Transaction,
	uncles []*types.Header) {
	if e.rewardsOn {
		e.inner.Finalize(chain, header, statedb, txs, uncles)
	} else {
		e.accumulateRewards(chain.Config(), statedb, header, uncles)
		header.Root = statedb.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	}
}

func (e *NoRewardEngine) FinalizeAndAssemble(chain consensus.ChainReader, header *types.Header, statedb *state.StateDB, txs []*types.Transaction,
	uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	if e.rewardsOn {
		return e.inner.FinalizeAndAssemble(chain, header, statedb, txs, uncles, receipts)
	} else {
		e.accumulateRewards(chain.Config(), statedb, header, uncles)
		header.Root = statedb.IntermediateRoot(chain.Config().IsEIP158(header.Number))

		// Header seems complete, assemble into a block and return
		return types.NewBlock(header, txs, uncles, receipts), nil
	}
}

func (e *NoRewardEngine) Seal(chain consensus.ChainReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	return e.inner.Seal(chain, block, results, stop)
}

func (e *NoRewardEngine) SealHash(header *types.Header) common.Hash {
	return e.inner.SealHash(header)
}

func (e *NoRewardEngine) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	return e.inner.CalcDifficulty(chain, time, parent)
}

func (e *NoRewardEngine) APIs(chain consensus.ChainReader) []rpc.API {
	return e.inner.APIs(chain)
}

func (e *NoRewardEngine) Close() error {
	return e.inner.Close()
}

func (api *RetestethAPI) SetChainParams(ctx context.Context, chainParams ChainParams) (bool, error) {

	if api.testclient != nil {
		api.testclient.Close()
	}

	api.testclient = newTester(&chainParams.Genesis)
	api.testclient.ethereum.SetEtherbase(chainParams.Genesis.Coinbase)
	api.testclient.ethereum.Miner().SetExtra(chainParams.Genesis.ExtraData)

	nonceLock := new(ethapi.AddrLocker) // !!!new
	api.rpchandler.RegisterName("eth", ethapi.NewPublicBlockChainAPI(api.testclient.ethereum.APIBackend))
	api.rpchandler.RegisterName("eth", ethapi.NewPublicTransactionPoolAPI(api.testclient.ethereum.APIBackend, nonceLock))
	api.rpchandler.RegisterName("debug", eth.NewPrivateDebugAPI(api.testclient.ethereum))
	api.rpchandler.RegisterName("web3", node.NewPublicWeb3API(api.testclient.stack))

	return true, nil
}

func (api *RetestethAPI) MineBlocks(ctx context.Context, number uint64) (bool, error) {
	for i := 0; i < int(number); i++ {
		if err := api.mineBlock(); err != nil {
			return false, err
		}
	}
	fmt.Printf("Mined %d blocks\n", number)
	return true, nil
}

func (api *RetestethAPI) mineBlock() error {
	//api.testclient.ethereum.Etherbase()
	//api.testclient.ethereum.Miner().Start()
	return nil
	//return api.importBlock(block)
}

func (api *RetestethAPI) currentNumber() uint64 {
	if current := api.testclient.ethereum.BlockChain().CurrentBlock(); current != nil {
		return current.NumberU64()
	}
	return 0
}


func (api *RetestethAPI) importBlock(block *types.Block) error {
	if _, err := api.testclient.ethereum.BlockChain().InsertChain([]*types.Block{block}); err != nil {
		return err
	}
	fmt.Printf("Imported block %d,  head is %d\n", block.NumberU64(), api.testclient.ethereum.BlockChain().CurrentBlock().Number())
	return nil
}

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

func (api *RetestethAPI) RewindToBlock(ctx context.Context, newHead uint64) (bool, error) {
	if err := api.testclient.ethereum.BlockChain().SetHead(newHead); err != nil {
		return false, err
	}
	// When we rewind, the transaction pool should be cleaned out.
	api.txMap = make(map[common.Address]map[uint64]*types.Transaction)
	api.txSenders = make(map[common.Address]struct{})
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
		//Miner: miner.Config{
		//	Etherbase: common.HexToAddress(testAddress),
		//},
		Ethash: ethash.Config{
			PowMode: ethash.ModeFake,
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
	listener, rpcHandler, err := rpc.StartHTTPEndpoint(httpEndpoint, rpcAPI, []string{"test"}, cors, vhosts, RetestethHTTPTimeouts)
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

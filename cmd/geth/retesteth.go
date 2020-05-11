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
	"math/big"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"

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

type RetestethEthAPI interface {
	SendRawTransaction(ctx context.Context, rawTx hexutil.Bytes) (common.Hash, error)
	BlockNumber(ctx context.Context) (uint64, error)
	GetBlockByNumber(ctx context.Context, blockNr math.HexOrDecimal64, fullTx bool) (map[string]interface{}, error)
	GetBlockByHash(ctx context.Context, blockHash common.Hash, fullTx bool) (map[string]interface{}, error)
	GetBalance(ctx context.Context, address common.Address, blockNr math.HexOrDecimal64) (*math.HexOrDecimal256, error)
	GetCode(ctx context.Context, address common.Address, blockNr math.HexOrDecimal64) (hexutil.Bytes, error)
	GetTransactionCount(ctx context.Context, address common.Address, blockNr math.HexOrDecimal64) (uint64, error)
}

type RetestethDebugAPI interface {
	AccountRange(ctx context.Context,
		blockHashOrNumber *math.HexOrDecimal256, txIndex uint64,
		addressHash *math.HexOrDecimal256, maxResults uint64,
	) (AccountRangeResult, error)
	StorageRangeAt(ctx context.Context,
		blockHashOrNumber *math.HexOrDecimal256, txIndex uint64,
		address common.Address,
		begin *math.HexOrDecimal256, maxResults uint64,
	) (StorageRangeResult, error)
}

type RetestWeb3API interface {
	ClientVersion(ctx context.Context) (string, error)
}

type RetestethAPI struct {
	ethDb         ethdb.Database
	db            state.Database
	chainConfig   *params.ChainConfig
	author        common.Address
	extraData     []byte
	genesisHash   common.Hash
	engine        *NoRewardEngine
	blockchain    *core.BlockChain
	txMap         map[common.Address]map[uint64]*types.Transaction // Sender -> Nonce -> Transaction
	txSenders     map[common.Address]struct{}                      // Set of transaction senders
	blockInterval uint64
}

type ChainParams struct {
	SealEngine string                            `json:"sealEngine"`
	Params     CParamsParams                     `json:"params"`
	Genesis    CParamsGenesis                    `json:"genesis"`
	Accounts   map[common.Address]CParamsAccount `json:"accounts"`
}

type CParamsParams struct {
	AccountStartNonce          math.HexOrDecimal64   `json:"accountStartNonce"`
	HomesteadForkBlock         *math.HexOrDecimal64  `json:"homesteadForkBlock"`
	EIP150ForkBlock            *math.HexOrDecimal64  `json:"EIP150ForkBlock"`
	EIP158ForkBlock            *math.HexOrDecimal64  `json:"EIP158ForkBlock"`
	DaoHardforkBlock           *math.HexOrDecimal64  `json:"daoHardforkBlock"`
	ByzantiumForkBlock         *math.HexOrDecimal64  `json:"byzantiumForkBlock"`
	ConstantinopleForkBlock    *math.HexOrDecimal64  `json:"constantinopleForkBlock"`
	ConstantinopleFixForkBlock *math.HexOrDecimal64  `json:"constantinopleFixForkBlock"`
	IstanbulBlock              *math.HexOrDecimal64  `json:"istanbulForkBlock"`
	ChainID                    *math.HexOrDecimal256 `json:"chainID"`
	MaximumExtraDataSize       math.HexOrDecimal64   `json:"maximumExtraDataSize"`
	TieBreakingGas             bool                  `json:"tieBreakingGas"`
	MinGasLimit                math.HexOrDecimal64   `json:"minGasLimit"`
	MaxGasLimit                math.HexOrDecimal64   `json:"maxGasLimit"`
	GasLimitBoundDivisor       math.HexOrDecimal64   `json:"gasLimitBoundDivisor"`
	MinimumDifficulty          math.HexOrDecimal256  `json:"minimumDifficulty"`
	DifficultyBoundDivisor     math.HexOrDecimal256  `json:"difficultyBoundDivisor"`
	DurationLimit              math.HexOrDecimal256  `json:"durationLimit"`
	BlockReward                math.HexOrDecimal256  `json:"blockReward"`
	NetworkID                  math.HexOrDecimal256  `json:"networkID"`
}

type CParamsGenesis struct {
	Nonce      math.HexOrDecimal64   `json:"nonce"`
	Difficulty *math.HexOrDecimal256 `json:"difficulty"`
	MixHash    *math.HexOrDecimal256 `json:"mixHash"`
	Author     common.Address        `json:"author"`
	Timestamp  math.HexOrDecimal64   `json:"timestamp"`
	ParentHash common.Hash           `json:"parentHash"`
	ExtraData  hexutil.Bytes         `json:"extraData"`
	GasLimit   math.HexOrDecimal64   `json:"gasLimit"`
}

type CParamsAccount struct {
	Balance     *math.HexOrDecimal256 `json:"balance"`
	Precompiled *CPAccountPrecompiled `json:"precompiled"`
	Code        hexutil.Bytes         `json:"code"`
	Storage     map[string]string     `json:"storage"`
	Nonce       *math.HexOrDecimal64  `json:"nonce"`
}

type CPAccountPrecompiled struct {
	Name          string                `json:"name"`
	StartingBlock math.HexOrDecimal64   `json:"startingBlock"`
	Linear        *CPAPrecompiledLinear `json:"linear"`
}

type CPAPrecompiledLinear struct {
	Base uint64 `json:"base"`
	Word uint64 `json:"word"`
}

type AccountRangeResult struct {
	AddressMap map[common.Hash]common.Address `json:"addressMap"`
	NextKey    common.Hash                    `json:"nextKey"`
}

type StorageRangeResult struct {
	Complete bool                   `json:"complete"`
	Storage  map[common.Hash]SRItem `json:"storage"`
}

type SRItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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
	// Clean up
	if api.blockchain != nil {
		api.blockchain.Stop()
	}
	if api.engine != nil {
		api.engine.Close()
	}
	if api.ethDb != nil {
		api.ethDb.Close()
	}
	ethDb := rawdb.NewMemoryDatabase()
	accounts := make(core.GenesisAlloc)
	for address, account := range chainParams.Accounts {
		balance := big.NewInt(0)
		if account.Balance != nil {
			balance.Set((*big.Int)(account.Balance))
		}
		var nonce uint64
		if account.Nonce != nil {
			nonce = uint64(*account.Nonce)
		}
		if account.Precompiled == nil || account.Balance != nil {
			storage := make(map[common.Hash]common.Hash)
			for k, v := range account.Storage {
				storage[common.HexToHash(k)] = common.HexToHash(v)
			}
			accounts[address] = core.GenesisAccount{
				Balance: balance,
				Code:    account.Code,
				Nonce:   nonce,
				Storage: storage,
			}
		}
	}
	chainId := big.NewInt(1)
	if chainParams.Params.ChainID != nil {
		chainId.Set((*big.Int)(chainParams.Params.ChainID))
	}
	var (
		homesteadBlock      *big.Int
		daoForkBlock        *big.Int
		eip150Block         *big.Int
		eip155Block         *big.Int
		eip158Block         *big.Int
		byzantiumBlock      *big.Int
		constantinopleBlock *big.Int
		petersburgBlock     *big.Int
		istanbulBlock       *big.Int
	)
	if chainParams.Params.HomesteadForkBlock != nil {
		homesteadBlock = big.NewInt(int64(*chainParams.Params.HomesteadForkBlock))
	}
	if chainParams.Params.DaoHardforkBlock != nil {
		daoForkBlock = big.NewInt(int64(*chainParams.Params.DaoHardforkBlock))
	}
	if chainParams.Params.EIP150ForkBlock != nil {
		eip150Block = big.NewInt(int64(*chainParams.Params.EIP150ForkBlock))
	}
	if chainParams.Params.EIP158ForkBlock != nil {
		eip158Block = big.NewInt(int64(*chainParams.Params.EIP158ForkBlock))
		eip155Block = eip158Block
	}
	if chainParams.Params.ByzantiumForkBlock != nil {
		byzantiumBlock = big.NewInt(int64(*chainParams.Params.ByzantiumForkBlock))
	}
	if chainParams.Params.ConstantinopleForkBlock != nil {
		constantinopleBlock = big.NewInt(int64(*chainParams.Params.ConstantinopleForkBlock))
	}
	if chainParams.Params.ConstantinopleFixForkBlock != nil {
		petersburgBlock = big.NewInt(int64(*chainParams.Params.ConstantinopleFixForkBlock))
	}
	if constantinopleBlock != nil && petersburgBlock == nil {
		petersburgBlock = big.NewInt(100000000000)
	}
	if chainParams.Params.IstanbulBlock != nil {
		istanbulBlock = big.NewInt(int64(*chainParams.Params.IstanbulBlock))
	}

	genesis := &core.Genesis{
		Config: &params.ChainConfig{
			ChainID:             chainId,
			HomesteadBlock:      homesteadBlock,
			DAOForkBlock:        daoForkBlock,
			DAOForkSupport:      true,
			EIP150Block:         eip150Block,
			EIP155Block:         eip155Block,
			EIP158Block:         eip158Block,
			ByzantiumBlock:      byzantiumBlock,
			ConstantinopleBlock: constantinopleBlock,
			PetersburgBlock:     petersburgBlock,
			IstanbulBlock:       istanbulBlock,
		},
		Nonce:      uint64(chainParams.Genesis.Nonce),
		Timestamp:  uint64(chainParams.Genesis.Timestamp),
		ExtraData:  chainParams.Genesis.ExtraData,
		GasLimit:   uint64(chainParams.Genesis.GasLimit),
		Difficulty: big.NewInt(0).Set((*big.Int)(chainParams.Genesis.Difficulty)),
		Mixhash:    common.BigToHash((*big.Int)(chainParams.Genesis.MixHash)),
		Coinbase:   chainParams.Genesis.Author,
		ParentHash: chainParams.Genesis.ParentHash,
		Alloc:      accounts,
	}
	chainConfig, genesisHash, err := core.SetupGenesisBlock(ethDb, genesis)
	if err != nil {
		return false, err
	}
	fmt.Printf("Chain config: %v\n", chainConfig)

	var inner consensus.Engine
	switch chainParams.SealEngine {
	case "NoProof", "NoReward":
		inner = ethash.NewFaker()
	case "Ethash":
		inner = ethash.New(ethash.Config{
			CacheDir:         "ethash",
			CachesInMem:      2,
			CachesOnDisk:     3,
			CachesLockMmap:   false,
			DatasetsInMem:    1,
			DatasetsOnDisk:   2,
			DatasetsLockMmap: false,
		}, nil, false)
	default:
		return false, fmt.Errorf("unrecognised seal engine: %s", chainParams.SealEngine)
	}
	engine := &NoRewardEngine{inner: inner, rewardsOn: chainParams.SealEngine != "NoReward"}

	blockchain, err := core.NewBlockChain(ethDb, nil, chainConfig, engine, vm.Config{}, nil, nil)
	if err != nil {
		return false, err
	}

	api.chainConfig = chainConfig
	api.genesisHash = genesisHash
	api.author = chainParams.Genesis.Author
	api.extraData = chainParams.Genesis.ExtraData
	api.ethDb = ethDb
	api.engine = engine
	api.blockchain = blockchain
	api.db = state.NewDatabase(api.ethDb)
	api.txMap = make(map[common.Address]map[uint64]*types.Transaction)
	api.txSenders = make(map[common.Address]struct{})
	api.blockInterval = 0
	return true, nil
}

func (api *RetestethAPI) SendRawTransaction(ctx context.Context, rawTx hexutil.Bytes) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(rawTx, tx); err != nil {
		// Return nil is not by mistake - some tests include sending transaction where gasLimit overflows uint64
		return common.Hash{}, nil
	}
	signer := types.MakeSigner(api.chainConfig, big.NewInt(int64(api.currentNumber())))
	sender, err := types.Sender(signer, tx)
	if err != nil {
		return common.Hash{}, err
	}
	if nonceMap, ok := api.txMap[sender]; ok {
		nonceMap[tx.Nonce()] = tx
	} else {
		nonceMap = make(map[uint64]*types.Transaction)
		nonceMap[tx.Nonce()] = tx
		api.txMap[sender] = nonceMap
	}
	api.txSenders[sender] = struct{}{}
	return tx.Hash(), nil
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

func (api *RetestethAPI) currentNumber() uint64 {
	if current := api.blockchain.CurrentBlock(); current != nil {
		return current.NumberU64()
	}
	return 0
}

func (api *RetestethAPI) mineBlock() error {
	number := api.currentNumber()
	parentHash := rawdb.ReadCanonicalHash(api.ethDb, number)
	parent := rawdb.ReadBlock(api.ethDb, parentHash, number)
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
	if api.engine != nil {
		api.engine.Prepare(api.blockchain, header)
	}
	// If we are care about TheDAO hard-fork check whether to override the extra-data or not
	if daoBlock := api.chainConfig.DAOForkBlock; daoBlock != nil {
		// Check whether the block is among the fork extra-override range
		limit := new(big.Int).Add(daoBlock, params.DAOForkExtraRange)
		if header.Number.Cmp(daoBlock) >= 0 && header.Number.Cmp(limit) < 0 {
			// Depending whether we support or oppose the fork, override differently
			if api.chainConfig.DAOForkSupport {
				header.Extra = common.CopyBytes(params.DAOForkBlockExtra)
			} else if bytes.Equal(header.Extra, params.DAOForkBlockExtra) {
				header.Extra = []byte{} // If miner opposes, don't let it use the reserved extra-data
			}
		}
	}
	statedb, err := api.blockchain.StateAt(parent.Root())
	if err != nil {
		return err
	}
	if api.chainConfig.DAOForkSupport && api.chainConfig.DAOForkBlock != nil && api.chainConfig.DAOForkBlock.Cmp(header.Number) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	gasPool := new(core.GasPool).AddGas(header.GasLimit)
	txCount := 0
	var txs []*types.Transaction
	var receipts []*types.Receipt
	var blockFull = gasPool.Gas() < params.TxGas
	for address := range api.txSenders {
		if blockFull {
			break
		}
		m := api.txMap[address]
		for nonce := statedb.GetNonce(address); ; nonce++ {
			if tx, ok := m[nonce]; ok {
				// Try to apply transactions to the state
				statedb.Prepare(tx.Hash(), common.Hash{}, txCount)
				snap := statedb.Snapshot()

				receipt, err := core.ApplyTransaction(
					api.chainConfig,
					api.blockchain,
					&api.author,
					gasPool,
					statedb,
					header, tx, &header.GasUsed, *api.blockchain.GetVMConfig(),
				)
				if err != nil {
					statedb.RevertToSnapshot(snap)
					break
				}
				txs = append(txs, tx)
				receipts = append(receipts, receipt)
				delete(m, nonce)
				if len(m) == 0 {
					// Last tx for the sender
					delete(api.txMap, address)
					delete(api.txSenders, address)
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
	block, err := api.engine.FinalizeAndAssemble(api.blockchain, header, statedb, txs, []*types.Header{}, receipts)
	if err != nil {
		return err
	}
	return api.importBlock(block)
}

func (api *RetestethAPI) importBlock(block *types.Block) error {
	if _, err := api.blockchain.InsertChain([]*types.Block{block}); err != nil {
		return err
	}
	fmt.Printf("Imported block %d,  head is %d\n", block.NumberU64(), api.currentNumber())
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
	fmt.Printf("Importing block %d with parent hash: %x, genesisHash: %x\n", block.NumberU64(), block.ParentHash(), api.genesisHash)
	if err := api.importBlock(block); err != nil {
		return common.Hash{}, err
	}
	return block.Hash(), nil
}

func (api *RetestethAPI) RewindToBlock(ctx context.Context, newHead uint64) (bool, error) {
	if err := api.blockchain.SetHead(newHead); err != nil {
		return false, err
	}
	// When we rewind, the transaction pool should be cleaned out.
	api.txMap = make(map[common.Address]map[uint64]*types.Transaction)
	api.txSenders = make(map[common.Address]struct{})
	return true, nil
}

var emptyListHash common.Hash = common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347")

func (api *RetestethAPI) GetLogHash(ctx context.Context, txHash common.Hash) (common.Hash, error) {
	receipt, _, _, _ := rawdb.ReadReceipt(api.ethDb, txHash, api.chainConfig)
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

func (api *RetestethAPI) BlockNumber(ctx context.Context) (uint64, error) {
	return api.currentNumber(), nil
}

func (api *RetestethAPI) GetBlockByNumber(ctx context.Context, blockNr math.HexOrDecimal64, fullTx bool) (map[string]interface{}, error) {
	block := api.blockchain.GetBlockByNumber(uint64(blockNr))
	if block != nil {
		response, err := RPCMarshalBlock(block, true, fullTx)
		if err != nil {
			return nil, err
		}
		response["author"] = response["miner"]
		response["totalDifficulty"] = (*hexutil.Big)(api.blockchain.GetTd(block.Hash(), uint64(blockNr)))
		return response, err
	}
	return nil, fmt.Errorf("block %d not found", blockNr)
}

func (api *RetestethAPI) GetBlockByHash(ctx context.Context, blockHash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block := api.blockchain.GetBlockByHash(blockHash)
	if block != nil {
		response, err := RPCMarshalBlock(block, true, fullTx)
		if err != nil {
			return nil, err
		}
		response["author"] = response["miner"]
		response["totalDifficulty"] = (*hexutil.Big)(api.blockchain.GetTd(block.Hash(), block.Number().Uint64()))
		return response, err
	}
	return nil, fmt.Errorf("block 0x%x not found", blockHash)
}

func (api *RetestethAPI) AccountRange(ctx context.Context,
	blockHashOrNumber *math.HexOrDecimal256, txIndex uint64,
	addressHash *math.HexOrDecimal256, maxResults uint64,
) (AccountRangeResult, error) {
	var (
		header *types.Header
		block  *types.Block
	)
	if (*big.Int)(blockHashOrNumber).Cmp(big.NewInt(math.MaxInt64)) > 0 {
		blockHash := common.BigToHash((*big.Int)(blockHashOrNumber))
		header = api.blockchain.GetHeaderByHash(blockHash)
		block = api.blockchain.GetBlockByHash(blockHash)
		//fmt.Printf("Account range: %x, txIndex %d, start: %x, maxResults: %d\n", blockHash, txIndex, common.BigToHash((*big.Int)(addressHash)), maxResults)
	} else {
		blockNumber := (*big.Int)(blockHashOrNumber).Uint64()
		header = api.blockchain.GetHeaderByNumber(blockNumber)
		block = api.blockchain.GetBlockByNumber(blockNumber)
		//fmt.Printf("Account range: %d, txIndex %d, start: %x, maxResults: %d\n", blockNumber, txIndex, common.BigToHash((*big.Int)(addressHash)), maxResults)
	}
	parentHeader := api.blockchain.GetHeaderByHash(header.ParentHash)
	var root common.Hash
	var statedb *state.StateDB
	var err error
	if parentHeader == nil || int(txIndex) >= len(block.Transactions()) {
		root = header.Root
		statedb, err = api.blockchain.StateAt(root)
		if err != nil {
			return AccountRangeResult{}, err
		}
	} else {
		root = parentHeader.Root
		statedb, err = api.blockchain.StateAt(root)
		if err != nil {
			return AccountRangeResult{}, err
		}
		// Recompute transactions up to the target index.
		signer := types.MakeSigner(api.blockchain.Config(), block.Number())
		for idx, tx := range block.Transactions() {
			// Assemble the transaction call message and return if the requested offset
			msg, _ := tx.AsMessage(signer)
			context := core.NewEVMContext(msg, block.Header(), api.blockchain, nil)
			// Not yet the searched for transaction, execute on top of the current state
			vmenv := vm.NewEVM(context, statedb, api.blockchain.Config(), vm.Config{})
			if _, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(tx.Gas())); err != nil {
				return AccountRangeResult{}, fmt.Errorf("transaction %#x failed: %v", tx.Hash(), err)
			}
			// Ensure any modifications are committed to the state
			// Only delete empty objects if EIP158/161 (a.k.a Spurious Dragon) is in effect
			root = statedb.IntermediateRoot(vmenv.ChainConfig().IsEIP158(block.Number()))
			if idx == int(txIndex) {
				// This is to make sure root can be opened by OpenTrie
				root, err = statedb.Commit(api.chainConfig.IsEIP158(block.Number()))
				if err != nil {
					return AccountRangeResult{}, err
				}
				break
			}
		}
	}
	accountTrie, err := statedb.Database().OpenTrie(root)
	if err != nil {
		return AccountRangeResult{}, err
	}
	it := trie.NewIterator(accountTrie.NodeIterator(common.BigToHash((*big.Int)(addressHash)).Bytes()))
	result := AccountRangeResult{AddressMap: make(map[common.Hash]common.Address)}
	for i := 0; i < int(maxResults) && it.Next(); i++ {
		if preimage := accountTrie.GetKey(it.Key); preimage != nil {
			result.AddressMap[common.BytesToHash(it.Key)] = common.BytesToAddress(preimage)
		}
	}
	//fmt.Printf("Number of entries returned: %d\n", len(result.AddressMap))
	// Add the 'next key' so clients can continue downloading.
	if it.Next() {
		next := common.BytesToHash(it.Key)
		result.NextKey = next
	}
	return result, nil
}

func (api *RetestethAPI) GetBalance(ctx context.Context, address common.Address, blockNr math.HexOrDecimal64) (*math.HexOrDecimal256, error) {
	//fmt.Printf("GetBalance %x, block %d\n", address, blockNr)
	header := api.blockchain.GetHeaderByNumber(uint64(blockNr))
	statedb, err := api.blockchain.StateAt(header.Root)
	if err != nil {
		return nil, err
	}
	return (*math.HexOrDecimal256)(statedb.GetBalance(address)), nil
}

func (api *RetestethAPI) GetCode(ctx context.Context, address common.Address, blockNr math.HexOrDecimal64) (hexutil.Bytes, error) {
	header := api.blockchain.GetHeaderByNumber(uint64(blockNr))
	statedb, err := api.blockchain.StateAt(header.Root)
	if err != nil {
		return nil, err
	}
	return statedb.GetCode(address), nil
}

func (api *RetestethAPI) GetTransactionCount(ctx context.Context, address common.Address, blockNr math.HexOrDecimal64) (uint64, error) {
	header := api.blockchain.GetHeaderByNumber(uint64(blockNr))
	statedb, err := api.blockchain.StateAt(header.Root)
	if err != nil {
		return 0, err
	}
	return statedb.GetNonce(address), nil
}

func (api *RetestethAPI) StorageRangeAt(ctx context.Context,
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
		header = api.blockchain.GetHeaderByHash(blockHash)
		block = api.blockchain.GetBlockByHash(blockHash)
		//fmt.Printf("Storage range: %x, txIndex %d, addr: %x, start: %x, maxResults: %d\n",
		//	blockHash, txIndex, address, common.BigToHash((*big.Int)(begin)), maxResults)
	} else {
		blockNumber := (*big.Int)(blockHashOrNumber).Uint64()
		header = api.blockchain.GetHeaderByNumber(blockNumber)
		block = api.blockchain.GetBlockByNumber(blockNumber)
		//fmt.Printf("Storage range: %d, txIndex %d, addr: %x, start: %x, maxResults: %d\n",
		//	blockNumber, txIndex, address, common.BigToHash((*big.Int)(begin)), maxResults)
	}
	parentHeader := api.blockchain.GetHeaderByHash(header.ParentHash)
	var root common.Hash
	var statedb *state.StateDB
	var err error
	if parentHeader == nil || int(txIndex) >= len(block.Transactions()) {
		root = header.Root
		statedb, err = api.blockchain.StateAt(root)
		if err != nil {
			return StorageRangeResult{}, err
		}
	} else {
		root = parentHeader.Root
		statedb, err = api.blockchain.StateAt(root)
		if err != nil {
			return StorageRangeResult{}, err
		}
		// Recompute transactions up to the target index.
		signer := types.MakeSigner(api.blockchain.Config(), block.Number())
		for idx, tx := range block.Transactions() {
			// Assemble the transaction call message and return if the requested offset
			msg, _ := tx.AsMessage(signer)
			context := core.NewEVMContext(msg, block.Header(), api.blockchain, nil)
			// Not yet the searched for transaction, execute on top of the current state
			vmenv := vm.NewEVM(context, statedb, api.blockchain.Config(), vm.Config{})
			if _, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(tx.Gas())); err != nil {
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

func (api *RetestethAPI) ClientVersion(ctx context.Context) (string, error) {
	return "Geth-" + params.VersionWithCommit(gitCommit, gitDate), nil
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

func retesteth(ctx *cli.Context) error {
	log.Info("Welcome to retesteth!")
	// register signer API with server
	var (
		extapiURL string
	)
	apiImpl := &RetestethAPI{}
	var testApi RetestethTestAPI = apiImpl
	var ethApi RetestethEthAPI = apiImpl
	var debugApi RetestethDebugAPI = apiImpl
	var web3Api RetestWeb3API = apiImpl
	rpcAPI := []rpc.API{
		{
			Namespace: "test",
			Public:    true,
			Service:   testApi,
			Version:   "1.0",
		},
		{
			Namespace: "eth",
			Public:    true,
			Service:   ethApi,
			Version:   "1.0",
		},
		{
			Namespace: "debug",
			Public:    true,
			Service:   debugApi,
			Version:   "1.0",
		},
		{
			Namespace: "web3",
			Public:    true,
			Service:   web3Api,
			Version:   "1.0",
		},
	}
	vhosts := splitAndTrim(ctx.GlobalString(utils.HTTPVirtualHostsFlag.Name))
	cors := splitAndTrim(ctx.GlobalString(utils.HTTPCORSDomainFlag.Name))

	// register apis and create handler stack
	srv := rpc.NewServer()
	err := node.RegisterApisFromWhitelist(rpcAPI, []string{"test", "eth", "debug", "web3"}, srv, false)
	if err != nil {
		utils.Fatalf("Could not register RPC apis: %w", err)
	}
	handler := node.NewHTTPHandlerStack(srv, cors, vhosts)

	// start http server
	var RetestethHTTPTimeouts = rpc.HTTPTimeouts{
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	httpEndpoint := fmt.Sprintf("%s:%d", ctx.GlobalString(utils.HTTPListenAddrFlag.Name), ctx.Int(rpcPortFlag.Name))
	httpServer, _, err := node.StartHTTPEndpoint(httpEndpoint, RetestethHTTPTimeouts, handler)
	if err != nil {
		utils.Fatalf("Could not start RPC api: %v", err)
	}
	extapiURL = fmt.Sprintf("http://%s", httpEndpoint)
	log.Info("HTTP endpoint opened", "url", extapiURL)

	defer func() {
		// Don't bother imposing a timeout here.
		httpServer.Shutdown(context.Background())
		log.Info("HTTP endpoint closed", "url", httpEndpoint)
	}()

	abortChan := make(chan os.Signal, 11)
	signal.Notify(abortChan, os.Interrupt)

	sig := <-abortChan
	log.Info("Exiting...", "signal", sig)
	return nil
}

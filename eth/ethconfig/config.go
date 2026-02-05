// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package ethconfig contains the configuration of the ETH and LES protocols.
package ethconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/history"
	"github.com/ethereum/go-ethereum/core/txpool/blobpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/eth/gasprice"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// FullNodeGPO contains default gasprice oracle settings for full node.
var FullNodeGPO = gasprice.Config{
	Blocks:           20,
	Percentile:       60,
	MaxHeaderHistory: 1024,
	MaxBlockHistory:  1024,
	MaxPrice:         gasprice.DefaultMaxPrice,
	IgnorePrice:      gasprice.DefaultIgnorePrice,
}

// Defaults contains default settings for use on the Ethereum main net.
var Defaults = Config{
	HistoryMode:             history.KeepAll,
	SyncMode:                SnapSync,
	NetworkId:               0, // enable auto configuration of networkID == chainID
	TxLookupLimit:           2350000,
	TransactionHistory:      2350000,
	LogHistory:              2350000,
	StateHistory:            pathdb.Defaults.StateHistory,
	TrienodeHistory:         pathdb.Defaults.TrienodeHistory,
	NodeFullValueCheckpoint: pathdb.Defaults.FullValueCheckpoint,
	DatabaseCache:           512,
	TrieCleanCache:          154,
	TrieDirtyCache:          256,
	TrieTimeout:             60 * time.Minute,
	SnapshotCache:           102,
	FilterLogCacheSize:      32,
	LogQueryLimit:           1000,
	Miner:                   miner.DefaultConfig,
	TxPool:                  legacypool.DefaultConfig,
	BlobPool:                blobpool.DefaultConfig,
	RPCGasCap:               50000000,
	RPCEVMTimeout:           5 * time.Second,
	GPO:                     FullNodeGPO,
	RPCTxFeeCap:             1, // 1 ether
	TxSyncDefaultTimeout:    20 * time.Second,
	TxSyncMaxTimeout:        1 * time.Minute,
	SlowBlockThreshold:      time.Second * 2,
	RangeLimit:              0,
	PartialState:            DefaultPartialStateConfig(),
}

//go:generate go run github.com/fjl/gencodec -type Config -formats toml -out gen_config.go

// Config contains configuration options for ETH and LES protocols.
type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the Ethereum main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Network ID separates blockchains on the peer-to-peer networking level. When left
	// zero, the chain ID is used as network ID.
	NetworkId uint64
	SyncMode  SyncMode

	// HistoryMode configures chain history retention.
	HistoryMode history.HistoryMode

	// This can be set to list of enrtree:// URLs which will be queried for
	// nodes to connect to.
	EthDiscoveryURLs  []string
	SnapDiscoveryURLs []string

	// State options.
	NoPruning  bool // Whether to disable pruning and flush everything to disk
	NoPrefetch bool // Whether to disable prefetching and only load state on demand

	// Deprecated: use 'TransactionHistory' instead.
	TxLookupLimit uint64 `toml:",omitempty"` // The maximum number of blocks from head whose tx indices are reserved.

	TransactionHistory   uint64 `toml:",omitempty"` // The maximum number of blocks from head whose tx indices are reserved.
	LogHistory           uint64 `toml:",omitempty"` // The maximum number of blocks from head where a log search index is maintained.
	LogNoHistory         bool   `toml:",omitempty"` // No log search index is maintained.
	LogExportCheckpoints string // export log index checkpoints to file
	StateHistory         uint64 `toml:",omitempty"` // The maximum number of blocks from head whose state histories are reserved.
	TrienodeHistory      int64  `toml:",omitempty"` // Number of blocks from the chain head for which trienode histories are retained

	// The frequency of full-value encoding. For example, a value of 16 means
	// that, on average, for a given trie node across its 16 consecutive historical
	// versions, only one version is stored in full format, while the others
	// are stored in diff mode for storage compression.
	NodeFullValueCheckpoint uint32 `toml:",omitempty"`

	// State scheme represents the scheme used to store ethereum states and trie
	// nodes on top. It can be 'hash', 'path', or none which means use the scheme
	// consistent with persistent state.
	StateScheme string `toml:",omitempty"`

	// RequiredBlocks is a set of block number -> hash mappings which must be in the
	// canonical chain of all remote peers. Setting the option makes geth verify the
	// presence of these blocks for every new peer connection.
	RequiredBlocks map[uint64]common.Hash `toml:"-"`

	// SlowBlockThreshold is the block execution speed threshold (Mgas/s)
	// below which detailed statistics are logged.
	SlowBlockThreshold time.Duration `toml:",omitempty"`

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int
	DatabaseFreezer    string
	DatabaseEra        string

	TrieCleanCache int
	TrieDirtyCache int
	TrieTimeout    time.Duration
	SnapshotCache  int
	Preimages      bool

	// This is the number of blocks for which logs will be cached in the filter system.
	FilterLogCacheSize int

	// This is the maximum number of addresses or topics allowed in filter criteria
	// for eth_getLogs.
	LogQueryLimit int

	// Mining options
	Miner miner.Config

	// Transaction pool options
	TxPool   legacypool.Config
	BlobPool blobpool.Config

	// Gas Price Oracle options
	GPO gasprice.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Enables collection of witness trie access statistics
	EnableWitnessStats bool

	// Generate execution witnesses and self-check against them (testing purpose)
	StatelessSelfValidation bool

	// Enables tracking of state size
	EnableStateSizeTracking bool

	// Enables VM tracing
	VMTrace           string
	VMTraceJsonConfig string

	// RPCGasCap is the global gas cap for eth-call variants.
	RPCGasCap uint64

	// RPCEVMTimeout is the global timeout for eth-call.
	RPCEVMTimeout time.Duration

	// RPCTxFeeCap is the global transaction fee (price * gas limit) cap for
	// send-transaction variants. The unit is ether.
	RPCTxFeeCap float64

	// OverrideOsaka (TODO: remove after the fork)
	OverrideOsaka *uint64 `toml:",omitempty"`

	// OverrideBPO1 (TODO: remove after the fork)
	OverrideBPO1 *uint64 `toml:",omitempty"`

	// OverrideBPO2 (TODO: remove after the fork)
	OverrideBPO2 *uint64 `toml:",omitempty"`

	// OverrideVerkle (TODO: remove after the fork)
	OverrideVerkle *uint64 `toml:",omitempty"`

	// EIP-7966: eth_sendRawTransactionSync timeouts
	TxSyncDefaultTimeout time.Duration `toml:",omitempty"`
	TxSyncMaxTimeout     time.Duration `toml:",omitempty"`

	// RangeLimit restricts the maximum range (end - start) for range queries.
	RangeLimit uint64 `toml:",omitempty"`

	BALExecutionMode int

	// PartialState configures partial statefulness mode for reduced storage.
	PartialState PartialStateConfig
}

// DefaultChainRetention is the default number of recent blocks for which
// bodies and receipts are retained in partial state mode. Older blocks only
// keep their headers. 1024 blocks (~3.4 hours at 12s/block) is sufficient
// for reorg handling and recent receipt lookups. Configurable via
// --partial-state.chain-retention.
const DefaultChainRetention = 1024

// PartialStateConfig configures partial statefulness mode.
// When enabled, the node stores all accounts but only storage for configured contracts.
// State updates are applied via Block Access Lists (BALs) per EIP-7928.
type PartialStateConfig struct {
	// Enabled activates partial statefulness mode
	Enabled bool

	// Contracts is the list of contracts to track storage for
	Contracts []common.Address

	// ContractsFile is the path to a JSON file containing contract addresses
	ContractsFile string `toml:",omitempty"`

	// BALRetention is the number of blocks to keep BAL history for reorg handling
	BALRetention uint64

	// ChainRetention is the number of recent blocks to retain bodies and
	// receipts for. Older blocks only keep their headers. During sync, bodies
	// and receipts outside this window are never downloaded. After sync, the
	// freezer enforces a rolling window, deleting aged-out data. Set to 0 to
	// keep all chain history.
	ChainRetention uint64
}

// DefaultPartialStateConfig returns the default partial state configuration.
func DefaultPartialStateConfig() PartialStateConfig {
	return PartialStateConfig{
		Enabled:        false,
		Contracts:      nil,
		ContractsFile:  "",
		BALRetention:   256,
		ChainRetention: DefaultChainRetention,
	}
}

// LoadPartialStateContracts loads contract addresses from a JSON file
// and merges them with any directly configured addresses.
func (c *PartialStateConfig) LoadPartialStateContracts() error {
	if c.ContractsFile == "" {
		return nil
	}
	return c.loadContractsFromFile(c.ContractsFile)
}

// loadContractsFromFile reads contract addresses from a JSON file.
// File format:
//
//	{
//	  "version": 1,
//	  "contracts": [
//	    {"address": "0x...", "name": "WETH", "comment": "Wrapped Ether"},
//	    {"address": "0x...", "name": "USDC"}
//	  ]
//	}
func (c *PartialStateConfig) loadContractsFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read contracts file: %w", err)
	}

	var file struct {
		Version   int `json:"version"`
		Contracts []struct {
			Address string `json:"address"`
			Name    string `json:"name,omitempty"`
			Comment string `json:"comment,omitempty"`
		} `json:"contracts"`
	}

	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse contracts file: %w", err)
	}

	// Validate version
	if file.Version != 1 {
		return fmt.Errorf("unsupported contracts file version: %d", file.Version)
	}

	// Merge contracts from file with directly configured ones
	seen := make(map[common.Address]struct{})
	for _, addr := range c.Contracts {
		seen[addr] = struct{}{}
	}

	for _, contract := range file.Contracts {
		addr := common.HexToAddress(contract.Address)
		if addr == (common.Address{}) {
			return fmt.Errorf("invalid contract address in file: %s", contract.Address)
		}
		if _, exists := seen[addr]; !exists {
			c.Contracts = append(c.Contracts, addr)
			seen[addr] = struct{}{}
		}
	}

	return nil
}

// Validate checks the configuration for errors.
func (c *PartialStateConfig) Validate() error {
	if !c.Enabled {
		return nil // Nothing to validate if disabled
	}

	// Load contracts from file if specified
	if err := c.LoadPartialStateContracts(); err != nil {
		return err
	}

	// Validate BAL retention
	if c.BALRetention < 64 {
		return fmt.Errorf("BAL retention must be at least 64 blocks (for BLOCKHASH support), got %d", c.BALRetention)
	}

	return nil
}

// CreateConsensusEngine creates a consensus engine for the given chain config.
// Clique is allowed for now to live standalone, but ethash is forbidden and can
// only exist on already merged networks.
func CreateConsensusEngine(config *params.ChainConfig, db ethdb.Database) (consensus.Engine, error) {
	if config.TerminalTotalDifficulty == nil {
		log.Error("Geth only supports PoS networks. Please transition legacy networks using Geth v1.13.x.")
		return nil, errors.New("'terminalTotalDifficulty' is not set in genesis block")
	}
	// Wrap previously supported consensus engines into their post-merge counterpart
	if config.Clique != nil {
		return beacon.New(clique.New(config.Clique, db)), nil
	}
	return beacon.New(ethash.NewFaker()), nil
}

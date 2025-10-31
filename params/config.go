// Copyright 2016 The go-ethereum Authors
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

package params

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params/forks"
)

// Genesis hashes to enforce below configs on.
var (
	MainnetGenesisHash = common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3")
	HoleskyGenesisHash = common.HexToHash("0xb5f7f912443c940f21fd611f12828d75b534364ed9e95ca4e307729a4661bde4")
	SepoliaGenesisHash = common.HexToHash("0x25a5cc106eea7138acab33231d7160d69cb777ee0c2c553fcddf5138993e6dd9")
	HoodiGenesisHash   = common.HexToHash("0xbbe312868b376a3001692a646dd2d7d1e4406380dfd86b98aa8a34d1557c971b")
)

func newUint64(val uint64) *uint64 { return &val }

var (
	MainnetTerminalTotalDifficulty, _ = new(big.Int).SetString("58_750_000_000_000_000_000_000", 0)

	// MainnetChainConfig is the chain parameters to run a node on the main network.
	MainnetChainConfig = &ChainConfig{
		ChainID:                 big.NewInt(1),
		HomesteadBlock:          big.NewInt(1_150_000),
		DAOForkBlock:            big.NewInt(1_920_000),
		DAOForkSupport:          true,
		EIP150Block:             big.NewInt(2_463_000),
		EIP155Block:             big.NewInt(2_675_000),
		EIP158Block:             big.NewInt(2_675_000),
		ByzantiumBlock:          big.NewInt(4_370_000),
		ConstantinopleBlock:     big.NewInt(7_280_000),
		PetersburgBlock:         big.NewInt(7_280_000),
		IstanbulBlock:           big.NewInt(9_069_000),
		MuirGlacierBlock:        big.NewInt(9_200_000),
		BerlinBlock:             big.NewInt(12_244_000),
		LondonBlock:             big.NewInt(12_965_000),
		ArrowGlacierBlock:       big.NewInt(13_773_000),
		GrayGlacierBlock:        big.NewInt(15_050_000),
		TerminalTotalDifficulty: MainnetTerminalTotalDifficulty, // 58_750_000_000_000_000_000_000
		ShanghaiTime:            newUint64(1681338455),
		CancunTime:              newUint64(1710338135),
		PragueTime:              newUint64(1746612311),
		OsakaTime:               newUint64(1764798551),
		BPO1Time:                newUint64(1765290071),
		BPO2Time:                newUint64(1767747671),
		DepositContractAddress:  common.HexToAddress("0x00000000219ab540356cbb839cbe05303d7705fa"),
		Ethash:                  new(EthashConfig),
		BlobScheduleConfig: &BlobScheduleConfig{
			Cancun: DefaultCancunBlobConfig,
			Prague: DefaultPragueBlobConfig,
			Osaka:  DefaultOsakaBlobConfig,
			BPO1:   DefaultBPO1BlobConfig,
			BPO2:   DefaultBPO2BlobConfig,
		},
	}
	// HoleskyChainConfig contains the chain parameters to run a node on the Holesky test network.
	HoleskyChainConfig = &ChainConfig{
		ChainID:                 big.NewInt(17000),
		HomesteadBlock:          big.NewInt(0),
		DAOForkBlock:            nil,
		DAOForkSupport:          true,
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        nil,
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       nil,
		GrayGlacierBlock:        nil,
		TerminalTotalDifficulty: big.NewInt(0),
		MergeNetsplitBlock:      nil,
		ShanghaiTime:            newUint64(1696000704),
		CancunTime:              newUint64(1707305664),
		PragueTime:              newUint64(1740434112),
		OsakaTime:               newUint64(1759308480),
		BPO1Time:                newUint64(1759800000),
		BPO2Time:                newUint64(1760389824),
		DepositContractAddress:  common.HexToAddress("0x4242424242424242424242424242424242424242"),
		Ethash:                  new(EthashConfig),
		BlobScheduleConfig: &BlobScheduleConfig{
			Cancun: DefaultCancunBlobConfig,
			Prague: DefaultPragueBlobConfig,
			Osaka:  DefaultOsakaBlobConfig,
			BPO1:   DefaultBPO1BlobConfig,
			BPO2:   DefaultBPO2BlobConfig,
		},
	}
	// SepoliaChainConfig contains the chain parameters to run a node on the Sepolia test network.
	SepoliaChainConfig = &ChainConfig{
		ChainID:                 big.NewInt(11155111),
		HomesteadBlock:          big.NewInt(0),
		DAOForkBlock:            nil,
		DAOForkSupport:          true,
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       nil,
		GrayGlacierBlock:        nil,
		TerminalTotalDifficulty: big.NewInt(17_000_000_000_000_000),
		MergeNetsplitBlock:      big.NewInt(1735371),
		ShanghaiTime:            newUint64(1677557088),
		CancunTime:              newUint64(1706655072),
		PragueTime:              newUint64(1741159776),
		OsakaTime:               newUint64(1760427360),
		BPO1Time:                newUint64(1761017184),
		BPO2Time:                newUint64(1761607008),
		DepositContractAddress:  common.HexToAddress("0x7f02c3e3c98b133055b8b348b2ac625669ed295d"),
		Ethash:                  new(EthashConfig),
		BlobScheduleConfig: &BlobScheduleConfig{
			Cancun: DefaultCancunBlobConfig,
			Prague: DefaultPragueBlobConfig,
			Osaka:  DefaultOsakaBlobConfig,
			BPO1:   DefaultBPO1BlobConfig,
			BPO2:   DefaultBPO2BlobConfig,
		},
	}
	// HoodiChainConfig contains the chain parameters to run a node on the Hoodi test network.
	HoodiChainConfig = &ChainConfig{
		ChainID:                 big.NewInt(560048),
		HomesteadBlock:          big.NewInt(0),
		DAOForkBlock:            nil,
		DAOForkSupport:          true,
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       nil,
		GrayGlacierBlock:        nil,
		TerminalTotalDifficulty: big.NewInt(0),
		MergeNetsplitBlock:      big.NewInt(0),
		ShanghaiTime:            newUint64(0),
		CancunTime:              newUint64(0),
		PragueTime:              newUint64(1742999832),
		OsakaTime:               newUint64(1761677592),
		BPO1Time:                newUint64(1762365720),
		BPO2Time:                newUint64(1762955544),
		DepositContractAddress:  common.HexToAddress("0x00000000219ab540356cBB839Cbe05303d7705Fa"),
		Ethash:                  new(EthashConfig),
		BlobScheduleConfig: &BlobScheduleConfig{
			Cancun: DefaultCancunBlobConfig,
			Prague: DefaultPragueBlobConfig,
			Osaka:  DefaultOsakaBlobConfig,
			BPO1:   DefaultBPO1BlobConfig,
			BPO2:   DefaultBPO2BlobConfig,
		},
	}
	// AllEthashProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Ethash consensus.
	AllEthashProtocolChanges = &ChainConfig{
		ChainID:                 big.NewInt(1337),
		HomesteadBlock:          big.NewInt(0),
		DAOForkBlock:            nil,
		DAOForkSupport:          false,
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       big.NewInt(0),
		GrayGlacierBlock:        big.NewInt(0),
		TerminalTotalDifficulty: big.NewInt(math.MaxInt64),
		MergeNetsplitBlock:      nil,
		ShanghaiTime:            nil,
		CancunTime:              nil,
		PragueTime:              nil,
		OsakaTime:               nil,
		VerkleTime:              nil,
		Ethash:                  new(EthashConfig),
		Clique:                  nil,
	}

	AllDevChainProtocolChanges = &ChainConfig{
		ChainID:                 big.NewInt(1337),
		HomesteadBlock:          big.NewInt(0),
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       big.NewInt(0),
		GrayGlacierBlock:        big.NewInt(0),
		ShanghaiTime:            newUint64(0),
		CancunTime:              newUint64(0),
		TerminalTotalDifficulty: big.NewInt(0),
		PragueTime:              newUint64(0),
		OsakaTime:               newUint64(0),
		BlobScheduleConfig: &BlobScheduleConfig{
			Cancun: DefaultCancunBlobConfig,
			Prague: DefaultPragueBlobConfig,
			Osaka:  DefaultOsakaBlobConfig,
		},
	}

	// AllCliqueProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Clique consensus.
	AllCliqueProtocolChanges = &ChainConfig{
		ChainID:                 big.NewInt(1337),
		HomesteadBlock:          big.NewInt(0),
		DAOForkBlock:            nil,
		DAOForkSupport:          false,
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       nil,
		GrayGlacierBlock:        nil,
		MergeNetsplitBlock:      nil,
		ShanghaiTime:            nil,
		CancunTime:              nil,
		PragueTime:              nil,
		OsakaTime:               nil,
		VerkleTime:              nil,
		TerminalTotalDifficulty: big.NewInt(math.MaxInt64),
		Ethash:                  nil,
		Clique:                  &CliqueConfig{Period: 0, Epoch: 30000},
	}

	// TestChainConfig contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers for testing purposes.
	TestChainConfig = &ChainConfig{
		ChainID:                 big.NewInt(1),
		HomesteadBlock:          big.NewInt(0),
		DAOForkBlock:            nil,
		DAOForkSupport:          false,
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       big.NewInt(0),
		GrayGlacierBlock:        big.NewInt(0),
		MergeNetsplitBlock:      nil,
		ShanghaiTime:            nil,
		CancunTime:              nil,
		PragueTime:              nil,
		OsakaTime:               nil,
		VerkleTime:              nil,
		TerminalTotalDifficulty: big.NewInt(math.MaxInt64),
		Ethash:                  new(EthashConfig),
		Clique:                  nil,
	}

	// MergedTestChainConfig contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers for testing purposes.
	MergedTestChainConfig = &ChainConfig{
		ChainID:                 big.NewInt(1),
		HomesteadBlock:          big.NewInt(0),
		DAOForkBlock:            nil,
		DAOForkSupport:          false,
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       big.NewInt(0),
		GrayGlacierBlock:        big.NewInt(0),
		MergeNetsplitBlock:      big.NewInt(0),
		ShanghaiTime:            newUint64(0),
		CancunTime:              newUint64(0),
		PragueTime:              newUint64(0),
		OsakaTime:               newUint64(0),
		VerkleTime:              nil,
		TerminalTotalDifficulty: big.NewInt(0),
		Ethash:                  new(EthashConfig),
		Clique:                  nil,
		BlobScheduleConfig: &BlobScheduleConfig{
			Cancun: DefaultCancunBlobConfig,
			Prague: DefaultPragueBlobConfig,
			Osaka:  DefaultOsakaBlobConfig,
		},
	}

	// NonActivatedConfig defines the chain configuration without activating
	// any protocol change (EIPs).
	NonActivatedConfig = &ChainConfig{
		ChainID:                 big.NewInt(1),
		HomesteadBlock:          nil,
		DAOForkBlock:            nil,
		DAOForkSupport:          false,
		EIP150Block:             nil,
		EIP155Block:             nil,
		EIP158Block:             nil,
		ByzantiumBlock:          nil,
		ConstantinopleBlock:     nil,
		PetersburgBlock:         nil,
		IstanbulBlock:           nil,
		MuirGlacierBlock:        nil,
		BerlinBlock:             nil,
		LondonBlock:             nil,
		ArrowGlacierBlock:       nil,
		GrayGlacierBlock:        nil,
		MergeNetsplitBlock:      nil,
		ShanghaiTime:            nil,
		CancunTime:              nil,
		PragueTime:              nil,
		OsakaTime:               nil,
		VerkleTime:              nil,
		TerminalTotalDifficulty: big.NewInt(math.MaxInt64),
		Ethash:                  new(EthashConfig),
		Clique:                  nil,
	}
	TestRules = TestChainConfig.Rules(new(big.Int), false, 0)
)

var (
	// DefaultCancunBlobConfig is the default blob configuration for the Cancun fork.
	DefaultCancunBlobConfig = &BlobConfig{
		Target:         3,
		Max:            6,
		UpdateFraction: 3338477,
	}
	// DefaultPragueBlobConfig is the default blob configuration for the Prague fork.
	DefaultPragueBlobConfig = &BlobConfig{
		Target:         6,
		Max:            9,
		UpdateFraction: 5007716,
	}
	// DefaultOsakaBlobConfig is the default blob configuration for the Osaka fork.
	DefaultOsakaBlobConfig = &BlobConfig{
		Target:         6,
		Max:            9,
		UpdateFraction: 5007716,
	}
	// DefaultBPO1BlobConfig is the default blob configuration for the BPO1 fork.
	DefaultBPO1BlobConfig = &BlobConfig{
		Target:         10,
		Max:            15,
		UpdateFraction: 8346193,
	}
	// DefaultBPO2BlobConfig is the default blob configuration for the BPO2 fork.
	DefaultBPO2BlobConfig = &BlobConfig{
		Target:         14,
		Max:            21,
		UpdateFraction: 11684671,
	}
	// DefaultBPO3BlobConfig is the default blob configuration for the BPO3 fork.
	DefaultBPO3BlobConfig = &BlobConfig{
		Target:         21,
		Max:            32,
		UpdateFraction: 20609697,
	}
	// DefaultBPO4BlobConfig is the default blob configuration for the BPO4 fork.
	DefaultBPO4BlobConfig = &BlobConfig{
		Target:         14,
		Max:            21,
		UpdateFraction: 13739630,
	}
	// DefaultBlobSchedule is the latest configured blob schedule for Ethereum mainnet.
	DefaultBlobSchedule = &BlobScheduleConfig{
		Cancun: DefaultCancunBlobConfig,
		Prague: DefaultPragueBlobConfig,
		Osaka:  DefaultOsakaBlobConfig,
	}
)

// NetworkNames are user friendly names to use in the chain spec banner.
var NetworkNames = map[string]string{
	MainnetChainConfig.ChainID.String(): "mainnet",
	SepoliaChainConfig.ChainID.String(): "sepolia",
	HoleskyChainConfig.ChainID.String(): "holesky",
	HoodiChainConfig.ChainID.String():   "hoodi",
}

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	ChainID *big.Int `json:"chainId"` // chainId identifies the current chain and is used for replay protection

	HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)

	DAOForkBlock   *big.Int `json:"daoForkBlock,omitempty"`   // TheDAO hard-fork switch block (nil = no fork)
	DAOForkSupport bool     `json:"daoForkSupport,omitempty"` // Whether the nodes supports or opposes the DAO hard-fork

	// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
	EIP150Block *big.Int `json:"eip150Block,omitempty"` // EIP150 HF block (nil = no fork)
	EIP155Block *big.Int `json:"eip155Block,omitempty"` // EIP155 HF block
	EIP158Block *big.Int `json:"eip158Block,omitempty"` // EIP158 HF block

	ByzantiumBlock      *big.Int `json:"byzantiumBlock,omitempty"`      // Byzantium switch block (nil = no fork, 0 = already on byzantium)
	ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"` // Constantinople switch block (nil = no fork, 0 = already activated)
	PetersburgBlock     *big.Int `json:"petersburgBlock,omitempty"`     // Petersburg switch block (nil = same as Constantinople)
	IstanbulBlock       *big.Int `json:"istanbulBlock,omitempty"`       // Istanbul switch block (nil = no fork, 0 = already on istanbul)
	MuirGlacierBlock    *big.Int `json:"muirGlacierBlock,omitempty"`    // Eip-2384 (bomb delay) switch block (nil = no fork, 0 = already activated)
	BerlinBlock         *big.Int `json:"berlinBlock,omitempty"`         // Berlin switch block (nil = no fork, 0 = already on berlin)
	LondonBlock         *big.Int `json:"londonBlock,omitempty"`         // London switch block (nil = no fork, 0 = already on london)
	ArrowGlacierBlock   *big.Int `json:"arrowGlacierBlock,omitempty"`   // Eip-4345 (bomb delay) switch block (nil = no fork, 0 = already activated)
	GrayGlacierBlock    *big.Int `json:"grayGlacierBlock,omitempty"`    // Eip-5133 (bomb delay) switch block (nil = no fork, 0 = already activated)
	MergeNetsplitBlock  *big.Int `json:"mergeNetsplitBlock,omitempty"`  // Virtual fork after The Merge to use as a network splitter

	// Fork scheduling was switched from blocks to timestamps here

	ShanghaiTime  *uint64 `json:"shanghaiTime,omitempty"`  // Shanghai switch time (nil = no fork, 0 = already on shanghai)
	CancunTime    *uint64 `json:"cancunTime,omitempty"`    // Cancun switch time (nil = no fork, 0 = already on cancun)
	PragueTime    *uint64 `json:"pragueTime,omitempty"`    // Prague switch time (nil = no fork, 0 = already on prague)
	OsakaTime     *uint64 `json:"osakaTime,omitempty"`     // Osaka switch time (nil = no fork, 0 = already on osaka)
	BPO1Time      *uint64 `json:"bpo1Time,omitempty"`      // BPO1 switch time (nil = no fork, 0 = already on bpo1)
	BPO2Time      *uint64 `json:"bpo2Time,omitempty"`      // BPO2 switch time (nil = no fork, 0 = already on bpo2)
	BPO3Time      *uint64 `json:"bpo3Time,omitempty"`      // BPO3 switch time (nil = no fork, 0 = already on bpo3)
	BPO4Time      *uint64 `json:"bpo4Time,omitempty"`      // BPO4 switch time (nil = no fork, 0 = already on bpo4)
	BPO5Time      *uint64 `json:"bpo5Time,omitempty"`      // BPO5 switch time (nil = no fork, 0 = already on bpo5)
	AmsterdamTime *uint64 `json:"amsterdamTime,omitempty"` // Amsterdam switch time (nil = no fork, 0 = already on amsterdam)
	VerkleTime    *uint64 `json:"verkleTime,omitempty"`    // Verkle switch time (nil = no fork, 0 = already on verkle)

	// TerminalTotalDifficulty is the amount of total difficulty reached by
	// the network that triggers the consensus upgrade.
	TerminalTotalDifficulty *big.Int `json:"terminalTotalDifficulty,omitempty"`

	DepositContractAddress common.Address `json:"depositContractAddress,omitempty"`

	// EnableVerkleAtGenesis is a flag that specifies whether the network uses
	// the Verkle tree starting from the genesis block. If set to true, the
	// genesis state will be committed using the Verkle tree, eliminating the
	// need for any Verkle transition later.
	//
	// This is a temporary flag only for verkle devnet testing, where verkle is
	// activated at genesis, and the configured activation date has already passed.
	//
	// In production networks (mainnet and public testnets), verkle activation
	// always occurs after the genesis block, making this flag irrelevant in
	// those cases.
	EnableVerkleAtGenesis bool `json:"enableVerkleAtGenesis,omitempty"`

	// Various consensus engines
	Ethash             *EthashConfig       `json:"ethash,omitempty"`
	Clique             *CliqueConfig       `json:"clique,omitempty"`
	BlobScheduleConfig *BlobScheduleConfig `json:"blobSchedule,omitempty"`
}

// EthashConfig is the consensus engine configs for proof-of-work based sealing.
type EthashConfig struct{}

// String implements the stringer interface, returning the consensus engine details.
func (c EthashConfig) String() string {
	return "ethash"
}

// CliqueConfig is the consensus engine configs for proof-of-authority based sealing.
type CliqueConfig struct {
	Period uint64 `json:"period"` // Number of seconds between blocks to enforce
	Epoch  uint64 `json:"epoch"`  // Epoch length to reset votes and checkpoint
}

// String implements the stringer interface, returning the consensus engine details.
func (c CliqueConfig) String() string {
	return fmt.Sprintf("clique(period: %d, epoch: %d)", c.Period, c.Epoch)
}

// String implements the fmt.Stringer interface, returning a string representation
// of ChainConfig.
func (c *ChainConfig) String() string {
	result := fmt.Sprintf("ChainConfig{ChainID: %v", c.ChainID)

	// Add block-based forks
	if c.HomesteadBlock != nil {
		result += fmt.Sprintf(", HomesteadBlock: %v", c.HomesteadBlock)
	}
	if c.DAOForkBlock != nil {
		result += fmt.Sprintf(", DAOForkBlock: %v", c.DAOForkBlock)
	}
	if c.EIP150Block != nil {
		result += fmt.Sprintf(", EIP150Block: %v", c.EIP150Block)
	}
	if c.EIP155Block != nil {
		result += fmt.Sprintf(", EIP155Block: %v", c.EIP155Block)
	}
	if c.EIP158Block != nil {
		result += fmt.Sprintf(", EIP158Block: %v", c.EIP158Block)
	}
	if c.ByzantiumBlock != nil {
		result += fmt.Sprintf(", ByzantiumBlock: %v", c.ByzantiumBlock)
	}
	if c.ConstantinopleBlock != nil {
		result += fmt.Sprintf(", ConstantinopleBlock: %v", c.ConstantinopleBlock)
	}
	if c.PetersburgBlock != nil {
		result += fmt.Sprintf(", PetersburgBlock: %v", c.PetersburgBlock)
	}
	if c.IstanbulBlock != nil {
		result += fmt.Sprintf(", IstanbulBlock: %v", c.IstanbulBlock)
	}
	if c.MuirGlacierBlock != nil {
		result += fmt.Sprintf(", MuirGlacierBlock: %v", c.MuirGlacierBlock)
	}
	if c.BerlinBlock != nil {
		result += fmt.Sprintf(", BerlinBlock: %v", c.BerlinBlock)
	}
	if c.LondonBlock != nil {
		result += fmt.Sprintf(", LondonBlock: %v", c.LondonBlock)
	}
	if c.ArrowGlacierBlock != nil {
		result += fmt.Sprintf(", ArrowGlacierBlock: %v", c.ArrowGlacierBlock)
	}
	if c.GrayGlacierBlock != nil {
		result += fmt.Sprintf(", GrayGlacierBlock: %v", c.GrayGlacierBlock)
	}
	if c.MergeNetsplitBlock != nil {
		result += fmt.Sprintf(", MergeNetsplitBlock: %v", c.MergeNetsplitBlock)
	}

	// Add timestamp-based forks
	if c.ShanghaiTime != nil {
		result += fmt.Sprintf(", ShanghaiTime: %v", *c.ShanghaiTime)
	}
	if c.CancunTime != nil {
		result += fmt.Sprintf(", CancunTime: %v", *c.CancunTime)
	}
	if c.PragueTime != nil {
		result += fmt.Sprintf(", PragueTime: %v", *c.PragueTime)
	}
	if c.OsakaTime != nil {
		result += fmt.Sprintf(", OsakaTime: %v", *c.OsakaTime)
	}
	if c.BPO1Time != nil {
		result += fmt.Sprintf(", BPO1Time: %v", *c.BPO1Time)
	}
	if c.BPO2Time != nil {
		result += fmt.Sprintf(", BPO2Time: %v", *c.BPO2Time)
	}
	if c.BPO3Time != nil {
		result += fmt.Sprintf(", BPO3Time: %v", *c.BPO3Time)
	}
	if c.BPO4Time != nil {
		result += fmt.Sprintf(", BPO4Time: %v", *c.BPO4Time)
	}
	if c.BPO5Time != nil {
		result += fmt.Sprintf(", BPO5Time: %v", *c.BPO5Time)
	}
	if c.AmsterdamTime != nil {
		result += fmt.Sprintf(", AmsterdamTime: %v", *c.AmsterdamTime)
	}
	if c.VerkleTime != nil {
		result += fmt.Sprintf(", VerkleTime: %v", *c.VerkleTime)
	}
	result += "}"
	return result
}

// Description returns a human-readable description of ChainConfig.
func (c *ChainConfig) Description() string {
	var banner string

	// Create some basic network config output
	network := NetworkNames[c.ChainID.String()]
	if network == "" {
		network = "unknown"
	}
	banner += fmt.Sprintf("Chain ID:  %v (%s)\n", c.ChainID, network)
	switch {
	case c.Ethash != nil:
		banner += "Consensus: Beacon (proof-of-stake), merged from Ethash (proof-of-work)\n"
	case c.Clique != nil:
		banner += "Consensus: Beacon (proof-of-stake), merged from Clique (proof-of-authority)\n"
	default:
		banner += "Consensus: unknown\n"
	}
	banner += "\n"

	// Create a list of forks with a short description of them. Forks that only
	// makes sense for mainnet should be optional at printing to avoid bloating
	// the output for testnets and private networks.
	banner += "Pre-Merge hard forks (block based):\n"
	banner += fmt.Sprintf(" - Homestead:                   #%-8v\n", c.HomesteadBlock)
	if c.DAOForkBlock != nil {
		banner += fmt.Sprintf(" - DAO Fork:                    #%-8v\n", c.DAOForkBlock)
	}
	banner += fmt.Sprintf(" - Tangerine Whistle (EIP 150): #%-8v\n", c.EIP150Block)
	banner += fmt.Sprintf(" - Spurious Dragon/1 (EIP 155): #%-8v\n", c.EIP155Block)
	banner += fmt.Sprintf(" - Spurious Dragon/2 (EIP 158): #%-8v\n", c.EIP158Block)
	banner += fmt.Sprintf(" - Byzantium:                   #%-8v\n", c.ByzantiumBlock)
	banner += fmt.Sprintf(" - Constantinople:              #%-8v\n", c.ConstantinopleBlock)
	banner += fmt.Sprintf(" - Petersburg:                  #%-8v\n", c.PetersburgBlock)
	banner += fmt.Sprintf(" - Istanbul:                    #%-8v\n", c.IstanbulBlock)
	if c.MuirGlacierBlock != nil {
		banner += fmt.Sprintf(" - Muir Glacier:                #%-8v\n", c.MuirGlacierBlock)
	}
	banner += fmt.Sprintf(" - Berlin:                      #%-8v\n", c.BerlinBlock)
	banner += fmt.Sprintf(" - London:                      #%-8v\n", c.LondonBlock)
	if c.ArrowGlacierBlock != nil {
		banner += fmt.Sprintf(" - Arrow Glacier:               #%-8v\n", c.ArrowGlacierBlock)
	}
	if c.GrayGlacierBlock != nil {
		banner += fmt.Sprintf(" - Gray Glacier:                #%-8v\n", c.GrayGlacierBlock)
	}
	banner += "\n"

	// Add a special section for the merge as it's non-obvious
	banner += "Merge configured:\n"
	banner += fmt.Sprintf(" - Total terminal difficulty:  %v\n", c.TerminalTotalDifficulty)
	if c.MergeNetsplitBlock != nil {
		banner += fmt.Sprintf(" - Merge netsplit block:       #%-8v\n", c.MergeNetsplitBlock)
	}
	banner += "\n"

	// Create a list of forks post-merge
	banner += "Post-Merge hard forks (timestamp based):\n"
	if c.ShanghaiTime != nil {
		banner += fmt.Sprintf(" - Shanghai:                    @%-10v\n", *c.ShanghaiTime)
	}
	if c.CancunTime != nil {
		banner += fmt.Sprintf(" - Cancun:                      @%-10v blob: (%s)\n", *c.CancunTime, c.BlobScheduleConfig.Cancun)
	}
	if c.PragueTime != nil {
		banner += fmt.Sprintf(" - Prague:                      @%-10v blob: (%s)\n", *c.PragueTime, c.BlobScheduleConfig.Prague)
	}
	if c.OsakaTime != nil {
		banner += fmt.Sprintf(" - Osaka:                       @%-10v blob: (%s)\n", *c.OsakaTime, c.BlobScheduleConfig.Osaka)
	}
	if c.BPO1Time != nil {
		banner += fmt.Sprintf(" - BPO1:                        @%-10v blob: (%s)\n", *c.BPO1Time, c.BlobScheduleConfig.BPO1)
	}
	if c.BPO2Time != nil {
		banner += fmt.Sprintf(" - BPO2:                        @%-10v blob: (%s)\n", *c.BPO2Time, c.BlobScheduleConfig.BPO2)
	}
	if c.BPO3Time != nil {
		banner += fmt.Sprintf(" - BPO3:                        @%-10v blob: (%s)\n", *c.BPO3Time, c.BlobScheduleConfig.BPO3)
	}
	if c.BPO4Time != nil {
		banner += fmt.Sprintf(" - BPO4:                        @%-10v blob: (%s)\n", *c.BPO4Time, c.BlobScheduleConfig.BPO4)
	}
	if c.BPO5Time != nil {
		banner += fmt.Sprintf(" - BPO5:                        @%-10v blob: (%s)\n", *c.BPO5Time, c.BlobScheduleConfig.BPO5)
	}
	if c.AmsterdamTime != nil {
		banner += fmt.Sprintf(" - Amsterdam:									 @%-10v blob: (%s)\n", *c.AmsterdamTime, c.BlobScheduleConfig.Amsterdam)
	}
	if c.VerkleTime != nil {
		banner += fmt.Sprintf(" - Verkle:                      @%-10v blob: (%s)\n", *c.VerkleTime, c.BlobScheduleConfig.Verkle)
	}
	banner += fmt.Sprintf("\nAll fork specifications can be found at https://ethereum.github.io/execution-specs/src/ethereum/forks/\n")
	return banner
}

// BlobConfig specifies the target and max blobs per block for the associated fork.
type BlobConfig struct {
	Target         int    `json:"target"`
	Max            int    `json:"max"`
	UpdateFraction uint64 `json:"baseFeeUpdateFraction"`
}

// String implement fmt.Stringer, returning string format blob config.
func (bc *BlobConfig) String() string {
	if bc == nil {
		return "nil"
	}
	return fmt.Sprintf("target: %d, max: %d, fraction: %d", bc.Target, bc.Max, bc.UpdateFraction)
}

// BlobScheduleConfig determines target and max number of blobs allow per fork.
type BlobScheduleConfig struct {
	Cancun    *BlobConfig `json:"cancun,omitempty"`
	Prague    *BlobConfig `json:"prague,omitempty"`
	Osaka     *BlobConfig `json:"osaka,omitempty"`
	Verkle    *BlobConfig `json:"verkle,omitempty"`
	BPO1      *BlobConfig `json:"bpo1,omitempty"`
	BPO2      *BlobConfig `json:"bpo2,omitempty"`
	BPO3      *BlobConfig `json:"bpo3,omitempty"`
	BPO4      *BlobConfig `json:"bpo4,omitempty"`
	BPO5      *BlobConfig `json:"bpo5,omitempty"`
	Amsterdam *BlobConfig `json:"amsterdam,omitempty"`
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	return isBlockForked(c.HomesteadBlock, num)
}

// IsDAOFork returns whether num is either equal to the DAO fork block or greater.
func (c *ChainConfig) IsDAOFork(num *big.Int) bool {
	return isBlockForked(c.DAOForkBlock, num)
}

// IsEIP150 returns whether num is either equal to the EIP150 fork block or greater.
func (c *ChainConfig) IsEIP150(num *big.Int) bool {
	return isBlockForked(c.EIP150Block, num)
}

// IsEIP155 returns whether num is either equal to the EIP155 fork block or greater.
func (c *ChainConfig) IsEIP155(num *big.Int) bool {
	return isBlockForked(c.EIP155Block, num)
}

// IsEIP158 returns whether num is either equal to the EIP158 fork block or greater.
func (c *ChainConfig) IsEIP158(num *big.Int) bool {
	return isBlockForked(c.EIP158Block, num)
}

// IsByzantium returns whether num is either equal to the Byzantium fork block or greater.
func (c *ChainConfig) IsByzantium(num *big.Int) bool {
	return isBlockForked(c.ByzantiumBlock, num)
}

// IsConstantinople returns whether num is either equal to the Constantinople fork block or greater.
func (c *ChainConfig) IsConstantinople(num *big.Int) bool {
	return isBlockForked(c.ConstantinopleBlock, num)
}

// IsMuirGlacier returns whether num is either equal to the Muir Glacier (EIP-2384) fork block or greater.
func (c *ChainConfig) IsMuirGlacier(num *big.Int) bool {
	return isBlockForked(c.MuirGlacierBlock, num)
}

// IsPetersburg returns whether num is either
// - equal to or greater than the PetersburgBlock fork block,
// - OR is nil, and Constantinople is active
func (c *ChainConfig) IsPetersburg(num *big.Int) bool {
	return isBlockForked(c.PetersburgBlock, num) || c.PetersburgBlock == nil && isBlockForked(c.ConstantinopleBlock, num)
}

// IsIstanbul returns whether num is either equal to the Istanbul fork block or greater.
func (c *ChainConfig) IsIstanbul(num *big.Int) bool {
	return isBlockForked(c.IstanbulBlock, num)
}

// IsBerlin returns whether num is either equal to the Berlin fork block or greater.
func (c *ChainConfig) IsBerlin(num *big.Int) bool {
	return isBlockForked(c.BerlinBlock, num)
}

// IsLondon returns whether num is either equal to the London fork block or greater.
func (c *ChainConfig) IsLondon(num *big.Int) bool {
	return isBlockForked(c.LondonBlock, num)
}

// IsArrowGlacier returns whether num is either equal to the Arrow Glacier (EIP-4345) fork block or greater.
func (c *ChainConfig) IsArrowGlacier(num *big.Int) bool {
	return isBlockForked(c.ArrowGlacierBlock, num)
}

// IsGrayGlacier returns whether num is either equal to the Gray Glacier (EIP-5133) fork block or greater.
func (c *ChainConfig) IsGrayGlacier(num *big.Int) bool {
	return isBlockForked(c.GrayGlacierBlock, num)
}

// IsTerminalPoWBlock returns whether the given block is the last block of PoW stage.
func (c *ChainConfig) IsTerminalPoWBlock(parentTotalDiff *big.Int, totalDiff *big.Int) bool {
	if c.TerminalTotalDifficulty == nil {
		return false
	}
	return parentTotalDiff.Cmp(c.TerminalTotalDifficulty) < 0 && totalDiff.Cmp(c.TerminalTotalDifficulty) >= 0
}

// IsPostMerge reports whether the given block number is assumed to be post-merge.
// Here we check the MergeNetsplitBlock to allow configuring networks with a PoW or
// PoA chain for unit testing purposes.
func (c *ChainConfig) IsPostMerge(blockNum uint64, timestamp uint64) bool {
	mergedAtGenesis := c.TerminalTotalDifficulty != nil && c.TerminalTotalDifficulty.Sign() == 0
	return mergedAtGenesis ||
		c.MergeNetsplitBlock != nil && blockNum >= c.MergeNetsplitBlock.Uint64() ||
		c.ShanghaiTime != nil && timestamp >= *c.ShanghaiTime
}

// IsShanghai returns whether time is either equal to the Shanghai fork time or greater.
func (c *ChainConfig) IsShanghai(num *big.Int, time uint64) bool {
	return c.IsLondon(num) && isTimestampForked(c.ShanghaiTime, time)
}

// IsCancun returns whether time is either equal to the Cancun fork time or greater.
func (c *ChainConfig) IsCancun(num *big.Int, time uint64) bool {
	return c.IsLondon(num) && isTimestampForked(c.CancunTime, time)
}

// IsPrague returns whether time is either equal to the Prague fork time or greater.
func (c *ChainConfig) IsPrague(num *big.Int, time uint64) bool {
	return c.IsLondon(num) && isTimestampForked(c.PragueTime, time)
}

// IsOsaka returns whether time is either equal to the Osaka fork time or greater.
func (c *ChainConfig) IsOsaka(num *big.Int, time uint64) bool {
	return c.IsLondon(num) && isTimestampForked(c.OsakaTime, time)
}

// IsBPO1 returns whether time is either equal to the BPO1 fork time or greater.
func (c *ChainConfig) IsBPO1(num *big.Int, time uint64) bool {
	return c.IsLondon(num) && isTimestampForked(c.BPO1Time, time)
}

// IsBPO2 returns whether time is either equal to the BPO2 fork time or greater.
func (c *ChainConfig) IsBPO2(num *big.Int, time uint64) bool {
	return c.IsLondon(num) && isTimestampForked(c.BPO2Time, time)
}

// IsBPO3 returns whether time is either equal to the BPO3 fork time or greater.
func (c *ChainConfig) IsBPO3(num *big.Int, time uint64) bool {
	return c.IsLondon(num) && isTimestampForked(c.BPO3Time, time)
}

// IsBPO4 returns whether time is either equal to the BPO4 fork time or greater.
func (c *ChainConfig) IsBPO4(num *big.Int, time uint64) bool {
	return c.IsLondon(num) && isTimestampForked(c.BPO4Time, time)
}

// IsBPO5 returns whether time is either equal to the BPO5 fork time or greater.
func (c *ChainConfig) IsBPO5(num *big.Int, time uint64) bool {
	return c.IsLondon(num) && isTimestampForked(c.BPO5Time, time)
}

// IsAmsterdam returns whether time is either equal to the Amsterdam fork time or greater.
func (c *ChainConfig) IsAmsterdam(num *big.Int, time uint64) bool {
	return c.IsLondon(num) && isTimestampForked(c.AmsterdamTime, time)
}

// IsVerkle returns whether time is either equal to the Verkle fork time or greater.
func (c *ChainConfig) IsVerkle(num *big.Int, time uint64) bool {
	return c.IsLondon(num) && isTimestampForked(c.VerkleTime, time)
}

// IsVerkleGenesis checks whether the verkle fork is activated at the genesis block.
//
// Verkle mode is considered enabled if the verkle fork time is configured,
// regardless of whether the local time has surpassed the fork activation time.
// This is a temporary workaround for verkle devnet testing, where verkle is
// activated at genesis, and the configured activation date has already passed.
//
// In production networks (mainnet and public testnets), verkle activation
// always occurs after the genesis block, making this function irrelevant in
// those cases.
func (c *ChainConfig) IsVerkleGenesis() bool {
	return c.EnableVerkleAtGenesis
}

// IsEIP4762 returns whether eip 4762 has been activated at given block.
func (c *ChainConfig) IsEIP4762(num *big.Int, time uint64) bool {
	return c.IsVerkle(num, time)
}

// CheckCompatible checks whether scheduled fork transitions have been imported
// with a mismatching chain configuration.
func (c *ChainConfig) CheckCompatible(newcfg *ChainConfig, height uint64, time uint64) *ConfigCompatError {
	var (
		bhead = new(big.Int).SetUint64(height)
		btime = time
	)
	// Iterate checkCompatible to find the lowest conflict.
	var lasterr *ConfigCompatError
	for {
		err := c.checkCompatible(newcfg, bhead, btime)
		if err == nil || (lasterr != nil && err.RewindToBlock == lasterr.RewindToBlock && err.RewindToTime == lasterr.RewindToTime) {
			break
		}
		lasterr = err

		if err.RewindToTime > 0 {
			btime = err.RewindToTime
		} else {
			bhead.SetUint64(err.RewindToBlock)
		}
	}
	return lasterr
}

// CheckConfigForkOrder checks that we don't "skip" any forks, geth isn't pluggable enough
// to guarantee that forks can be implemented in a different order than on official networks
func (c *ChainConfig) CheckConfigForkOrder() error {
	type fork struct {
		name      string
		block     *big.Int // forks up to - and including the merge - were defined with block numbers
		timestamp *uint64  // forks after the merge are scheduled using timestamps
		optional  bool     // if true, the fork may be nil and next fork is still allowed
	}
	var lastFork fork
	for _, cur := range []fork{
		{name: "homesteadBlock", block: c.HomesteadBlock},
		{name: "daoForkBlock", block: c.DAOForkBlock, optional: true},
		{name: "eip150Block", block: c.EIP150Block},
		{name: "eip155Block", block: c.EIP155Block},
		{name: "eip158Block", block: c.EIP158Block},
		{name: "byzantiumBlock", block: c.ByzantiumBlock},
		{name: "constantinopleBlock", block: c.ConstantinopleBlock},
		{name: "petersburgBlock", block: c.PetersburgBlock},
		{name: "istanbulBlock", block: c.IstanbulBlock},
		{name: "muirGlacierBlock", block: c.MuirGlacierBlock, optional: true},
		{name: "berlinBlock", block: c.BerlinBlock},
		{name: "londonBlock", block: c.LondonBlock},
		{name: "arrowGlacierBlock", block: c.ArrowGlacierBlock, optional: true},
		{name: "grayGlacierBlock", block: c.GrayGlacierBlock, optional: true},
		{name: "mergeNetsplitBlock", block: c.MergeNetsplitBlock, optional: true},
		{name: "shanghaiTime", timestamp: c.ShanghaiTime},
		{name: "cancunTime", timestamp: c.CancunTime, optional: true},
		{name: "pragueTime", timestamp: c.PragueTime, optional: true},
		{name: "osakaTime", timestamp: c.OsakaTime, optional: true},
		{name: "verkleTime", timestamp: c.VerkleTime, optional: true},
		{name: "bpo1", timestamp: c.BPO1Time, optional: true},
		{name: "bpo2", timestamp: c.BPO2Time, optional: true},
		{name: "bpo3", timestamp: c.BPO3Time, optional: true},
		{name: "bpo4", timestamp: c.BPO4Time, optional: true},
		{name: "bpo5", timestamp: c.BPO5Time, optional: true},
		{name: "amsterdam", timestamp: c.AmsterdamTime, optional: true},
	} {
		if lastFork.name != "" {
			switch {
			// Non-optional forks must all be present in the chain config up to the last defined fork
			case lastFork.block == nil && lastFork.timestamp == nil && (cur.block != nil || cur.timestamp != nil):
				if cur.block != nil {
					return fmt.Errorf("unsupported fork ordering: %v not enabled, but %v enabled at block %v",
						lastFork.name, cur.name, cur.block)
				} else {
					return fmt.Errorf("unsupported fork ordering: %v not enabled, but %v enabled at timestamp %v",
						lastFork.name, cur.name, *cur.timestamp)
				}

			// Fork (whether defined by block or timestamp) must follow the fork definition sequence
			case (lastFork.block != nil && cur.block != nil) || (lastFork.timestamp != nil && cur.timestamp != nil):
				if lastFork.block != nil && lastFork.block.Cmp(cur.block) > 0 {
					return fmt.Errorf("unsupported fork ordering: %v enabled at block %v, but %v enabled at block %v",
						lastFork.name, lastFork.block, cur.name, cur.block)
				} else if lastFork.timestamp != nil && *lastFork.timestamp > *cur.timestamp {
					return fmt.Errorf("unsupported fork ordering: %v enabled at timestamp %v, but %v enabled at timestamp %v",
						lastFork.name, *lastFork.timestamp, cur.name, *cur.timestamp)
				}

				// Timestamp based forks can follow block based ones, but not the other way around
				if lastFork.timestamp != nil && cur.block != nil {
					return fmt.Errorf("unsupported fork ordering: %v used timestamp ordering, but %v reverted to block ordering",
						lastFork.name, cur.name)
				}
			}
		}
		// If it was optional and not set, then ignore it
		if !cur.optional || (cur.block != nil || cur.timestamp != nil) {
			lastFork = cur
		}
	}

	// Check that all forks with blobs explicitly define the blob schedule configuration.
	bsc := c.BlobScheduleConfig
	if bsc == nil {
		bsc = new(BlobScheduleConfig)
	}
	for _, cur := range []struct {
		name      string
		timestamp *uint64
		config    *BlobConfig
	}{
		{name: "cancun", timestamp: c.CancunTime, config: bsc.Cancun},
		{name: "prague", timestamp: c.PragueTime, config: bsc.Prague},
		{name: "osaka", timestamp: c.OsakaTime, config: bsc.Osaka},
		{name: "bpo1", timestamp: c.BPO1Time, config: bsc.BPO1},
		{name: "bpo2", timestamp: c.BPO2Time, config: bsc.BPO2},
		{name: "bpo3", timestamp: c.BPO3Time, config: bsc.BPO3},
		{name: "bpo4", timestamp: c.BPO4Time, config: bsc.BPO4},
		{name: "bpo5", timestamp: c.BPO5Time, config: bsc.BPO5},
		{name: "amsterdam", timestamp: c.AmsterdamTime, config: bsc.Amsterdam},
	} {
		if cur.config != nil {
			if err := cur.config.validate(); err != nil {
				return fmt.Errorf("invalid chain configuration in blobSchedule for fork %q: %v", cur.name, err)
			}
		}
		if cur.timestamp != nil {
			// If the fork is configured, a blob schedule must be defined for it.
			if cur.config == nil {
				return fmt.Errorf("invalid chain configuration: missing entry for fork %q in blobSchedule", cur.name)
			}
		}
	}
	return nil
}

func (bc *BlobConfig) validate() error {
	if bc.Max < 0 {
		return errors.New("max < 0")
	}
	if bc.Target < 0 {
		return errors.New("target < 0")
	}
	if bc.UpdateFraction == 0 {
		return errors.New("update fraction must be defined and non-zero")
	}
	return nil
}

func (c *ChainConfig) checkCompatible(newcfg *ChainConfig, headNumber *big.Int, headTimestamp uint64) *ConfigCompatError {
	if isForkBlockIncompatible(c.HomesteadBlock, newcfg.HomesteadBlock, headNumber) {
		return newBlockCompatError("Homestead fork block", c.HomesteadBlock, newcfg.HomesteadBlock)
	}
	if isForkBlockIncompatible(c.DAOForkBlock, newcfg.DAOForkBlock, headNumber) {
		return newBlockCompatError("DAO fork block", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if c.IsDAOFork(headNumber) && c.DAOForkSupport != newcfg.DAOForkSupport {
		return newBlockCompatError("DAO fork support flag", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if isForkBlockIncompatible(c.EIP150Block, newcfg.EIP150Block, headNumber) {
		return newBlockCompatError("EIP150 fork block", c.EIP150Block, newcfg.EIP150Block)
	}
	if isForkBlockIncompatible(c.EIP155Block, newcfg.EIP155Block, headNumber) {
		return newBlockCompatError("EIP155 fork block", c.EIP155Block, newcfg.EIP155Block)
	}
	if isForkBlockIncompatible(c.EIP158Block, newcfg.EIP158Block, headNumber) {
		return newBlockCompatError("EIP158 fork block", c.EIP158Block, newcfg.EIP158Block)
	}
	if c.IsEIP158(headNumber) && !configBlockEqual(c.ChainID, newcfg.ChainID) {
		return newBlockCompatError("EIP158 chain ID", c.EIP158Block, newcfg.EIP158Block)
	}
	if isForkBlockIncompatible(c.ByzantiumBlock, newcfg.ByzantiumBlock, headNumber) {
		return newBlockCompatError("Byzantium fork block", c.ByzantiumBlock, newcfg.ByzantiumBlock)
	}
	if isForkBlockIncompatible(c.ConstantinopleBlock, newcfg.ConstantinopleBlock, headNumber) {
		return newBlockCompatError("Constantinople fork block", c.ConstantinopleBlock, newcfg.ConstantinopleBlock)
	}
	if isForkBlockIncompatible(c.PetersburgBlock, newcfg.PetersburgBlock, headNumber) {
		// the only case where we allow Petersburg to be set in the past is if it is equal to Constantinople
		// mainly to satisfy fork ordering requirements which state that Petersburg fork be set if Constantinople fork is set
		if isForkBlockIncompatible(c.ConstantinopleBlock, newcfg.PetersburgBlock, headNumber) {
			return newBlockCompatError("Petersburg fork block", c.PetersburgBlock, newcfg.PetersburgBlock)
		}
	}
	if isForkBlockIncompatible(c.IstanbulBlock, newcfg.IstanbulBlock, headNumber) {
		return newBlockCompatError("Istanbul fork block", c.IstanbulBlock, newcfg.IstanbulBlock)
	}
	if isForkBlockIncompatible(c.MuirGlacierBlock, newcfg.MuirGlacierBlock, headNumber) {
		return newBlockCompatError("Muir Glacier fork block", c.MuirGlacierBlock, newcfg.MuirGlacierBlock)
	}
	if isForkBlockIncompatible(c.BerlinBlock, newcfg.BerlinBlock, headNumber) {
		return newBlockCompatError("Berlin fork block", c.BerlinBlock, newcfg.BerlinBlock)
	}
	if isForkBlockIncompatible(c.LondonBlock, newcfg.LondonBlock, headNumber) {
		return newBlockCompatError("London fork block", c.LondonBlock, newcfg.LondonBlock)
	}
	if isForkBlockIncompatible(c.ArrowGlacierBlock, newcfg.ArrowGlacierBlock, headNumber) {
		return newBlockCompatError("Arrow Glacier fork block", c.ArrowGlacierBlock, newcfg.ArrowGlacierBlock)
	}
	if isForkBlockIncompatible(c.GrayGlacierBlock, newcfg.GrayGlacierBlock, headNumber) {
		return newBlockCompatError("Gray Glacier fork block", c.GrayGlacierBlock, newcfg.GrayGlacierBlock)
	}
	if isForkBlockIncompatible(c.MergeNetsplitBlock, newcfg.MergeNetsplitBlock, headNumber) {
		return newBlockCompatError("Merge netsplit fork block", c.MergeNetsplitBlock, newcfg.MergeNetsplitBlock)
	}
	if isForkTimestampIncompatible(c.ShanghaiTime, newcfg.ShanghaiTime, headTimestamp) {
		return newTimestampCompatError("Shanghai fork timestamp", c.ShanghaiTime, newcfg.ShanghaiTime)
	}
	if isForkTimestampIncompatible(c.CancunTime, newcfg.CancunTime, headTimestamp) {
		return newTimestampCompatError("Cancun fork timestamp", c.CancunTime, newcfg.CancunTime)
	}
	if isForkTimestampIncompatible(c.PragueTime, newcfg.PragueTime, headTimestamp) {
		return newTimestampCompatError("Prague fork timestamp", c.PragueTime, newcfg.PragueTime)
	}
	if isForkTimestampIncompatible(c.OsakaTime, newcfg.OsakaTime, headTimestamp) {
		return newTimestampCompatError("Osaka fork timestamp", c.OsakaTime, newcfg.OsakaTime)
	}
	if isForkTimestampIncompatible(c.VerkleTime, newcfg.VerkleTime, headTimestamp) {
		return newTimestampCompatError("Verkle fork timestamp", c.VerkleTime, newcfg.VerkleTime)
	}
	if isForkTimestampIncompatible(c.BPO1Time, newcfg.BPO1Time, headTimestamp) {
		return newTimestampCompatError("BPO1 fork timestamp", c.BPO1Time, newcfg.BPO1Time)
	}
	if isForkTimestampIncompatible(c.BPO2Time, newcfg.BPO2Time, headTimestamp) {
		return newTimestampCompatError("BPO2 fork timestamp", c.BPO2Time, newcfg.BPO2Time)
	}
	if isForkTimestampIncompatible(c.BPO3Time, newcfg.BPO3Time, headTimestamp) {
		return newTimestampCompatError("BPO3 fork timestamp", c.BPO3Time, newcfg.BPO3Time)
	}
	if isForkTimestampIncompatible(c.BPO4Time, newcfg.BPO4Time, headTimestamp) {
		return newTimestampCompatError("BPO4 fork timestamp", c.BPO4Time, newcfg.BPO4Time)
	}
	if isForkTimestampIncompatible(c.BPO5Time, newcfg.BPO5Time, headTimestamp) {
		return newTimestampCompatError("BPO5 fork timestamp", c.BPO5Time, newcfg.BPO5Time)
	}
	if isForkTimestampIncompatible(c.AmsterdamTime, newcfg.AmsterdamTime, headTimestamp) {
		return newTimestampCompatError("Amsterdam fork timestamp", c.AmsterdamTime, newcfg.AmsterdamTime)
	}
	return nil
}

// BaseFeeChangeDenominator bounds the amount the base fee can change between blocks.
func (c *ChainConfig) BaseFeeChangeDenominator() uint64 {
	return DefaultBaseFeeChangeDenominator
}

// ElasticityMultiplier bounds the maximum gas limit an EIP-1559 block may have.
func (c *ChainConfig) ElasticityMultiplier() uint64 {
	return DefaultElasticityMultiplier
}

// LatestFork returns the latest time-based fork that would be active for the given time.
func (c *ChainConfig) LatestFork(time uint64) forks.Fork {
	// Assume last non-time-based fork has passed.
	london := c.LondonBlock

	switch {
	case c.IsAmsterdam(london, time):
		return forks.Amsterdam
	case c.IsBPO5(london, time):
		return forks.BPO5
	case c.IsBPO4(london, time):
		return forks.BPO4
	case c.IsBPO3(london, time):
		return forks.BPO3
	case c.IsBPO2(london, time):
		return forks.BPO2
	case c.IsBPO1(london, time):
		return forks.BPO1
	case c.IsOsaka(london, time):
		return forks.Osaka
	case c.IsPrague(london, time):
		return forks.Prague
	case c.IsCancun(london, time):
		return forks.Cancun
	case c.IsShanghai(london, time):
		return forks.Shanghai
	default:
		return forks.Paris
	}
}

// BlobConfig returns the blob config associated with the provided fork.
func (c *ChainConfig) BlobConfig(fork forks.Fork) *BlobConfig {
	switch fork {
	case forks.BPO5:
		return c.BlobScheduleConfig.BPO5
	case forks.BPO4:
		return c.BlobScheduleConfig.BPO4
	case forks.BPO3:
		return c.BlobScheduleConfig.BPO3
	case forks.BPO2:
		return c.BlobScheduleConfig.BPO2
	case forks.BPO1:
		return c.BlobScheduleConfig.BPO1
	case forks.Osaka:
		return c.BlobScheduleConfig.Osaka
	case forks.Prague:
		return c.BlobScheduleConfig.Prague
	case forks.Cancun:
		return c.BlobScheduleConfig.Cancun
	default:
		return nil
	}
}

// ActiveSystemContracts returns the currently active system contracts at the
// given timestamp.
func (c *ChainConfig) ActiveSystemContracts(time uint64) map[string]common.Address {
	fork := c.LatestFork(time)
	active := make(map[string]common.Address)
	if fork >= forks.Osaka {
		// no new system contracts
	}
	if fork >= forks.Prague {
		active["CONSOLIDATION_REQUEST_PREDEPLOY_ADDRESS"] = ConsolidationQueueAddress
		active["DEPOSIT_CONTRACT_ADDRESS"] = c.DepositContractAddress
		active["HISTORY_STORAGE_ADDRESS"] = HistoryStorageAddress
		active["WITHDRAWAL_REQUEST_PREDEPLOY_ADDRESS"] = WithdrawalQueueAddress
	}
	if fork >= forks.Cancun {
		active["BEACON_ROOTS_ADDRESS"] = BeaconRootsAddress
	}
	return active
}

// Timestamp returns the timestamp associated with the fork or returns nil if
// the fork isn't defined or isn't a time-based fork.
func (c *ChainConfig) Timestamp(fork forks.Fork) *uint64 {
	switch {
	case fork == forks.BPO5:
		return c.BPO5Time
	case fork == forks.BPO4:
		return c.BPO4Time
	case fork == forks.BPO3:
		return c.BPO3Time
	case fork == forks.BPO2:
		return c.BPO2Time
	case fork == forks.BPO1:
		return c.BPO1Time
	case fork == forks.Osaka:
		return c.OsakaTime
	case fork == forks.Prague:
		return c.PragueTime
	case fork == forks.Cancun:
		return c.CancunTime
	case fork == forks.Shanghai:
		return c.ShanghaiTime
	default:
		return nil
	}
}

// isForkBlockIncompatible returns true if a fork scheduled at block s1 cannot be
// rescheduled to block s2 because head is already past the fork.
func isForkBlockIncompatible(s1, s2, head *big.Int) bool {
	return (isBlockForked(s1, head) || isBlockForked(s2, head)) && !configBlockEqual(s1, s2)
}

// isBlockForked returns whether a fork scheduled at block s is active at the
// given head block. Whilst this method is the same as isTimestampForked, they
// are explicitly separate for clearer reading.
func isBlockForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}

func configBlockEqual(x, y *big.Int) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return x.Cmp(y) == 0
}

// isForkTimestampIncompatible returns true if a fork scheduled at timestamp s1
// cannot be rescheduled to timestamp s2 because head is already past the fork.
func isForkTimestampIncompatible(s1, s2 *uint64, head uint64) bool {
	return (isTimestampForked(s1, head) || isTimestampForked(s2, head)) && !configTimestampEqual(s1, s2)
}

// isTimestampForked returns whether a fork scheduled at timestamp s is active
// at the given head timestamp. Whilst this method is the same as isBlockForked,
// they are explicitly separate for clearer reading.
func isTimestampForked(s *uint64, head uint64) bool {
	if s == nil {
		return false
	}
	return *s <= head
}

func configTimestampEqual(x, y *uint64) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return *x == *y
}

// ConfigCompatError is raised if the locally-stored blockchain is initialised with a
// ChainConfig that would alter the past.
type ConfigCompatError struct {
	What string

	// block numbers of the stored and new configurations if block based forking
	StoredBlock, NewBlock *big.Int

	// timestamps of the stored and new configurations if time based forking
	StoredTime, NewTime *uint64

	// the block number to which the local chain must be rewound to correct the error
	RewindToBlock uint64

	// the timestamp to which the local chain must be rewound to correct the error
	RewindToTime uint64
}

func newBlockCompatError(what string, storedblock, newblock *big.Int) *ConfigCompatError {
	var rew *big.Int
	switch {
	case storedblock == nil:
		rew = newblock
	case newblock == nil || storedblock.Cmp(newblock) < 0:
		rew = storedblock
	default:
		rew = newblock
	}
	err := &ConfigCompatError{
		What:          what,
		StoredBlock:   storedblock,
		NewBlock:      newblock,
		RewindToBlock: 0,
	}
	if rew != nil && rew.Sign() > 0 {
		err.RewindToBlock = rew.Uint64() - 1
	}
	return err
}

func newTimestampCompatError(what string, storedtime, newtime *uint64) *ConfigCompatError {
	var rew *uint64
	switch {
	case storedtime == nil:
		rew = newtime
	case newtime == nil || *storedtime < *newtime:
		rew = storedtime
	default:
		rew = newtime
	}
	err := &ConfigCompatError{
		What:         what,
		StoredTime:   storedtime,
		NewTime:      newtime,
		RewindToTime: 0,
	}
	if rew != nil && *rew != 0 {
		err.RewindToTime = *rew - 1
	}
	return err
}

func (err *ConfigCompatError) Error() string {
	if err.StoredBlock != nil {
		return fmt.Sprintf("mismatching %s in database (have block %d, want block %d, rewindto block %d)", err.What, err.StoredBlock, err.NewBlock, err.RewindToBlock)
	}

	if err.StoredTime == nil && err.NewTime == nil {
		return ""
	} else if err.StoredTime == nil && err.NewTime != nil {
		return fmt.Sprintf("mismatching %s in database (have timestamp nil, want timestamp %d, rewindto timestamp %d)", err.What, *err.NewTime, err.RewindToTime)
	} else if err.StoredTime != nil && err.NewTime == nil {
		return fmt.Sprintf("mismatching %s in database (have timestamp %d, want timestamp nil, rewindto timestamp %d)", err.What, *err.StoredTime, err.RewindToTime)
	}
	return fmt.Sprintf("mismatching %s in database (have timestamp %d, want timestamp %d, rewindto timestamp %d)", err.What, *err.StoredTime, *err.NewTime, err.RewindToTime)
}

// Rules wraps ChainConfig and is merely syntactic sugar or can be used for functions
// that do not have or require information about the block.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
type Rules struct {
	ChainID                                                 *big.Int
	IsHomestead, IsEIP150, IsEIP155, IsEIP158               bool
	IsEIP2929, IsEIP4762                                    bool
	IsByzantium, IsConstantinople, IsPetersburg, IsIstanbul bool
	IsBerlin, IsLondon                                      bool
	IsMerge, IsShanghai, IsCancun, IsPrague, IsOsaka        bool
	IsAmsterdam, IsVerkle                                   bool
}

// Rules ensures c's ChainID is not nil.
func (c *ChainConfig) Rules(num *big.Int, isMerge bool, timestamp uint64) Rules {
	chainID := c.ChainID
	if chainID == nil {
		chainID = new(big.Int)
	}
	// disallow setting Merge out of order
	isMerge = isMerge && c.IsLondon(num)
	isVerkle := isMerge && c.IsVerkle(num, timestamp)
	return Rules{
		ChainID:          new(big.Int).Set(chainID),
		IsHomestead:      c.IsHomestead(num),
		IsEIP150:         c.IsEIP150(num),
		IsEIP155:         c.IsEIP155(num),
		IsEIP158:         c.IsEIP158(num),
		IsByzantium:      c.IsByzantium(num),
		IsConstantinople: c.IsConstantinople(num),
		IsPetersburg:     c.IsPetersburg(num),
		IsIstanbul:       c.IsIstanbul(num),
		IsBerlin:         c.IsBerlin(num),
		IsEIP2929:        c.IsBerlin(num) && !isVerkle,
		IsLondon:         c.IsLondon(num),
		IsMerge:          isMerge,
		IsShanghai:       isMerge && c.IsShanghai(num, timestamp),
		IsCancun:         isMerge && c.IsCancun(num, timestamp),
		IsPrague:         isMerge && c.IsPrague(num, timestamp),
		IsOsaka:          isMerge && c.IsOsaka(num, timestamp),
		IsAmsterdam:      isMerge && c.IsAmsterdam(num, timestamp),
		IsVerkle:         isVerkle,
		IsEIP4762:        isVerkle,
	}
}

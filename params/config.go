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
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/log"
)

const (
	ConsensusEngineVersion1 = "v1"
	ConsensusEngineVersion2 = "v2"
	Default                 = 0
)

var (
	XDCMainnetGenesisHash = common.HexToHash("4a9d748bd78a8d0385b67788c2435dcdb914f98a96250b68863a1f8b7642d6b1") // XDC Mainnet genesis hash to enforce below configs on
	MainnetGenesisHash    = common.HexToHash("8d13370621558f4ed0da587934473c0404729f28b0ff1d50e5fdd840457a2f17") // Mainnet genesis hash to enforce below configs on
	TestnetGenesisHash    = common.HexToHash("bdea512b4f12ff1135ec92c00dc047ffb93890c2ea1aa0eefe9b013d80640075") // Testnet genesis hash to enforce below configs on
	DevnetGenesisHash     = common.HexToHash("ab6fd3cb7d1a489e03250c7d14c2d6d819a6a528d6380b31e8410951964ef423") // Devnet genesis hash to enforce below configs on
)

var (
	MainnetV2Configs = map[uint64]*V2Config{
		Default: {
			MaxMasternodes:       108,
			SwitchRound:          0,
			CertThreshold:        0.667,
			TimeoutSyncThreshold: 3,
			TimeoutPeriod:        30,
			MinePeriod:           2,
			ExpTimeoutConfig:     ExpTimeoutConfig{Base: 1.0, MaxExponent: 0},
		},
		2000: {
			MaxMasternodes:       108,
			SwitchRound:          2000,
			CertThreshold:        0.667,
			TimeoutSyncThreshold: 2,
			TimeoutPeriod:        600,
			MinePeriod:           2,
			ExpTimeoutConfig:     ExpTimeoutConfig{Base: 1.0, MaxExponent: 0},
		},
		8000: {
			MaxMasternodes:       108,
			SwitchRound:          8000,
			CertThreshold:        0.667,
			TimeoutSyncThreshold: 2,
			TimeoutPeriod:        60,
			MinePeriod:           2,
			ExpTimeoutConfig:     ExpTimeoutConfig{Base: 1.0, MaxExponent: 0},
		},
		220000: {
			MaxMasternodes:       108,
			SwitchRound:          220000,
			CertThreshold:        0.667,
			TimeoutSyncThreshold: 2,
			TimeoutPeriod:        30,
			MinePeriod:           2,
			ExpTimeoutConfig:     ExpTimeoutConfig{Base: 1.0, MaxExponent: 0},
		},
		460000: {
			MaxMasternodes:       108,
			SwitchRound:          460000,
			CertThreshold:        0.667,
			TimeoutSyncThreshold: 2,
			TimeoutPeriod:        20,
			MinePeriod:           2,
			ExpTimeoutConfig:     ExpTimeoutConfig{Base: 1.0, MaxExponent: 0},
		},
	}

	TestnetV2Configs = map[uint64]*V2Config{
		Default: {
			MaxMasternodes:       15,
			SwitchRound:          0,
			CertThreshold:        0.45,
			TimeoutSyncThreshold: 3,
			TimeoutPeriod:        60,
			MinePeriod:           2,
			ExpTimeoutConfig:     ExpTimeoutConfig{Base: 1.0, MaxExponent: 0},
		},
		900000: {
			MaxMasternodes:       108,
			SwitchRound:          900000,
			CertThreshold:        0.667,
			TimeoutSyncThreshold: 3,
			TimeoutPeriod:        60,
			MinePeriod:           2,
			ExpTimeoutConfig:     ExpTimeoutConfig{Base: 1.0, MaxExponent: 0},
		},
	}

	DevnetV2Configs = map[uint64]*V2Config{
		Default: {
			MaxMasternodes:       108,
			SwitchRound:          0,
			CertThreshold:        0.667,
			TimeoutSyncThreshold: 3,
			TimeoutPeriod:        5,
			MinePeriod:           2,
			ExpTimeoutConfig:     ExpTimeoutConfig{Base: 2.0, MaxExponent: 5},
			MasternodeReward:     5000,
			ProtectorReward:      4000,
			ObserverReward:       1000,
		},
		9999999999: {
			MaxMasternodes:       15,
			MaxProtectorNodes:    2,
			SwitchRound:          9999999999,
			CertThreshold:        0.667,
			TimeoutSyncThreshold: 3,
			TimeoutPeriod:        5,
			MinePeriod:           2,
			ExpTimeoutConfig:     ExpTimeoutConfig{Base: 2.0, MaxExponent: 5},
			MasternodeReward:     5000,
			ProtectorReward:      4000,
			ObserverReward:       1000,
		},
	}

	UnitTestV2Configs = map[uint64]*V2Config{
		Default: {
			MaxMasternodes:       18,
			SwitchRound:          0,
			CertThreshold:        0.667,
			TimeoutSyncThreshold: 2,
			TimeoutPeriod:        4,
			MinePeriod:           2,
			ExpTimeoutConfig:     ExpTimeoutConfig{Base: 1.0, MaxExponent: 0},
		},
		10: {
			MaxMasternodes:       18,
			SwitchRound:          10,
			CertThreshold:        0.667,
			TimeoutSyncThreshold: 2,
			TimeoutPeriod:        4,
			MinePeriod:           3,
			ExpTimeoutConfig:     ExpTimeoutConfig{Base: 1.0, MaxExponent: 0},
		},
		900: {
			MaxMasternodes:       20,
			MaxProtectorNodes:    17,
			SwitchRound:          900,
			CertThreshold:        0.667,
			TimeoutSyncThreshold: 4,
			TimeoutPeriod:        5,
			MinePeriod:           2,
			ExpTimeoutConfig:     ExpTimeoutConfig{Base: 1.0, MaxExponent: 0},
			MasternodeReward:     500, // double as Reward
			ProtectorReward:      400,
			ObserverReward:       300,
		},
	}

	// XDPoSChain mainnet config
	XDCMainnetChainConfig = &ChainConfig{
		ChainId:        big.NewInt(50),
		HomesteadBlock: big.NewInt(1),
		EIP150Block:    big.NewInt(2),
		EIP155Block:    big.NewInt(3),
		EIP158Block:    big.NewInt(3),
		ByzantiumBlock: big.NewInt(4),
		XDPoS: &XDPoSConfig{
			Period:              2,
			Epoch:               900,
			Reward:              5000,
			RewardCheckpoint:    900,
			Gap:                 450,
			FoudationWalletAddr: common.HexToAddress("xdc92a289fe95a85c53b8d0d113cbaef0c1ec98ac65"),
			V2: &V2{
				SwitchEpoch:   common.MaintnetConstant.TIPV2SwitchBlock.Uint64() / 900,
				SwitchBlock:   common.MaintnetConstant.TIPV2SwitchBlock,
				CurrentConfig: MainnetV2Configs[0],
				AllConfigs:    MainnetV2Configs,
			},
		},
	}

	// MainnetChainConfig is the chain parameters to run a node on the main network.
	MainnetChainConfig = &ChainConfig{
		ChainId:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(1150000),
		DAOForkBlock:        big.NewInt(1920000),
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(2463000),
		EIP155Block:         big.NewInt(2675000),
		EIP158Block:         big.NewInt(2675000),
		ByzantiumBlock:      big.NewInt(4370000),
		ConstantinopleBlock: nil,
		Ethash:              new(EthashConfig),
	}

	// TestnetChainConfig contains the chain parameters to run a node on the Ropsten test network.
	TestnetChainConfig = &ChainConfig{
		ChainId:             big.NewInt(51),
		HomesteadBlock:      big.NewInt(1),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(2),
		EIP155Block:         big.NewInt(3),
		EIP158Block:         big.NewInt(3),
		ByzantiumBlock:      big.NewInt(4),
		ConstantinopleBlock: nil,
		XDPoS: &XDPoSConfig{
			Period:              2,
			Epoch:               900,
			Reward:              5000,
			RewardCheckpoint:    900,
			Gap:                 450,
			FoudationWalletAddr: common.HexToAddress("xdc746249c61f5832c5eed53172776b460491bdcd5c"),
			V2: &V2{
				SwitchEpoch:   common.TestnetConstant.TIPV2SwitchBlock.Uint64() / 900,
				SwitchBlock:   common.TestnetConstant.TIPV2SwitchBlock,
				CurrentConfig: TestnetV2Configs[0],
				AllConfigs:    TestnetV2Configs,
			},
		},
	}

	// DevnetChainConfig contains the chain parameters to run a node on the Ropsten test network.
	DevnetChainConfig = &ChainConfig{
		ChainId:        big.NewInt(551),
		HomesteadBlock: big.NewInt(1),
		EIP150Block:    big.NewInt(2),
		EIP155Block:    big.NewInt(3),
		EIP158Block:    big.NewInt(3),
		ByzantiumBlock: big.NewInt(4),
		XDPoS: &XDPoSConfig{
			Period:              2,
			Epoch:               900,
			Reward:              5000,
			RewardCheckpoint:    900,
			Gap:                 450,
			FoudationWalletAddr: common.HexToAddress("0x746249c61f5832c5eed53172776b460491bdcd5c"),
			V2: &V2{
				SwitchEpoch:   common.DevnetConstant.TIPV2SwitchBlock.Uint64() / 900,
				SwitchBlock:   common.DevnetConstant.TIPV2SwitchBlock,
				CurrentConfig: DevnetV2Configs[0],
				AllConfigs:    DevnetV2Configs,
			},
		},
	}

	// AllEthashProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Ethash consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllEthashProtocolChanges = &ChainConfig{
		ChainId:             big.NewInt(1337),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: nil,
		Ethash:              new(EthashConfig),
		Clique:              nil,
		XDPoS:               nil,
	}

	// AllXDPoSProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the XDPoS consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllXDPoSProtocolChanges = &ChainConfig{
		ChainId:             big.NewInt(89),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: nil,
		Ethash:              nil,
		Clique:              nil,
		XDPoS:               &XDPoSConfig{Period: 0, Epoch: 900},
	}

	AllCliqueProtocolChanges = &ChainConfig{
		ChainId:             big.NewInt(1337),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: nil,
		Ethash:              nil,
		Clique:              &CliqueConfig{Period: 0, Epoch: 900},
		XDPoS:               nil,
	}

	// XDPoS config with v2 engine after block 901
	TestXDPoSMockChainConfig = &ChainConfig{
		ChainId:             big.NewInt(1337),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		Ethash:              new(EthashConfig),
		Clique:              nil,
		XDPoS: &XDPoSConfig{
			Epoch:               900,
			Gap:                 450,
			SkipV1Validation:    true,
			FoudationWalletAddr: common.HexToAddress("0x0000000000000000000000000000000000000068"),
			Reward:              250,
			V2: &V2{
				SwitchEpoch:   1,
				SwitchBlock:   big.NewInt(900),
				CurrentConfig: UnitTestV2Configs[0],
				AllConfigs:    UnitTestV2Configs,
			},
		},
	}

	TestChainConfig = &ChainConfig{
		ChainId:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: nil,
		Ethash:              new(EthashConfig),
		Clique:              nil,
		XDPoS:               nil,
	}
	TestRules = TestChainConfig.Rules(new(big.Int))
)

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	ChainId *big.Int `json:"chainId"` // Chain id identifies the current chain and is used for replay protection

	HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)

	DAOForkBlock   *big.Int `json:"daoForkBlock,omitempty"`   // TheDAO hard-fork switch block (nil = no fork)
	DAOForkSupport bool     `json:"daoForkSupport,omitempty"` // Whether the nodes supports or opposes the DAO hard-fork

	// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
	EIP150Block *big.Int `json:"eip150Block,omitempty"` // EIP150 HF block (nil = no fork)
	EIP155Block *big.Int `json:"eip155Block,omitempty"` // EIP155 HF block
	EIP158Block *big.Int `json:"eip158Block,omitempty"` // EIP158 HF block

	ByzantiumBlock      *big.Int `json:"byzantiumBlock,omitempty"`      // Byzantium switch block (nil = no fork, 0 = already on byzantium)
	ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"` // Constantinople switch block (nil = no fork, 0 = already activated)

	PetersburgBlock *big.Int `json:"petersburgBlock,omitempty"`
	IstanbulBlock   *big.Int `json:"istanbulBlock,omitempty"`
	BerlinBlock     *big.Int `json:"berlinBlock,omitempty"`
	LondonBlock     *big.Int `json:"londonBlock,omitempty"`
	MergeBlock      *big.Int `json:"mergeBlock,omitempty"`
	ShanghaiBlock   *big.Int `json:"shanghaiBlock,omitempty"`
	Eip1559Block    *big.Int `json:"eip1559Block,omitempty"`
	CancunBlock     *big.Int `json:"cancunBlock,omitempty"`

	// Various consensus engines
	Ethash *EthashConfig `json:"ethash,omitempty"`
	Clique *CliqueConfig `json:"clique,omitempty"`
	XDPoS  *XDPoSConfig  `json:"XDPoS,omitempty"`
}

// EthashConfig is the consensus engine configs for proof-of-work based sealing.
type EthashConfig struct{}

// String implements the stringer interface, returning the consensus engine details.
func (c *EthashConfig) String() string {
	return "ethash"
}

// CliqueConfig is the consensus engine configs for proof-of-authority based sealing.
type CliqueConfig struct {
	Period uint64 `json:"period"` // Number of seconds between blocks to enforce
	Epoch  uint64 `json:"epoch"`  // Epoch length to reset votes and checkpoint
}

// String implements the stringer interface, returning the consensus engine details.
func (c *CliqueConfig) String() string {
	return "clique"
}

// XDPoSConfig is the consensus engine configs for delegated-proof-of-stake based sealing.
type XDPoSConfig struct {
	Period              uint64         `json:"period"`              // Number of seconds between blocks to enforce
	Epoch               uint64         `json:"epoch"`               // Epoch length to reset votes and checkpoint
	Reward              uint64         `json:"reward"`              // Block reward - unit Ether
	RewardCheckpoint    uint64         `json:"rewardCheckpoint"`    // Checkpoint block for calculate rewards.
	Gap                 uint64         `json:"gap"`                 // Gap time preparing for the next epoch
	FoudationWalletAddr common.Address `json:"foudationWalletAddr"` // Foundation Address Wallet
	SkipV1Validation    bool           //Skip Block Validation for testing purpose, V1 consensus only
	V2                  *V2            `json:"v2"`
}

type V2 struct {
	lock sync.RWMutex // Protects the signer fields

	SwitchEpoch   uint64
	SwitchBlock   *big.Int             `json:"switchBlock"`
	CurrentConfig *V2Config            `json:"config"`
	AllConfigs    map[uint64]*V2Config `json:"allConfigs"`
	configIndex   []uint64             //list of switch block of configs

	SkipV2Validation bool //Skip Block Validation for testing purpose, V2 consensus only
}

type V2Config struct {
	MaxMasternodes       int     `json:"maxMasternodes"`       // v2 max masternodes
	MaxProtectorNodes    int     `json:"maxProtectorNodes"`    // v2 max ProtectorNodes
	SwitchRound          uint64  `json:"switchRound"`          // v1 to v2 switch block number
	MinePeriod           int     `json:"minePeriod"`           // Miner mine period to mine a block
	TimeoutSyncThreshold int     `json:"timeoutSyncThreshold"` // send syncInfo after number of timeout
	TimeoutPeriod        int     `json:"timeoutPeriod"`        // Duration in ms
	CertThreshold        float64 `json:"certificateThreshold"` // Necessary number of messages from master nodes to form a certificate

	MasternodeReward uint64 `json:"masternodeReward"` // Block reward for master nodes (core validators) - unit Ether
	ProtectorReward  uint64 `json:"protectorReward"`  // Block reward for protectors - unit Ether
	ObserverReward   uint64 `json:"observerReward"`   // Block reward for observer - unit Ether

	ExpTimeoutConfig ExpTimeoutConfig `json:"expTimeoutConfig"`
}

type ExpTimeoutConfig struct {
	Base        float64 `json:"base"`        // base in base^exponent
	MaxExponent uint8   `json:"maxExponent"` // max exponent in base^exponent
}

func (c *XDPoSConfig) String() string {
	return "XDPoS"
}

func (c *XDPoSConfig) Description(indent int) string {
	if c == nil {
		return "XDPoS: <nil>"
	}

	banner := "XDPoS\n"
	prefix := strings.Repeat(" ", indent)
	banner += fmt.Sprintf("%s- Period: %v\n", prefix, c.Period)
	banner += fmt.Sprintf("%s- Epoch: %v\n", prefix, c.Epoch)
	banner += fmt.Sprintf("%s- Reward: %v\n", prefix, c.Reward)
	banner += fmt.Sprintf("%s- RewardCheckpoint: %v\n", prefix, c.RewardCheckpoint)
	banner += fmt.Sprintf("%s- Gap: %v\n", prefix, c.Gap)
	banner += fmt.Sprintf("%s- FoudationWalletAddr: %v\n", prefix, c.FoudationWalletAddr.Hex())
	banner += fmt.Sprintf("%s- SkipV1Validation: %v\n", prefix, c.SkipV1Validation)
	banner += fmt.Sprintf("%s- %s", prefix, c.V2.Description(indent+2))
	return banner
}

func (v2 *V2) Description(indent int) string {
	if v2 == nil {
		return "V2: <nil>"
	}

	banner := "V2:\n"
	prefix := strings.Repeat(" ", indent)
	banner += fmt.Sprintf("%s- SwitchEpoch: %v\n", prefix, v2.SwitchEpoch)
	banner += fmt.Sprintf("%s- SwitchBlock: %v\n", prefix, v2.SwitchBlock)
	banner += fmt.Sprintf("%s- SkipV2Validation: %v\n", prefix, v2.SkipV2Validation)
	banner += fmt.Sprintf("%s- %s", prefix, v2.CurrentConfig.Description("CurrentConfig", indent+2))
	return banner
}

func (c *V2Config) Description(name string, indent int) string {
	if c == nil {
		return name + ": <nil>"
	}

	banner := name + ":\n"
	prefix := strings.Repeat(" ", indent)
	banner += fmt.Sprintf("%s- MaxMasternodes: %v\n", prefix, c.MaxMasternodes)
	banner += fmt.Sprintf("%s- SwitchRound: %v\n", prefix, c.SwitchRound)
	banner += fmt.Sprintf("%s- MinePeriod: %v\n", prefix, c.MinePeriod)
	banner += fmt.Sprintf("%s- TimeoutSyncThreshold: %v\n", prefix, c.TimeoutSyncThreshold)
	banner += fmt.Sprintf("%s- TimeoutPeriod: %v\n", prefix, c.TimeoutPeriod)
	banner += fmt.Sprintf("%s- CertThreshold: %v", prefix, c.CertThreshold)
	return banner
}

func (c *XDPoSConfig) BlockConsensusVersion(num *big.Int, extraByte []byte, extraCheck bool) string {
	if c.V2 != nil && c.V2.SwitchBlock != nil && num.Cmp(c.V2.SwitchBlock) > 0 {
		return ConsensusEngineVersion2
	}
	return ConsensusEngineVersion1
}

func (v *V2) UpdateConfig(round uint64) {
	v.lock.Lock()
	defer v.lock.Unlock()

	var index uint64

	//find the right config
	for i := range v.configIndex {
		if v.configIndex[i] <= round {
			index = v.configIndex[i]
			break
		}
	}
	// update to current config
	log.Info("[updateV2Config] Update config", "index", index, "round", round, "SwitchRound", v.AllConfigs[index].SwitchRound)
	v.CurrentConfig = v.AllConfigs[index]
}

func (v *V2) Config(round uint64) *V2Config {
	configRound := round
	var index uint64

	//find the right config
	for i := range v.configIndex {
		if v.configIndex[i] <= configRound {
			index = v.configIndex[i]
			break
		}
	}
	return v.AllConfigs[index]
}

func (v *V2) BuildConfigIndex() {
	var list []uint64

	for i := range v.AllConfigs {
		list = append(list, i)
	}

	// sort, sort lib doesn't support type uint64, it's ok to have O(n^2)  because only few items in the list
	// Make it descending order
	for i := 0; i < len(list)-1; i++ {
		for j := i + 1; j < len(list); j++ {
			if list[i] < list[j] {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	log.Info("[BuildConfigIndex] config list", "list", list)
	v.configIndex = list
}

func (v *V2) ConfigIndex() []uint64 {
	return v.configIndex
}

// Description returns a human-readable description of ChainConfig.
func (c *ChainConfig) Description() string {
	var engine interface{}
	switch {
	case c.Ethash != nil:
		engine = c.Ethash
	case c.XDPoS != nil:
		engine = c.XDPoS.Description(4)
	default:
		engine = "unknown"
	}
	berlinBlock := common.BerlinBlock
	if c.BerlinBlock != nil {
		berlinBlock = c.BerlinBlock
	}
	londonBlock := common.LondonBlock
	if c.LondonBlock != nil {
		londonBlock = c.LondonBlock
	}
	mergeBlock := common.MergeBlock
	if c.MergeBlock != nil {
		mergeBlock = c.MergeBlock
	}
	shanghaiBlock := common.ShanghaiBlock
	if c.ShanghaiBlock != nil {
		shanghaiBlock = c.ShanghaiBlock
	}
	eip1559Block := common.Eip1559Block
	if c.Eip1559Block != nil {
		eip1559Block = c.Eip1559Block
	}
	cancunBlock := common.CancunBlock
	if c.CancunBlock != nil {
		cancunBlock = c.CancunBlock
	}

	var banner = "Chain configuration:\n"
	banner += fmt.Sprintf("  - ChainID:                     %-8v\n", c.ChainId)
	banner += fmt.Sprintf("  - Homestead:                   %-8v\n", c.HomesteadBlock)
	banner += fmt.Sprintf("  - DAO Fork:                    %-8v\n", c.DAOForkBlock)
	banner += fmt.Sprintf("  - DAO Support:                 %-8v\n", c.DAOForkSupport)
	banner += fmt.Sprintf("  - Tangerine Whistle (EIP 150): %-8v\n", c.EIP150Block)
	banner += fmt.Sprintf("  - Spurious Dragon (EIP 155):   %-8v\n", c.EIP155Block)
	banner += fmt.Sprintf("  - Byzantium:                   %-8v\n", c.ByzantiumBlock)
	banner += fmt.Sprintf("  - Constantinople:              %-8v\n", c.ConstantinopleBlock)
	banner += fmt.Sprintf("  - Petersburg:                  %-8v\n", c.PetersburgBlock)
	banner += fmt.Sprintf("  - Istanbul:                    %-8v\n", c.IstanbulBlock)
	banner += fmt.Sprintf("  - Berlin:                      %-8v\n", berlinBlock)
	banner += fmt.Sprintf("  - London:                      %-8v\n", londonBlock)
	banner += fmt.Sprintf("  - Merge:                       %-8v\n", mergeBlock)
	banner += fmt.Sprintf("  - Shanghai:                    %-8v\n", shanghaiBlock)
	banner += fmt.Sprintf("  - Eip1559:                     %-8v\n", eip1559Block)
	banner += fmt.Sprintf("  - Cancun:                      %-8v\n", cancunBlock)
	banner += fmt.Sprintf("  - Engine:                      %v", engine)
	return banner
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	return isForked(c.HomesteadBlock, num)
}

// IsDAO returns whether num is either equal to the DAO fork block or greater.
func (c *ChainConfig) IsDAOFork(num *big.Int) bool {
	return isForked(c.DAOForkBlock, num)
}

func (c *ChainConfig) IsEIP150(num *big.Int) bool {
	return isForked(c.EIP150Block, num)
}

func (c *ChainConfig) IsEIP155(num *big.Int) bool {
	return isForked(c.EIP155Block, num)
}

func (c *ChainConfig) IsEIP158(num *big.Int) bool {
	return isForked(c.EIP158Block, num)
}

func (c *ChainConfig) IsByzantium(num *big.Int) bool {
	return isForked(c.ByzantiumBlock, num)
}

func (c *ChainConfig) IsConstantinople(num *big.Int) bool {
	return isForked(c.ConstantinopleBlock, num)
}

// IsPetersburg returns whether num is either
// - equal to or greater than the PetersburgBlock fork block,
// - OR is nil, and Constantinople is active
func (c *ChainConfig) IsPetersburg(num *big.Int) bool {
	return isForked(common.TIPXDCXCancellationFee, num) || isForked(c.PetersburgBlock, num)
}

// IsIstanbul returns whether num is either equal to the Istanbul fork block or greater.
func (c *ChainConfig) IsIstanbul(num *big.Int) bool {
	return isForked(common.TIPXDCXCancellationFee, num) || isForked(c.IstanbulBlock, num)
}

// IsBerlin returns whether num is either equal to the Berlin fork block or greater.
func (c *ChainConfig) IsBerlin(num *big.Int) bool {
	return isForked(common.BerlinBlock, num) || isForked(c.BerlinBlock, num)
}

// IsLondon returns whether num is either equal to the London fork block or greater.
func (c *ChainConfig) IsLondon(num *big.Int) bool {
	return isForked(common.LondonBlock, num) || isForked(c.LondonBlock, num)
}

// IsMerge returns whether num is either equal to the Merge fork block or greater.
// Different from Geth which uses `block.difficulty != nil`
func (c *ChainConfig) IsMerge(num *big.Int) bool {
	return isForked(common.MergeBlock, num) || isForked(c.MergeBlock, num)
}

// IsShanghai returns whether num is either equal to the Shanghai fork block or greater.
func (c *ChainConfig) IsShanghai(num *big.Int) bool {
	return isForked(common.ShanghaiBlock, num) || isForked(c.ShanghaiBlock, num)
}

func (c *ChainConfig) IsEIP1559(num *big.Int) bool {
	return isForked(common.Eip1559Block, num) || isForked(c.Eip1559Block, num)
}

func (c *ChainConfig) IsCancun(num *big.Int) bool {
	return isForked(common.CancunBlock, num) || isForked(c.CancunBlock, num)
}

func (c *ChainConfig) IsTIP2019(num *big.Int) bool {
	return isForked(common.TIP2019Block, num)
}

func (c *ChainConfig) IsTIPSigning(num *big.Int) bool {
	return isForked(common.TIPSigning, num)
}

func (c *ChainConfig) IsTIPRandomize(num *big.Int) bool {
	return isForked(common.TIPRandomize, num)
}

// IsTIPIncreaseMasternodes using for increase masternodes from 18 to 40

// Time update: 23-07-2019
func (c *ChainConfig) IsTIPIncreaseMasternodes(num *big.Int) bool {
	return isForked(common.TIPIncreaseMasternodes, num)
}

func (c *ChainConfig) IsTIPNoHalvingMNReward(num *big.Int) bool {
	return isForked(common.TIPNoHalvingMNReward, num)
}
func (c *ChainConfig) IsTIPXDCX(num *big.Int) bool {
	return isForked(common.TIPXDCX, num)
}
func (c *ChainConfig) IsTIPXDCXMiner(num *big.Int) bool {
	return isForked(common.TIPXDCX, num) && !isForked(common.TIPXDCXMinerDisable, num)
}

func (c *ChainConfig) IsTIPXDCXReceiver(num *big.Int) bool {
	return isForked(common.TIPXDCX, num) && !isForked(common.TIPXDCXReceiverDisable, num)
}

func (c *ChainConfig) IsXDCxDisable(num *big.Int) bool {
	return isForked(common.TIPXDCXMinerDisable, num)
}

func (c *ChainConfig) IsTIPXDCXLending(num *big.Int) bool {
	return isForked(common.TIPXDCXLending, num)
}

func (c *ChainConfig) IsTIPXDCXCancellationFee(num *big.Int) bool {
	return isForked(common.TIPXDCXCancellationFee, num)
}

func (c *ChainConfig) IsTIPUpgradeReward(num *big.Int) bool {
	return isForked(common.TIPUpgradeReward, num)
}

// GasTable returns the gas table corresponding to the current phase (homestead or homestead reprice).
//
// The returned GasTable's fields shouldn't, under any circumstances, be changed.
func (c *ChainConfig) GasTable(num *big.Int) GasTable {
	if num == nil {
		return GasTableHomestead
	}
	switch {
	case c.IsEIP158(num):
		return GasTableEIP158
	case c.IsEIP150(num):
		return GasTableEIP150
	default:
		return GasTableHomestead
	}
}

// CheckCompatible checks whether scheduled fork transitions have been imported
// with a mismatching chain configuration.
func (c *ChainConfig) CheckCompatible(newcfg *ChainConfig, height uint64) *ConfigCompatError {
	bhead := new(big.Int).SetUint64(height)

	// Iterate checkCompatible to find the lowest conflict.
	var lasterr *ConfigCompatError
	for {
		err := c.checkCompatible(newcfg, bhead)
		if err == nil || (lasterr != nil && err.RewindTo == lasterr.RewindTo) {
			break
		}
		lasterr = err
		bhead.SetUint64(err.RewindTo)
	}
	return lasterr
}

func (c *ChainConfig) checkCompatible(newcfg *ChainConfig, head *big.Int) *ConfigCompatError {
	if isForkIncompatible(c.HomesteadBlock, newcfg.HomesteadBlock, head) {
		return newCompatError("Homestead fork block", c.HomesteadBlock, newcfg.HomesteadBlock)
	}
	if isForkIncompatible(c.DAOForkBlock, newcfg.DAOForkBlock, head) {
		return newCompatError("DAO fork block", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if c.IsDAOFork(head) && c.DAOForkSupport != newcfg.DAOForkSupport {
		return newCompatError("DAO fork support flag", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if isForkIncompatible(c.EIP150Block, newcfg.EIP150Block, head) {
		return newCompatError("EIP150 fork block", c.EIP150Block, newcfg.EIP150Block)
	}
	if isForkIncompatible(c.EIP155Block, newcfg.EIP155Block, head) {
		return newCompatError("EIP155 fork block", c.EIP155Block, newcfg.EIP155Block)
	}
	if isForkIncompatible(c.EIP158Block, newcfg.EIP158Block, head) {
		return newCompatError("EIP158 fork block", c.EIP158Block, newcfg.EIP158Block)
	}
	if c.IsEIP158(head) && !configNumEqual(c.ChainId, newcfg.ChainId) {
		return newCompatError("EIP158 chain ID", c.EIP158Block, newcfg.EIP158Block)
	}
	if isForkIncompatible(c.ByzantiumBlock, newcfg.ByzantiumBlock, head) {
		return newCompatError("Byzantium fork block", c.ByzantiumBlock, newcfg.ByzantiumBlock)
	}
	if isForkIncompatible(c.ConstantinopleBlock, newcfg.ConstantinopleBlock, head) {
		return newCompatError("Constantinople fork block", c.ConstantinopleBlock, newcfg.ConstantinopleBlock)
	}
	if isForkIncompatible(c.PetersburgBlock, newcfg.PetersburgBlock, head) {
		// the only case where we allow Petersburg to be set in the past is if it is equal to Constantinople
		// mainly to satisfy fork ordering requirements which state that Petersburg fork be set if Constantinople fork is set
		if isForkIncompatible(c.ConstantinopleBlock, newcfg.PetersburgBlock, head) {
			return newCompatError("Petersburg fork block", c.PetersburgBlock, newcfg.PetersburgBlock)
		}
	}
	if isForkIncompatible(c.IstanbulBlock, newcfg.IstanbulBlock, head) {
		return newCompatError("Istanbul fork block", c.IstanbulBlock, newcfg.IstanbulBlock)
	}
	if isForkIncompatible(c.BerlinBlock, newcfg.BerlinBlock, head) {
		return newCompatError("Berlin fork block", c.BerlinBlock, newcfg.BerlinBlock)
	}
	if isForkIncompatible(c.LondonBlock, newcfg.LondonBlock, head) {
		return newCompatError("London fork block", c.LondonBlock, newcfg.LondonBlock)
	}
	if isForkIncompatible(c.ShanghaiBlock, newcfg.ShanghaiBlock, head) {
		return newCompatError("Shanghai fork timestamp", c.ShanghaiBlock, newcfg.ShanghaiBlock)
	}
	if isForkIncompatible(c.CancunBlock, newcfg.CancunBlock, head) {
		return newCompatError("Cancun fork block", c.CancunBlock, newcfg.CancunBlock)
	}
	return nil
}

// isForkIncompatible returns true if a fork scheduled at s1 cannot be rescheduled to
// block s2 because head is already past the fork.
func isForkIncompatible(s1, s2, head *big.Int) bool {
	return (isForked(s1, head) || isForked(s2, head)) && !configNumEqual(s1, s2)
}

// isForked returns whether a fork scheduled at block s is active at the given head block.
func isForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}

func configNumEqual(x, y *big.Int) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return x.Cmp(y) == 0
}

// ConfigCompatError is raised if the locally-stored blockchain is initialised with a
// ChainConfig that would alter the past.
type ConfigCompatError struct {
	What string
	// block numbers of the stored and new configurations
	StoredConfig, NewConfig *big.Int
	// the block number to which the local chain must be rewound to correct the error
	RewindTo uint64
}

func newCompatError(what string, storedblock, newblock *big.Int) *ConfigCompatError {
	var rew *big.Int
	switch {
	case storedblock == nil:
		rew = newblock
	case newblock == nil || storedblock.Cmp(newblock) < 0:
		rew = storedblock
	default:
		rew = newblock
	}
	err := &ConfigCompatError{what, storedblock, newblock, 0}
	if rew != nil && rew.Sign() > 0 {
		err.RewindTo = rew.Uint64() - 1
	}
	return err
}

func (err *ConfigCompatError) Error() string {
	return fmt.Sprintf("mismatching %s in database (have %d, want %d, rewindto %d)", err.What, err.StoredConfig, err.NewConfig, err.RewindTo)
}

// Rules wraps ChainConfig and is merely syntatic sugar or can be used for functions
// that do not have or require information about the block.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
type Rules struct {
	ChainId                                                 *big.Int
	IsHomestead, IsEIP150, IsEIP155, IsEIP158               bool
	IsByzantium, IsConstantinople, IsPetersburg, IsIstanbul bool
	IsBerlin, IsLondon                                      bool
	IsMerge, IsShanghai                                     bool
	IsXDCxDisable                                           bool
	IsEIP1559                                               bool
	IsCancun                                                bool
}

func (c *ChainConfig) Rules(num *big.Int) Rules {
	chainId := c.ChainId
	if chainId == nil {
		chainId = new(big.Int)
	}
	return Rules{
		ChainId:          new(big.Int).Set(chainId),
		IsHomestead:      c.IsHomestead(num),
		IsEIP150:         c.IsEIP150(num),
		IsEIP155:         c.IsEIP155(num),
		IsEIP158:         c.IsEIP158(num),
		IsByzantium:      c.IsByzantium(num),
		IsConstantinople: c.IsConstantinople(num),
		IsPetersburg:     c.IsPetersburg(num),
		IsIstanbul:       c.IsIstanbul(num),
		IsBerlin:         c.IsBerlin(num),
		IsLondon:         c.IsLondon(num),
		IsMerge:          c.IsMerge(num),
		IsShanghai:       c.IsShanghai(num),
		IsXDCxDisable:    c.IsXDCxDisable(num),
		IsEIP1559:        c.IsEIP1559(num),
		IsCancun:         c.IsCancun(num),
	}
}

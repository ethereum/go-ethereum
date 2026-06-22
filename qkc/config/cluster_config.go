// Ported verbatim from github.com/QuarkChain/goquarkchain/cluster/config (byte-compatible).

package config

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"

	ethcom "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/qkc/account"
	"github.com/ethereum/go-ethereum/qkc/common"
)

var (
	slavePort uint16 = 38000
)

type ClusterConfig struct {
	P2PPort                  uint16            `json:"P2P_PORT"`
	JSONRPCPort              uint16            `json:"JSON_RPC_PORT"`
	JSONRPCHOST              string            `json:"JSON_RPC_HOST"`
	PrivateJSONRPCPort       uint16            `json:"PRIVATE_JSON_RPC_PORT"`
	PrivateJSONRPCHOST       string            `json:"PRIVATE_JSON_RPC_HOST"`
	EnableTransactionHistory bool              `json:"ENABLE_TRANSACTION_HISTORY"`
	DbPathRoot               string            `json:"DB_PATH_ROOT"`
	LogLevel                 string            `json:"LOG_LEVEL"`
	StartSimulatedMining     bool              `json:"START_SIMULATED_MINING"`
	Clean                    bool              `json:"CLEAN"`
	GenesisDir               string            `json:"GENESIS_DIR"`
	Quarkchain               *QuarkChainConfig `json:"QUARKCHAIN"`
	Master                   *MasterConfig     `json:"MASTER"`
	SlaveList                []*SlaveConfig    `json:"SLAVE_LIST"`
	SimpleNetwork            *SimpleNetwork    `json:"SIMPLE_NETWORK,omitempty"`
	P2P                      *P2PConfig        `json:"P2P,omitempty"`
	Monitoring               *MonitoringConfig `json:"MONITORING"`
	CheckDB                  bool
	CheckDBRBlockFrom        int
	CheckDBRBlockTo          int
	CheckDBRBlockBatch       int
	NoPruning                bool
}

func NewClusterConfig() *ClusterConfig {
	var ret = ClusterConfig{
		P2PPort:                  DefaultP2PPort,
		JSONRPCPort:              DefaultPubRpcPort,
		JSONRPCHOST:              "0.0.0.0",
		PrivateJSONRPCPort:       DefaultPrivRpcPort,
		PrivateJSONRPCHOST:       DefaultHost,
		EnableTransactionHistory: false,
		DbPathRoot:               "./db",
		LogLevel:                 "info",
		StartSimulatedMining:     false,
		Clean:                    false,
		GenesisDir:               "../genesis_data",
		Quarkchain:               NewQuarkChainConfig(),
		Master:                   NewMasterConfig(),
		SimpleNetwork:            NewSimpleNetwork(),
		P2P:                      NewP2PConfig(),
		Monitoring:               NewMonitoringConfig(),
		CheckDB:                  false,
		CheckDBRBlockFrom:        -1,
		CheckDBRBlockTo:          0,
		CheckDBRBlockBatch:       10,
	}

	fullShardIds := ret.Quarkchain.GetGenesisShardIds()

	for i := 0; i < DefaultNumSlaves; i++ {
		slave := NewDefaultSlaveConfig()
		slave.Port = slavePort + uint16(i)
		slave.ID = fmt.Sprintf("S%d", i)
		slave.FullShardList = append(slave.FullShardList, getFullShardIdListFromSlaveIndex(fullShardIds, DefaultNumSlaves, i)...)
		ret.SlaveList = append(ret.SlaveList, slave)
	}
	return &ret
}

func getFullShardIdListFromSlaveIndex(list []uint32, slaveNumber int, slaveIndex int) []uint32 {
	ans := make([]uint32, 0)
	for index := 0; index < len(list); index++ {
		if index%slaveNumber == slaveIndex {
			ans = append(ans, list[index])
		}
	}
	return ans
}

func (c *ClusterConfig) GetSlaveConfig(id string) (*SlaveConfig, error) {
	if c.SlaveList == nil {
		return nil, errors.New("slave config is empty")
	}
	for _, slave := range c.SlaveList {
		if slave != nil && slave.ID == id {
			return slave, nil
		}
	}
	return nil, fmt.Errorf("slave %s is not in cluster config", id)
}

func (c *ClusterConfig) BackWardChainMaskList() error {
	setFullShardFromChainMask := true
	for _, slave := range c.SlaveList {
		for _, vv := range slave.FullShardList {
			if vv != 0 {
				setFullShardFromChainMask = false
			}
		}
	}
	if !setFullShardFromChainMask {
		return nil
	}
	for _, v := range c.Quarkchain.Chains {
		if v.ShardSize != 1 {
			return errors.New("CHAIN_MASK_LIST:only works if every chain has 1 shard only")
		}
	}
	for _, v := range c.SlaveList {
		for _, m := range v.ChainMaskListForBackward {
			bitMask := uint32(1<<(common.IntLeftMostBit(m)-1) - 1)
			v.FullShardList = make([]uint32, 0)
			for _, chainConfig := range c.Quarkchain.Chains {
				if chainConfig.ChainID&bitMask == m&bitMask {
					v.FullShardList = append(v.FullShardList, chainConfig.ChainID<<16+1)
				}
			}
		}
		sort.Sort(fullShardList(v.FullShardList))
	}
	return nil
}

type fullShardList []uint32

func (a fullShardList) Len() int           { return len(a) }
func (a fullShardList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a fullShardList) Less(i, j int) bool { return a[i] < a[j] }

type QuarkChainConfig struct {
	ChainSize                             uint32      `json:"CHAIN_SIZE"`
	MaxNeighbors                          uint32      `json:"MAX_NEIGHBORS"`
	NetworkID                             uint32      `json:"NETWORK_ID"`
	TransactionQueueSizeLimitPerShard     uint64      `json:"TRANSACTION_QUEUE_SIZE_LIMIT_PER_SHARD"`
	BlockExtraDataSizeLimit               uint32      `json:"BLOCK_EXTRA_DATA_SIZE_LIMIT"`
	GuardianPublicKey                     []byte      `json:"-"`
	RootSignerPrivateKey                  []byte      `json:"-"`
	P2PProtocolVersion                    uint32      `json:"P2P_PROTOCOL_VERSION"`
	P2PCommandSizeLimit                   uint32      `json:"P2P_COMMAND_SIZE_LIMIT"`
	SkipRootDifficultyCheck               bool        `json:"SKIP_ROOT_DIFFICULTY_CHECK"`
	SkipRootCoinbaseCheck                 bool        `json:"SKIP_ROOT_COINBASE_CHECK"`
	SkipMinorDifficultyCheck              bool        `json:"SKIP_MINOR_DIFFICULTY_CHECK"`
	GenesisToken                          string      `json:"GENESIS_TOKEN"`
	Root                                  *RootConfig `json:"ROOT"`
	shards                                map[uint32]*ShardConfig
	Chains                                map[uint32]*ChainConfig `json:"-"`
	RewardTaxRate                         *big.Rat                `json:"-"`
	LocalFeeRate                          *big.Rat                `json:"-"`
	RewardCalculateRate                   *big.Rat                `json:"-"`
	BlockRewardDecayFactor                *big.Rat                `json:"-"`
	chainIdToShardSize                    map[uint32]uint32
	chainIdToShardIds                     map[uint32][]uint32
	defaultChainTokenID                   uint64
	EnableEvmTimeStamp                    uint64      `json:"ENABLE_EVM_TIMESTAMP"`
	EnableQkcHashXHeight                  uint64      `json:"ENABLE_QKCHASHX_HEIGHT"`
	EnableNonReservedNativeTokenTimestamp uint64      `json:"ENABLE_NON_RESERVED_NATIVE_TOKEN_TIMESTAMP"`
	EnableGeneralNativeTokenTimestamp     uint64      `json:"ENABLE_GENERAL_NATIVE_TOKEN_TIMESTAMP"`
	EnablePoswStakingDecayTimestamp       uint64      `json:"ENABLE_POSW_STAKING_DECAY_TIMESTAMP"`
	EnableEIP155SignerTimestamp           uint64      `json:"ENABLE_EIP155_SIGNER_TIMESTAMP"`
	BaseEthChainID                        uint32      `json:"BASE_ETH_CHAIN_ID"`
	DisablePowCheck                       bool        `json:"DISABLE_POW_CHECK"`
	XShardGasDDOSFixRootHeight            uint64      `json:"XSHARD_GAS_DDOS_FIX_ROOT_HEIGHT"`
	MinTXPoolGasPrice                     *big.Int    `json:"MIN_TX_POOL_GAS_PRICE"`
	MinMiningGasPrice                     *big.Int    `json:"MIN_MINING_GAS_PRICE"`
	RootChainPoSWContractBytecodeHash     ethcom.Hash `json:"-"`
}

type QuarkChainConfigAlias QuarkChainConfig
type jsonConfig struct {
	QuarkChainConfigAlias
	GuardianPublicKey                 string         `json:"GUARDIAN_PUBLIC_KEY"`
	RootSignerPrivateKey              string         `json:"ROOT_SIGNER_PRIVATE_KEY"`
	Chains                            []*ChainConfig `json:"CHAINS"`
	RewardTaxRate                     float64        `json:"REWARD_TAX_RATE"`
	BlockRewardDecayFactor            float64        `json:"BLOCK_REWARD_DECAY_FACTOR"`
	RootChainPoSWContractBytecodeHash string         `json:"ROOT_CHAIN_POSW_CONTRACT_BYTECODE_HASH"`
}

func (q *QuarkChainConfig) MarshalJSON() ([]byte, error) {
	rewardTaxRate, _ := q.RewardTaxRate.Float64()
	BlockRewardDecayFactor, _ := q.BlockRewardDecayFactor.Float64()
	chains := make([]*ChainConfig, 0, len(q.Chains))
	rootChainPoSWContractBytecodeHash := ethcom.Bytes2Hex(q.RootChainPoSWContractBytecodeHash[:])
	for _, chain := range q.Chains {
		chains = append(chains, chain)
	}
	jConfig := jsonConfig{
		QuarkChainConfigAlias(*q),
		hex.EncodeToString(q.GuardianPublicKey),
		hex.EncodeToString(q.RootSignerPrivateKey),
		chains,
		rewardTaxRate,
		BlockRewardDecayFactor,
		rootChainPoSWContractBytecodeHash,
	}
	return json.Marshal(jConfig)
}

func (q *QuarkChainConfig) UnmarshalJSON(input []byte) error {
	jConfig := &jsonConfig{}
	if err := json.Unmarshal(input, jConfig); err != nil {
		return err
	}
	sort.Slice(jConfig.Chains, func(i, j int) bool { return jConfig.Chains[i].ChainID < jConfig.Chains[j].ChainID })

	*q = QuarkChainConfig(jConfig.QuarkChainConfigAlias)
	q.Chains = make(map[uint32]*ChainConfig)
	q.shards = make(map[uint32]*ShardConfig)
	for _, chainCfg := range jConfig.Chains {
		q.Chains[chainCfg.ChainID] = chainCfg
		for shardID := uint32(0); shardID < chainCfg.ShardSize; shardID++ {
			var cfg = new(ChainConfig)
			_ = common.DeepCopy(cfg, chainCfg)
			shardCfg := NewShardConfig(cfg)
			shardCfg.SetRootConfig(q.Root)
			shardCfg.ShardID = shardID
			// Follow pyquarkchain (ClusterConfig.from_dict): the per-shard config is
			// derived from the chain config by (1) rewriting the coinbase into this
			// shard and (2) filtering GENESIS.ALLOC down to the addresses that belong
			// to this shard. goquarkchain's UnmarshalJSON does neither (it copies the
			// chain coinbase and the full alloc verbatim), but the goshard slave shares
			// its config with a pyquarkchain master, so the per-shard config must be
			// derived exactly as pyquarkchain derives it.
			shardCfg.CoinbaseAddress = chainCfg.CoinbaseAddress.AddressInShard(shardCfg.GetFullShardId())
			if chainCfg.Genesis != nil && shardCfg.Genesis != nil {
				alloc := make(map[account.Address]Allocation)
				for addr, allocation := range chainCfg.Genesis.Alloc {
					if addr.FullShardKey&(chainCfg.ShardSize-1) == shardID {
						alloc[addr] = allocation
					}
				}
				shardCfg.Genesis.Alloc = alloc
			}
			q.shards[shardCfg.GetFullShardId()] = shardCfg
		}
	}
	var denom int64 = 1000
	q.RewardTaxRate = big.NewRat(int64(jConfig.RewardTaxRate*float64(denom)), denom)
	one := big.NewRat(1, 1)
	q.LocalFeeRate = one.Sub(one, q.RewardTaxRate)
	q.RewardCalculateRate = new(big.Rat).Quo(q.RewardTaxRate, q.LocalFeeRate)
	q.BlockRewardDecayFactor = big.NewRat(int64(jConfig.BlockRewardDecayFactor*float64(denom)), denom)
	q.RootChainPoSWContractBytecodeHash = ethcom.HexToHash(jConfig.RootChainPoSWContractBytecodeHash)

	q.GuardianPublicKey = ethcom.FromHex(jConfig.GuardianPublicKey)
	q.RootSignerPrivateKey = ethcom.FromHex(jConfig.RootSignerPrivateKey)
	if len(q.GuardianPublicKey) == 64 {
		q.GuardianPublicKey = append([]byte{byte(0x4)}, q.GuardianPublicKey...)
	}
	q.initAndValidate()
	return nil
}

// Return the root block height at which the shard shall be created
func (q *QuarkChainConfig) GetGenesisRootHeight(fullShardId uint32) uint32 {
	return q.shards[fullShardId].Genesis.RootHeight
}

// GetGenesisShardIds returns a list of ids for shards that have GENESIS.
func (q *QuarkChainConfig) GetGenesisShardIds() []uint32 {
	var result []uint32
	for shardID := range q.shards {
		result = append(result, shardID)
	}
	return result
}

// Return a list of ids of the shards that have been initialized before a certain root height
func (q *QuarkChainConfig) GetInitializedShardIdsBeforeRootHeight(rootHeight uint32) []uint32 {
	var result []uint32
	for fullShardId, config := range q.shards {
		if config.Genesis != nil && config.Genesis.RootHeight < rootHeight {
			result = append(result, uint32(fullShardId))
		}
	}
	return result
}

func (q *QuarkChainConfig) Update(chainSize, shardSizePerChain, rootBlockTime, minorBlockTime uint32) {
	q.ChainSize = chainSize
	if q.Root == nil {
		q.Root = NewRootConfig()
	}
	q.Root.ConsensusType = PoWSimulate
	if q.Root.ConsensusConfig == nil {
		q.Root.ConsensusConfig = NewPOWConfig()
	}
	q.Root.ConsensusConfig.TargetBlockTime = rootBlockTime

	q.Chains = make(map[uint32]*ChainConfig)
	q.shards = make(map[uint32]*ShardConfig)
	for chainId := uint32(0); chainId < chainSize; chainId++ {
		chainCfg := NewChainConfig()
		chainCfg.ChainID = chainId
		chainCfg.ShardSize = shardSizePerChain
		chainCfg.ConsensusType = PoWSimulate
		chainCfg.ConsensusConfig = NewPOWConfig()
		chainCfg.ConsensusConfig.TargetBlockTime = minorBlockTime
		chainCfg.DefaultChainToken = DefaultToken
		q.Chains[chainId] = chainCfg
		for shardId := uint32(0); shardId < shardSizePerChain; shardId++ {
			var cfg = new(ChainConfig)
			_ = common.DeepCopy(cfg, chainCfg)
			shardCfg := NewShardConfig(cfg)
			shardCfg.SetRootConfig(q.Root)
			shardCfg.ShardID = shardId
			// shardCfg.CoinbaseAddress = account.CreatEmptyAddress(shardCfg.GetFullShardId())
			q.shards[shardCfg.GetFullShardId()] = shardCfg
		}
	}
	q.initAndValidate()
}

func (q *QuarkChainConfig) initAndValidate() {
	if q.MinMiningGasPrice == nil {
		q.MinMiningGasPrice = new(big.Int).SetUint64(1000000000)
	}
	if q.MinTXPoolGasPrice == nil {
		q.MinTXPoolGasPrice = new(big.Int).SetUint64(1000000000)
	}
	if q.XShardGasDDOSFixRootHeight == 0 {
		q.XShardGasDDOSFixRootHeight = 90000
	}
	if len(q.GuardianPublicKey) != 65 && len(q.GuardianPublicKey) != 0 {
		fmt.Println("len", len(q.GuardianPublicKey))
		panic("GuardianPublicKey should 0 or 65")
	}
	q.chainIdToShardSize = make(map[uint32]uint32)
	q.chainIdToShardIds = make(map[uint32][]uint32)

	for fullShardId, shardCfg := range q.shards {
		chainID := shardCfg.ChainID
		shardSize := shardCfg.ShardSize
		shardID := shardCfg.ShardID
		realID := (chainID << 16) | shardSize | shardID

		if fullShardId != realID {
			panic(fmt.Sprintf("full_shard_id is not right, target=%d, actual=%d", realID, fullShardId))
		} else {
			q.chainIdToShardSize[chainID] = shardSize
		}
		if q.chainIdToShardIds[chainID] == nil {
			q.chainIdToShardIds[chainID] = make([]uint32, 0)
		}
		q.chainIdToShardIds[chainID] = append(q.chainIdToShardIds[chainID], shardID)
	}
	chainIDMap := make(map[uint32]uint32)
	for chainID, shardIDs := range q.chainIdToShardIds {
		chainIDMap[chainID] = chainID
		shardSize, err := q.GetShardSizeByChainId(chainID)
		if err != nil {
			panic(err)
		}
		if len(shardIDs) != int(shardSize) {
			panic(fmt.Sprintf("shard_size length is not right, target=%d, actual=%d", shardSize, len(shardIDs)))
		}
		for i := uint32(0); i < shardSize; i++ {
			exist := false
			for _, shardID := range shardIDs {
				if i == shardID {
					exist = true
					break
				}
			}
			if !exist {
				panic(fmt.Sprintf("shard ids is not right, target=%d, actual=%d", i, shardIDs[i]))
			}
		}
	}
	for i := uint32(0); i < q.ChainSize; i++ {
		if _, ok := chainIDMap[i]; !ok {
			panic(fmt.Sprintf("chain id %d is missing from the configuration", i))
		}
	}
}

func (q *QuarkChainConfig) GetShardConfigByFullShardID(fullShardID uint32) *ShardConfig {
	return q.shards[fullShardID]
}

func (q *QuarkChainConfig) IsSameFullShard(key1, key2 uint32) bool {
	id1, err := q.GetFullShardIdByFullShardKey(key1)
	if err != nil {
		return false
	}
	id2, err := q.GetFullShardIdByFullShardKey(key2)
	if err != nil {
		return false
	}
	return id1 == id2
}

func (q *QuarkChainConfig) GetFullShardIdByFullShardKey(fullShardKey uint32) (uint32, error) {
	chainID := fullShardKey >> 16
	shardSize, err := q.GetShardSizeByChainId(chainID)
	if err != nil {
		return 0, err
	}
	shardID := fullShardKey & (shardSize - 1)
	return (chainID << 16) | shardSize | shardID, nil
}

func (q *QuarkChainConfig) GetShardSizeByChainId(ID uint32) (uint32, error) {
	data, ok := q.chainIdToShardSize[ID]
	if !ok {
		return 0, errors.New("no such chainID")
	}
	return data, nil
}

func NewQuarkChainConfig() *QuarkChainConfig {
	var ret = QuarkChainConfig{
		ChainSize:                             3,
		MaxNeighbors:                          32,
		NetworkID:                             3,
		TransactionQueueSizeLimitPerShard:     10000,
		BlockExtraDataSizeLimit:               1024,
		GuardianPublicKey:                     ethcom.FromHex("04ab856abd0983a82972021e454fcf66ed5940ed595b0898bcd75cbe2d0a51a00f5358b566df22395a2a8bf6c022c1d51a2c3defe654e91a8d244947783029694d"),
		RootSignerPrivateKey:                  nil,
		P2PProtocolVersion:                    0,
		P2PCommandSizeLimit:                   DefaultP2PCmddSizeLimit,
		SkipRootDifficultyCheck:               false,
		SkipRootCoinbaseCheck:                 false,
		SkipMinorDifficultyCheck:              false,
		GenesisToken:                          DefaultToken,
		RewardTaxRate:                         new(big.Rat).SetFloat64(0.5),
		BlockRewardDecayFactor:                new(big.Rat).SetFloat64(0.5),
		Root:                                  NewRootConfig(),
		MinTXPoolGasPrice:                     new(big.Int).SetUint64(1000000000),
		MinMiningGasPrice:                     new(big.Int).SetUint64(1000000000),
		XShardGasDDOSFixRootHeight:            90000,
		EnableEvmTimeStamp:                    1569567600,
		EnableNonReservedNativeTokenTimestamp: math.MaxUint64,
		EnableGeneralNativeTokenTimestamp:     math.MaxUint64,
		RootChainPoSWContractBytecodeHash:     ethcom.HexToHash("0000000000000000000000000000000000000000000000000000000000000000"),
	}

	ret.Root.ConsensusType = PoWSimulate
	ret.Root.ConsensusConfig = NewPOWConfig()
	ret.Root.ConsensusConfig.TargetBlockTime = 10

	one := big.NewRat(1, 1)
	ret.LocalFeeRate = one.Sub(one, ret.RewardTaxRate)
	ret.RewardCalculateRate = new(big.Rat).Quo(ret.RewardTaxRate, ret.LocalFeeRate)

	ret.Chains = make(map[uint32]*ChainConfig)
	ret.shards = make(map[uint32]*ShardConfig)
	for chainID := uint32(0); chainID < ret.ChainSize; chainID++ {
		cfg := NewChainConfig()
		cfg.ChainID = chainID
		cfg.ConsensusType = PoWSimulate
		cfg.ConsensusConfig = NewPOWConfig()
		cfg.ConsensusConfig.TargetBlockTime = 3
		ret.Chains[chainID] = cfg
		for shardID := uint32(0); shardID < cfg.ShardSize; shardID++ {
			var chainCfg = new(ChainConfig)
			_ = common.DeepCopy(chainCfg, cfg)
			shardCfg := NewShardConfig(chainCfg)
			shardCfg.SetRootConfig(ret.Root)
			shardCfg.ShardID = shardID
			shardCfg.CoinbaseAddress = account.CreatEmptyAddress(shardCfg.GetFullShardId())
			ret.shards[shardCfg.GetFullShardId()] = shardCfg
		}
	}
	ret.initAndValidate()
	return &ret
}

func (q *QuarkChainConfig) SetShardsAndValidate(shards map[uint32]*ShardConfig) { // only used in gen config
	q.shards = shards
	q.initAndValidate()
}

func (q *QuarkChainConfig) GetDefaultChainTokenID() uint64 {
	if q.defaultChainTokenID == 0 {
		q.defaultChainTokenID = common.TokenIDEncode(q.GenesisToken)
	}
	return q.defaultChainTokenID
}

func (q *QuarkChainConfig) GasLimit(fullShardID uint32) (*big.Int, error) {
	data, ok := q.shards[fullShardID]
	if !ok {
		return nil, fmt.Errorf("no such fullShardID %v", fullShardID)
	}
	return new(big.Int).SetUint64(data.Genesis.GasLimit), nil
}

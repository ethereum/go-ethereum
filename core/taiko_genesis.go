package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	taikoGenesis "github.com/ethereum/go-ethereum/core/taiko_genesis"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	InternalDevnetOntakeBlock = new(big.Int).SetUint64(0)
	PreconfDevnetOntakeBlock  = common.Big0
	HeklaOntakeBlock          = new(big.Int).SetUint64(840_512)
	MainnetOntakeBlock        = new(big.Int).SetUint64(538_304)

	InternalDevnetPacayaBlock = new(big.Int).SetUint64(10)
	PreconfDevnetPacayaBlock  = common.Big0
	HeklaPacayaBlock          = new(big.Int).SetUint64(1_299_888)
	MainnetPacayaBlock        = new(big.Int).SetUint64(1_166_000)
)

// TaikoGenesisBlock returns the Taiko network genesis block configs.
func TaikoGenesisBlock(networkID uint64) *Genesis {
	chainConfig := params.TaikoChainConfig

	var allocJSON []byte
	switch networkID {
	case params.TaikoMainnetNetworkID.Uint64():
		chainConfig.ChainID = params.TaikoMainnetNetworkID
		chainConfig.OntakeBlock = MainnetOntakeBlock
		chainConfig.PacayaBlock = MainnetPacayaBlock
		allocJSON = taikoGenesis.MainnetGenesisAllocJSON
	case params.TaikoInternalL2ANetworkID.Uint64():
		chainConfig.ChainID = params.TaikoInternalL2ANetworkID
		chainConfig.OntakeBlock = InternalDevnetOntakeBlock
		chainConfig.PacayaBlock = InternalDevnetPacayaBlock
		allocJSON = taikoGenesis.InternalL2AGenesisAllocJSON
	case params.TaikoInternalL2BNetworkID.Uint64():
		chainConfig.ChainID = params.TaikoInternalL2BNetworkID
		allocJSON = taikoGenesis.InternalL2BGenesisAllocJSON
	case params.SnaefellsjokullNetworkID.Uint64():
		chainConfig.ChainID = params.SnaefellsjokullNetworkID
		allocJSON = taikoGenesis.SnaefellsjokullGenesisAllocJSON
	case params.AskjaNetworkID.Uint64():
		chainConfig.ChainID = params.AskjaNetworkID
		allocJSON = taikoGenesis.AskjaGenesisAllocJSON
	case params.GrimsvotnNetworkID.Uint64():
		chainConfig.ChainID = params.GrimsvotnNetworkID
		allocJSON = taikoGenesis.GrimsvotnGenesisAllocJSON
	case params.EldfellNetworkID.Uint64():
		chainConfig.ChainID = params.EldfellNetworkID
		allocJSON = taikoGenesis.EldfellGenesisAllocJSON
	case params.JolnirNetworkID.Uint64():
		chainConfig.ChainID = params.JolnirNetworkID
		allocJSON = taikoGenesis.JolnirGenesisAllocJSON
	case params.KatlaNetworkID.Uint64():
		chainConfig.ChainID = params.KatlaNetworkID
		allocJSON = taikoGenesis.KatlaGenesisAllocJSON
	case params.HeklaNetworkID.Uint64():
		chainConfig.ChainID = params.HeklaNetworkID
		chainConfig.OntakeBlock = HeklaOntakeBlock
		chainConfig.PacayaBlock = HeklaPacayaBlock
		allocJSON = taikoGenesis.HeklaGenesisAllocJSON
	case params.PreconfDevnetNetworkID.Uint64():
		chainConfig.ChainID = params.PreconfDevnetNetworkID
		chainConfig.OntakeBlock = PreconfDevnetOntakeBlock
		chainConfig.PacayaBlock = PreconfDevnetPacayaBlock
		allocJSON = taikoGenesis.PreconfDevnetGenesisAllocJSON
	default:
		chainConfig.ChainID = params.TaikoInternalL2ANetworkID
		chainConfig.OntakeBlock = InternalDevnetOntakeBlock
		chainConfig.PacayaBlock = InternalDevnetPacayaBlock
		allocJSON = taikoGenesis.InternalL2AGenesisAllocJSON
	}

	var alloc GenesisAlloc
	if err := alloc.UnmarshalJSON(allocJSON); err != nil {
		log.Crit("unmarshal alloc json error", "error", err)
	}

	return &Genesis{
		Config:     chainConfig,
		ExtraData:  []byte{},
		GasLimit:   uint64(15_000_000),
		Difficulty: common.Big0,
		Alloc:      alloc,
		GasUsed:    0,
		BaseFee:    new(big.Int).SetUint64(10_000_000),
	}
}

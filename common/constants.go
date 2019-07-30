package common

import "math/big"

const (
	RewardMasterPercent        = 90
	RewardVoterPercent         = 0
	RewardFoundationPercent    = 10
	HexSignMethod              = "e341eaa4"
	HexSetSecret               = "34d38600"
	HexSetOpening              = "e11f5ba2"
	EpocBlockSecret            = 800
	EpocBlockOpening           = 850
	EpocBlockRandomize         = 900
	MaxMasternodes             = 18
	MaxMasternodesV2           = 108
	LimitPenaltyEpoch          = 4
	BlocksPerYear              = uint64(15768000)
	LimitThresholdNonceInQueue = 10
	DefaultMinGasPrice         = 2500
	MergeSignRange             = 15
	RangeReturnSigner          = 150
	MinimunMinerBlockPerEpoch  = 1
)

var TIP2019Block = big.NewInt(1)
var TIPSigning = big.NewInt(3000000)
var TIPRandomize = big.NewInt(3464000)
var TIPIncreaseMasternodes = big.NewInt(5000000) // Upgrade MN Count at Block.
var IsTestnet bool = false
var StoreRewardFolder string
var RollbackHash Hash
var MinGasPrice int64

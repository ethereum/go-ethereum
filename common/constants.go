package common

import "math/big"

const (
	RewardMasterPercent        = 40
	RewardVoterPercent         = 50
	RewardFoundationPercent    = 10
	HexSignMethod              = "e341eaa4"
	HexSetSecret               = "34d38600"
	HexSetOpening              = "e11f5ba2"
	EpocBlockSecret            = 800
	EpocBlockOpening           = 850
	EpocBlockRandomize         = 900
	MaxMasternodes             = 150
	LimitPenaltyEpoch          = 4
	BlocksPerYear              = uint64(15768000)
	LimitThresholdNonceInQueue = 10
	MinGasPrice                = 2500
	MergeSignRange             = 15
)

var TIP2019Block = big.NewInt(1050000)
var IsTestnet bool = false
var StoreRewardFolder string
var RollbackHash Hash

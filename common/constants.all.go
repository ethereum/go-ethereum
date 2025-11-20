package common

import (
	"maps"
	"math/big"
)

// non-const variables for all network.
var (
	IsTestnet      bool = false
	Enable0xPrefix bool = true

	RollbackNumber = uint64(0)

	StoreRewardFolder string

	TRC21GasPriceBefore = big.NewInt(2500)
	TRC21GasPrice       = big.NewInt(250000000)
	MinGasPrice         = big.NewInt(DefaultMinGasPrice)

	// XDCx and XDCxlending
	BasePrice         = big.NewInt(1000000000000000000)               // 1
	RelayerLockedFund = big.NewInt(20000)                             // 20000 XDC
	XDCXBaseFee       = big.NewInt(10000)                             // 1 / XDCXBaseFee
	XDCXBaseCancelFee = new(big.Int).Mul(XDCXBaseFee, big.NewInt(10)) // 1/ (XDCXBaseFee *10)

	// XDCx
	RelayerFee       = big.NewInt(1000000000000000) // 0.001
	RelayerCancelFee = big.NewInt(100000000000000)  // 0.0001

	// XDCxlending
	RateTopUp               = big.NewInt(90) // 90%
	BaseTopUp               = big.NewInt(100)
	BaseRecall              = big.NewInt(100)
	BaseLendingInterest     = big.NewInt(100000000)         // 1e8
	RelayerLendingFee       = big.NewInt(10000000000000000) // 0.01
	RelayerLendingCancelFee = big.NewInt(1000000000000000)  // 0.001
)

type constant struct {
	chainID           uint64
	blackListHFNumber uint64
	maxMasternodesV2  int // Last v1 masternodes

	tip2019Block           *big.Int
	tipSigning             *big.Int
	tipRandomize           *big.Int
	tipNoHalvingMNReward   *big.Int // hard fork no halving masternodes reward
	tipXDCX                *big.Int
	tipXDCXLending         *big.Int
	tipXDCXCancellationFee *big.Int
	tipTRC21Fee            *big.Int
	tipIncreaseMasternodes *big.Int // Upgrade MN Count at Block.
	berlinBlock            *big.Int
	londonBlock            *big.Int
	mergeBlock             *big.Int
	shanghaiBlock          *big.Int
	blockNumberGas50x      *big.Int
	TIPV2SwitchBlock       *big.Int
	tipXDCXMinerDisable    *big.Int
	tipXDCXReceiverDisable *big.Int
	eip1559Block           *big.Int
	cancunBlock            *big.Int

	trc21IssuerSMC         Address
	xdcxListingSMC         Address
	relayerRegistrationSMC Address
	lendingRegistrationSMC Address

	ignoreSignerCheckBlockArray map[uint64]struct{}

	blacklist map[Address]struct{}
}

// variables for specific networks, copy values from maintnet constant to pass tests
var (
	BlackListHFNumber = MainnetConstant.blackListHFNumber
	MaxMasternodesV2  = MainnetConstant.maxMasternodesV2 // Last v1 masternodes

	TIP2019Block           = MainnetConstant.tip2019Block
	TIPSigning             = MainnetConstant.tipSigning
	TIPRandomize           = MainnetConstant.tipRandomize
	TIPNoHalvingMNReward   = MainnetConstant.tipNoHalvingMNReward
	TIPXDCX                = MainnetConstant.tipXDCX
	TIPXDCXLending         = MainnetConstant.tipXDCXLending
	TIPXDCXCancellationFee = MainnetConstant.tipXDCXCancellationFee
	TIPTRC21Fee            = MainnetConstant.tipTRC21Fee
	TIPIncreaseMasternodes = MainnetConstant.tipIncreaseMasternodes
	BerlinBlock            = MainnetConstant.berlinBlock
	LondonBlock            = MainnetConstant.londonBlock
	MergeBlock             = MainnetConstant.mergeBlock
	ShanghaiBlock          = MainnetConstant.shanghaiBlock
	BlockNumberGas50x      = MainnetConstant.blockNumberGas50x
	TIPXDCXMinerDisable    = MainnetConstant.tipXDCXMinerDisable
	TIPXDCXReceiverDisable = MainnetConstant.tipXDCXReceiverDisable
	Eip1559Block           = MainnetConstant.eip1559Block
	CancunBlock            = MainnetConstant.cancunBlock

	TRC21IssuerSMC         = MainnetConstant.trc21IssuerSMC
	XDCXListingSMC         = MainnetConstant.xdcxListingSMC
	RelayerRegistrationSMC = MainnetConstant.relayerRegistrationSMC
	LendingRegistrationSMC = MainnetConstant.lendingRegistrationSMC

	ignoreSignerCheckBlockArray = MainnetConstant.ignoreSignerCheckBlockArray
	blacklist                   = MainnetConstant.blacklist
)

func IsIgnoreSignerCheckBlock(blockNumber uint64) bool {
	_, ok := ignoreSignerCheckBlockArray[blockNumber]
	return ok
}

func IsInBlacklist(address *Address) bool {
	if address == nil {
		return false
	}
	_, ok := blacklist[*address]
	return ok
}

// CopyConstants only handles testnet, devnet, local network.
// It skips mainnet since the default value is from mainnet.
func CopyConstants(chainID uint64) {
	var c *constant
	if chainID == TestnetConstant.chainID {
		c = &TestnetConstant
		IsTestnet = true
	} else if chainID == DevnetConstant.chainID {
		c = &DevnetConstant
	} else if chainID == localConstant.chainID {
		c = &localConstant
	} else {
		return
	}

	MaxMasternodesV2 = c.maxMasternodesV2
	BlackListHFNumber = c.blackListHFNumber
	TIP2019Block = c.tip2019Block
	TIPSigning = c.tipSigning
	TIPRandomize = c.tipRandomize
	TIPNoHalvingMNReward = c.tipNoHalvingMNReward
	TIPXDCX = c.tipXDCX
	TIPXDCXLending = c.tipXDCXLending
	TIPXDCXCancellationFee = c.tipXDCXCancellationFee
	TIPTRC21Fee = c.tipTRC21Fee
	TIPIncreaseMasternodes = c.tipIncreaseMasternodes
	BerlinBlock = c.berlinBlock
	LondonBlock = c.londonBlock
	MergeBlock = c.mergeBlock
	ShanghaiBlock = c.shanghaiBlock
	BlockNumberGas50x = c.blockNumberGas50x
	TIPXDCXMinerDisable = c.tipXDCXMinerDisable
	TIPXDCXReceiverDisable = c.tipXDCXReceiverDisable
	Eip1559Block = c.eip1559Block
	CancunBlock = c.cancunBlock

	TRC21IssuerSMC = c.trc21IssuerSMC
	XDCXListingSMC = c.xdcxListingSMC
	RelayerRegistrationSMC = c.relayerRegistrationSMC
	LendingRegistrationSMC = c.lendingRegistrationSMC

	clear(ignoreSignerCheckBlockArray)
	maps.Copy(ignoreSignerCheckBlockArray, c.ignoreSignerCheckBlockArray)

	clear(blacklist)
	maps.Copy(blacklist, c.blacklist)
}

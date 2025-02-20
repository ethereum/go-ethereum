package common

import (
	"math/big"
)

var localConstant = constant{
	chainID:           5151,
	maxMasternodesV2:  108,
	blackListHFNumber: 0,

	tip2019Block:                  big.NewInt(0),
	tipSigning:                    big.NewInt(0),
	tipRandomize:                  big.NewInt(0),
	tipNoHalvingMNReward:          big.NewInt(0),
	tipXDCX:                       big.NewInt(0),
	tipXDCXLending:                big.NewInt(0),
	tipXDCXCancellationFee:        big.NewInt(0),
	tipXDCXCancellationFeeTestnet: big.NewInt(0),
	tipTRC21Fee:                   big.NewInt(13523400),
	tipTRC21FeeTestnet:            big.NewInt(225000),
	tipIncreaseMasternodes:        big.NewInt(0),
	berlinBlock:                   big.NewInt(0),
	londonBlock:                   big.NewInt(0),
	mergeBlock:                    big.NewInt(0),
	shanghaiBlock:                 big.NewInt(0),
	blockNumberGas50x:             big.NewInt(0),
	TIPV2SwitchBlock:              big.NewInt(0),
	tipXDCXMinerDisable:           big.NewInt(0),
	tipXDCXReceiverDisable:        big.NewInt(0),
	eip1559Block:                  big.NewInt(0),
	cancunBlock:                   big.NewInt(9999999999),

	trc21IssuerSMCTestNet: HexToAddress("0x0E2C88753131CE01c7551B726b28BFD04e44003F"),
	trc21IssuerSMC:        HexToAddress("0x8c0faeb5C6bEd2129b8674F262Fd45c4e9468bee"),
	xdcxListingSMC:        HexToAddress("0xDE34dD0f536170993E8CFF639DdFfCF1A85D3E53"),
	xdcxListingSMCTestNet: HexToAddress("0x14B2Bf043b9c31827A472CE4F94294fE9a6277e0"),

	relayerRegistrationSMC:        HexToAddress("0x16c63b79f9C8784168103C0b74E6A59EC2de4a02"),
	relayerRegistrationSMCTestnet: HexToAddress("0xA1996F69f47ba14Cb7f661010A7C31974277958c"),
	lendingRegistrationSMC:        HexToAddress("0x7d761afd7ff65a79e4173897594a194e3c506e57"),
	lendingRegistrationSMCTestnet: HexToAddress("0x28d7fC2Cf5c18203aaCD7459EFC6Af0643C97bE8"),

	ignoreSignerCheckBlockArray: map[uint64]struct{}{},

	blacklist: map[Address]struct{}{},
}

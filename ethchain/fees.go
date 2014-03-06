package ethchain

import (
	"math/big"
)

var TxFeeRat *big.Int = big.NewInt(100000000000000)

var TxFee *big.Int = big.NewInt(100)
var StepFee *big.Int = big.NewInt(1)
var StoreFee *big.Int = big.NewInt(5)
var DataFee *big.Int = big.NewInt(20)
var ExtroFee *big.Int = big.NewInt(40)
var CryptoFee *big.Int = big.NewInt(20)
var ContractFee *big.Int = big.NewInt(100)

var BlockReward *big.Int = big.NewInt(1.5e+18)
var UncleReward *big.Int = big.NewInt(1.125e+18)
var UncleInclusionReward *big.Int = big.NewInt(1.875e+17)

var Period1Reward *big.Int = new(big.Int)
var Period2Reward *big.Int = new(big.Int)
var Period3Reward *big.Int = new(big.Int)
var Period4Reward *big.Int = new(big.Int)

func InitFees() {
	StepFee.Mul(StepFee, TxFeeRat)
	StoreFee.Mul(StoreFee, TxFeeRat)
	DataFee.Mul(DataFee, TxFeeRat)
	ExtroFee.Mul(ExtroFee, TxFeeRat)
	CryptoFee.Mul(CryptoFee, TxFeeRat)
	ContractFee.Mul(ContractFee, TxFeeRat)
}

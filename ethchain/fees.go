package ethchain

import (
	"math/big"
)

var TxFeeRat *big.Int = big.NewInt(100000000000000)

var TxFee *big.Int = big.NewInt(100)
var StepFee *big.Int = big.NewInt(1)
var StoreFee *big.Int = big.NewInt(0)
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
	/*
		// Base for 2**64
		b60 := new(big.Int)
		b60.Exp(big.NewInt(2), big.NewInt(64), big.NewInt(0))
		// Base for 2**80
		b80 := new(big.Int)
		b80.Exp(big.NewInt(2), big.NewInt(80), big.NewInt(0))

		StepFee.Exp(big.NewInt(10), big.NewInt(16), big.NewInt(0))
		//StepFee.Div(b60, big.NewInt(64))
		//fmt.Println("StepFee:", StepFee)

		TxFee.Exp(big.NewInt(2), big.NewInt(64), big.NewInt(0))
		//fmt.Println("TxFee:", TxFee)

		ContractFee.Exp(big.NewInt(2), big.NewInt(64), big.NewInt(0))
		//fmt.Println("ContractFee:", ContractFee)

		MemFee.Div(b60, big.NewInt(4))
		//fmt.Println("MemFee:", MemFee)

		DataFee.Div(b60, big.NewInt(16))
		//fmt.Println("DataFee:", DataFee)

		CryptoFee.Div(b60, big.NewInt(16))
		//fmt.Println("CrytoFee:", CryptoFee)

		ExtroFee.Div(b60, big.NewInt(16))
		//fmt.Println("ExtroFee:", ExtroFee)

		Period1Reward.Mul(b80, big.NewInt(1024))
		//fmt.Println("Period1Reward:", Period1Reward)

		Period2Reward.Mul(b80, big.NewInt(512))
		//fmt.Println("Period2Reward:", Period2Reward)

		Period3Reward.Mul(b80, big.NewInt(256))
		//fmt.Println("Period3Reward:", Period3Reward)

		Period4Reward.Mul(b80, big.NewInt(128))
		//fmt.Println("Period4Reward:", Period4Reward)
	*/
}

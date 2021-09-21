package lendingstate

import (
	"github.com/XinFinOrg/XDPoSChain/common"
	"math/big"
	"reflect"
	"testing"
)

func TestCalculateInterestRate(t *testing.T) {
	type args struct {
		repayTime       uint64
		liquidationTime uint64
		term            uint64
		apr             uint64
	}
	tests := []struct {
		name string
		args args
		want *big.Int
	}{
		// apr = 10% per year
		// term 365 days
		// repay after one day
		// have to pay interest for a half of year
		// I = APR *(T + T1) / 2 / 365 = 10% * (365 + 1) / 2 /365 = 5,01369863 %
		// 1e8 is decimal of interestRate
		{
			"term 365 days: early repay",
			args{
				repayTime:       86400,
				liquidationTime: common.OneYear,
				term:            common.OneYear,
				apr:             10 * 1e8,
			},
			new(big.Int).SetUint64(501369863),
		},

		// apr = 10% per year (365 days)
		// term: 365 days
		// repay at the end
		// pay full interestRate 10%
		// I = APR *(T + T1) / 2 / 365 = 10% * (365 + 365) / 2 /365 = 10 %
		// 1e8 is decimal of interestRate
		{
			"term 365 days: repay at the end",
			args{
				repayTime:       common.OneYear,
				liquidationTime: common.OneYear,
				term:            common.OneYear,
				apr:             10 * 1e8,
			},
			new(big.Int).SetUint64(10 * 1e8),
		},

		// apr = 10% per year
		// term 30 days
		// repay after one day
		// have to pay interest for a half of year
		// I = APR *(T + T1) / 2 / 365 = 10% * (30 + 1) / 2 /365 = 0,424657534 %
		// 1e8 is decimal of interestRate
		{
			"term 30 days: early repay",
			args{
				repayTime:       86400,
				liquidationTime: 30 * 86400,
				term:            30 * 86400,
				apr:             10 * 1e8,
			},
			new(big.Int).SetUint64(42465753),
		},

		// apr = 10% per year (365 days)
		// term: 30 days
		// repay at the end
		// pay full interestRate 10%
		// I = APR *(T + T1) / 2 / 365 = 10% * (30 + 30) / 2 /365 = 0,821917808 %
		// 1e8 is decimal of interestRate
		{
			"term 30 days: repay at the end",
			args{
				repayTime:       30 * 86400,
				liquidationTime: 30 * 86400,
				term:            30 * 86400,
				apr:             10 * 1e8,
			},
			new(big.Int).SetUint64(82191780),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateInterestRate(tt.args.repayTime, tt.args.liquidationTime, tt.args.term, tt.args.apr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CalculateInterestRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSettleBalance(t *testing.T) {
	lendQuantity, _ := new(big.Int).SetString("1000000000000000000000", 10)        // 1000
	fee, _ := new(big.Int).SetString("10000000000000000000", 10)                   // 10
	lendQuantityExcluded, _ := new(big.Int).SetString("990000000000000000000", 10) // 990
	lendTokenNotXDC := common.HexToAddress("0x0000000000000000000000000000000000000033")
	collateral := common.HexToAddress("0x0000000000000000000000000000000000000022")
	collateralLocked, _ := new(big.Int).SetString("1000000000000000000000", 10) // 1000
	collateralLocked = new(big.Int).Mul(big.NewInt(150), collateralLocked)
	collateralLocked = new(big.Int).Div(collateralLocked, big.NewInt(100))

	type GetSettleBalanceArg struct {
		isXDCXLendingFork      bool
		takerSide              string
		lendTokenXDCPrice      *big.Int
		collateralPrice        *big.Int
		depositRate            *big.Int
		borrowFeeRate          *big.Int
		lendingToken           common.Address
		collateralToken        common.Address
		lendTokenDecimal       *big.Int
		collateralTokenDecimal *big.Int
		quantityToLend         *big.Int
	}
	tests := []struct {
		name    string
		args    GetSettleBalanceArg
		want    *LendingSettleBalance
		wantErr bool
	}{
		{
			"collateralPrice nil",
			GetSettleBalanceArg{
				true,
				Borrowing,
				common.BasePrice,
				common.Big0,
				big.NewInt(150),
				big.NewInt(10000), // 100%
				common.Address{},
				common.Address{},
				common.BasePrice,
				common.BasePrice,
				lendQuantity,
			},
			nil,
			true,
		},
		{
			"quantityToLend = borrowFee, taker BORROW",
			GetSettleBalanceArg{
				true,
				Borrowing,
				common.BasePrice,
				common.BasePrice,
				big.NewInt(150),
				big.NewInt(10000), // 100%
				common.Address{},
				common.Address{},
				common.BasePrice,
				common.BasePrice,
				lendQuantity,
			},
			nil,
			true,
		},

		{
			"LendToken is XDC, quantity too small, taker BORROW",
			GetSettleBalanceArg{
				true,
				Borrowing,
				common.BasePrice,
				common.BasePrice,
				big.NewInt(150),
				big.NewInt(100), // 1%
				common.HexToAddress(common.XDCNativeAddress),
				common.Address{},
				common.BasePrice,
				common.BasePrice,
				common.BasePrice,
			},
			nil,
			true,
		},
		{
			"LendToken is not XDC, quantity too small, taker BORROW",
			GetSettleBalanceArg{
				true,
				Borrowing,
				common.BasePrice,
				common.BasePrice,
				big.NewInt(150),
				big.NewInt(100), // 1%
				common.Address{},
				common.Address{},
				common.BasePrice,
				common.BasePrice,
				common.BasePrice,
			},
			nil,
			true,
		},

		{
			"LendToken is not XDC, no error, taker BORROW",
			GetSettleBalanceArg{
				true,
				Borrowing,
				common.BasePrice,
				common.BasePrice,
				big.NewInt(150),
				big.NewInt(100), // 1%
				lendTokenNotXDC,
				collateral,
				common.BasePrice,
				common.BasePrice,
				lendQuantity,
			},
			&LendingSettleBalance{
				Taker: TradeResult{
					Fee:      fee,
					InToken:  lendTokenNotXDC,
					InTotal:  lendQuantityExcluded,
					OutToken: collateral,
					OutTotal: collateralLocked,
				},
				Maker: TradeResult{
					Fee:      common.Big0,
					InToken:  common.Address{},
					InTotal:  common.Big0,
					OutToken: lendTokenNotXDC,
					OutTotal: lendQuantity,
				},
				CollateralLockedAmount: collateralLocked,
			},
			false,
		},

		{
			"LendToken is not XDC, no error, taker INVEST",
			GetSettleBalanceArg{
				true,
				Investing,
				common.BasePrice,
				common.BasePrice,
				big.NewInt(150),
				big.NewInt(100), // 1%
				lendTokenNotXDC,
				collateral,
				common.BasePrice,
				common.BasePrice,
				lendQuantity,
			},
			&LendingSettleBalance{
				Maker: TradeResult{
					Fee:      fee,
					InToken:  lendTokenNotXDC,
					InTotal:  lendQuantityExcluded,
					OutToken: collateral,
					OutTotal: collateralLocked,
				},
				Taker: TradeResult{
					Fee:      common.Big0,
					InToken:  common.Address{},
					InTotal:  common.Big0,
					OutToken: lendTokenNotXDC,
					OutTotal: lendQuantity,
				},
				CollateralLockedAmount: collateralLocked,
			},
			false,
		},
		{
			"LendToken is XDC, no error, taker invest",
			GetSettleBalanceArg{
				true,
				Investing,
				common.BasePrice,
				common.BasePrice,
				big.NewInt(150),
				big.NewInt(100), // 1%
				common.HexToAddress(common.XDCNativeAddress),
				collateral,
				common.BasePrice,
				common.BasePrice,
				lendQuantity,
			},
			&LendingSettleBalance{
				Taker: TradeResult{
					Fee:      common.Big0,
					InToken:  common.Address{},
					InTotal:  common.Big0,
					OutToken: common.HexToAddress(common.XDCNativeAddress),
					OutTotal: lendQuantity,
				},
				Maker: TradeResult{
					Fee:      fee,
					InToken:  common.HexToAddress(common.XDCNativeAddress),
					InTotal:  lendQuantityExcluded,
					OutToken: collateral,
					OutTotal: collateralLocked,
				},
				CollateralLockedAmount: collateralLocked,
			},
			false,
		},

		{
			"LendToken is XDC, no error, taker Borrow",
			GetSettleBalanceArg{
				true,
				Borrowing,
				common.BasePrice,
				common.BasePrice,
				big.NewInt(150),
				big.NewInt(100), // 1%
				common.HexToAddress(common.XDCNativeAddress),
				collateral,
				common.BasePrice,
				common.BasePrice,
				lendQuantity,
			},
			&LendingSettleBalance{
				Maker: TradeResult{
					Fee:      common.Big0,
					InToken:  common.Address{},
					InTotal:  common.Big0,
					OutToken: common.HexToAddress(common.XDCNativeAddress),
					OutTotal: lendQuantity,
				},
				Taker: TradeResult{
					Fee:      fee,
					InToken:  common.HexToAddress(common.XDCNativeAddress),
					InTotal:  lendQuantityExcluded,
					OutToken: collateral,
					OutTotal: collateralLocked,
				},
				CollateralLockedAmount: collateralLocked,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSettleBalance(tt.args.isXDCXLendingFork, tt.args.takerSide, tt.args.lendTokenXDCPrice, tt.args.collateralPrice, tt.args.depositRate, tt.args.borrowFeeRate, tt.args.lendingToken, tt.args.collateralToken, tt.args.lendTokenDecimal, tt.args.collateralTokenDecimal, tt.args.quantityToLend)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSettleBalance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != nil {
				t.Log(tt.want.String())
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSettleBalance() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateTotalRepayValue(t *testing.T) {
	type CalculateTotalRepayValueArg struct {
		finalizeTime    uint64
		liquidationTime uint64
		term            uint64
		apr             uint64
		tradeAmount     *big.Int
	}
	totalRepayOneYearRepayEarly, _ := new(big.Int).SetString("1050136986300000000000", 10)
	totalRepayOneYearRepayInTime, _ := new(big.Int).SetString("1100000000000000000000", 10)
	totalRepay30DaysRepayEarly, _ := new(big.Int).SetString("1004246575300000000000", 10)
	totalRepay30DaysRepayInTime, _ := new(big.Int).SetString("1008219178000000000000", 10)

	tradeAmount := new(big.Int).Mul(big.NewInt(1000), common.BasePrice)
	tests := []struct {
		name string
		args CalculateTotalRepayValueArg
		want *big.Int
	}{
		// apr = 10% per year
		// term 365 days
		// repay after one day
		// have to pay interest for a half of year
		// I = APR *(T + T1) / 2 / 365 = 10% * (365 + 1) / 2 /365 = 5,01369863 %
		// 1e8 is decimal of interestRate
		// amount 1000 USDT
		// -> totalRepay: 1000 * (1 + 5,01369863 %) = 1050,1369863
		{
			"term 365 days, 1000 USDT: early repay",
			CalculateTotalRepayValueArg{
				finalizeTime:    86400,
				liquidationTime: common.OneYear,
				term:            common.OneYear,
				apr:             10 * 1e8,
				tradeAmount:     tradeAmount,
			},
			totalRepayOneYearRepayEarly,
		},

		// apr = 10% per year (365 days)
		// term: 365 days
		// repay at the end
		// pay full interestRate 10%
		// I = APR *(T + T1) / 2 / 365 = 10% * (365 + 365) / 2 /365 = 10 %
		// 1e8 is decimal of interestRate
		// -> totalRepay: 1000 * (1 + 10 %) = 1100
		{
			"term 365 days: repay at the end",
			CalculateTotalRepayValueArg{
				finalizeTime:    common.OneYear,
				liquidationTime: common.OneYear,
				term:            common.OneYear,
				apr:             10 * 1e8,
				tradeAmount:     tradeAmount,
			},
			totalRepayOneYearRepayInTime,
		},

		// apr = 10% per year
		// term 30 days
		// repay after one day
		// have to pay interest for a half of year
		// I = APR *(T + T1) / 2 / 365 = 10% * (30 + 1) / 2 /365 = 0,424657534 %
		// 1e8 is decimal of interestRate
		// -> totalRepay: 1000 * (1 + 0,424657534 %) = 1004,2465753
		{
			"term 30 days: early repay",
			CalculateTotalRepayValueArg{
				finalizeTime:    86400,
				liquidationTime: 30 * 86400,
				term:            30 * 86400,
				apr:             10 * 1e8,
				tradeAmount:     tradeAmount,
			},
			totalRepay30DaysRepayEarly,
		},

		// apr = 10% per year (365 days)
		// term: 30 days
		// repay at the end
		// pay full interestRate 10%
		// I = APR *(T + T1) / 2 / 365 = 10% * (30 + 30) / 2 /365 = 0,821917808 %
		// 1e8 is decimal of interestRate
		// -> totalRepay: 1000 * (1 + 0,821917808 %) = 1008,2191780
		{
			"term 30 days: repay at the end",
			CalculateTotalRepayValueArg{
				finalizeTime:    30 * 86400,
				liquidationTime: 30 * 86400,
				term:            30 * 86400,
				apr:             10 * 1e8,
				tradeAmount:     tradeAmount,
			},
			totalRepay30DaysRepayInTime,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateTotalRepayValue(tt.args.finalizeTime, tt.args.liquidationTime, tt.args.term, tt.args.apr, tt.args.tradeAmount); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CalculateTotalRepayValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

package tradingstate

import (
	"github.com/XinFinOrg/XDPoSChain/common"
	"math/big"
	"reflect"
	"testing"
)

func TestGetSettleBalance(t *testing.T) {
	testToken := common.HexToAddress("0x0000000000000000000000000000000000000022")
	testFee, _ := new(big.Int).SetString("1000000000000000000", 10)
	tradeQuantity, _ := new(big.Int).SetString("1000000000000000000000", 10)
	tradeQuantityIncludedFee, _ := new(big.Int).SetString("1001000000000000000000", 10)
	tradeQuantityExcludedFee, _ := new(big.Int).SetString("999000000000000000000", 10)
	type GetSettleBalanceArg struct {
		quotePrice        *big.Int
		takerSide         string
		takerFeeRate      *big.Int
		baseToken         common.Address
		quoteToken        common.Address
		makerPrice        *big.Int
		makerFeeRate      *big.Int
		baseTokenDecimal  *big.Int
		quoteTokenDecimal *big.Int
		quantityToTrade   *big.Int
	}
	tests := []struct {
		name    string
		args    GetSettleBalanceArg
		want    *SettleBalance
		wantErr bool
	}{
		{
			"BUY tradeQuantity == fee",
			GetSettleBalanceArg{
				quotePrice:        common.BasePrice,
				takerSide:         Bid,
				takerFeeRate:      big.NewInt(10000), // feeRate 100%
				baseToken:         common.Address{},
				quoteToken:        common.Address{},
				makerPrice:        common.BasePrice,
				makerFeeRate:      big.NewInt(10000), // feeRate 100%
				baseTokenDecimal:  common.BasePrice,
				quoteTokenDecimal: common.BasePrice,
				quantityToTrade:   new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			nil,
			true,
		},
		{
			"BUY, quote is not XDC, makerFee <= 0.001 XDC",
			GetSettleBalanceArg{
				quotePrice:        common.BasePrice,
				takerSide:         Bid,
				takerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseToken:         testToken,
				quoteToken:        common.HexToAddress("0x0000000000000000000000000000000000000002"),
				makerPrice:        common.BasePrice,
				makerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseTokenDecimal:  common.BasePrice,
				quoteTokenDecimal: common.BasePrice,
				quantityToTrade:   new(big.Int).Mul(big.NewInt(1), common.BasePrice),
			},
			nil,
			true,
		},
		{
			"BUY, quote is not XDC, takerFee <= 0.001 XDC",
			GetSettleBalanceArg{
				quotePrice:        common.BasePrice,
				takerSide:         Bid,
				takerFeeRate:      big.NewInt(5), // feeRate 0.05%
				baseToken:         testToken,
				quoteToken:        common.HexToAddress("0x0000000000000000000000000000000000000002"),
				makerPrice:        common.BasePrice,
				makerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseTokenDecimal:  common.BasePrice,
				quoteTokenDecimal: common.BasePrice,
				quantityToTrade:   new(big.Int).Mul(big.NewInt(2), common.BasePrice),
			},
			nil,
			true,
		},
		{
			"BUY, quote is XDC, makerFee <= 0.001 XDC",
			GetSettleBalanceArg{
				quotePrice:        common.BasePrice,
				takerSide:         Bid,
				takerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseToken:         testToken,
				quoteToken:        common.HexToAddress(common.XDCNativeAddress),
				makerPrice:        common.BasePrice,
				makerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseTokenDecimal:  common.BasePrice,
				quoteTokenDecimal: common.BasePrice,
				quantityToTrade:   new(big.Int).Mul(big.NewInt(1), common.BasePrice),
			},
			nil,
			true,
		},
		{
			"BUY, quote is XDC, takerFee <= 0.001 XDC",
			GetSettleBalanceArg{
				quotePrice:        common.BasePrice,
				takerSide:         Bid,
				takerFeeRate:      big.NewInt(5), // feeRate 0.05%
				baseToken:         testToken,
				quoteToken:        common.HexToAddress(common.XDCNativeAddress),
				makerPrice:        common.BasePrice,
				makerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseTokenDecimal:  common.BasePrice,
				quoteTokenDecimal: common.BasePrice,
				quantityToTrade:   new(big.Int).Mul(big.NewInt(2), common.BasePrice),
			},
			nil,
			true,
		},

		{
			"BUY, no error",
			GetSettleBalanceArg{
				quotePrice:        common.BasePrice,
				takerSide:         Bid,
				takerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseToken:         testToken,
				quoteToken:        common.HexToAddress(common.XDCNativeAddress),
				makerPrice:        common.BasePrice,
				makerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseTokenDecimal:  common.BasePrice,
				quoteTokenDecimal: common.BasePrice,
				quantityToTrade:   new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			&SettleBalance{
				Taker: TradeResult{Fee: testFee, InToken: testToken, InTotal: tradeQuantity, OutToken: common.HexToAddress(common.XDCNativeAddress), OutTotal: tradeQuantityIncludedFee},
				Maker: TradeResult{Fee: testFee, InToken: common.HexToAddress(common.XDCNativeAddress), InTotal: tradeQuantityExcludedFee, OutToken: testToken, OutTotal: tradeQuantity},
			},
			false,
		},

		{
			"SELL tradeQuantity == fee",
			GetSettleBalanceArg{
				quotePrice:        common.BasePrice,
				takerSide:         Ask,
				takerFeeRate:      big.NewInt(10000), // feeRate 100%
				baseToken:         testToken,
				quoteToken:        common.Address{},
				makerPrice:        common.BasePrice,
				makerFeeRate:      big.NewInt(10000), // feeRate 100%
				baseTokenDecimal:  common.BasePrice,
				quoteTokenDecimal: common.BasePrice,
				quantityToTrade:   new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			nil,
			true,
		},
		{
			"SELL, quote is not XDC, makerFee <= 0.001 XDC",
			GetSettleBalanceArg{
				quotePrice:        common.BasePrice,
				takerSide:         Ask,
				takerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseToken:         testToken,
				quoteToken:        common.HexToAddress("0x0000000000000000000000000000000000000002"),
				makerPrice:        common.BasePrice,
				makerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseTokenDecimal:  common.BasePrice,
				quoteTokenDecimal: common.BasePrice,
				quantityToTrade:   new(big.Int).Mul(big.NewInt(1), common.BasePrice),
			},
			nil,
			true,
		},
		{
			"SELL, quote is not XDC, takerFee <= 0.001 XDC",
			GetSettleBalanceArg{
				quotePrice:        common.BasePrice,
				takerSide:         Ask,
				takerFeeRate:      big.NewInt(5), // feeRate 0.05%
				baseToken:         testToken,
				quoteToken:        common.HexToAddress("0x0000000000000000000000000000000000000002"),
				makerPrice:        common.BasePrice,
				makerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseTokenDecimal:  common.BasePrice,
				quoteTokenDecimal: common.BasePrice,
				quantityToTrade:   new(big.Int).Mul(big.NewInt(2), common.BasePrice),
			},
			nil,
			true,
		},
		{
			"SELL, quote is XDC, makerFee <= 0.001 XDC",
			GetSettleBalanceArg{
				quotePrice:        common.BasePrice,
				takerSide:         Ask,
				takerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseToken:         testToken,
				quoteToken:        common.HexToAddress(common.XDCNativeAddress),
				makerPrice:        common.BasePrice,
				makerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseTokenDecimal:  common.BasePrice,
				quoteTokenDecimal: common.BasePrice,
				quantityToTrade:   new(big.Int).Mul(big.NewInt(1), common.BasePrice),
			},
			nil,
			true,
		},
		{
			"SELL, quote is XDC, takerFee <= 0.001 XDC",
			GetSettleBalanceArg{
				quotePrice:        common.BasePrice,
				takerSide:         Ask,
				takerFeeRate:      big.NewInt(5), // feeRate 0.05%
				baseToken:         testToken,
				quoteToken:        common.HexToAddress(common.XDCNativeAddress),
				makerPrice:        common.BasePrice,
				makerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseTokenDecimal:  common.BasePrice,
				quoteTokenDecimal: common.BasePrice,
				quantityToTrade:   new(big.Int).Mul(big.NewInt(2), common.BasePrice),
			},
			nil,
			true,
		},

		{
			"SELL, no error",
			GetSettleBalanceArg{
				quotePrice:        common.BasePrice,
				takerSide:         Ask,
				takerFeeRate:      big.NewInt(10), // feeRate 15%
				baseToken:         testToken,
				quoteToken:        common.HexToAddress(common.XDCNativeAddress),
				makerPrice:        common.BasePrice,
				makerFeeRate:      big.NewInt(10), // feeRate 0.1%
				baseTokenDecimal:  common.BasePrice,
				quoteTokenDecimal: common.BasePrice,
				quantityToTrade:   new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			&SettleBalance{
				Maker: TradeResult{Fee: testFee, InToken: testToken, InTotal: tradeQuantity, OutToken: common.HexToAddress(common.XDCNativeAddress), OutTotal: tradeQuantityIncludedFee},
				Taker: TradeResult{Fee: testFee, InToken: common.HexToAddress(common.XDCNativeAddress), InTotal: tradeQuantityExcludedFee, OutToken: testToken, OutTotal: tradeQuantity},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSettleBalance(tt.args.quotePrice, tt.args.takerSide, tt.args.takerFeeRate, tt.args.baseToken, tt.args.quoteToken, tt.args.makerPrice, tt.args.makerFeeRate, tt.args.baseTokenDecimal, tt.args.quoteTokenDecimal, tt.args.quantityToTrade)
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

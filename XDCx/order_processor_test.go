package XDCx

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/XDCx/tradingstate"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/node"
)

func Test_getCancelFeeV1(t *testing.T) {
	type CancelFeeArg struct {
		baseTokenDecimal *big.Int
		feeRate          *big.Int
		order            *tradingstate.OrderItem
	}
	tests := []struct {
		name string
		args CancelFeeArg
		want *big.Int
	}{
		// zero fee test: SELL
		{
			"zero fee getCancelFeeV1: SELL",
			CancelFeeArg{
				baseTokenDecimal: common.Big1,
				feeRate:          common.Big0,
				order: &tradingstate.OrderItem{
					Quantity: new(big.Int).SetUint64(10000),
					Side:     tradingstate.Ask,
				},
			},
			common.Big0,
		},

		// zero fee test: BUY
		{
			"zero fee getCancelFeeV1: BUY",
			CancelFeeArg{
				baseTokenDecimal: common.Big1,
				feeRate:          common.Big0,
				order: &tradingstate.OrderItem{
					Quantity: new(big.Int).SetUint64(10000),
					Price:    new(big.Int).SetUint64(1),
					Side:     tradingstate.Bid,
				},
			},
			common.Big0,
		},

		// test getCancelFee: SELL
		{
			"test getCancelFeeV1:: SELL",
			CancelFeeArg{
				baseTokenDecimal: common.Big1,
				feeRate:          new(big.Int).SetUint64(10), // 10/10000= 0.1%
				order: &tradingstate.OrderItem{
					Quantity: new(big.Int).SetUint64(10000),
					Side:     tradingstate.Ask,
				},
			},
			common.Big1,
		},

		// test getCancelFee:: BUY
		{
			"test getCancelFeeV1:: BUY",
			CancelFeeArg{
				baseTokenDecimal: common.Big1,
				feeRate:          new(big.Int).SetUint64(10), // 10/10000= 0.1%
				order: &tradingstate.OrderItem{
					Quantity: new(big.Int).SetUint64(10000),
					Price:    new(big.Int).SetUint64(1),
					Side:     tradingstate.Bid,
				},
			},
			common.Big1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getCancelFeeV1(tt.args.baseTokenDecimal, tt.args.feeRate, tt.args.order); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCancelFeeV1() = %v, quantity %v", got, tt.want)
			}
		})
	}
}

func Test_getCancelFee(t *testing.T) {
	stack, err := node.New(&node.DefaultConfig)
	if err != nil {
		t.Fatalf("could not create new node: %v", err)
	}
	XDCx := New(stack, &DefaultConfig)
	defer stack.Close()
	db := rawdb.NewMemoryDatabase()
	stateCache := tradingstate.NewDatabase(db)
	tradingStateDb, _ := tradingstate.New(types.EmptyRootHash, stateCache)

	testTokenA := common.HexToAddress("0x1000000000000000000000000000000000000002")
	testTokenB := common.HexToAddress("0x1100000000000000000000000000000000000003")
	// set decimal
	// tokenA has decimal 10^18
	XDCx.SetTokenDecimal(testTokenA, common.BasePrice)
	// tokenB has decimal 10^8
	tokenBDecimal := new(big.Int).Exp(big.NewInt(10), big.NewInt(8), nil)
	XDCx.SetTokenDecimal(testTokenB, tokenBDecimal)

	// set tokenAPrice = 1 XDC
	tradingStateDb.SetMediumPriceBeforeEpoch(tradingstate.GetTradingOrderBookHash(testTokenA, common.XDCNativeAddressBinary), common.BasePrice)
	// set tokenBPrice = 1 XDC
	tradingStateDb.SetMediumPriceBeforeEpoch(tradingstate.GetTradingOrderBookHash(common.XDCNativeAddressBinary, testTokenB), tokenBDecimal)

	type CancelFeeArg struct {
		feeRate *big.Int
		order   *tradingstate.OrderItem
	}
	tests := []struct {
		name string
		args CancelFeeArg
		want *big.Int
	}{

		// BASE: testTokenA,
		// QUOTE: XDC

		// zero fee test: SELL
		{
			"TokenA/XDC zero fee test: SELL",
			CancelFeeArg{
				feeRate: common.Big0,
				order: &tradingstate.OrderItem{
					BaseToken:  testTokenA,
					QuoteToken: common.XDCNativeAddressBinary,
					Quantity:   new(big.Int).SetUint64(10000),
					Side:       tradingstate.Ask,
				},
			},
			common.Big0,
		},

		// zero fee test: BUY
		{
			"TokenA/XDC zero fee test: BUY",
			CancelFeeArg{
				feeRate: common.Big0,
				order: &tradingstate.OrderItem{
					BaseToken:  testTokenA,
					QuoteToken: common.XDCNativeAddressBinary,
					Quantity:   new(big.Int).SetUint64(10000),
					Side:       tradingstate.Bid,
				},
			},
			common.Big0,
		},

		// test getCancelFee: SELL
		{
			"TokenA/XDC test getCancelFee:: SELL",
			CancelFeeArg{
				feeRate: new(big.Int).SetUint64(10), // 10/10000= 0.1%
				order: &tradingstate.OrderItem{
					BaseToken:  common.XDCNativeAddressBinary,
					QuoteToken: testTokenA,
					Quantity:   new(big.Int).SetUint64(10000),
					Side:       tradingstate.Ask,
				},
			},
			common.RelayerCancelFee,
		},

		// test getCancelFee:: BUY
		{
			"TokenA/XDC test getCancelFee:: BUY",
			CancelFeeArg{
				feeRate: new(big.Int).SetUint64(10), // 10/10000= 0.1%
				order: &tradingstate.OrderItem{
					Quantity:   new(big.Int).SetUint64(10000),
					BaseToken:  common.XDCNativeAddressBinary,
					QuoteToken: testTokenA,
					Side:       tradingstate.Bid,
				},
			},
			common.RelayerCancelFee,
		},

		// BASE: XDC
		// QUOTE: testTokenA
		// zero fee test: SELL
		{
			"XDC/TokenA zero fee test: SELL",
			CancelFeeArg{
				feeRate: common.Big0,
				order: &tradingstate.OrderItem{
					BaseToken:  common.XDCNativeAddressBinary,
					QuoteToken: testTokenA,
					Quantity:   new(big.Int).SetUint64(10000),
					Side:       tradingstate.Ask,
				},
			},
			common.Big0,
		},

		// zero fee test: BUY
		{
			"XDC/TokenA zero fee test: BUY",
			CancelFeeArg{
				feeRate: common.Big0,
				order: &tradingstate.OrderItem{
					BaseToken:  common.XDCNativeAddressBinary,
					QuoteToken: testTokenA,
					Quantity:   new(big.Int).SetUint64(10000),
					Side:       tradingstate.Bid,
				},
			},
			common.Big0,
		},

		// test getCancelFee: SELL
		{
			"XDC/TokenA test getCancelFee:: SELL",
			CancelFeeArg{
				feeRate: new(big.Int).SetUint64(10), // 10/10000= 0.1%
				order: &tradingstate.OrderItem{
					BaseToken:  common.XDCNativeAddressBinary,
					QuoteToken: testTokenA,
					Quantity:   new(big.Int).SetUint64(10000),
					Side:       tradingstate.Ask,
				},
			},
			common.RelayerCancelFee,
		},

		// test getCancelFee:: BUY
		{
			"XDC/TokenA test getCancelFee:: BUY",
			CancelFeeArg{
				feeRate: new(big.Int).SetUint64(10), // 10/10000= 0.1%
				order: &tradingstate.OrderItem{
					Quantity:   new(big.Int).SetUint64(10000),
					BaseToken:  common.XDCNativeAddressBinary,
					QuoteToken: testTokenA,
					Side:       tradingstate.Bid,
				},
			},
			common.RelayerCancelFee,
		},

		// BASE: testTokenB
		// QUOTE: testTokenA
		// zero fee test: SELL
		{
			"TokenB/TokenA zero fee test: SELL",
			CancelFeeArg{
				feeRate: common.Big0,
				order: &tradingstate.OrderItem{
					BaseToken:  testTokenB,
					QuoteToken: testTokenA,
					Quantity:   new(big.Int).SetUint64(10000),
					Side:       tradingstate.Ask,
				},
			},
			common.Big0,
		},

		// zero fee test: BUY
		{
			"TokenB/TokenA zero fee test: BUY",
			CancelFeeArg{
				feeRate: common.Big0,
				order: &tradingstate.OrderItem{
					BaseToken:  testTokenB,
					QuoteToken: testTokenA,
					Quantity:   new(big.Int).SetUint64(10000),
					Side:       tradingstate.Bid,
				},
			},
			common.Big0,
		},

		// test getCancelFee: SELL
		{
			"TokenB/TokenA test getCancelFee:: SELL",
			CancelFeeArg{
				feeRate: new(big.Int).SetUint64(10), // 10/10000= 0.1%
				order: &tradingstate.OrderItem{
					BaseToken:  testTokenB,
					QuoteToken: testTokenA,
					Quantity:   new(big.Int).SetUint64(10000),
					Side:       tradingstate.Ask,
				},
			},
			new(big.Int).Exp(big.NewInt(10), big.NewInt(4), nil),
		},

		// test getCancelFee:: BUY
		{
			"TokenB/TokenA test getCancelFee:: BUY",
			CancelFeeArg{
				feeRate: new(big.Int).SetUint64(10), // 10/10000= 0.1%
				order: &tradingstate.OrderItem{
					Quantity:   new(big.Int).SetUint64(10000),
					BaseToken:  testTokenB,
					QuoteToken: testTokenA,
					Side:       tradingstate.Bid,
				},
			},
			common.RelayerCancelFee,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := XDCx.getCancelFee(nil, nil, tradingStateDb, tt.args.order, tt.args.feeRate); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCancelFee() = %v, quantity %v", got, tt.want)
			}
		})
	}

	// testcase: can't get price of token in XDC
	testTokenC := common.HexToAddress("0x1200000000000000000000000000000000000004")
	XDCx.SetTokenDecimal(testTokenC, big.NewInt(1))
	tokenCOrder := CancelFeeArg{
		feeRate: new(big.Int).SetUint64(10), // 10/10000= 0.1%
		order: &tradingstate.OrderItem{
			Quantity:   new(big.Int).SetUint64(10000),
			BaseToken:  testTokenC,
			QuoteToken: testTokenA,
			Side:       tradingstate.Ask,
		},
	}
	if fee, _ := XDCx.getCancelFee(nil, nil, tradingStateDb, tokenCOrder.order, tokenCOrder.feeRate); fee != nil && fee.Sign() != 0 {
		t.Errorf("getCancelFee() = %v, want %v", fee, common.Big0)
	}

	// testcase: invalid token decimal
	testTokenD := common.HexToAddress("0x1300000000000000000000000000000000000005")
	XDCx.SetTokenDecimal(testTokenD, big.NewInt(0))
	tokenDOrder := CancelFeeArg{
		feeRate: new(big.Int).SetUint64(10), // 10/10000= 0.1%
		order: &tradingstate.OrderItem{
			Quantity:   new(big.Int).SetUint64(10000),
			BaseToken:  testTokenD,
			QuoteToken: testTokenA,
			Side:       tradingstate.Ask,
		},
	}
	if fee, _ := XDCx.getCancelFee(nil, nil, tradingStateDb, tokenDOrder.order, tokenDOrder.feeRate); fee != nil && fee.Sign() != 0 {
		t.Errorf("getCancelFee() = %v, want %v", fee, common.Big0)
	}

}

func TestGetTradeQuantity(t *testing.T) {
	type GetTradeQuantityArg struct {
		takerSide        string
		takerFeeRate     *big.Int
		takerBalance     *big.Int
		makerPrice       *big.Int
		makerFeeRate     *big.Int
		makerBalance     *big.Int
		baseTokenDecimal *big.Int
		quantityToTrade  *big.Int
	}
	tests := []struct {
		name        string
		args        GetTradeQuantityArg
		quantity    *big.Int
		rejectMaker bool
	}{
		{
			"BUY: feeRate = 0, price 1, quantity 1000, taker balance 1000, maker balance 1000",
			GetTradeQuantityArg{
				takerSide:        tradingstate.Bid,
				takerFeeRate:     common.Big0,
				takerBalance:     new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
				makerPrice:       common.BasePrice,
				makerFeeRate:     common.Big0,
				makerBalance:     new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
				baseTokenDecimal: common.BasePrice,
				quantityToTrade:  new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			false,
		},
		{
			"BUY: feeRate = 0, price 1, quantity 1000, taker balance 1000, maker balance 900 -> reject maker",
			GetTradeQuantityArg{
				takerSide:        tradingstate.Bid,
				takerFeeRate:     common.Big0,
				takerBalance:     new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
				makerPrice:       common.BasePrice,
				makerFeeRate:     common.Big0,
				makerBalance:     new(big.Int).Mul(big.NewInt(900), common.BasePrice),
				baseTokenDecimal: common.BasePrice,
				quantityToTrade:  new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			new(big.Int).Mul(big.NewInt(900), common.BasePrice),
			true,
		},
		{
			"BUY: feeRate = 0, price 1, quantity 1000, taker balance 900, maker balance 1000 -> reject taker",
			GetTradeQuantityArg{
				takerSide:        tradingstate.Bid,
				takerFeeRate:     common.Big0,
				takerBalance:     new(big.Int).Mul(big.NewInt(900), common.BasePrice),
				makerPrice:       common.BasePrice,
				makerFeeRate:     common.Big0,
				makerBalance:     new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
				baseTokenDecimal: common.BasePrice,
				quantityToTrade:  new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			new(big.Int).Mul(big.NewInt(900), common.BasePrice),
			false,
		},
		{
			"BUY: feeRate = 0, price 1, quantity 1000, taker balance 0, maker balance 1000 -> reject taker",
			GetTradeQuantityArg{
				takerSide:        tradingstate.Bid,
				takerFeeRate:     common.Big0,
				takerBalance:     common.Big0,
				makerPrice:       common.BasePrice,
				makerFeeRate:     common.Big0,
				makerBalance:     new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
				baseTokenDecimal: common.BasePrice,
				quantityToTrade:  new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			common.Big0,
			false,
		},
		{
			"BUY: feeRate = 0, price 1, quantity 1000, taker balance 0, maker balance 0 -> reject both taker",
			GetTradeQuantityArg{
				takerSide:        tradingstate.Bid,
				takerFeeRate:     common.Big0,
				takerBalance:     common.Big0,
				makerPrice:       common.BasePrice,
				makerFeeRate:     common.Big0,
				makerBalance:     common.Big0,
				baseTokenDecimal: common.BasePrice,
				quantityToTrade:  new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			common.Big0,
			false,
		},
		{
			"BUY: feeRate = 0, price 1, quantity 1000, taker balance 500, maker balance 100 -> reject both taker, maker",
			GetTradeQuantityArg{
				takerSide:        tradingstate.Bid,
				takerFeeRate:     common.Big0,
				takerBalance:     new(big.Int).Mul(big.NewInt(500), common.BasePrice),
				makerPrice:       common.BasePrice,
				makerFeeRate:     common.Big0,
				makerBalance:     new(big.Int).Mul(big.NewInt(100), common.BasePrice),
				baseTokenDecimal: common.BasePrice,
				quantityToTrade:  new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			new(big.Int).Mul(big.NewInt(100), common.BasePrice),
			true,
		},

		{
			"SELL: feeRate = 0, price 1, quantity 1000, taker balance 1000, maker balance 1000",
			GetTradeQuantityArg{
				takerSide:        tradingstate.Ask,
				takerFeeRate:     common.Big0,
				takerBalance:     new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
				makerPrice:       common.BasePrice,
				makerFeeRate:     common.Big0,
				makerBalance:     new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
				baseTokenDecimal: common.BasePrice,
				quantityToTrade:  new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			false,
		},
		{
			"SELL: feeRate = 0, price 1, quantity 1000, taker balance 1000, maker balance 900 -> reject maker",
			GetTradeQuantityArg{
				takerSide:        tradingstate.Ask,
				takerFeeRate:     common.Big0,
				takerBalance:     new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
				makerPrice:       common.BasePrice,
				makerFeeRate:     common.Big0,
				makerBalance:     new(big.Int).Mul(big.NewInt(900), common.BasePrice),
				baseTokenDecimal: common.BasePrice,
				quantityToTrade:  new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			new(big.Int).Mul(big.NewInt(900), common.BasePrice),
			true,
		},
		{
			"SELL: feeRate = 0, price 1, quantity 1000, taker balance 900, maker balance 1000 -> reject taker",
			GetTradeQuantityArg{
				takerSide:        tradingstate.Ask,
				takerFeeRate:     common.Big0,
				takerBalance:     new(big.Int).Mul(big.NewInt(900), common.BasePrice),
				makerPrice:       common.BasePrice,
				makerFeeRate:     common.Big0,
				makerBalance:     new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
				baseTokenDecimal: common.BasePrice,
				quantityToTrade:  new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			new(big.Int).Mul(big.NewInt(900), common.BasePrice),
			false,
		},
		{
			"SELL: feeRate = 0, price 1, quantity 1000, taker balance 0, maker balance 1000 -> reject taker",
			GetTradeQuantityArg{
				takerSide:        tradingstate.Ask,
				takerFeeRate:     common.Big0,
				takerBalance:     common.Big0,
				makerPrice:       common.BasePrice,
				makerFeeRate:     common.Big0,
				makerBalance:     new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
				baseTokenDecimal: common.BasePrice,
				quantityToTrade:  new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			common.Big0,
			false,
		},
		{
			"SELL: feeRate = 0, price 1, quantity 1000, taker balance 0, maker balance 0 -> reject maker",
			GetTradeQuantityArg{
				takerSide:        tradingstate.Ask,
				takerFeeRate:     common.Big0,
				takerBalance:     common.Big0,
				makerPrice:       common.BasePrice,
				makerFeeRate:     common.Big0,
				makerBalance:     common.Big0,
				baseTokenDecimal: common.BasePrice,
				quantityToTrade:  new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			common.Big0,
			true,
		},
		{
			"SELL: feeRate = 0, price 1, quantity 1000, taker balance 500, maker balance 100 -> reject both taker, maker",
			GetTradeQuantityArg{
				takerSide:        tradingstate.Ask,
				takerFeeRate:     common.Big0,
				takerBalance:     new(big.Int).Mul(big.NewInt(500), common.BasePrice),
				makerPrice:       common.BasePrice,
				makerFeeRate:     common.Big0,
				makerBalance:     new(big.Int).Mul(big.NewInt(100), common.BasePrice),
				baseTokenDecimal: common.BasePrice,
				quantityToTrade:  new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			new(big.Int).Mul(big.NewInt(100), common.BasePrice),
			true,
		},
		{
			"SELL: feeRate = 0, price 1, quantity 1000, taker balance 0, maker balance 100 -> reject both taker, maker",
			GetTradeQuantityArg{
				takerSide:        tradingstate.Ask,
				takerFeeRate:     common.Big0,
				takerBalance:     common.Big0,
				makerPrice:       common.BasePrice,
				makerFeeRate:     common.Big0,
				makerBalance:     new(big.Int).Mul(big.NewInt(100), common.BasePrice),
				baseTokenDecimal: common.BasePrice,
				quantityToTrade:  new(big.Int).Mul(big.NewInt(1000), common.BasePrice),
			},
			common.Big0,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := GetTradeQuantity(tt.args.takerSide, tt.args.takerFeeRate, tt.args.takerBalance, tt.args.makerPrice, tt.args.makerFeeRate, tt.args.makerBalance, tt.args.baseTokenDecimal, tt.args.quantityToTrade)
			if !reflect.DeepEqual(got, tt.quantity) {
				t.Errorf("GetTradeQuantity() got = %v, quantity %v", got, tt.quantity)
			}
			if got1 != tt.rejectMaker {
				t.Errorf("GetTradeQuantity() got1 = %v, quantity %v", got1, tt.rejectMaker)
			}
		})
	}
}

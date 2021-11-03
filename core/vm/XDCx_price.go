package vm

import (
	"github.com/XinFinOrg/XDPoSChain/XDCx/tradingstate"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
)

const XDCXPriceNumberOfBytesReturn = 32

// XDCxPrice implements a pre-compile contract to get token price in XDCx

type XDCxLastPrice struct {
	tradingStateDB *tradingstate.TradingStateDB
}
type XDCxEpochPrice struct {
	tradingStateDB *tradingstate.TradingStateDB
}

func (t *XDCxLastPrice) RequiredGas(input []byte) uint64 {
	return params.XDCXPriceGas
}

func (t *XDCxLastPrice) Run(input []byte) ([]byte, error) {
	// input includes baseTokenAddress, quoteTokenAddress
	if t.tradingStateDB != nil && len(input) == 64 {
		base := common.BytesToAddress(input[12:32]) // 20 bytes from 13-32
		quote := common.BytesToAddress(input[44:])  // 20 bytes from 45-64
		price := t.tradingStateDB.GetLastPrice(tradingstate.GetTradingOrderBookHash(base, quote))
		if price != nil {
			log.Debug("Run GetLastPrice", "base", base.Hex(), "quote", quote.Hex(), "price", price)
			return common.LeftPadBytes(price.Bytes(), XDCXPriceNumberOfBytesReturn), nil
		}
	}
	return common.LeftPadBytes([]byte{}, XDCXPriceNumberOfBytesReturn), nil
}

func (t *XDCxLastPrice) SetTradingState(tradingStateDB *tradingstate.TradingStateDB) {
	if tradingStateDB != nil {
		t.tradingStateDB = tradingStateDB.Copy()
	} else {
		t.tradingStateDB = nil
	}
}

func (t *XDCxEpochPrice) RequiredGas(input []byte) uint64 {
	return params.XDCXPriceGas
}

func (t *XDCxEpochPrice) Run(input []byte) ([]byte, error) {
	// input includes baseTokenAddress, quoteTokenAddress
	if t.tradingStateDB != nil && len(input) == 64 {
		base := common.BytesToAddress(input[12:32]) // 20 bytes from 13-32
		quote := common.BytesToAddress(input[44:])  // 20 bytes from 45-64
		price := t.tradingStateDB.GetMediumPriceBeforeEpoch(tradingstate.GetTradingOrderBookHash(base, quote))
		if price != nil {
			log.Debug("Run GetEpochPrice", "base", base.Hex(), "quote", quote.Hex(), "price", price)
			return common.LeftPadBytes(price.Bytes(), XDCXPriceNumberOfBytesReturn), nil
		}
	}
	return common.LeftPadBytes([]byte{}, XDCXPriceNumberOfBytesReturn), nil
}

func (t *XDCxEpochPrice) SetTradingState(tradingStateDB *tradingstate.TradingStateDB) {
	if tradingStateDB != nil {
		t.tradingStateDB = tradingStateDB.Copy()
	} else {
		t.tradingStateDB = nil
	}
}

package XDCx

import (
	"encoding/json"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"math/big"
	"strconv"
	"time"

	"github.com/XinFinOrg/XDPoSChain/consensus"

	"fmt"

	"github.com/XinFinOrg/XDPoSChain/XDCx/tradingstate"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/log"
)

func (XDCx *XDCX) CommitOrder(header *types.Header, coinbase common.Address, chain consensus.ChainContext, statedb *state.StateDB, tradingStateDB *tradingstate.TradingStateDB, orderBook common.Hash, order *tradingstate.OrderItem) ([]map[string]string, []*tradingstate.OrderItem, error) {
	XDCxSnap := tradingStateDB.Snapshot()
	dbSnap := statedb.Snapshot()
	trades, rejects, err := XDCx.ApplyOrder(header, coinbase, chain, statedb, tradingStateDB, orderBook, order)
	if err != nil {
		tradingStateDB.RevertToSnapshot(XDCxSnap)
		statedb.RevertToSnapshot(dbSnap)
		return nil, nil, err
	}
	return trades, rejects, err
}

func (XDCx *XDCX) ApplyOrder(header *types.Header, coinbase common.Address, chain consensus.ChainContext, statedb *state.StateDB, tradingStateDB *tradingstate.TradingStateDB, orderBook common.Hash, order *tradingstate.OrderItem) ([]map[string]string, []*tradingstate.OrderItem, error) {
	var (
		rejects []*tradingstate.OrderItem
		trades  []map[string]string
		err     error
	)
	nonce := tradingStateDB.GetNonce(order.UserAddress.Hash())
	log.Debug("ApplyOrder", "addr", order.UserAddress, "statenonce", nonce, "ordernonce", order.Nonce)
	if big.NewInt(int64(nonce)).Cmp(order.Nonce) == -1 {
		log.Debug("ApplyOrder ErrNonceTooHigh", "nonce", order.Nonce)
		return nil, nil, ErrNonceTooHigh
	} else if big.NewInt(int64(nonce)).Cmp(order.Nonce) == 1 {
		log.Debug("ApplyOrder ErrNonceTooLow", "nonce", order.Nonce)
		return nil, nil, ErrNonceTooLow
	}
	// increase nonce
	log.Debug("ApplyOrder set nonce", "nonce", nonce+1, "addr", order.UserAddress.Hex(), "status", order.Status, "oldnonce", nonce)
	tradingStateDB.SetNonce(order.UserAddress.Hash(), nonce+1)
	XDCxSnap := tradingStateDB.Snapshot()
	dbSnap := statedb.Snapshot()
	defer func() {
		if err != nil {
			tradingStateDB.RevertToSnapshot(XDCxSnap)
			statedb.RevertToSnapshot(dbSnap)
		}
	}()

	if err := order.VerifyOrder(statedb); err != nil {
		rejects = append(rejects, order)
		return trades, rejects, nil
	}
	if order.Status == tradingstate.OrderStatusCancelled {
		err, reject := XDCx.ProcessCancelOrder(header, tradingStateDB, statedb, chain, coinbase, orderBook, order)
		if err != nil || reject {
			log.Debug("Reject cancelled order", "err", err)
			rejects = append(rejects, order)
		}
		return trades, rejects, nil
	}
	if order.Type != tradingstate.Market {
		if order.Price.Sign() == 0 || common.BigToHash(order.Price).Big().Cmp(order.Price) != 0 {
			log.Debug("Reject order price invalid", "price", order.Price)
			rejects = append(rejects, order)
			return trades, rejects, nil
		}
	}
	if order.Quantity.Sign() == 0 || common.BigToHash(order.Quantity).Big().Cmp(order.Quantity) != 0 {
		log.Debug("Reject order quantity invalid", "quantity", order.Quantity)
		rejects = append(rejects, order)
		return trades, rejects, nil
	}
	orderType := order.Type
	// if we do not use auto-increment orderid, we must set price slot to avoid conflict
	if orderType == tradingstate.Market {
		log.Debug("Process maket order", "side", order.Side, "quantity", order.Quantity, "price", order.Price)
		trades, rejects, err = XDCx.processMarketOrder(coinbase, chain, statedb, tradingStateDB, orderBook, order)
		if err != nil {
			log.Debug("Reject market order", "err", err, "order", tradingstate.ToJSON(order))
			trades = []map[string]string{}
			rejects = append(rejects, order)
		}
	} else {
		log.Debug("Process limit order", "side", order.Side, "quantity", order.Quantity, "price", order.Price)
		trades, rejects, err = XDCx.processLimitOrder(coinbase, chain, statedb, tradingStateDB, orderBook, order)
		if err != nil {
			log.Debug("Reject limit order", "err", err, "order", tradingstate.ToJSON(order))
			trades = []map[string]string{}
			rejects = append(rejects, order)
		}
	}

	return trades, rejects, nil
}

// processMarketOrder : process the market order
func (XDCx *XDCX) processMarketOrder(coinbase common.Address, chain consensus.ChainContext, statedb *state.StateDB, tradingStateDB *tradingstate.TradingStateDB, orderBook common.Hash, order *tradingstate.OrderItem) ([]map[string]string, []*tradingstate.OrderItem, error) {
	var (
		trades     []map[string]string
		newTrades  []map[string]string
		rejects    []*tradingstate.OrderItem
		newRejects []*tradingstate.OrderItem
		err        error
	)
	quantityToTrade := order.Quantity
	side := order.Side
	// speedup the comparison, do not assign because it is pointer
	zero := tradingstate.Zero
	if side == tradingstate.Bid {
		bestPrice, volume := tradingStateDB.GetBestAskPrice(orderBook)
		log.Debug("processMarketOrder ", "side", side, "bestPrice", bestPrice, "quantityToTrade", quantityToTrade, "volume", volume)
		for quantityToTrade.Cmp(zero) > 0 && bestPrice.Cmp(zero) > 0 {
			quantityToTrade, newTrades, newRejects, err = XDCx.processOrderList(coinbase, chain, statedb, tradingStateDB, tradingstate.Ask, orderBook, bestPrice, quantityToTrade, order)
			if err != nil {
				return nil, nil, err
			}
			trades = append(trades, newTrades...)
			rejects = append(rejects, newRejects...)
			bestPrice, volume = tradingStateDB.GetBestAskPrice(orderBook)
			log.Debug("processMarketOrder ", "side", side, "bestPrice", bestPrice, "quantityToTrade", quantityToTrade, "volume", volume)
		}
	} else {
		bestPrice, volume := tradingStateDB.GetBestBidPrice(orderBook)
		log.Debug("processMarketOrder ", "side", side, "bestPrice", bestPrice, "quantityToTrade", quantityToTrade, "volume", volume)
		for quantityToTrade.Cmp(zero) > 0 && bestPrice.Cmp(zero) > 0 {
			quantityToTrade, newTrades, newRejects, err = XDCx.processOrderList(coinbase, chain, statedb, tradingStateDB, tradingstate.Bid, orderBook, bestPrice, quantityToTrade, order)
			if err != nil {
				return nil, nil, err
			}
			trades = append(trades, newTrades...)
			rejects = append(rejects, newRejects...)
			bestPrice, volume = tradingStateDB.GetBestBidPrice(orderBook)
			log.Debug("processMarketOrder ", "side", side, "bestPrice", bestPrice, "quantityToTrade", quantityToTrade, "volume", volume)
		}
	}
	return trades, rejects, nil
}

// processLimitOrder : process the limit order, can change the quote
// If not care for performance, we should make a copy of quote to prevent further reference problem
func (XDCx *XDCX) processLimitOrder(coinbase common.Address, chain consensus.ChainContext, statedb *state.StateDB, tradingStateDB *tradingstate.TradingStateDB, orderBook common.Hash, order *tradingstate.OrderItem) ([]map[string]string, []*tradingstate.OrderItem, error) {
	var (
		trades     []map[string]string
		newTrades  []map[string]string
		rejects    []*tradingstate.OrderItem
		newRejects []*tradingstate.OrderItem
		err        error
	)
	quantityToTrade := order.Quantity
	side := order.Side
	price := order.Price

	// speedup the comparison, do not assign because it is pointer
	zero := tradingstate.Zero

	if side == tradingstate.Bid {
		minPrice, volume := tradingStateDB.GetBestAskPrice(orderBook)
		log.Debug("processLimitOrder ", "side", side, "minPrice", minPrice, "orderPrice", price, "volume", volume)
		for quantityToTrade.Cmp(zero) > 0 && price.Cmp(minPrice) >= 0 && minPrice.Cmp(zero) > 0 {
			log.Debug("Min price in asks tree", "price", minPrice.String())
			quantityToTrade, newTrades, newRejects, err = XDCx.processOrderList(coinbase, chain, statedb, tradingStateDB, tradingstate.Ask, orderBook, minPrice, quantityToTrade, order)
			if err != nil {
				return nil, nil, err
			}
			trades = append(trades, newTrades...)
			rejects = append(rejects, newRejects...)
			log.Debug("New trade found", "newTrades", newTrades, "quantityToTrade", quantityToTrade)
			minPrice, volume = tradingStateDB.GetBestAskPrice(orderBook)
			log.Debug("processLimitOrder ", "side", side, "minPrice", minPrice, "orderPrice", price, "volume", volume)
		}
	} else {
		maxPrice, volume := tradingStateDB.GetBestBidPrice(orderBook)
		log.Debug("processLimitOrder ", "side", side, "maxPrice", maxPrice, "orderPrice", price, "volume", volume)
		for quantityToTrade.Cmp(zero) > 0 && price.Cmp(maxPrice) <= 0 && maxPrice.Cmp(zero) > 0 {
			log.Debug("Max price in bids tree", "price", maxPrice.String())
			quantityToTrade, newTrades, newRejects, err = XDCx.processOrderList(coinbase, chain, statedb, tradingStateDB, tradingstate.Bid, orderBook, maxPrice, quantityToTrade, order)
			if err != nil {
				return nil, nil, err
			}
			trades = append(trades, newTrades...)
			rejects = append(rejects, newRejects...)
			log.Debug("New trade found", "newTrades", newTrades, "quantityToTrade", quantityToTrade)
			maxPrice, volume = tradingStateDB.GetBestBidPrice(orderBook)
			log.Debug("processLimitOrder ", "side", side, "maxPrice", maxPrice, "orderPrice", price, "volume", volume)
		}
	}
	if quantityToTrade.Cmp(zero) > 0 {
		orderId := tradingStateDB.GetNonce(orderBook)
		order.OrderID = orderId + 1
		order.Quantity = quantityToTrade
		tradingStateDB.SetNonce(orderBook, orderId+1)
		orderIdHash := common.BigToHash(new(big.Int).SetUint64(order.OrderID))
		tradingStateDB.InsertOrderItem(orderBook, orderIdHash, *order)
		log.Debug("After matching, order (unmatched part) is now added to tree", "side", order.Side, "order", order)
	}
	return trades, rejects, nil
}

// processOrderList : process the order list
func (XDCx *XDCX) processOrderList(coinbase common.Address, chain consensus.ChainContext, statedb *state.StateDB, tradingStateDB *tradingstate.TradingStateDB, side string, orderBook common.Hash, price *big.Int, quantityStillToTrade *big.Int, order *tradingstate.OrderItem) (*big.Int, []map[string]string, []*tradingstate.OrderItem, error) {
	quantityToTrade := tradingstate.CloneBigInt(quantityStillToTrade)
	log.Debug("Process matching between order and orderlist", "quantityToTrade", quantityToTrade)
	var (
		trades []map[string]string

		rejects []*tradingstate.OrderItem
	)
	for quantityToTrade.Sign() > 0 {
		orderId, amount, _ := tradingStateDB.GetBestOrderIdAndAmount(orderBook, price, side)
		var oldestOrder tradingstate.OrderItem
		if amount.Sign() > 0 {
			oldestOrder = tradingStateDB.GetOrder(orderBook, orderId)
		}
		log.Debug("found order ", "orderId ", orderId, "side", oldestOrder.Side, "amount", amount)
		if oldestOrder.Quantity == nil || oldestOrder.Quantity.Sign() == 0 && amount.Sign() == 0 {
			break
		}
		var (
			tradedQuantity    *big.Int
			maxTradedQuantity *big.Int
		)
		if quantityToTrade.Cmp(amount) <= 0 {
			maxTradedQuantity = tradingstate.CloneBigInt(quantityToTrade)
		} else {
			maxTradedQuantity = tradingstate.CloneBigInt(amount)
		}
		var quotePrice *big.Int
		if oldestOrder.QuoteToken.String() != common.XDCNativeAddress {
			quotePrice = tradingStateDB.GetLastPrice(tradingstate.GetTradingOrderBookHash(oldestOrder.QuoteToken, common.HexToAddress(common.XDCNativeAddress)))
			log.Debug("TryGet quotePrice QuoteToken/XDC", "quotePrice", quotePrice)
			if quotePrice == nil || quotePrice.Sign() == 0 {
				inversePrice := tradingStateDB.GetLastPrice(tradingstate.GetTradingOrderBookHash(common.HexToAddress(common.XDCNativeAddress), oldestOrder.QuoteToken))
				quoteTokenDecimal, err := XDCx.GetTokenDecimal(chain, statedb, oldestOrder.QuoteToken)
				if err != nil || quoteTokenDecimal.Sign() == 0 {
					return nil, nil, nil, fmt.Errorf("Fail to get tokenDecimal. Token: %v . Err: %v", oldestOrder.QuoteToken.String(), err)
				}
				log.Debug("TryGet inversePrice XDC/QuoteToken", "inversePrice", inversePrice)
				if inversePrice != nil && inversePrice.Sign() > 0 {
					quotePrice = new(big.Int).Mul(common.BasePrice, quoteTokenDecimal)
					quotePrice = new(big.Int).Div(quotePrice, inversePrice)
					log.Debug("TryGet quotePrice after get inversePrice XDC/QuoteToken", "quotePrice", quotePrice, "quoteTokenDecimal", quoteTokenDecimal)
				}
			}
		} else {
			quotePrice = common.BasePrice
		}
		tradedQuantity, rejectMaker, settleBalanceResult, err := XDCx.getTradeQuantity(quotePrice, coinbase, chain, statedb, order, &oldestOrder, maxTradedQuantity)
		if err != nil && err == tradingstate.ErrQuantityTradeTooSmall {
			if tradedQuantity.Cmp(maxTradedQuantity) == 0 {
				if quantityToTrade.Cmp(amount) == 0 { // reject Taker & maker
					rejects = append(rejects, order)
					quantityToTrade = tradingstate.Zero
					rejects = append(rejects, &oldestOrder)
					err = tradingStateDB.CancelOrder(orderBook, &oldestOrder)
					if err != nil {
						return nil, nil, nil, err
					}
					break
				} else if quantityToTrade.Cmp(amount) < 0 { // reject Taker
					rejects = append(rejects, order)
					quantityToTrade = tradingstate.Zero
					break
				} else { // reject maker
					rejects = append(rejects, &oldestOrder)
					err = tradingStateDB.CancelOrder(orderBook, &oldestOrder)
					if err != nil {
						return nil, nil, nil, err
					}
					continue
				}
			} else {
				if rejectMaker { // reject maker
					rejects = append(rejects, &oldestOrder)
					err = tradingStateDB.CancelOrder(orderBook, &oldestOrder)
					if err != nil {
						return nil, nil, nil, err
					}
					continue
				} else { // reject Taker
					rejects = append(rejects, order)
					quantityToTrade = tradingstate.Zero
					break
				}
			}
		} else if err != nil {
			return nil, nil, nil, err
		}
		if tradedQuantity.Sign() == 0 && !rejectMaker {
			log.Debug("Reject order Taker ", "tradedQuantity", tradedQuantity, "rejectMaker", rejectMaker)
			rejects = append(rejects, order)
			quantityToTrade = tradingstate.Zero
			break
		}
		if tradedQuantity.Sign() > 0 {
			quantityToTrade = tradingstate.Sub(quantityToTrade, tradedQuantity)
			tradingStateDB.SubAmountOrderItem(orderBook, orderId, price, tradedQuantity, side)
			tradingStateDB.SetLastPrice(orderBook, price)
			log.Debug("Update quantity for orderId", "orderId", orderId.Hex())
			log.Debug("TRADE", "orderBook", orderBook, "Taker price", price, "maker price", order.Price, "Amount", tradedQuantity, "orderId", orderId, "side", side)

			tradeRecord := make(map[string]string)
			tradeRecord[tradingstate.TradeTakerOrderHash] = order.Hash.Hex()
			tradeRecord[tradingstate.TradeMakerOrderHash] = oldestOrder.Hash.Hex()
			tradeRecord[tradingstate.TradeTimestamp] = strconv.FormatInt(time.Now().Unix(), 10)
			tradeRecord[tradingstate.TradeQuantity] = tradedQuantity.String()
			tradeRecord[tradingstate.TradeMakerExchange] = oldestOrder.ExchangeAddress.String()
			tradeRecord[tradingstate.TradeMaker] = oldestOrder.UserAddress.String()
			tradeRecord[tradingstate.TradeBaseToken] = oldestOrder.BaseToken.String()
			tradeRecord[tradingstate.TradeQuoteToken] = oldestOrder.QuoteToken.String()
			if settleBalanceResult != nil {
				tradeRecord[tradingstate.MakerFee] = settleBalanceResult.Maker.Fee.Text(10)
				tradeRecord[tradingstate.TakerFee] = settleBalanceResult.Taker.Fee.Text(10)
			}
			// maker price is actual price
			// Taker price is offer price
			// tradedPrice is always actual price
			tradeRecord[tradingstate.TradePrice] = oldestOrder.Price.String()
			tradeRecord[tradingstate.MakerOrderType] = oldestOrder.Type
			trades = append(trades, tradeRecord)

			oldAveragePrice, oldTotalQuantity := tradingStateDB.GetMediumPriceAndTotalAmount(orderBook)

			var newAveragePrice, newTotalQuantity *big.Int
			if oldAveragePrice == nil || oldAveragePrice.Sign() <= 0 || oldTotalQuantity == nil || oldTotalQuantity.Sign() <= 0 {
				newAveragePrice = price
				newTotalQuantity = tradedQuantity
			} else {
				//volume = price * quantity
				//=> price = volume /quantity
				// averagePrice = totalVolume / totalQuantity
				// averagePrice = (oldVolume + newTradeVolume) / (oldQuantity + newTradeQuantity)
				// FIXME: average price formula
				// https://user-images.githubusercontent.com/17243442/72722447-ecb83700-3bb0-11ea-9273-1c1028dbade0.jpg

				oldVolume := new(big.Int).Mul(oldAveragePrice, oldTotalQuantity)
				newTradeVolume := new(big.Int).Mul(price, tradedQuantity)
				newTotalQuantity = new(big.Int).Add(oldTotalQuantity, tradedQuantity)
				newAveragePrice = new(big.Int).Div(new(big.Int).Add(oldVolume, newTradeVolume), newTotalQuantity)
			}

			tradingStateDB.SetMediumPrice(orderBook, newAveragePrice, newTotalQuantity)
		}
		if rejectMaker {
			rejects = append(rejects, &oldestOrder)
			err := tradingStateDB.CancelOrder(orderBook, &oldestOrder)
			if err != nil {
				return nil, nil, nil, err
			}
		}
	}
	return quantityToTrade, trades, rejects, nil
}

func (XDCx *XDCX) getTradeQuantity(quotePrice *big.Int, coinbase common.Address, chain consensus.ChainContext, statedb *state.StateDB, takerOrder *tradingstate.OrderItem, makerOrder *tradingstate.OrderItem, quantityToTrade *big.Int) (*big.Int, bool, *tradingstate.SettleBalance, error) {
	baseTokenDecimal, err := XDCx.GetTokenDecimal(chain, statedb, makerOrder.BaseToken)
	if err != nil || baseTokenDecimal.Sign() == 0 {
		return tradingstate.Zero, false, nil, fmt.Errorf("Fail to get tokenDecimal. Token: %v . Err: %v", makerOrder.BaseToken.String(), err)
	}
	quoteTokenDecimal, err := XDCx.GetTokenDecimal(chain, statedb, makerOrder.QuoteToken)
	if err != nil || quoteTokenDecimal.Sign() == 0 {
		return tradingstate.Zero, false, nil, fmt.Errorf("Fail to get tokenDecimal. Token: %v . Err: %v", makerOrder.QuoteToken.String(), err)
	}
	if makerOrder.QuoteToken.String() == common.XDCNativeAddress {
		quotePrice = quoteTokenDecimal
	}
	if takerOrder.ExchangeAddress.String() == makerOrder.ExchangeAddress.String() {
		if err := tradingstate.CheckRelayerFee(takerOrder.ExchangeAddress, new(big.Int).Mul(common.RelayerFee, big.NewInt(2)), statedb); err != nil {
			log.Debug("Reject order Taker Exchnage = Maker Exchange , relayer not enough fee ", "err", err)
			return tradingstate.Zero, false, nil, nil
		}
	} else {
		if err := tradingstate.CheckRelayerFee(takerOrder.ExchangeAddress, common.RelayerFee, statedb); err != nil {
			log.Debug("Reject order Taker , relayer not enough fee ", "err", err)
			return tradingstate.Zero, false, nil, nil
		}
		if err := tradingstate.CheckRelayerFee(makerOrder.ExchangeAddress, common.RelayerFee, statedb); err != nil {
			log.Debug("Reject order maker , relayer not enough fee ", "err", err)
			return tradingstate.Zero, true, nil, nil
		}
	}
	takerFeeRate := tradingstate.GetExRelayerFee(takerOrder.ExchangeAddress, statedb)
	makerFeeRate := tradingstate.GetExRelayerFee(makerOrder.ExchangeAddress, statedb)
	var takerBalance, makerBalance *big.Int
	switch takerOrder.Side {
	case tradingstate.Bid:
		takerBalance = tradingstate.GetTokenBalance(takerOrder.UserAddress, makerOrder.QuoteToken, statedb)
		makerBalance = tradingstate.GetTokenBalance(makerOrder.UserAddress, makerOrder.BaseToken, statedb)
	case tradingstate.Ask:
		takerBalance = tradingstate.GetTokenBalance(takerOrder.UserAddress, makerOrder.BaseToken, statedb)
		makerBalance = tradingstate.GetTokenBalance(makerOrder.UserAddress, makerOrder.QuoteToken, statedb)
	default:
		takerBalance = big.NewInt(0)
		makerBalance = big.NewInt(0)
	}
	quantity, rejectMaker := GetTradeQuantity(takerOrder.Side, takerFeeRate, takerBalance, makerOrder.Price, makerFeeRate, makerBalance, baseTokenDecimal, quantityToTrade)
	log.Debug("GetTradeQuantity", "side", takerOrder.Side, "takerBalance", takerBalance, "makerBalance", makerBalance, "BaseToken", makerOrder.BaseToken, "QuoteToken", makerOrder.QuoteToken, "quantity", quantity, "rejectMaker", rejectMaker, "quotePrice", quotePrice)
	var settleBalanceResult *tradingstate.SettleBalance
	if quantity.Sign() > 0 {
		// Apply Match Order
		settleBalanceResult, err = tradingstate.GetSettleBalance(quotePrice, takerOrder.Side, takerFeeRate, makerOrder.BaseToken, makerOrder.QuoteToken, makerOrder.Price, makerFeeRate, baseTokenDecimal, quoteTokenDecimal, quantity)
		log.Debug("GetSettleBalance", "settleBalanceResult", settleBalanceResult, "err", err)
		if err == nil {
			err = DoSettleBalance(coinbase, takerOrder, makerOrder, settleBalanceResult, statedb)
		}
		return quantity, rejectMaker, settleBalanceResult, err
	}
	return quantity, rejectMaker, settleBalanceResult, nil
}

func GetTradeQuantity(takerSide string, takerFeeRate *big.Int, takerBalance *big.Int, makerPrice *big.Int, makerFeeRate *big.Int, makerBalance *big.Int, baseTokenDecimal *big.Int, quantityToTrade *big.Int) (*big.Int, bool) {
	if takerSide == tradingstate.Bid {
		// maker InQuantity quoteTokenQuantity=(quantityToTrade*maker.Price/baseTokenDecimal)
		quoteTokenQuantity := new(big.Int).Mul(quantityToTrade, makerPrice)
		quoteTokenQuantity = new(big.Int).Div(quoteTokenQuantity, baseTokenDecimal)
		// Fee
		// charge on the token he/she has before the trade, in this case: quoteToken
		// charge on the token he/she has before the trade, in this case: baseToken
		// takerFee = quoteTokenQuantity*takerFeeRate/baseFee=(quantityToTrade*maker.Price/baseTokenDecimal) * makerFeeRate/baseFee
		takerFee := big.NewInt(0).Mul(quoteTokenQuantity, takerFeeRate)
		takerFee = big.NewInt(0).Div(takerFee, common.XDCXBaseFee)
		//takerOutTotal= quoteTokenQuantity + takerFee =  quantityToTrade*maker.Price/baseTokenDecimal + quantityToTrade*maker.Price/baseTokenDecimal * takerFeeRate/baseFee
		// = quantityToTrade *  maker.Price/baseTokenDecimal ( 1 +  takerFeeRate/baseFee)
		// = quantityToTrade * maker.Price * (baseFee + takerFeeRate ) / ( baseTokenDecimal * baseFee)
		takerOutTotal := new(big.Int).Add(quoteTokenQuantity, takerFee)
		makerOutTotal := quantityToTrade
		if takerBalance.Cmp(takerOutTotal) >= 0 && makerBalance.Cmp(makerOutTotal) >= 0 {
			return quantityToTrade, false
		} else if takerBalance.Cmp(takerOutTotal) < 0 && makerBalance.Cmp(makerOutTotal) >= 0 {
			newQuantityTrade := new(big.Int).Mul(takerBalance, baseTokenDecimal)
			newQuantityTrade = new(big.Int).Mul(newQuantityTrade, common.XDCXBaseFee)
			newQuantityTrade = new(big.Int).Div(newQuantityTrade, new(big.Int).Add(common.XDCXBaseFee, takerFeeRate))
			newQuantityTrade = new(big.Int).Div(newQuantityTrade, makerPrice)
			if newQuantityTrade.Sign() == 0 {
				log.Debug("Reject order Taker , not enough balance ", "takerSide", takerSide, "takerBalance", takerBalance, "takerOutTotal", takerOutTotal)
			}
			return newQuantityTrade, false
		} else if takerBalance.Cmp(takerOutTotal) >= 0 && makerBalance.Cmp(makerOutTotal) < 0 {
			log.Debug("Reject order maker , not enough balance ", "makerBalance", makerBalance, " makerOutTotal", makerOutTotal)
			return makerBalance, true
		} else {
			// takerBalance.Cmp(takerOutTotal) < 0 && makerBalance.Cmp(makerOutTotal) < 0
			newQuantityTrade := new(big.Int).Mul(takerBalance, baseTokenDecimal)
			newQuantityTrade = new(big.Int).Mul(newQuantityTrade, common.XDCXBaseFee)
			newQuantityTrade = new(big.Int).Div(newQuantityTrade, new(big.Int).Add(common.XDCXBaseFee, takerFeeRate))
			newQuantityTrade = new(big.Int).Div(newQuantityTrade, makerPrice)
			if newQuantityTrade.Cmp(makerBalance) <= 0 {
				if newQuantityTrade.Sign() == 0 {
					log.Debug("Reject order Taker , not enough balance ", "takerSide", takerSide, "takerBalance", takerBalance, "makerBalance", makerBalance, " newQuantityTrade ", newQuantityTrade)
				}
				return newQuantityTrade, false
			}
			log.Debug("Reject order maker , not enough balance ", "takerSide", takerSide, "takerBalance", takerBalance, "makerBalance", makerBalance, " newQuantityTrade ", newQuantityTrade)
			return makerBalance, true
		}
	} else {
		// Taker InQuantity
		// quoteTokenQuantity = quantityToTrade * makerPrice / baseTokenDecimal
		quoteTokenQuantity := new(big.Int).Mul(quantityToTrade, makerPrice)
		quoteTokenQuantity = new(big.Int).Div(quoteTokenQuantity, baseTokenDecimal)
		// maker InQuantity

		// Fee
		// charge on the token he/she has before the trade, in this case: baseToken
		// makerFee = quoteTokenQuantity * makerFeeRate / baseFee = quantityToTrade * makerPrice / baseTokenDecimal * makerFeeRate / baseFee
		// charge on the token he/she has before the trade, in this case: quoteToken
		makerFee := new(big.Int).Mul(quoteTokenQuantity, makerFeeRate)
		makerFee = new(big.Int).Div(makerFee, common.XDCXBaseFee)

		takerOutTotal := quantityToTrade
		// makerOutTotal = quoteTokenQuantity + makerFee  = quantityToTrade * makerPrice / baseTokenDecimal + quantityToTrade * makerPrice / baseTokenDecimal * makerFeeRate / baseFee
		// =  quantityToTrade * makerPrice / baseTokenDecimal * (1+makerFeeRate / baseFee)
		// = quantityToTrade  * makerPrice * (baseFee + makerFeeRate) / ( baseTokenDecimal * baseFee )
		makerOutTotal := new(big.Int).Add(quoteTokenQuantity, makerFee)
		if takerBalance.Cmp(takerOutTotal) >= 0 && makerBalance.Cmp(makerOutTotal) >= 0 {
			return quantityToTrade, false
		} else if takerBalance.Cmp(takerOutTotal) < 0 && makerBalance.Cmp(makerOutTotal) >= 0 {
			if takerBalance.Sign() == 0 {
				log.Debug("Reject order Taker , not enough balance ", "takerSide", takerSide, "takerBalance", takerBalance, "takerOutTotal", takerOutTotal)
			}
			return takerBalance, false
		} else if takerBalance.Cmp(takerOutTotal) >= 0 && makerBalance.Cmp(makerOutTotal) < 0 {
			newQuantityTrade := new(big.Int).Mul(makerBalance, baseTokenDecimal)
			newQuantityTrade = new(big.Int).Mul(newQuantityTrade, common.XDCXBaseFee)
			newQuantityTrade = new(big.Int).Div(newQuantityTrade, new(big.Int).Add(common.XDCXBaseFee, makerFeeRate))
			newQuantityTrade = new(big.Int).Div(newQuantityTrade, makerPrice)
			log.Debug("Reject order maker , not enough balance ", "makerBalance", makerBalance, " makerOutTotal", makerOutTotal)
			return newQuantityTrade, true
		} else {
			// takerBalance.Cmp(takerOutTotal) < 0 && makerBalance.Cmp(makerOutTotal) < 0
			newQuantityTrade := new(big.Int).Mul(makerBalance, baseTokenDecimal)
			newQuantityTrade = new(big.Int).Mul(newQuantityTrade, common.XDCXBaseFee)
			newQuantityTrade = new(big.Int).Div(newQuantityTrade, new(big.Int).Add(common.XDCXBaseFee, makerFeeRate))
			newQuantityTrade = new(big.Int).Div(newQuantityTrade, makerPrice)
			if newQuantityTrade.Cmp(takerBalance) <= 0 {
				log.Debug("Reject order maker , not enough balance ", "takerSide", takerSide, "takerBalance", takerBalance, "makerBalance", makerBalance, " newQuantityTrade ", newQuantityTrade)
				return newQuantityTrade, true
			}
			if takerBalance.Sign() == 0 {
				log.Debug("Reject order Taker , not enough balance ", "takerSide", takerSide, "takerBalance", takerBalance, "makerBalance", makerBalance, " newQuantityTrade ", newQuantityTrade)
			}
			return takerBalance, false
		}
	}
}

func DoSettleBalance(coinbase common.Address, takerOrder, makerOrder *tradingstate.OrderItem, settleBalance *tradingstate.SettleBalance, statedb *state.StateDB) error {
	takerExOwner := tradingstate.GetRelayerOwner(takerOrder.ExchangeAddress, statedb)
	makerExOwner := tradingstate.GetRelayerOwner(makerOrder.ExchangeAddress, statedb)
	matchingFee := big.NewInt(0)
	// masternodes charges fee of both 2 relayers. If maker and Taker are on same relayer, that relayer is charged fee twice
	matchingFee = new(big.Int).Add(matchingFee, common.RelayerFee)
	matchingFee = new(big.Int).Add(matchingFee, common.RelayerFee)

	if common.EmptyHash(takerExOwner.Hash()) || common.EmptyHash(makerExOwner.Hash()) {
		return fmt.Errorf("Echange owner empty , Taker: %v , maker : %v ", takerExOwner, makerExOwner)
	}
	mapBalances := map[common.Address]map[common.Address]*big.Int{}
	//Checking balance
	newTakerInTotal, err := tradingstate.CheckAddTokenBalance(takerOrder.UserAddress, settleBalance.Taker.InTotal, settleBalance.Taker.InToken, statedb, mapBalances)
	if err != nil {
		return err
	}
	if mapBalances[settleBalance.Taker.InToken] == nil {
		mapBalances[settleBalance.Taker.InToken] = map[common.Address]*big.Int{}
	}
	mapBalances[settleBalance.Taker.InToken][takerOrder.UserAddress] = newTakerInTotal
	newTakerOutTotal, err := tradingstate.CheckSubTokenBalance(takerOrder.UserAddress, settleBalance.Taker.OutTotal, settleBalance.Taker.OutToken, statedb, mapBalances)
	if err != nil {
		return err
	}
	if mapBalances[settleBalance.Taker.OutToken] == nil {
		mapBalances[settleBalance.Taker.OutToken] = map[common.Address]*big.Int{}
	}
	mapBalances[settleBalance.Taker.OutToken][takerOrder.UserAddress] = newTakerOutTotal
	newMakerInTotal, err := tradingstate.CheckAddTokenBalance(makerOrder.UserAddress, settleBalance.Maker.InTotal, settleBalance.Maker.InToken, statedb, mapBalances)
	if err != nil {
		return err
	}
	if mapBalances[settleBalance.Maker.InToken] == nil {
		mapBalances[settleBalance.Maker.InToken] = map[common.Address]*big.Int{}
	}
	mapBalances[settleBalance.Maker.InToken][makerOrder.UserAddress] = newMakerInTotal
	newMakerOutTotal, err := tradingstate.CheckSubTokenBalance(makerOrder.UserAddress, settleBalance.Maker.OutTotal, settleBalance.Maker.OutToken, statedb, mapBalances)
	if err != nil {
		return err
	}
	if mapBalances[settleBalance.Maker.OutToken] == nil {
		mapBalances[settleBalance.Maker.OutToken] = map[common.Address]*big.Int{}
	}
	mapBalances[settleBalance.Maker.OutToken][makerOrder.UserAddress] = newMakerOutTotal
	newTakerFee, err := tradingstate.CheckAddTokenBalance(takerExOwner, settleBalance.Taker.Fee, makerOrder.QuoteToken, statedb, mapBalances)
	if err != nil {
		return err
	}
	if mapBalances[makerOrder.QuoteToken] == nil {
		mapBalances[makerOrder.QuoteToken] = map[common.Address]*big.Int{}
	}
	mapBalances[makerOrder.QuoteToken][takerExOwner] = newTakerFee
	newMakerFee, err := tradingstate.CheckAddTokenBalance(makerExOwner, settleBalance.Maker.Fee, makerOrder.QuoteToken, statedb, mapBalances)
	if err != nil {
		return err
	}
	mapBalances[makerOrder.QuoteToken][makerExOwner] = newMakerFee

	mapRelayerFee := map[common.Address]*big.Int{}
	newRelayerTakerFee, err := tradingstate.CheckSubRelayerFee(takerOrder.ExchangeAddress, common.RelayerFee, statedb, mapRelayerFee)
	if err != nil {
		return err
	}
	mapRelayerFee[takerOrder.ExchangeAddress] = newRelayerTakerFee
	newRelayerMakerFee, err := tradingstate.CheckSubRelayerFee(makerOrder.ExchangeAddress, common.RelayerFee, statedb, mapRelayerFee)
	if err != nil {
		return err
	}
	mapRelayerFee[makerOrder.ExchangeAddress] = newRelayerMakerFee
	tradingstate.SetSubRelayerFee(takerOrder.ExchangeAddress, newRelayerTakerFee, common.RelayerFee, statedb)
	tradingstate.SetSubRelayerFee(makerOrder.ExchangeAddress, newRelayerMakerFee, common.RelayerFee, statedb)

	masternodeOwner := statedb.GetOwner(coinbase)
	statedb.AddBalance(masternodeOwner, matchingFee)

	tradingstate.SetTokenBalance(takerOrder.UserAddress, newTakerInTotal, settleBalance.Taker.InToken, statedb)
	tradingstate.SetTokenBalance(takerOrder.UserAddress, newTakerOutTotal, settleBalance.Taker.OutToken, statedb)

	tradingstate.SetTokenBalance(makerOrder.UserAddress, newMakerInTotal, settleBalance.Maker.InToken, statedb)
	tradingstate.SetTokenBalance(makerOrder.UserAddress, newMakerOutTotal, settleBalance.Maker.OutToken, statedb)

	// add balance for relayers
	//log.Debug("ApplyXDCXMatchedTransaction settle fee for relayers",
	//	"takerRelayerOwner", takerExOwner,
	//	"takerFeeToken", quoteToken, "takerFee", settleBalanceResult[takerAddr][XDCx.Fee].(*big.Int),
	//	"makerRelayerOwner", makerExOwner,
	//	"makerFeeToken", quoteToken, "makerFee", settleBalanceResult[makerAddr][XDCx.Fee].(*big.Int))
	// takerFee
	tradingstate.SetTokenBalance(takerExOwner, newTakerFee, makerOrder.QuoteToken, statedb)
	tradingstate.SetTokenBalance(makerExOwner, newMakerFee, makerOrder.QuoteToken, statedb)
	return nil
}

func (XDCx *XDCX) ProcessCancelOrder(header *types.Header, tradingStateDB *tradingstate.TradingStateDB, statedb *state.StateDB, chain consensus.ChainContext, coinbase common.Address, orderBook common.Hash, order *tradingstate.OrderItem) (error, bool) {
	if err := tradingstate.CheckRelayerFee(order.ExchangeAddress, common.RelayerCancelFee, statedb); err != nil {
		log.Debug("Relayer not enough fee when cancel order", "err", err)
		return nil, true
	}
	baseTokenDecimal, err := XDCx.GetTokenDecimal(chain, statedb, order.BaseToken)
	if err != nil || baseTokenDecimal.Sign() == 0 {
		log.Debug("Fail to get tokenDecimal ", "Token", order.BaseToken.String(), "err", err)
		return err, false
	}
	// order: basic order information (includes orderId, orderHash, baseToken, quoteToken) which user send to XDCx to cancel order
	// originOrder: full order information getting from order trie
	originOrder := tradingStateDB.GetOrder(orderBook, common.BigToHash(new(big.Int).SetUint64(order.OrderID)))
	if originOrder == tradingstate.EmptyOrder {
		return fmt.Errorf("order not found. OrderId: %v. Base: %s. Quote: %s", order.OrderID, order.BaseToken.Hex(), order.QuoteToken.Hex()), false
	}
	var tokenBalance *big.Int
	switch originOrder.Side {
	case tradingstate.Ask:
		tokenBalance = tradingstate.GetTokenBalance(originOrder.UserAddress, originOrder.BaseToken, statedb)
	case tradingstate.Bid:
		tokenBalance = tradingstate.GetTokenBalance(originOrder.UserAddress, originOrder.QuoteToken, statedb)
	default:
		log.Debug("Not found order side", "Side", originOrder.Side)
		return nil, false
	}
	log.Debug("ProcessCancelOrder", "baseToken", originOrder.BaseToken, "quoteToken", originOrder.QuoteToken)
	feeRate := tradingstate.GetExRelayerFee(originOrder.ExchangeAddress, statedb)
	tokenCancelFee, tokenPriceInXDC := common.Big0, common.Big0
	if !chain.Config().IsTIPXDCXCancellationFee(header.Number) {
		tokenCancelFee = getCancelFeeV1(baseTokenDecimal, feeRate, &originOrder)
	} else {
		tokenCancelFee, tokenPriceInXDC = XDCx.getCancelFee(chain, statedb, tradingStateDB, &originOrder, feeRate)
	}
	if tokenBalance.Cmp(tokenCancelFee) < 0 {
		log.Debug("User not enough balance when cancel order", "Side", originOrder.Side, "balance", tokenBalance, "fee", tokenCancelFee)
		return nil, true
	}

	err = tradingStateDB.CancelOrder(orderBook, order)
	if err != nil {
		log.Debug("Error when cancel order", "order", order)
		return err, false
	}
	// relayers pay XDC for masternode
	tradingstate.SubRelayerFee(originOrder.ExchangeAddress, common.RelayerCancelFee, statedb)
	masternodeOwner := statedb.GetOwner(coinbase)
	// relayers pay XDC for masternode
	statedb.AddBalance(masternodeOwner, common.RelayerCancelFee)

	relayerOwner := tradingstate.GetRelayerOwner(originOrder.ExchangeAddress, statedb)
	switch originOrder.Side {
	case tradingstate.Ask:
		// users pay token (which they have) for relayer
		tradingstate.SubTokenBalance(originOrder.UserAddress, tokenCancelFee, originOrder.BaseToken, statedb)
		tradingstate.AddTokenBalance(relayerOwner, tokenCancelFee, originOrder.BaseToken, statedb)
	case tradingstate.Bid:
		// users pay token (which they have) for relayer
		tradingstate.SubTokenBalance(originOrder.UserAddress, tokenCancelFee, originOrder.QuoteToken, statedb)
		tradingstate.AddTokenBalance(relayerOwner, tokenCancelFee, originOrder.QuoteToken, statedb)
	default:
	}
	// update cancel fee
	extraData, _ := json.Marshal(struct {
		CancelFee       string
		TokenPriceInXDC string
	}{
		CancelFee:       tokenCancelFee.Text(10),
		TokenPriceInXDC: tokenPriceInXDC.Text(10),
	})
	order.ExtraData = string(extraData)

	return nil, false
}

// cancellation fee = 1/10 trading fee
// deprecated after hardfork at TIPXDCXCancellationFee
func getCancelFeeV1(baseTokenDecimal *big.Int, feeRate *big.Int, order *tradingstate.OrderItem) *big.Int {
	cancelFee := big.NewInt(0)
	if order.Side == tradingstate.Ask {
		// SELL 1 BTC => XDC ,,
		// order.Quantity =1 && fee rate =2
		// ==> cancel fee = 2/10000
		// order.Quantity already included baseToken decimal
		cancelFee = new(big.Int).Mul(order.Quantity, feeRate)
		cancelFee = new(big.Int).Div(cancelFee, common.XDCXBaseCancelFee)
	} else {
		// BUY 1 BTC => XDC with Price : 10000
		// quoteTokenQuantity = 10000 && fee rate =2
		// => cancel fee =2
		quoteTokenQuantity := new(big.Int).Mul(order.Quantity, order.Price)
		quoteTokenQuantity = new(big.Int).Div(quoteTokenQuantity, baseTokenDecimal)
		// Fee
		// makerFee = quoteTokenQuantity * feeRate / baseFee = quantityToTrade * makerPrice / baseTokenDecimal * feeRate / baseFee
		cancelFee = new(big.Int).Mul(quoteTokenQuantity, feeRate)
		cancelFee = new(big.Int).Div(cancelFee, common.XDCXBaseCancelFee)
	}
	return cancelFee
}

// return tokenQuantity, tokenPriceInXDC
func (XDCx *XDCX) getCancelFee(chain consensus.ChainContext, statedb *state.StateDB, tradingStateDb *tradingstate.TradingStateDB, order *tradingstate.OrderItem, feeRate *big.Int) (*big.Int, *big.Int) {
	if feeRate == nil || feeRate.Sign() == 0 {
		return common.Big0, common.Big0
	}
	cancelFee := big.NewInt(0)
	tokenPriceInXDC := big.NewInt(0)
	var err error
	if order.Side == tradingstate.Ask {
		cancelFee, tokenPriceInXDC, err = XDCx.ConvertXDCToToken(chain, statedb, tradingStateDb, order.BaseToken, common.RelayerCancelFee)
	} else {
		cancelFee, tokenPriceInXDC, err = XDCx.ConvertXDCToToken(chain, statedb, tradingStateDb, order.QuoteToken, common.RelayerCancelFee)
	}
	if err != nil {
		return common.Big0, common.Big0
	}
	return cancelFee, tokenPriceInXDC
}

func (XDCx *XDCX) UpdateMediumPriceBeforeEpoch(epochNumber uint64, tradingStateDB *tradingstate.TradingStateDB, statedb *state.StateDB) error {
	mapPairs, err := tradingstate.GetAllTradingPairs(statedb)
	log.Debug("UpdateMediumPriceBeforeEpoch", "len(mapPairs)", len(mapPairs))

	if err != nil {
		return err
	}
	epochPriceResult := map[common.Hash]*big.Int{}
	for orderbook := range mapPairs {
		oldMediumPriceBeforeEpoch := tradingStateDB.GetMediumPriceBeforeEpoch(orderbook)
		mediumPriceCurrent, _ := tradingStateDB.GetMediumPriceAndTotalAmount(orderbook)

		// if there is no trade in this epoch, use average price of last epoch
		epochPriceResult[orderbook] = oldMediumPriceBeforeEpoch
		if mediumPriceCurrent.Sign() > 0 && mediumPriceCurrent.Cmp(oldMediumPriceBeforeEpoch) != 0 {
			log.Debug("UpdateMediumPriceBeforeEpoch", "mediumPriceCurrent", mediumPriceCurrent)
			tradingStateDB.SetMediumPriceBeforeEpoch(orderbook, mediumPriceCurrent)
			epochPriceResult[orderbook] = mediumPriceCurrent
		}
		tradingStateDB.SetMediumPrice(orderbook, tradingstate.Zero, tradingstate.Zero)
	}
	if XDCx.IsSDKNode() {
		if err := XDCx.LogEpochPrice(epochNumber, epochPriceResult); err != nil {
			log.Error("failed to update epochPrice", "err", err)
		}
	}
	return nil
}

// put average price of epoch to mongodb for tracking liquidation trades
// epochPriceResult: a map of epoch average price, key is orderbook hash , value is epoch average price
// orderbook hash genereted from baseToken, quoteToken at XDPoSChain/XDCx/tradingstate/common.go:214
func (XDCx *XDCX) LogEpochPrice(epochNumber uint64, epochPriceResult map[common.Hash]*big.Int) error {
	db := XDCx.GetMongoDB()
	db.InitBulk()

	for orderbook, price := range epochPriceResult {
		if price.Sign() <= 0 {
			continue
		}
		epochPriceItem := &tradingstate.EpochPriceItem{
			Epoch:     epochNumber,
			Orderbook: orderbook,
			Price:     price,
		}
		epochPriceItem.Hash = epochPriceItem.ComputeHash()
		if err := db.PutObject(epochPriceItem.Hash, epochPriceItem); err != nil {
			return err
		}
	}
	if err := db.CommitBulk(); err != nil {
		return err
	}
	return nil
}

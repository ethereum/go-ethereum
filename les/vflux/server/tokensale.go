// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package server

import (
	"math"
	"math/big"
	"math/bits"
	"sync"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/les/vflux"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
)

// tokenSale handles vflux requests for currency deposit and token buy/sell/price query operations
type tokenSale struct {
	bt                              *balanceTracker
	ndb                             *nodeDB
	clock                           mclock.Clock
	lock                            sync.Mutex
	lastUpdate                      mclock.AbsTime
	bc                              bondingCurve
	maxLimit                        int64
	limitAdjustRate, bpAdjustFactor float64 // per nanosecond
	minLogBasePrice                 utils.Fixed64
	currencyId                      string
	paymentReceivers                map[string]paymentReceiver
}

// paymentReceiver connects the payment receiver to the token sale module
type paymentReceiver interface {
	Deposit(batch ethdb.Batch, address, data []byte) (amount *big.Int, reply []byte)
}

// newTokenSale creates a new tokenSale module
func newTokenSale(bt *balanceTracker, ndb *nodeDB, clock mclock.Clock, currencyId string, minBasePrice float64) *tokenSale {
	minLogBasePrice := utils.Float64ToFixed64(math.Log2(minBasePrice))
	return &tokenSale{
		bt:               bt,
		ndb:              ndb,
		clock:            clock,
		lastUpdate:       clock.Now(),
		bc:               newBondingCurve(linHyperCurve, 1, minLogBasePrice),
		currencyId:       currencyId,
		minLogBasePrice:  minLogBasePrice,
		paymentReceivers: make(map[string]paymentReceiver),
	}
}

// addPaymentReceiver adds a new payment receiver to the token sale module
func (t *tokenSale) addPaymentReceiver(id string, pm paymentReceiver) {
	t.lock.Lock()
	t.paymentReceivers[id] = pm
	t.lock.Unlock()
}

// adjustCurve tries to adjust the bonding curve scaling parameters according to
// the current adjust rates. Should be called before using the bonding curve or
// changing the adjust rate parameters.
func (t *tokenSale) adjustCurve() {
	now := t.clock.Now()
	dt := now - t.lastUpdate
	t.lastUpdate = now

	targetLimit := t.bc.tokenLimit + int64(t.limitAdjustRate*float64(dt))
	if targetLimit > t.maxLimit {
		targetLimit = t.maxLimit
	}
	bpAdjust := t.bpAdjustFactor * (float64(t.bc.tokenAmount)/float64(t.bc.tokenLimit)*2 - 1)
	targetLogBasePrice := t.bc.logBasePrice + utils.Fixed64(bpAdjust*float64(dt))
	if targetLogBasePrice < t.minLogBasePrice {
		targetLogBasePrice = t.minLogBasePrice
	}
	t.bc.adjust(int64(t.bt.TotalTokenAmount()), targetLimit, targetLogBasePrice)
}

// setLimitAdjustRate sets a linear per-nanosecond adjust rate and a max cap for the token limit
func (t *tokenSale) setLimitAdjustRate(maxLimit int64, limitAdjustRate float64) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.adjustCurve()
	t.maxLimit = maxLimit
	t.limitAdjustRate = limitAdjustRate
}

// setBasePriceAdjustFactor sets the base price adjust factor
// Note: the exponential adjust rate is calculated as bpAdjustFactor*(tokenAmount/tokenLimit*2-1)
func (t *tokenSale) setBasePriceAdjustFactor(bpAdjustFactor float64) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.adjustCurve()
	t.bpAdjustFactor = bpAdjustFactor * float64(utils.Float64ToFixed64(1)) * math.Log(2)
}

// handle handles a vflux request
func (t *tokenSale) handle(id enode.ID, address string, name string, data []byte) []byte {
	switch name {
	case vflux.GetBalanceName:
		return t.serveGetBalance(id, data)
	case vflux.PriceQueryName:
		return t.servePriceQuery(id, data)
	case vflux.DepositName:
		return t.serveDeposit(id, data)
	case vflux.ExchangeName:
		return t.serveExchange(id, data)
	}
	return nil
}

// serveGetBalance serves a vflux token/currency balance request
func (t *tokenSale) serveGetBalance(id enode.ID, data []byte) []byte {
	var (
		req    vflux.GetBalanceRequest
		result vflux.GetBalanceReply
	)
	if rlp.DecodeBytes(data, &req) != nil {
		return nil
	}

	t.bt.BalanceOperation(id, "", nil, func(balance AtomicBalanceOperator) {
		tokenBalance, _ := balance.GetBalance()
		result.TokenBalance.SetInt64(int64(tokenBalance))
		if req.CurrencyId == t.currencyId && req.VerifySignature(id) {
			currencyBalance := t.ndb.getCurrencyBalance(req.CurrencyId, req.PaymentAddress)
			result.CurrencyBalance.SetBigInt(currencyBalance)
			result.LastSerial = balance.SerialNumber()
		} else {
			// if no valid currency ID and account are specified or the request is not
			// signed with the account key then a minus infinity currency balance is returned.
			result.CurrencyBalance.SetInf(-1)
		}
	})
	res, _ := rlp.EncodeToBytes(&result)
	return res
}

// servePriceQuery serves a vflux price query request
func (t *tokenSale) servePriceQuery(id enode.ID, data []byte) []byte {
	var req vflux.PriceQueryRequest
	if rlp.DecodeBytes(data, &req) != nil || req.CurrencyId != t.currencyId {
		return nil
	}
	if l := len(req.TokenAmounts); l == 0 || l > vflux.PriceQueryMaxLen {
		return nil
	}

	result := make(vflux.PriceQueryReply, len(req.TokenAmounts))
	t.bt.BalanceOperation(id, "", nil, func(balance AtomicBalanceOperator) {
		t.lock.Lock()
		defer t.lock.Unlock()

		t.adjustCurve()
		tokenBalance, _ := balance.GetBalance()
		for i, a := range req.TokenAmounts {
			amount := a.Int64()
			if amount < -int64(tokenBalance) {
				// Note: since the token amounts decrease continuously and it is not
				// possible to do an exact query for selling all tokens, a sale query
				// for more than the client's token balance results in a valid answer
				// corresponding to the entire balance.
				amount = -int64(tokenBalance)
			}
			if price := t.bc.price(amount); price != nil {
				result[i].SetBigInt(price)
			} else {
				result[i].SetInf(1)
			}
		}
	})
	res, _ := rlp.EncodeToBytes(&result)
	return res
}

// serveDeposit serves a vflux currency deposit request
func (t *tokenSale) serveDeposit(id enode.ID, data []byte) []byte {
	var (
		req    vflux.DepositRequest
		result vflux.DepositReply
	)
	if rlp.DecodeBytes(data, &req) != nil || req.CurrencyId != t.currencyId {
		return nil
	}

	batch := t.ndb.db.NewBatch()
	var success bool
	t.bt.BalanceOperation(id, "", batch, func(balance AtomicBalanceOperator) {
		t.lock.Lock()
		defer t.lock.Unlock()

		currencyBalance := t.ndb.getCurrencyBalance(req.CurrencyId, req.PaymentAddress) // big.Int
		if pm := t.paymentReceivers[req.PaymentReceiver]; pm != nil {
			amount, data := pm.Deposit(batch, req.PaymentAddress, req.PaymentData)
			currencyBalance.Add(currencyBalance, amount)
			t.ndb.setOrDelCurrencyBalance(batch, req.CurrencyId, req.PaymentAddress, currencyBalance)
			result.PaymentResponse = data
		}
		result.Balance.SetBigInt(currencyBalance)
	})
	if success {
		batch.Write()
	}
	res, _ := rlp.EncodeToBytes(&result)
	return res
}

// serveDeposit serves a vflux currency deposit request
func (t *tokenSale) serveExchange(id enode.ID, data []byte) []byte {
	var (
		req    vflux.ExchangeRequest
		result vflux.ExchangeReply
	)
	if rlp.DecodeBytes(data, &req) != nil || req.CurrencyId != t.currencyId || !req.VerifySignature(id) {
		return nil
	}

	batch := t.ndb.db.NewBatch()
	var success bool
	t.bt.BalanceOperation(id, "", batch, func(balance AtomicBalanceOperator) {
		t.lock.Lock()
		defer t.lock.Unlock()

		t.adjustCurve()
		currencyBalance := t.ndb.getCurrencyBalance(req.CurrencyId, req.PaymentAddress)
		result.LastSerial = balance.SerialNumber()
		if req.SerialNumber <= result.LastSerial {
			return
		}
		tokenBalance, _ := balance.GetBalance()

		minTokens, maxTokens := req.MinTokens.Int64(), req.MaxTokens.Int64()
		if minTokens < -int64(tokenBalance) {
			minTokens = -int64(tokenBalance)
		}
		mci := req.MaxCurrency.Inf()
		if minTokens <= maxTokens && mci != -1 {
			maxCurrency := currencyBalance
			if mci == 0 {
				mc := req.MaxCurrency.BigInt()
				if maxCurrency.Cmp(mc) > 0 {
					maxCurrency = mc
				}
			}
			tokensEx, currencyEx := t.bc.exchange(minTokens, maxTokens, maxCurrency)
			if currencyEx != nil {
				success = true
				tokenBalance += uint64(tokensEx)
				currencyBalance.Add(currencyBalance, currencyEx)
				if tokenBalance < 0 || currencyBalance.Sign() == -1 {
					utils.Error("tokenSale.serveExchange: negative token/currency balance")
				}
				balance.AddBalance(tokensEx)
				balance.SetSerialNumber(req.SerialNumber)
				t.ndb.setOrDelCurrencyBalance(batch, req.CurrencyId, req.PaymentAddress, currencyBalance)
				result.TokensEx.SetInt64(tokensEx)
				result.CurrencyEx.SetBigInt(currencyEx)
			}
		}
		result.TokenBalance.SetInt64(int64(tokenBalance))
		result.CurrencyBalance.SetBigInt(currencyBalance)
	})
	if success {
		batch.Write()
	}
	res, _ := rlp.EncodeToBytes(&result)
	return res
}

const (
	curveLogSize            = 14 // log2 size of the bonding curve
	curveSize               = 1 << curveLogSize
	curveLogValueMultiplier = 60 // log2 of fixed point multiplier
)

// curve stores a scalable bonding curve in a piecewise linear representation.
// Indices correspond to total token amount (relative to tokenLimit) while values
// correspond to locked currency amount (scaled by basePrice).
// Note: the curve should always be superlinear (the delta between adjacent values
// should never decrease). This is enforced by the generator function.
type curve [curveSize + 1]uint64 // +1 point so that values can be interpolated in the whole [0; curveSize] range

// value calculates linear interpolated curve value at curveSize*a/b where a<=b and b>0
func (c *curve) value(a, b uint64) uint64 {
	if a > b || b == 0 {
		utils.Error("curve.value: a > b || b == 0")
	}
	if a == b {
		return (*c)[curveSize]
	}
	// calculate r = a/b * 2^64
	r, _ := bits.Div64(a, 0, b)
	// use the upper bits as the curve index
	pos := int(r >> (64 - curveLogSize))
	// linear value interpolation between curve[pos] and curve[pos+1]
	y0 := (*c)[pos]
	y1 := (*c)[pos+1]
	// use the lower bits of r as a sub-position to interpolate between y0 and y1
	// subPos = frac(curveSize*a/b) * 2^64
	subPos := r << curveLogSize
	// the higher 64 bit word of the product equals (y1-y0)*frac(curveSize*a/b)
	dy, _ := bits.Mul64(y1-y0, subPos)
	return y0 + dy

}

// genCumulativeCurve generates a guaranteed monotonic curve with values proportional to the
// funds locked by the bonding curve mechanism as a function of the tokenAmount/tokenLimit ratio.
// Note: for each i the generator function should return the delta of the curve value between
// i and i+1. The delta values are also expected to be monotonically increasing, thereby
// ensuring that the curve is superlinear and curve.value(a, b) * b always increases if b is
// decreased. The generated values are multiplied by 2^curveLogValueMultiplier for fixed-point
// representation. The last value gen(curveSize-1) is allowed to cause an overflow as long
// as the superlinear property is maintained. The function panics if any of these conditions
// are broken.
func genCumulativeCurve(gen func(int) float64) *curve {
	var (
		c              curve
		lastDelta, sum uint64
	)
	fixedMult := math.Pow(2, curveLogValueMultiplier)
	for i := 0; i < curveSize; i++ {
		// calculate the delta value and convert to fixed point uint64
		v := gen(i)
		var u uint64
		if math.IsInf(v, 1) {
			u = math.MaxUint64
		} else {
			v *= fixedMult
			if v < math.MaxUint64 {
				u = uint64(v)
			} else {
				u = math.MaxUint64
			}
		}
		// avoid sum overflow
		if a := math.MaxUint64 - sum; u > a {
			u = a
		}
		// check superlinearity
		if u < lastDelta {
			panic(nil)
		}
		lastDelta = u
		// use the sum as the next curve point
		sum += u
		c[i+1] = sum
	}
	return &c
}

// linHyperCurveGenerator generates the linear-hyperbolical
func linHyperCurveGenerator(i int) float64 {
	return vflux.LinHyperIntegral(float64(i)/(curveSize/2), float64(1)/(curveSize/2)) / 2
}

var linHyperCurve = genCumulativeCurve(linHyperCurveGenerator)

// bondingCurve implements an adjustable bonding curve pricing mechanism. The locked currency
// vs. issued token amount curve is adjustably scaled along both axes. The token amount range
// goes from 0 to tokenLimit while the currency amounts are multiplied by the base price which
// is represented as a fixed point base 2 logarithm. When tokenAmount is decreased (issued
// tokens are being spent) the reduction of the corresponding locked currencyAmount is returned
// as earnings for the server. Locked currency amount is represented as an int64 with an
// optional bit shift applied to the external big.Int currency representation.
type bondingCurve struct {
	curve                                   *curve
	tokenAmount, tokenLimit, currencyAmount int64 // tokenLimit > 0
	logBasePrice                            utils.Fixed64

	limitShift, basePriceShift uint // priceFactor = (tokenLimit << limitShift) * (basePrice << basePriceShift)
	priceShift                 uint // currencyAmount = (curve.value * priceFactor) >> priceShift
	currencyShift              uint // external currency representation = currencyAmount << currencyShift
}

// newBondingCurve creates a new bondingCurve with the given initial scaling parameters
func newBondingCurve(curve *curve, tokenLimit int64, logBasePrice utils.Fixed64) bondingCurve {
	if tokenLimit < 1 {
		tokenLimit = 1
	}
	b := bondingCurve{
		curve:        curve,
		tokenLimit:   tokenLimit,
		logBasePrice: logBasePrice,
	}
	b.updateCurrencyAmount()
	b.adjust(0, tokenLimit, logBasePrice) // sets currencyShift
	return b
}

// calcCurrencyAmount calculates the locked currency amount corresponding to the given
// tokenAmount and curve parameters.
func (bc *bondingCurve) calcCurrencyAmount(tokenAmount, tokenLimit int64, logBasePrice utils.Fixed64) int64 {
	if tokenAmount >= tokenLimit {
		return math.MaxInt64
	}
	// calculate the bonding curve value at the current amount/limit ratio (which is multiplied by 2^curveLogValueMultiplier)
	cv := bc.curve.value(uint64(tokenAmount), uint64(tokenLimit))
	shiftedLimit := uint64(tokenLimit) << bc.limitShift
	shiftedBasePrice := uint64((logBasePrice + utils.Uint64ToFixed64(uint64(bc.basePriceShift))).Pow2())
	priceFactor, _ := bits.Mul64(shiftedLimit, shiftedBasePrice)
	m, _ := bits.Mul64(cv, priceFactor)
	return int64(m >> bc.priceShift)
}

// updateCurrencyAmount calculates the locked currency amount corresponding to the current
// tokenAmount and curve parameters and stores the result in bc.currencyAmount. This function
// should be called after modifying any of the parameters in order to ensure consistency.
func (bc *bondingCurve) updateCurrencyAmount() {
	bc.currencyAmount = bc.calcCurrencyAmount(bc.tokenAmount, bc.tokenLimit, bc.logBasePrice)
	if bc.currencyAmount == math.MaxInt64 {
		utils.Error("bondingCurve.updateCurrencyAmount: overflow")
	}
}

// price calculates the current price of the given token amount. The price is returned
// in external big.Int representation (currencyShify applied). If the purchase is
// not possible (total token amount would approach tokenLimit very closely) then nil
// is returned. Since the token amounts decrease continuously and it is not possible
// to do an exact query for selling all tokens, a sale query for more than the total
// existing amount of tokens results in a valid answer corresponding to the total amount.
// Note: adjust should be called before price in order to ensure that the latest curve
// parameters and token spending are applied.
func (bc *bondingCurve) price(tokenAmount int64) *big.Int {
	newTokenAmount := bc.tokenAmount
	if tokenAmount > math.MaxInt64-newTokenAmount {
		newTokenAmount = math.MaxInt64
	} else if tokenAmount < -newTokenAmount {
		newTokenAmount = 0
	} else {
		newTokenAmount += tokenAmount
	}
	newCurrencyAmount := bc.calcCurrencyAmount(newTokenAmount, bc.tokenLimit, bc.logBasePrice)
	if newCurrencyAmount == math.MaxInt64 {
		return nil
	}
	c := big.NewInt(newCurrencyAmount - bc.currencyAmount)
	c.Lsh(c, bc.currencyShift)
	return c
}

// exchange performs a token buy or sell operation, ensuring that tokenAmount and currencyAmount
// are moving on the curve with the current scaling parameters applied. Only an exchange operation
// can increase either tokenAmount or currencyAmount. The delta of tokenAmount is the highest
// possible amount between minAmount and maxAmount while the delta of currenctAmount is not
// greater than maxCost. If  If the exchange is successful then both deltas are returned.
// Both maxCost and the resulting change of currency amount are external big.Int representations
// and therefore currencyShift is applied.
// Note: adjust should be called before exchange in order to ensure that the latest curve
// parameters and token spending are applied.
func (bc *bondingCurve) exchange(minAmount, maxAmount int64, maxCost *big.Int) (int64, *big.Int) {
	var (
		mcost int64
		mc    big.Int
	)
	mc.Rsh(maxCost, bc.currencyShift) // floor(maxCost / 2^currencyShift)
	if mc.IsInt64() {
		mcost = mc.Int64()
	} else {
		if mc.Sign() == 1 {
			mcost = math.MaxInt64
		} else {
			mcost = math.MinInt64
		}
	}
	if mcost >= math.MaxInt64-bc.currencyAmount {
		mcost = math.MaxInt64 - bc.currencyAmount - 1
	}
	oldTokenAmount := bc.tokenAmount
	if minAmount < -bc.tokenAmount {
		minAmount = -bc.tokenAmount
	}
	if maxAmount < -bc.tokenAmount {
		maxAmount = -bc.tokenAmount
	}
	if maxAmount >= bc.tokenLimit {
		maxAmount = bc.tokenLimit - 1
	}
	if minAmount > maxAmount {
		return 0, nil
	}
	if maxAmount < math.MaxInt64-bc.tokenAmount {
		if c := bc.calcCurrencyAmount(bc.tokenAmount+maxAmount, bc.tokenLimit, bc.logBasePrice); c-bc.currencyAmount <= mcost {
			bc.tokenAmount += maxAmount
			cost := big.NewInt(c - bc.currencyAmount)
			cost.Lsh(cost, bc.currencyShift)
			bc.currencyAmount = c
			return maxAmount, cost
		}
	}
	newCurrencyAmount := bc.currencyAmount + mcost
	bc.tokenAmount = reverseFunction(oldTokenAmount+minAmount, oldTokenAmount+maxAmount, false, func(i int64) int64 {
		if c := bc.calcCurrencyAmount(i, bc.tokenLimit, bc.logBasePrice); c != math.MaxInt64 {
			return c - newCurrencyAmount
		} else {
			return math.MaxInt64
		}
	})
	newCurrencyAmount = bc.calcCurrencyAmount(bc.tokenAmount, bc.tokenLimit, bc.logBasePrice)
	if cost := newCurrencyAmount - bc.currencyAmount; cost <= mcost {
		bc.currencyAmount = newCurrencyAmount
		c := big.NewInt(cost)
		c.Lsh(c, bc.currencyShift)
		return bc.tokenAmount - oldTokenAmount, c
	}
	bc.tokenAmount = oldTokenAmount
	return 0, nil
}

// adjust updates tokenAmount according to token spending and tries to adjust curve scaling
// parameters (tokenLimit or logBasePrice) to the specified target values. If currencyAmount
// is decreased either due to token spending or curve scaling adjustment then the decrease is
// returned as earnings (currencyShift applied). If the desired curve adjustment would increase
// currencyAmount then the adjustments are carried out partially (to the extent that token
// spending allows). In this case no earnings are returned while the curve parameters are
// gradually converged to the target values.
func (bc *bondingCurve) adjust(tokenAmount, targetLimit int64, targetLogBasePrice utils.Fixed64) (earned *big.Int, success bool) {
	if targetLimit < 1 {
		targetLimit = 1
	}
	if tokenAmount < bc.tokenAmount {
		// token amount can only be increased by purchase
		bc.tokenAmount = tokenAmount
	}
	// find a base price shift value that is safe for any logBasePrice between the current and target values
	lbp := bc.logBasePrice
	if targetLogBasePrice > lbp {
		lbp = targetLogBasePrice
	}
	bc.basePriceShift = uint((utils.Uint64ToFixed64(64) - 1 - lbp).ToUint64())
	// find a limit shift value that is safe for any tokenLimit between the current and target values
	tl := bc.tokenLimit
	if targetLimit > tl {
		tl = targetLimit
	}
	bc.limitShift = uint(bits.LeadingZeros64(uint64(tl)))

	// calculate the external (left shifted) representation of the original currencyAmount
	// in order to be able to calculate final earnings and ensure they are not negative
	oldShiftedAmount := big.NewInt(bc.currencyAmount)
	oldShiftedAmount.Lsh(oldShiftedAmount, bc.currencyShift)

	// calculate final bit shift applied to currency amount
	// Note: tokenLimit is shifted left by limitShift, base price multiplied by
	// 2^basePriceShift in order to achieve highest precision. The fixed point
	// curve values are multiplied by 2^curveLogValueMultiplier.
	// The final shift applied to priceFactor*curve.value yields the currency amount
	// (the external big.Int representation).
	if shift := int(128 - curveLogValueMultiplier - bc.limitShift - bc.basePriceShift); shift >= 0 {
		// shift left between internal and external currency representation
		bc.currencyShift = uint(shift)
		bc.priceShift = 0
	} else {
		// shift priceFactor*curve.value right when calculating internal representation
		bc.currencyShift = 0
		bc.priceShift = uint(-shift)
	}

	// shift oldShiftedAmount back with the updated currencyShift
	// currencyAmount should not go over maxCurrencyAmount in order to ensure that
	// the final earned amount is not negative
	mca := big.NewInt(0)
	mca.Rsh(oldShiftedAmount, bc.currencyShift)
	if !mca.IsInt64() {
		// should never happen; new shift is chosen so that it is safe anywhere between old and target parameters
		utils.Error("bondingCurve.adjust: maxCurrencyAmount out of int64 range")
	}
	maxCurrencyAmount := mca.Int64()

	oldTokenLimit := bc.tokenLimit
	oldLogBasePrice := bc.logBasePrice
	defer func() {
		// Note: earned amount is defined as the decrease of the locked currency amount (external shifted representation)
		earned = big.NewInt(bc.currencyAmount)
		earned.Lsh(earned, bc.currencyShift)
		earned.Sub(oldShiftedAmount, earned)
		if earned.Sign() < 0 {
			utils.Error("bondingCurve.adjust: earned amount is negative")
		}
	}()
	bc.tokenLimit = targetLimit
	bc.logBasePrice = targetLogBasePrice
	bc.currencyAmount = bc.calcCurrencyAmount(bc.tokenAmount, bc.tokenLimit, bc.logBasePrice)
	if bc.currencyAmount <= maxCurrencyAmount {
		// currency amount did not increase, all adjustments are fully performed
		success = true
		return
	}
	if targetLimit >= oldTokenLimit {
		// currency amount exceeded old value only because of base price increase; revert and do a partial increase
		bc.logBasePrice = oldLogBasePrice
		bc.partialBasePriceIncrease(targetLogBasePrice, maxCurrencyAmount)
		bc.updateCurrencyAmount()
		return
	}
	if targetLogBasePrice <= oldLogBasePrice {
		// currency amount exceeded old value only because of limit decrease; revert and do a partial decrease
		bc.tokenLimit = oldTokenLimit
		bc.partialLimitDecrease(targetLimit, maxCurrencyAmount)
		bc.updateCurrencyAmount()
		return
	}
	// both parameter updates contributed to the currency amount being increased;
	// revert base price increase and try only decreasing limit first
	bc.logBasePrice = oldLogBasePrice
	if bc.calcCurrencyAmount(bc.tokenAmount, bc.tokenLimit, bc.logBasePrice) > maxCurrencyAmount {
		// limit decrease alone is too much; revert it, do a partial decrease and return
		bc.tokenLimit = oldTokenLimit
		bc.partialLimitDecrease(targetLimit, maxCurrencyAmount)
		bc.updateCurrencyAmount()
		return
	}
	// limit adjustment was fully performed; do a partial base price increase and return
	bc.partialBasePriceIncrease(targetLogBasePrice, maxCurrencyAmount)
	bc.updateCurrencyAmount()
	return
}

func (bc *bondingCurve) partialLimitDecrease(targetLimit, maxCurrencyAmount int64) {
	bc.tokenLimit = reverseFunction(targetLimit, bc.tokenLimit, true, func(i int64) int64 {
		if c := bc.calcCurrencyAmount(bc.tokenAmount, i, bc.logBasePrice); c != math.MaxInt64 {
			return maxCurrencyAmount - c
		} else {
			return math.MinInt64
		}
	})
}

func (bc *bondingCurve) partialBasePriceIncrease(targetLogBasePrice utils.Fixed64, maxCurrencyAmount int64) {
	bc.logBasePrice = utils.Fixed64(reverseFunction(int64(bc.logBasePrice), int64(targetLogBasePrice), false, func(i int64) int64 {
		if c := bc.calcCurrencyAmount(bc.tokenAmount, bc.tokenLimit, utils.Fixed64(i)); c != math.MaxInt64 {
			return c - maxCurrencyAmount
		} else {
			return math.MaxInt64
		}

	}))
}

// errFn should increase monotonically
func reverseFunction(min, max int64, upper bool, errFn func(int64) int64) int64 {
	minErr := errFn(min)
	maxErr := errFn(max)
	if minErr >= 0 {
		return min
	}
	if maxErr <= 0 {
		return max
	}
	for min < max-1 {
		var d float64
		if minErr == math.MinInt64 || maxErr == math.MaxInt64 {
			d = 0.5
		} else {
			d = float64(minErr) / (float64(minErr) - float64(maxErr)) // minErr < 0, maxErr > 0
			if d < 0.01 {
				d = 0.01
			}
			if d > 0.99 {
				d = 0.99
			}
		}
		mid := min + int64(float64(max-min)*d)
		if mid <= min {
			mid = min + 1
		}
		if mid >= max {
			mid = max - 1
		}
		midErr := errFn(mid)
		if midErr == 0 {
			return mid
		}
		if midErr < 0 {
			min, minErr = mid, midErr
		} else {
			max, maxErr = mid, midErr
		}
	}
	if upper {
		return max
	} else {
		return min
	}
}

type dummyPaymentReceiver struct {
	db     ethdb.KeyValueStore
	prefix []byte
}

func NewDummyPaymentReceiver(db ethdb.KeyValueStore, prefix []byte) *dummyPaymentReceiver {
	return &dummyPaymentReceiver{
		db:     db,
		prefix: prefix,
	}
}

func (pm *dummyPaymentReceiver) Deposit(batch ethdb.Batch, address, data []byte) (amount *big.Int, reply []byte) {
	key := append(pm.prefix, address...)
	lastEnc, _ := pm.db.Get(key)
	lastAmount := new(big.Int)
	lastAmount.SetBytes(lastEnc)
	amount = new(big.Int)
	amount.SetBytes(data)
	amount.Sub(amount, lastAmount)
	if amount.Sign() > 0 {
		pm.db.Put(key, data)
		return amount, data
	}
	return new(big.Int), lastEnc
}

package lendingstate

import (
	"encoding/json"
	"errors"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/log"
	"math/big"
)

const DefaultFeeRate = 100 // 100 / XDCXBaseFee = 100 / 10000 = 1%
var (
	ErrQuantityTradeTooSmall  = errors.New("quantity trade too small")
	ErrInvalidCollateralPrice = errors.New("unable to retrieve price of this collateral. Please try another collateral")
)

type TradeResult struct {
	Fee      *big.Int
	InToken  common.Address
	InTotal  *big.Int
	OutToken common.Address
	OutTotal *big.Int
}
type LendingSettleBalance struct {
	Taker                  TradeResult
	Maker                  TradeResult
	CollateralLockedAmount *big.Int
}

func (settleBalance *LendingSettleBalance) String() string {
	jsonData, _ := json.Marshal(settleBalance)
	return string(jsonData)
}

func GetSettleBalance(isXDCXLendingFork bool,
	takerSide string,
	lendTokenXDCPrice,
	collateralPrice,
	depositRate,
	borrowFeeRate *big.Int,
	lendingToken,
	collateralToken common.Address,
	lendTokenDecimal,
	collateralTokenDecimal *big.Int,
	quantityToLend *big.Int) (*LendingSettleBalance, error) {
	log.Debug("GetSettleBalance", "takerSide", takerSide, "borrowFeeRate", borrowFeeRate, "lendingToken", lendingToken, "collateralToken", collateralToken, "quantityToLend", quantityToLend)
	if collateralPrice == nil || collateralPrice.Sign() <= 0 {
		return nil, ErrInvalidCollateralPrice
	}

	//use the defaultFee to validate small orders
	defaultFee := new(big.Int).Mul(quantityToLend, new(big.Int).SetUint64(DefaultFeeRate))
	defaultFee = new(big.Int).Div(defaultFee, common.XDCXBaseFee)

	var result *LendingSettleBalance
	//result = map[common.Address]map[string]interface{}{}

	if !isXDCXLendingFork {
		if takerSide == Borrowing {
			// taker = Borrower : takerOutTotal = CollateralLockedAmount = quantityToLend * collateral Token Decimal/ CollateralPrice  * deposit rate
			takerOutTotal := new(big.Int).Mul(quantityToLend, collateralTokenDecimal)
			takerOutTotal = new(big.Int).Mul(takerOutTotal, depositRate) // eg: depositRate = 150%
			takerOutTotal = new(big.Int).Div(takerOutTotal, big.NewInt(100))
			takerOutTotal = new(big.Int).Div(takerOutTotal, collateralPrice)
			// Fee
			// takerFee = quantityToLend*borrowFeeRate/baseFee
			takerFee := new(big.Int).Mul(quantityToLend, borrowFeeRate)
			takerFee = new(big.Int).Div(takerFee, common.XDCXBaseFee)

			if quantityToLend.Cmp(takerFee) <= 0 || quantityToLend.Cmp(defaultFee) <= 0 {
				log.Debug("quantity lending too small", "quantityToLend", quantityToLend, "takerFee", takerFee)
				return result, ErrQuantityTradeTooSmall
			}
			if lendingToken.String() != common.XDCNativeAddress && lendTokenXDCPrice != nil && lendTokenXDCPrice.Cmp(common.Big0) > 0 {
				exTakerReceivedFee := new(big.Int).Mul(takerFee, lendTokenXDCPrice)
				exTakerReceivedFee = new(big.Int).Div(exTakerReceivedFee, lendTokenDecimal)

				defaultFeeInXDC := new(big.Int).Mul(defaultFee, lendTokenXDCPrice)
				defaultFeeInXDC = new(big.Int).Div(defaultFeeInXDC, lendTokenDecimal)

				if (exTakerReceivedFee.Cmp(common.RelayerLendingFee) <= 0 && exTakerReceivedFee.Sign() > 0) || defaultFeeInXDC.Cmp(common.RelayerLendingFee) <= 0 {
					log.Debug("takerFee too small", "quantityToLend", quantityToLend, "takerFee", takerFee, "exTakerReceivedFee", exTakerReceivedFee, "borrowFeeRate", borrowFeeRate, "defaultFeeInXDC", defaultFeeInXDC)
					return result, ErrQuantityTradeTooSmall
				}
			} else if lendingToken.String() == common.XDCNativeAddress {
				exTakerReceivedFee := takerFee
				if (exTakerReceivedFee.Cmp(common.RelayerLendingFee) <= 0 && exTakerReceivedFee.Sign() > 0) || defaultFee.Cmp(common.RelayerLendingFee) <= 0 {
					log.Debug("takerFee too small", "quantityToLend", quantityToLend, "takerFee", takerFee, "exTakerReceivedFee", exTakerReceivedFee, "borrowFeeRate", borrowFeeRate, "defaultFee", defaultFee)
					return result, ErrQuantityTradeTooSmall
				}
			}
			result = &LendingSettleBalance{
				//Borrower
				Taker: TradeResult{
					Fee:      takerFee,
					InToken:  lendingToken,
					InTotal:  new(big.Int).Sub(quantityToLend, takerFee),
					OutToken: collateralToken,
					OutTotal: takerOutTotal,
				},
				// Investor : makerOutTotal = quantityToLend
				Maker: TradeResult{
					Fee:      common.Big0,
					InToken:  common.Address{},
					InTotal:  common.Big0,
					OutToken: lendingToken,
					OutTotal: quantityToLend,
				},
				CollateralLockedAmount: takerOutTotal,
			}
		} else {
			// maker =  Borrower : makerOutTotal = CollateralLockedAmount = quantityToLend * collateral Token Decimal / CollateralPrice  * deposit rate
			makerOutTotal := new(big.Int).Mul(quantityToLend, collateralTokenDecimal)
			makerOutTotal = new(big.Int).Mul(makerOutTotal, depositRate) // eg: depositRate = 150%
			makerOutTotal = new(big.Int).Div(makerOutTotal, big.NewInt(100))
			makerOutTotal = new(big.Int).Div(makerOutTotal, collateralPrice)
			// Fee
			makerFee := new(big.Int).Mul(quantityToLend, borrowFeeRate)
			makerFee = new(big.Int).Div(makerFee, common.XDCXBaseFee)
			if quantityToLend.Cmp(makerFee) <= 0 || quantityToLend.Cmp(defaultFee) <= 0 {
				log.Debug("quantity lending too small", "quantityToLend", quantityToLend, "makerFee", makerFee)
				return result, ErrQuantityTradeTooSmall
			}
			if lendingToken.String() != common.XDCNativeAddress && lendTokenXDCPrice != nil && lendTokenXDCPrice.Cmp(common.Big0) > 0 {
				exMakerReceivedFee := new(big.Int).Mul(makerFee, lendTokenXDCPrice)
				exMakerReceivedFee = new(big.Int).Div(exMakerReceivedFee, lendTokenDecimal)

				defaultFeeInXDC := new(big.Int).Mul(defaultFee, lendTokenXDCPrice)
				defaultFeeInXDC = new(big.Int).Div(defaultFeeInXDC, lendTokenDecimal)

				if (exMakerReceivedFee.Cmp(common.RelayerLendingFee) <= 0 && exMakerReceivedFee.Sign() > 0) || defaultFeeInXDC.Cmp(common.RelayerLendingFee) <= 0 {
					log.Debug("makerFee too small", "quantityToLend", quantityToLend, "makerFee", makerFee, "exMakerReceivedFee", exMakerReceivedFee, "borrowFeeRate", borrowFeeRate, "defaultFeeInXDC", defaultFeeInXDC)
					return result, ErrQuantityTradeTooSmall
				}
			} else if lendingToken.String() == common.XDCNativeAddress {
				exMakerReceivedFee := makerFee
				if (exMakerReceivedFee.Cmp(common.RelayerLendingFee) <= 0 && exMakerReceivedFee.Sign() > 0) || defaultFee.Cmp(common.RelayerLendingFee) <= 0 {
					log.Debug("makerFee too small", "quantityToLend", quantityToLend, "makerFee", makerFee, "exMakerReceivedFee", exMakerReceivedFee, "borrowFeeRate", borrowFeeRate, "defaultFee", defaultFee)
					return result, ErrQuantityTradeTooSmall
				}
			}
			result = &LendingSettleBalance{
				Taker: TradeResult{
					Fee:      common.Big0,
					InToken:  common.Address{},
					InTotal:  common.Big0,
					OutToken: lendingToken,
					OutTotal: quantityToLend,
				},
				Maker: TradeResult{
					Fee:      makerFee,
					InToken:  lendingToken,
					InTotal:  new(big.Int).Add(quantityToLend, makerFee),
					OutToken: collateralToken,
					OutTotal: makerOutTotal,
				},
				CollateralLockedAmount: makerOutTotal,
			}
		}
	} else {

		collateralQuantity := new(big.Int).Mul(quantityToLend, collateralTokenDecimal)
		collateralQuantity = new(big.Int).Mul(collateralQuantity, depositRate) // eg: depositRate = 150%
		collateralQuantity = new(big.Int).Div(collateralQuantity, big.NewInt(100))
		collateralQuantity = new(big.Int).Div(collateralQuantity, collateralPrice)

		borrowFee := new(big.Int).Mul(quantityToLend, borrowFeeRate)
		borrowFee = new(big.Int).Div(borrowFee, common.XDCXBaseFee)

		if quantityToLend.Cmp(borrowFee) <= 0 || quantityToLend.Cmp(defaultFee) <= 0 {
			log.Debug("quantity lending too small", "quantityToLend", quantityToLend, "borrowFee", borrowFee)
			return result, ErrQuantityTradeTooSmall
		}
		if lendingToken.String() != common.XDCNativeAddress && lendTokenXDCPrice != nil && lendTokenXDCPrice.Cmp(common.Big0) > 0 {
			// exReceivedFee: the fee amount which borrowingRelayer will receive
			exReceivedFee := new(big.Int).Mul(borrowFee, lendTokenXDCPrice)
			exReceivedFee = new(big.Int).Div(exReceivedFee, lendTokenDecimal)

			defaultFeeInXDC := new(big.Int).Mul(defaultFee, lendTokenXDCPrice)
			defaultFeeInXDC = new(big.Int).Div(defaultFeeInXDC, lendTokenDecimal)

			if (exReceivedFee.Cmp(common.RelayerLendingFee) <= 0 && exReceivedFee.Sign() > 0) || defaultFeeInXDC.Cmp(common.RelayerLendingFee) <= 0 {
				log.Debug("takerFee too small", "quantityToLend", quantityToLend, "borrowFee", borrowFee, "exReceivedFee", exReceivedFee, "borrowFeeRate", borrowFeeRate, "defaultFeeInXDC", defaultFeeInXDC)
				return result, ErrQuantityTradeTooSmall
			}
		} else if lendingToken.String() == common.XDCNativeAddress {
			exReceivedFee := borrowFee
			if (exReceivedFee.Cmp(common.RelayerLendingFee) <= 0 && exReceivedFee.Sign() > 0) || defaultFee.Cmp(common.RelayerLendingFee) <= 0 {
				log.Debug("takerFee too small", "quantityToLend", quantityToLend, "borrowFee", borrowFee, "exReceivedFee", exReceivedFee, "borrowFeeRate", borrowFeeRate, "defaultFee", defaultFee)
				return result, ErrQuantityTradeTooSmall
			}
		}
		borrowerReceivedQuantity := new(big.Int).Sub(quantityToLend, borrowFee)
		borrowerTradeResult := TradeResult{
			Fee:      borrowFee,
			InToken:  lendingToken,
			InTotal:  borrowerReceivedQuantity,
			OutToken: collateralToken,
			OutTotal: collateralQuantity,
		}
		investorTradeResult := TradeResult{
			Fee:      common.Big0,
			InToken:  common.Address{},
			InTotal:  common.Big0,
			OutToken: lendingToken,
			OutTotal: quantityToLend,
		}
		if takerSide == Borrowing {
			result = &LendingSettleBalance{
				Taker:                  borrowerTradeResult,
				Maker:                  investorTradeResult,
				CollateralLockedAmount: collateralQuantity,
			}
		} else {
			result = &LendingSettleBalance{
				Taker:                  investorTradeResult,
				Maker:                  borrowerTradeResult,
				CollateralLockedAmount: collateralQuantity,
			}
		}
	}

	return result, nil
}

// apr: annual percentage rate
// this function returns actual interest rate base on borrowing time and apr
// I = APR *(T + T1) / 2 / 365
// T: term
// T1: borrowingTime
func CalculateInterestRate(finalizeTime, liquidationTime, term uint64, apr uint64) *big.Int {
	startBorrowingTime := liquidationTime - term
	borrowingTime := finalizeTime - startBorrowingTime

	// the time interval which borrower have to pay interest
	// (T + T1) / 2
	timeToPayInterest := new(big.Int).Add(new(big.Int).SetUint64(term), new(big.Int).SetUint64(borrowingTime))
	timeToPayInterest = new(big.Int).Div(timeToPayInterest, new(big.Int).SetUint64(2))

	interestRate := new(big.Int).SetUint64(apr)
	interestRate = new(big.Int).Mul(interestRate, timeToPayInterest)
	interestRate = new(big.Int).Div(interestRate, new(big.Int).SetUint64(common.OneYear))
	return interestRate
}

func CalculateTotalRepayValue(finalizeTime, liquidationTime, term uint64, apr uint64, tradeAmount *big.Int) *big.Int {
	interestRate := CalculateInterestRate(finalizeTime, liquidationTime, term, apr)

	// interest 10%
	// user should send: 10 * common.BaseLendingInterest
	// decimal = common.BaseLendingInterest * 100
	baseInterestDecimal := new(big.Int).Mul(common.BaseLendingInterest, new(big.Int).SetUint64(100))
	paymentBalance := new(big.Int).Mul(tradeAmount, new(big.Int).Add(baseInterestDecimal, interestRate))
	paymentBalance = new(big.Int).Div(paymentBalance, baseInterestDecimal)
	return paymentBalance
}

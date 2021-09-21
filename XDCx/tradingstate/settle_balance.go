package tradingstate

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/log"
)

const DefaultFeeRate = 10 // 10 / XDCXBaseFee = 10 / 10000 = 0.1%
var ErrQuantityTradeTooSmall = errors.New("quantity trade too small")

type TradeResult struct {
	Fee      *big.Int
	InToken  common.Address
	InTotal  *big.Int
	OutToken common.Address
	OutTotal *big.Int
}
type SettleBalance struct {
	Taker TradeResult
	Maker TradeResult
}

func (settleBalance *SettleBalance) String() string {
	jsonData, _ := json.Marshal(settleBalance)
	return string(jsonData)
}

func GetSettleBalance(quotePrice *big.Int, takerSide string, takerFeeRate *big.Int, baseToken, quoteToken common.Address, makerPrice *big.Int, makerFeeRate *big.Int, baseTokenDecimal *big.Int, quoteTokenDecimal *big.Int, quantityToTrade *big.Int) (*SettleBalance, error) {
	log.Debug("GetSettleBalance", "takerSide", takerSide, "takerFeeRate", takerFeeRate, "baseToken", baseToken, "quoteToken", quoteToken, "makerPrice", makerPrice, "makerFeeRate", makerFeeRate, "baseTokenDecimal", baseTokenDecimal, "quantityToTrade", quantityToTrade, "quotePrice", quotePrice)
	var result *SettleBalance
	//result = map[common.Address]map[string]interface{}{}

	// quoteTokenQuantity = quantityToTrade * makerPrice / baseTokenDecimal
	quoteTokenQuantity := new(big.Int).Mul(quantityToTrade, makerPrice)
	quoteTokenQuantity = new(big.Int).Div(quoteTokenQuantity, baseTokenDecimal)

	makerFee := new(big.Int).Mul(quoteTokenQuantity, makerFeeRate)
	makerFee = new(big.Int).Div(makerFee, common.XDCXBaseFee)
	takerFee := new(big.Int).Mul(quoteTokenQuantity, takerFeeRate)
	takerFee = new(big.Int).Div(takerFee, common.XDCXBaseFee)

	// use the defaultFee to validate small orders
	defaultFee := new(big.Int).Mul(quoteTokenQuantity, new(big.Int).SetUint64(DefaultFeeRate))
	defaultFee = new(big.Int).Div(defaultFee, common.XDCXBaseFee)

	if takerSide == Bid {
		if quoteTokenQuantity.Cmp(makerFee) <= 0 || quoteTokenQuantity.Cmp(defaultFee) <= 0 {
			log.Debug("quantity trade too small", "quoteTokenQuantity", quoteTokenQuantity, "makerFee", makerFee, "defaultFee", defaultFee)
			return result, ErrQuantityTradeTooSmall
		}
		if quoteToken.String() != common.XDCNativeAddress && quotePrice != nil && quotePrice.Cmp(common.Big0) > 0 {
			// defaultFeeInXDC
			defaultFeeInXDC := new(big.Int).Mul(defaultFee, quotePrice)
			defaultFeeInXDC = new(big.Int).Div(defaultFeeInXDC, quoteTokenDecimal)

			exMakerReceivedFee := new(big.Int).Mul(makerFee, quotePrice)
			exMakerReceivedFee = new(big.Int).Div(exMakerReceivedFee, quoteTokenDecimal)
			if (exMakerReceivedFee.Cmp(common.RelayerFee) <= 0 && exMakerReceivedFee.Sign() > 0) || defaultFeeInXDC.Cmp(common.RelayerFee) <= 0 {
				log.Debug("makerFee too small", "quoteTokenQuantity", quoteTokenQuantity, "makerFee", makerFee, "exMakerReceivedFee", exMakerReceivedFee, "quotePrice", quotePrice, "defaultFeeInXDC", defaultFeeInXDC)
				return result, ErrQuantityTradeTooSmall
			}
			exTakerReceivedFee := new(big.Int).Mul(takerFee, quotePrice)
			exTakerReceivedFee = new(big.Int).Div(exTakerReceivedFee, quoteTokenDecimal)
			if (exTakerReceivedFee.Cmp(common.RelayerFee) <= 0 && exTakerReceivedFee.Sign() > 0) || defaultFeeInXDC.Cmp(common.RelayerFee) <= 0 {
				log.Debug("takerFee too small", "quoteTokenQuantity", quoteTokenQuantity, "takerFee", takerFee, "exTakerReceivedFee", exTakerReceivedFee, "quotePrice", quotePrice, "defaultFeeInXDC", defaultFeeInXDC)
				return result, ErrQuantityTradeTooSmall
			}
		} else if quoteToken.String() == common.XDCNativeAddress {
			exMakerReceivedFee := makerFee
			if (exMakerReceivedFee.Cmp(common.RelayerFee) <= 0 && exMakerReceivedFee.Sign() > 0) || defaultFee.Cmp(common.RelayerFee) <= 0 {
				log.Debug("makerFee too small", "quantityToTrade", quantityToTrade, "makerFee", makerFee, "exMakerReceivedFee", exMakerReceivedFee, "makerFeeRate", makerFeeRate, "defaultFee", defaultFee)
				return result, ErrQuantityTradeTooSmall
			}
			exTakerReceivedFee := takerFee
			if (exTakerReceivedFee.Cmp(common.RelayerFee) <= 0 && exTakerReceivedFee.Sign() > 0) || defaultFee.Cmp(common.RelayerFee) <= 0 {
				log.Debug("takerFee too small", "quantityToTrade", quantityToTrade, "takerFee", takerFee, "exTakerReceivedFee", exTakerReceivedFee, "takerFeeRate", takerFeeRate, "defaultFee", defaultFee)
				return result, ErrQuantityTradeTooSmall
			}
		}
		inTotal := new(big.Int).Sub(quoteTokenQuantity, makerFee)
		//takerOutTotal= quoteTokenQuantity + takerFee =  quantityToTrade*maker.Price/baseTokenDecimal + quantityToTrade*maker.Price/baseTokenDecimal * takerFeeRate/baseFee
		// = quantityToTrade *  maker.Price/baseTokenDecimal ( 1 +  takerFeeRate/baseFee)
		// = quantityToTrade * maker.Price * (baseFee + takerFeeRate ) / ( baseTokenDecimal * baseFee)
		takerOutTotal := new(big.Int).Add(quoteTokenQuantity, takerFee)

		result = &SettleBalance{
			Taker: TradeResult{
				Fee:      takerFee,
				InToken:  baseToken,
				InTotal:  quantityToTrade,
				OutToken: quoteToken,
				OutTotal: takerOutTotal,
			},
			Maker: TradeResult{
				Fee:      makerFee,
				InToken:  quoteToken,
				InTotal:  inTotal,
				OutToken: baseToken,
				OutTotal: quantityToTrade,
			},
		}
	} else {
		if quoteTokenQuantity.Cmp(takerFee) <= 0 || quoteTokenQuantity.Cmp(defaultFee) <= 0 {
			log.Debug("quantity trade too small", "quoteTokenQuantity", quoteTokenQuantity, "takerFee", takerFee)
			return result, ErrQuantityTradeTooSmall
		}
		if quoteToken.String() != common.XDCNativeAddress && quotePrice != nil && quotePrice.Cmp(common.Big0) > 0 {
			// defaultFeeInXDC
			defaultFeeInXDC := new(big.Int).Mul(defaultFee, quotePrice)
			defaultFeeInXDC = new(big.Int).Div(defaultFeeInXDC, quoteTokenDecimal)

			exMakerReceivedFee := new(big.Int).Mul(makerFee, quotePrice)
			exMakerReceivedFee = new(big.Int).Div(exMakerReceivedFee, quoteTokenDecimal)
			log.Debug("exMakerReceivedFee", "quoteTokenQuantity", quoteTokenQuantity, "makerFee", makerFee, "exMakerReceivedFee", exMakerReceivedFee, "quotePrice", quotePrice)
			if (exMakerReceivedFee.Cmp(common.RelayerFee) <= 0 && exMakerReceivedFee.Sign() > 0) || defaultFeeInXDC.Cmp(common.RelayerFee) <= 0 {
				log.Debug("makerFee too small", "quoteTokenQuantity", quoteTokenQuantity, "makerFee", makerFee, "exMakerReceivedFee", exMakerReceivedFee, "quotePrice", quotePrice, "defaultMakerFeeInXDC", defaultFeeInXDC)
				return result, ErrQuantityTradeTooSmall
			}
			exTakerReceivedFee := new(big.Int).Mul(takerFee, quotePrice)
			exTakerReceivedFee = new(big.Int).Div(exTakerReceivedFee, quoteTokenDecimal)
			if (exTakerReceivedFee.Cmp(common.RelayerFee) <= 0 && exTakerReceivedFee.Sign() > 0) || defaultFeeInXDC.Cmp(common.RelayerFee) <= 0 {
				log.Debug("takerFee too small", "quoteTokenQuantity", quoteTokenQuantity, "takerFee", takerFee, "exTakerReceivedFee", exTakerReceivedFee, "quotePrice", quotePrice, "defaultFeeInXDC", defaultFeeInXDC)
				return result, ErrQuantityTradeTooSmall
			}
		} else if quoteToken.String() == common.XDCNativeAddress {
			exMakerReceivedFee := makerFee
			if (exMakerReceivedFee.Cmp(common.RelayerFee) <= 0 && exMakerReceivedFee.Sign() > 0) || defaultFee.Cmp(common.RelayerFee) <= 0 {
				log.Debug("makerFee too small", "quantityToTrade", quantityToTrade, "makerFee", makerFee, "exMakerReceivedFee", exMakerReceivedFee, "makerFeeRate", makerFeeRate, "defaultFee", defaultFee)
				return result, ErrQuantityTradeTooSmall
			}
			exTakerReceivedFee := takerFee
			if (exTakerReceivedFee.Cmp(common.RelayerFee) <= 0 && exTakerReceivedFee.Sign() > 0) || defaultFee.Cmp(common.RelayerFee) <= 0 {
				log.Debug("takerFee too small", "quantityToTrade", quantityToTrade, "takerFee", takerFee, "exTakerReceivedFee", exTakerReceivedFee, "takerFeeRate", takerFeeRate, "defaultFee", defaultFee)
				return result, ErrQuantityTradeTooSmall
			}
		}
		inTotal := new(big.Int).Sub(quoteTokenQuantity, takerFee)
		// makerOutTotal = quoteTokenQuantity + makerFee  = quantityToTrade * makerPrice / baseTokenDecimal + quantityToTrade * makerPrice / baseTokenDecimal * makerFeeRate / baseFee
		// =  quantityToTrade * makerPrice / baseTokenDecimal * (1+makerFeeRate / baseFee)
		// = quantityToTrade  * makerPrice * (baseFee + makerFeeRate) / ( baseTokenDecimal * baseFee )
		makerOutTotal := new(big.Int).Add(quoteTokenQuantity, makerFee)
		// Fee
		result = &SettleBalance{
			Taker: TradeResult{
				Fee:      takerFee,
				InToken:  quoteToken,
				InTotal:  inTotal,
				OutToken: baseToken,
				OutTotal: quantityToTrade,
			},
			Maker: TradeResult{
				Fee:      makerFee,
				InToken:  baseToken,
				InTotal:  quantityToTrade,
				OutToken: quoteToken,
				OutTotal: makerOutTotal,
			},
		}
	}
	return result, nil
}

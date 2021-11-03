package lendingstate

import (
	"fmt"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto/sha3"
	"github.com/globalsign/mgo/bson"
	"math/big"
	"strconv"
	"time"
)

const (
	Investing                  = "INVEST"
	Borrowing                  = "BORROW"
	TopUp                      = "TOPUP"
	Repay                      = "REPAY"
	Recall                     = "RECALL"
	LendingStatusNew           = "NEW"
	LendingStatusOpen          = "OPEN"
	LendingStatusReject        = "REJECTED"
	LendingStatusFilled        = "FILLED"
	LendingStatusPartialFilled = "PARTIAL_FILLED"
	LendingStatusCancelled     = "CANCELLED"
	Market                     = "MO"
	Limit                      = "LO"
)

var ValidInputLendingStatus = map[string]bool{
	LendingStatusNew:       true,
	LendingStatusCancelled: true,
}

var ValidInputLendingType = map[string]bool{
	Market: true,
	Limit:  true,
	Repay:  true,
	TopUp:  true,
	Recall: true,
}

// Signature struct
type Signature struct {
	V byte        `bson:"v" json:"v"`
	R common.Hash `bson:"r" json:"r"`
	S common.Hash `bson:"s" json:"s"`
}

type SignatureRecord struct {
	V byte   `bson:"v" json:"v"`
	R string `bson:"r" json:"r"`
	S string `bson:"s" json:"s"`
}

type LendingItem struct {
	Quantity        *big.Int       `bson:"quantity" json:"quantity"`
	Interest        *big.Int       `bson:"interest" json:"interest"`
	Side            string         `bson:"side" json:"side"` // INVESTING/BORROWING
	Type            string         `bson:"type" json:"type"` // LIMIT/MARKET
	LendingToken    common.Address `bson:"lendingToken" json:"lendingToken"`
	CollateralToken common.Address `bson:"collateralToken" json:"collateralToken"`
	AutoTopUp       bool           `bson:"autoTopUp" json:"autoTopUp"`
	FilledAmount    *big.Int       `bson:"filledAmount" json:"filledAmount"`
	Status          string         `bson:"status" json:"status"`
	Relayer         common.Address `bson:"relayer" json:"relayer"`
	Term            uint64         `bson:"term" json:"term"`
	UserAddress     common.Address `bson:"userAddress" json:"userAddress"`
	Signature       *Signature     `bson:"signature" json:"signature"`
	Hash            common.Hash    `bson:"hash" json:"hash"`
	TxHash          common.Hash    `bson:"txHash" json:"txHash"`
	Nonce           *big.Int       `bson:"nonce" json:"nonce"`
	CreatedAt       time.Time      `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time      `bson:"updatedAt" json:"updatedAt"`
	LendingId       uint64         `bson:"lendingId" json:"lendingId"`
	LendingTradeId  uint64         `bson:"tradeId" json:"tradeId"`
	ExtraData       string         `bson:"extraData" json:"extraData"`
}

type LendingItemBSON struct {
	Quantity        string           `bson:"quantity" json:"quantity"`
	Interest        string           `bson:"interest" json:"interest"`
	Side            string           `bson:"side" json:"side"` // INVESTING/BORROWING
	Type            string           `bson:"type" json:"type"` // LIMIT/MARKET
	LendingToken    string           `bson:"lendingToken" json:"lendingToken"`
	CollateralToken string           `bson:"collateralToken" json:"collateralToken"`
	AutoTopUp       bool             `bson:"autoTopUp" json:"autoTopUp"`
	FilledAmount    string           `bson:"filledAmount" json:"filledAmount"`
	Status          string           `bson:"status" json:"status"`
	Relayer         string           `bson:"relayer" json:"relayer"`
	Term            string           `bson:"term" json:"term"`
	UserAddress     string           `bson:"userAddress" json:"userAddress"`
	Signature       *SignatureRecord `bson:"signature" json:"signature"`
	Hash            string           `bson:"hash" json:"hash"`
	TxHash          string           `bson:"txHash" json:"txHash"`
	Nonce           string           `bson:"nonce" json:"nonce"`
	CreatedAt       time.Time        `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time        `bson:"updatedAt" json:"updatedAt"`
	LendingId       string           `bson:"lendingId" json:"lendingId"`
	LendingTradeId  string           `bson:"tradeId" json:"tradeId"`
	ExtraData       string           `bson:"extraData" json:"extraData"`
}

func (l *LendingItem) GetBSON() (interface{}, error) {
	lr := LendingItemBSON{
		Quantity:        l.Quantity.String(),
		Interest:        l.Interest.String(),
		Side:            l.Side,
		Type:            l.Type,
		LendingToken:    l.LendingToken.Hex(),
		CollateralToken: l.CollateralToken.Hex(),
		AutoTopUp:       l.AutoTopUp,
		Status:          l.Status,
		Relayer:         l.Relayer.Hex(),
		Term:            strconv.FormatUint(l.Term, 10),
		UserAddress:     l.UserAddress.Hex(),
		Hash:            l.Hash.Hex(),
		TxHash:          l.TxHash.Hex(),
		Nonce:           l.Nonce.String(),
		CreatedAt:       l.CreatedAt,
		UpdatedAt:       l.UpdatedAt,
		LendingId:       strconv.FormatUint(l.LendingId, 10),
		LendingTradeId:  strconv.FormatUint(l.LendingTradeId, 10),
		ExtraData:       l.ExtraData,
	}

	if l.FilledAmount != nil {
		lr.FilledAmount = l.FilledAmount.String()
	}

	if l.Signature != nil {
		lr.Signature = &SignatureRecord{
			V: l.Signature.V,
			R: l.Signature.R.Hex(),
			S: l.Signature.S.Hex(),
		}
	}

	return lr, nil
}

func (l *LendingItem) SetBSON(raw bson.Raw) error {
	decoded := new(LendingItemBSON)

	err := raw.Unmarshal(decoded)
	if err != nil {
		return err
	}
	if decoded.Quantity != "" {
		l.Quantity = ToBigInt(decoded.Quantity)
	}
	l.Interest = ToBigInt(decoded.Interest)
	l.Side = decoded.Side
	l.Type = decoded.Type
	l.LendingToken = common.HexToAddress(decoded.LendingToken)
	l.CollateralToken = common.HexToAddress(decoded.CollateralToken)
	l.AutoTopUp = decoded.AutoTopUp
	l.FilledAmount = ToBigInt(decoded.FilledAmount)
	l.Status = decoded.Status
	l.Relayer = common.HexToAddress(decoded.Relayer)
	term, err := strconv.ParseInt(decoded.Term, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse lendingItem.term. Err: %v", err)
	}
	l.Term = uint64(term)
	l.UserAddress = common.HexToAddress(decoded.UserAddress)

	if decoded.Signature != nil {
		l.Signature = &Signature{
			V: byte(decoded.Signature.V),
			R: common.HexToHash(decoded.Signature.R),
			S: common.HexToHash(decoded.Signature.S),
		}
	}

	l.Hash = common.HexToHash(decoded.Hash)
	l.TxHash = common.HexToHash(decoded.TxHash)
	l.Nonce = ToBigInt(decoded.Nonce)

	l.CreatedAt = decoded.CreatedAt
	l.UpdatedAt = decoded.UpdatedAt
	lendingId, err := strconv.ParseInt(decoded.LendingId, 10, 64)
	if err != nil {
		return err
	}
	l.LendingId = uint64(lendingId)
	lendingTradeId, err := strconv.ParseInt(decoded.LendingTradeId, 10, 64)
	if err != nil {
		return err
	}
	l.LendingTradeId = uint64(lendingTradeId)
	l.ExtraData = decoded.ExtraData
	return nil
}

func (l *LendingItem) VerifyLendingItem(state *state.StateDB) error {
	if err := l.VerifyLendingStatus(); err != nil {
		return err
	}
	if valid, _ := IsValidPair(state, l.Relayer, l.LendingToken, l.Term); valid == false {
		return fmt.Errorf("invalid pair . LendToken %s . Term: %v", l.LendingToken.Hex(), l.Term)
	}
	if l.Status == LendingStatusNew {
		if err := l.VerifyLendingType(); err != nil {
			return err
		}
		if l.Type != Repay {
			if err := l.VerifyLendingQuantity(); err != nil {
				return err
			}
		}
		if l.Type == Limit || l.Type == Market {
			if err := l.VerifyLendingSide(); err != nil {
				return err
			}
			if l.Side == Borrowing {
				if err := l.VerifyCollateral(state); err != nil {
					return err
				}
			}
		}
		if l.Type == Limit {
			if err := l.VerifyLendingInterest(); err != nil {
				return err
			}
		}
	}
	if !IsValidRelayer(state, l.Relayer) {
		return fmt.Errorf("VerifyLendingItem: invalid relayer. address: %s", l.Relayer.Hex())
	}
	if err := l.VerifyLendingSignature(); err != nil {
		return err
	}
	return nil
}

func (l *LendingItem) VerifyLendingSide() error {
	if l.Side != Borrowing && l.Side != Investing {
		return fmt.Errorf("VerifyLendingSide: invalid side . Side: %s", l.Side)
	}
	return nil
}

func (l *LendingItem) VerifyCollateral(state *state.StateDB) error {
	if l.CollateralToken.String() == EmptyAddress || l.CollateralToken.String() == l.LendingToken.String() {
		return fmt.Errorf("invalid collateral %s", l.CollateralToken.Hex())
	}
	validCollateral := false
	collateralList, _ := GetCollaterals(state, l.Relayer, l.LendingToken, l.Term)
	for _, collateral := range collateralList {
		if l.CollateralToken.String() == collateral.String() {
			validCollateral = true
			break
		}
	}
	if !validCollateral {
		return fmt.Errorf("invalid collateral %s", l.CollateralToken.Hex())
	}
	return nil
}

func (l *LendingItem) VerifyLendingInterest() error {
	if l.Interest == nil || l.Interest.Sign() <= 0 {
		return fmt.Errorf("VerifyLendingInterest: invalid interest. Interest: %v", l.Interest)
	}
	return nil
}

func (l *LendingItem) VerifyLendingQuantity() error {
	if l.Quantity == nil || l.Quantity.Sign() <= 0 {
		return fmt.Errorf("VerifyLendingQuantity: invalid quantity. Quantity: %v", l.Quantity)
	}
	return nil
}

func (l *LendingItem) VerifyLendingType() error {
	if valid, ok := ValidInputLendingType[l.Type]; !ok && !valid {
		return fmt.Errorf("VerifyLendingType: invalid lending type. Type: %s", l.Type)
	}
	return nil
}

func (l *LendingItem) VerifyLendingStatus() error {
	if valid, ok := ValidInputLendingStatus[l.Status]; !ok && !valid {
		return fmt.Errorf("VerifyLendingStatus: invalid lending status. Status: %s", l.Status)
	}
	return nil
}

func (l *LendingItem) ComputeHash() common.Hash {
	sha := sha3.NewKeccak256()
	if l.Status == LendingStatusNew {
		sha.Write(l.Relayer.Bytes())
		sha.Write(l.UserAddress.Bytes())
		sha.Write(l.LendingToken.Bytes())
		sha.Write(l.CollateralToken.Bytes())
		sha.Write([]byte(strconv.FormatInt(int64(l.Term), 10)))
		sha.Write(common.BigToHash(l.Quantity).Bytes())
		if l.Type == Limit {
			if l.Interest != nil {
				sha.Write(common.BigToHash(l.Interest).Bytes())
			}
		}
		sha.Write(common.BigToHash(l.EncodedSide()).Bytes())
		sha.Write([]byte(l.Status))
		sha.Write([]byte(l.Type))
		sha.Write(common.BigToHash(l.Nonce).Bytes())
	} else if l.Status == LendingStatusCancelled {
		sha.Write(l.Hash.Bytes())
		sha.Write(common.BigToHash(l.Nonce).Bytes())
		sha.Write(l.UserAddress.Bytes())
		sha.Write(common.BigToHash(big.NewInt(int64(l.LendingId))).Bytes())
		sha.Write([]byte(l.Status))
		sha.Write(l.Relayer.Bytes())
		sha.Write(l.LendingToken.Bytes())
		sha.Write(l.CollateralToken.Bytes())
	} else {
		return common.Hash{}
	}

	return common.BytesToHash(sha.Sum(nil))
}

func (l *LendingItem) EncodedSide() *big.Int {
	if l.Side == Borrowing {
		return big.NewInt(0)
	}
	return big.NewInt(1)
}

//verify signatures
func (l *LendingItem) VerifyLendingSignature() error {
	V := big.NewInt(int64(l.Signature.V))
	R := l.Signature.R.Big()
	S := l.Signature.S.Big()

	//(nonce uint64, quantity *big.Int, interest, duration uint64, relayerAddress, userAddress, lendingToken, collateralToken common.Address, status, side, typeLending string, hash common.Hash, id uint64
	tx := types.NewLendingTransaction(l.Nonce.Uint64(), l.Quantity, l.Interest.Uint64(), l.Term, l.Relayer, l.UserAddress,
		l.LendingToken, l.CollateralToken, l.AutoTopUp, l.Status, l.Side, l.Type, l.Hash, l.LendingId, l.LendingTradeId, l.ExtraData)
	tx.ImportSignature(V, R, S)
	from, _ := types.LendingSender(types.LendingTxSigner{}, tx)
	if from != tx.UserAddress() {
		return fmt.Errorf("verify lending item: invalid signature")
	}
	return nil
}

func VerifyBalance(isXDCXLendingFork bool, statedb *state.StateDB, lendingStateDb *LendingStateDB,
	orderType, side, status string, userAddress, relayer, lendingToken, collateralToken common.Address,
	quantity, lendingTokenDecimal, collateralTokenDecimal, lendTokenXDCPrice, collateralPrice *big.Int,
	term uint64, lendingId uint64, lendingTradeId uint64) error {
	borrowingFeeRate := GetFee(statedb, relayer)
	switch orderType {
	case TopUp:
		lendingBook := GetLendingOrderBookHash(lendingToken, term)
		lendingTrade := lendingStateDb.GetLendingTrade(lendingBook, common.Uint64ToHash(lendingTradeId))
		if lendingTrade == EmptyLendingTrade {
			return fmt.Errorf("VerifyBalance: process deposit for emptyLendingTrade is not allowed. lendingTradeId: %v", lendingTradeId)
		}
		tokenBalance := GetTokenBalance(lendingTrade.Borrower, lendingTrade.CollateralToken, statedb)
		if tokenBalance.Cmp(quantity) < 0 {
			return fmt.Errorf("VerifyBalance: not enough balance to process deposit for lendingTrade."+
				"lendingTradeId: %v. Token: %s. ExpectedBalance: %s. ActualBalance: %s",
				lendingTradeId, lendingTrade.CollateralToken.Hex(), quantity.String(), tokenBalance.String())
		}
	case Repay:
		lendingBook := GetLendingOrderBookHash(lendingToken, term)
		lendingTrade := lendingStateDb.GetLendingTrade(lendingBook, common.Uint64ToHash(lendingTradeId))
		if lendingTrade == EmptyLendingTrade {
			return fmt.Errorf("VerifyBalance: process payment for emptyLendingTrade is not allowed. lendingTradeId: %v", lendingTradeId)
		}
		tokenBalance := GetTokenBalance(lendingTrade.Borrower, lendingTrade.LendingToken, statedb)
		paymentBalance := CalculateTotalRepayValue(uint64(time.Now().Unix()), lendingTrade.LiquidationTime, lendingTrade.Term, lendingTrade.Interest, lendingTrade.Amount)

		if tokenBalance.Cmp(paymentBalance) < 0 {
			return fmt.Errorf("VerifyBalance: not enough balance to process payment for lendingTrade."+
				"lendingTradeId: %v. Token: %s. ExpectedBalance: %s. ActualBalance: %s",
				lendingTradeId, lendingTrade.LendingToken.Hex(), paymentBalance.String(), tokenBalance.String())

		}
	case Market, Limit:
		switch side {
		case Investing:
			switch status {
			case LendingStatusNew:
				// make sure that investor have enough lendingToken
				if balance := GetTokenBalance(userAddress, lendingToken, statedb); balance.Cmp(quantity) < 0 {
					return fmt.Errorf("VerifyBalance: investor doesn't have enough lendingToken. User: %s. Token: %s. Expected: %v. Have: %v", userAddress.Hex(), lendingToken.Hex(), quantity, balance)
				}
				// check quantity: reject if it's too small
				if lendTokenXDCPrice != nil && lendTokenXDCPrice.Sign() > 0 {
					defaultFee := new(big.Int).Mul(quantity, new(big.Int).SetUint64(DefaultFeeRate))
					defaultFee = new(big.Int).Div(defaultFee, common.XDCXBaseFee)
					defaultFeeInXDC := common.Big0
					if lendingToken.String() != common.XDCNativeAddress {
						defaultFeeInXDC = new(big.Int).Mul(defaultFee, lendTokenXDCPrice)
						defaultFeeInXDC = new(big.Int).Div(defaultFeeInXDC, lendingTokenDecimal)
					} else {
						defaultFeeInXDC = defaultFee
					}
					if defaultFeeInXDC.Cmp(common.RelayerLendingFee) <= 0 {
						return ErrQuantityTradeTooSmall
					}

				}

			case LendingStatusCancelled:
				// in case of cancel, investor need to pay cancel fee in lendingToken
				// make sure actualBalance >= cancel fee
				lendingBook := GetLendingOrderBookHash(lendingToken, term)
				item := lendingStateDb.GetLendingOrder(lendingBook, common.BigToHash(new(big.Int).SetUint64(lendingId)))
				cancelFee := big.NewInt(0)
				cancelFee = new(big.Int).Mul(item.Quantity, borrowingFeeRate)
				cancelFee = new(big.Int).Div(cancelFee, common.XDCXBaseCancelFee)

				actualBalance := GetTokenBalance(userAddress, lendingToken, statedb)
				if actualBalance.Cmp(cancelFee) < 0 {
					return fmt.Errorf("VerifyBalance: investor doesn't have enough lendingToken to pay cancel fee. LendingToken: %s . ExpectedBalance: %s . ActualBalance: %s",
						lendingToken.Hex(), cancelFee.String(), actualBalance.String())
				}
			default:
				return fmt.Errorf("VerifyBalance: invalid status of investing lendingitem. Status: %s", status)
			}
			return nil
		case Borrowing:
			switch status {
			case LendingStatusNew:
				depositRate, _, _ := GetCollateralDetail(statedb, collateralToken)
				settleBalanceResult, err := GetSettleBalance(isXDCXLendingFork, Borrowing, lendTokenXDCPrice, collateralPrice, depositRate, borrowingFeeRate, lendingToken, collateralToken, lendingTokenDecimal, collateralTokenDecimal, quantity)
				if err != nil {
					return err
				}
				expectedBalance := settleBalanceResult.CollateralLockedAmount
				actualBalance := GetTokenBalance(userAddress, collateralToken, statedb)
				if actualBalance.Cmp(expectedBalance) < 0 {
					return fmt.Errorf("VerifyBalance: borrower doesn't have enough collateral token.  User: %s. CollateralToken: %s . ExpectedBalance: %s . ActualBalance: %s",
						userAddress.Hex(), collateralToken.Hex(), expectedBalance.String(), actualBalance.String())
				}
			case LendingStatusCancelled:
				lendingBook := GetLendingOrderBookHash(lendingToken, term)
				item := lendingStateDb.GetLendingOrder(lendingBook, common.BigToHash(new(big.Int).SetUint64(lendingId)))
				cancelFee := big.NewInt(0)
				// Fee ==  quantityToLend/base lend token decimal *price*borrowFee/LendingCancelFee
				cancelFee = new(big.Int).Div(item.Quantity, collateralPrice)
				cancelFee = new(big.Int).Mul(cancelFee, borrowingFeeRate)
				cancelFee = new(big.Int).Div(cancelFee, common.XDCXBaseCancelFee)
				actualBalance := GetTokenBalance(userAddress, collateralToken, statedb)
				if actualBalance.Cmp(cancelFee) < 0 {
					return fmt.Errorf("VerifyBalance: borrower doesn't have enough collateralToken to pay cancel fee. User: %s. CollateralToken: %s . ExpectedBalance: %s . ActualBalance: %s",
						userAddress.Hex(), lendingToken.Hex(), cancelFee.String(), actualBalance.String())
				}
			default:
				return fmt.Errorf("VerifyBalance: invalid status of borrowing lendingitem. Status: %s", status)
			}
			return nil
		default:
			return fmt.Errorf("VerifyBalance: unknown lending side")
		}
	default:
		return fmt.Errorf("VerifyBalance: unknown lending type")
	}
	return nil
}

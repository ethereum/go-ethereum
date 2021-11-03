package lendingstate

import (
	"fmt"
	"github.com/XinFinOrg/XDPoSChain/XDCx/tradingstate"
	"github.com/XinFinOrg/XDPoSChain/crypto/sha3"
	"math/big"
	"strconv"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/globalsign/mgo/bson"
)

const (
	TradeStatusOpen       = "OPEN"
	TradeStatusClosed     = "CLOSED"
	TradeStatusLiquidated = "LIQUIDATED"
)

type LendingTrade struct {
	Borrower               common.Address `bson:"borrower" json:"borrower"`
	Investor               common.Address `bson:"investor" json:"investor"`
	LendingToken           common.Address `bson:"lendingToken" json:"lendingToken"`
	CollateralToken        common.Address `bson:"collateralToken" json:"collateralToken"`
	BorrowingOrderHash     common.Hash    `bson:"borrowingOrderHash" json:"borrowingOrderHash"`
	InvestingOrderHash     common.Hash    `bson:"investingOrderHash" json:"investingOrderHash"`
	BorrowingRelayer       common.Address `bson:"borrowingRelayer" json:"borrowingRelayer"`
	InvestingRelayer       common.Address `bson:"investingRelayer" json:"investingRelayer"`
	Term                   uint64         `bson:"term" json:"term"`
	Interest               uint64         `bson:"interest" json:"interest"`
	CollateralPrice        *big.Int       `bson:"collateralPrice" json:"collateralPrice"`
	LiquidationPrice       *big.Int       `bson:"liquidationPrice" json:"liquidationPrice"`
	CollateralLockedAmount *big.Int       `bson:"collateralLockedAmount" json:"collateralLockedAmount"`
	AutoTopUp              bool           `bson:"autoTopUp" json:"autoTopUp"`
	LiquidationTime        uint64         `bson:"liquidationTime" json:"liquidationTime"`
	DepositRate            *big.Int       `bson:"depositRate" json:"depositRate"`
	LiquidationRate        *big.Int       `bson:"liquidationRate" json:"liquidationRate"`
	RecallRate             *big.Int       `bson:"recallRate" json:"recallRate"`
	Amount                 *big.Int       `bson:"amount" json:"amount"`
	BorrowingFee           *big.Int       `bson:"borrowingFee" json:"borrowingFee"`
	InvestingFee           *big.Int       `bson:"investingFee" json:"investingFee"`
	Status                 string         `bson:"status" json:"status"`
	TakerOrderSide         string         `bson:"takerOrderSide" json:"takerOrderSide"`
	TakerOrderType         string         `bson:"takerOrderType" json:"takerOrderType"`
	MakerOrderType         string         `bson:"makerOrderType" json:"makerOrderType"`
	TradeId                uint64         `bson:"tradeId" json:"tradeId"`
	Hash                   common.Hash    `bson:"hash" json:"hash"`
	TxHash                 common.Hash    `bson:"txHash" json:"txHash"`
	ExtraData              string         `bson:"extraData" json:"extraData"`
	CreatedAt              time.Time      `bson:"createdAt" json:"createdAt"`
	UpdatedAt              time.Time      `bson:"updatedAt" json:"updatedAt"`
}

type LendingTradeBSON struct {
	Borrower               string    `bson:"borrower" json:"borrower"`
	Investor               string    `bson:"investor" json:"investor"`
	LendingToken           string    `bson:"lendingToken" json:"lendingToken"`
	CollateralToken        string    `bson:"collateralToken" json:"collateralToken"`
	BorrowingOrderHash     string    `bson:"borrowingOrderHash" json:"borrowingOrderHash"`
	InvestingOrderHash     string    `bson:"investingOrderHash" json:"investingOrderHash"`
	BorrowingRelayer       string    `bson:"borrowingRelayer" json:"borrowingRelayer"`
	InvestingRelayer       string    `bson:"investingRelayer" json:"investingRelayer"`
	Term                   string    `bson:"term" json:"term"`
	Interest               string    `bson:"interest" json:"interest"`
	CollateralPrice        string    `bson:"collateralPrice" json:"collateralPrice"`
	LiquidationPrice       string    `bson:"liquidationPrice" json:"liquidationPrice"`
	LiquidationTime        string    `bson:"liquidationTime" json:"liquidationTime"`
	CollateralLockedAmount string    `bson:"collateralLockedAmount" json:"collateralLockedAmount"`
	AutoTopUp              bool      `bson:"autoTopUp" json:"autoTopUp"`
	DepositRate            string    `bson:"depositRate" json:"depositRate"`
	LiquidationRate        string    `bson:"liquidationRate" json:"liquidationRate"`
	RecallRate             string    `bson:"recallRate" json:"recallRate"`
	Amount                 string    `bson:"amount" json:"amount"`
	BorrowingFee           string    `bson:"borrowingFee" json:"borrowingFee"`
	InvestingFee           string    `bson:"investingFee" json:"investingFee"`
	Status                 string    `bson:"status" json:"status"`
	TakerOrderSide         string    `bson:"takerOrderSide" json:"takerOrderSide"`
	TakerOrderType         string    `bson:"takerOrderType" json:"takerOrderType"`
	MakerOrderType         string    `bson:"makerOrderType" json:"makerOrderType"`
	TradeId                string    `bson:"tradeId" json:"tradeId"`
	Hash                   string    `bson:"hash" json:"hash"`
	TxHash                 string    `bson:"txHash" json:"txHash"`
	ExtraData              string    `bson:"extraData" json:"extraData"`
	UpdatedAt              time.Time `bson:"updatedAt" json:"updatedAt"`
}

func (t *LendingTrade) GetBSON() (interface{}, error) {
	return bson.M{
		"$setOnInsert": bson.M{
			"createdAt": t.CreatedAt,
		},
		"$set": LendingTradeBSON{
			Borrower:               t.Borrower.Hex(),
			Investor:               t.Investor.Hex(),
			LendingToken:           t.LendingToken.Hex(),
			CollateralToken:        t.CollateralToken.Hex(),
			BorrowingOrderHash:     t.BorrowingOrderHash.Hex(),
			InvestingOrderHash:     t.InvestingOrderHash.Hex(),
			BorrowingRelayer:       t.BorrowingRelayer.Hex(),
			InvestingRelayer:       t.InvestingRelayer.Hex(),
			Term:                   strconv.FormatUint(t.Term, 10),
			Interest:               strconv.FormatUint(t.Interest, 10),
			CollateralPrice:        t.CollateralPrice.String(),
			LiquidationPrice:       t.LiquidationPrice.String(),
			LiquidationTime:        strconv.FormatUint(t.LiquidationTime, 10),
			CollateralLockedAmount: t.CollateralLockedAmount.String(),
			AutoTopUp:              t.AutoTopUp,
			DepositRate:            t.DepositRate.String(),
			LiquidationRate:        t.LiquidationRate.String(),
			RecallRate:             t.RecallRate.String(),
			Amount:                 t.Amount.String(),
			BorrowingFee:           t.BorrowingFee.String(),
			InvestingFee:           t.InvestingFee.String(),
			Status:                 t.Status,
			TakerOrderSide:         t.TakerOrderSide,
			TakerOrderType:         t.TakerOrderType,
			MakerOrderType:         t.MakerOrderType,
			TradeId:                strconv.FormatUint(t.TradeId, 10),
			Hash:                   t.Hash.Hex(),
			TxHash:                 t.TxHash.Hex(),
			ExtraData:              t.ExtraData,
			UpdatedAt:              t.UpdatedAt,
		},
	}, nil
}

func (t *LendingTrade) SetBSON(raw bson.Raw) error {
	decoded := new(LendingTradeBSON)

	err := raw.Unmarshal(decoded)
	if err != nil {
		return err
	}
	tradeId, err := strconv.ParseInt(decoded.TradeId, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse lendingItem.TradeId. Err: %v", err)
	}
	t.TradeId = uint64(tradeId)
	t.Borrower = common.HexToAddress(decoded.Borrower)
	t.Investor = common.HexToAddress(decoded.Investor)
	t.LendingToken = common.HexToAddress(decoded.LendingToken)
	t.CollateralToken = common.HexToAddress(decoded.CollateralToken)
	t.AutoTopUp = decoded.AutoTopUp
	t.BorrowingOrderHash = common.HexToHash(decoded.BorrowingOrderHash)
	t.InvestingOrderHash = common.HexToHash(decoded.InvestingOrderHash)
	t.BorrowingRelayer = common.HexToAddress(decoded.BorrowingRelayer)
	t.InvestingRelayer = common.HexToAddress(decoded.InvestingRelayer)
	term, err := strconv.ParseInt(decoded.Term, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse lendingItem.term. Err: %v", err)
	}
	t.Term = uint64(term)
	interest, err := strconv.ParseInt(decoded.Interest, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse lendingItem.interest. Err: %v", err)
	}
	t.Interest = uint64(interest)
	t.CollateralPrice = ToBigInt(decoded.CollateralPrice)
	t.LiquidationPrice = ToBigInt(decoded.LiquidationPrice)
	liquidationTime, err := strconv.ParseInt(decoded.LiquidationTime, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse lendingItem.LiquidationTime. Err: %v", err)
	}
	t.LiquidationTime = uint64(liquidationTime)
	t.CollateralLockedAmount = ToBigInt(decoded.CollateralLockedAmount)
	t.DepositRate = ToBigInt(decoded.DepositRate)
	t.LiquidationRate = ToBigInt(decoded.LiquidationRate)
	t.RecallRate = ToBigInt(decoded.RecallRate)
	t.Amount = tradingstate.ToBigInt(decoded.Amount)
	t.BorrowingFee = tradingstate.ToBigInt(decoded.BorrowingFee)
	t.InvestingFee = tradingstate.ToBigInt(decoded.InvestingFee)
	t.Status = decoded.Status
	t.TakerOrderSide = decoded.TakerOrderSide
	t.TakerOrderType = decoded.TakerOrderType
	t.MakerOrderType = decoded.MakerOrderType
	t.ExtraData = decoded.ExtraData
	t.Hash = common.HexToHash(decoded.Hash)
	t.TxHash = common.HexToHash(decoded.TxHash)
	t.UpdatedAt = decoded.UpdatedAt

	return nil
}

func (t *LendingTrade) ComputeHash() common.Hash {
	sha := sha3.NewKeccak256()
	sha.Write(t.InvestingOrderHash.Bytes())
	sha.Write(t.BorrowingOrderHash.Bytes())
	return common.BytesToHash(sha.Sum(nil))
}

package tradingstate

import (
	"math/big"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/crypto/sha3"
	"github.com/globalsign/mgo/bson"
)

const (
	TradeStatusSuccess = "SUCCESS"

	TradeTakerOrderHash = "takerOrderHash"
	TradeMakerOrderHash = "makerOrderHash"
	TradeTimestamp      = "timestamp"
	TradeQuantity       = "quantity"
	TradeMakerExchange  = "makerExAddr"
	TradeMaker          = "uAddr"
	TradeBaseToken      = "bToken"
	TradeQuoteToken     = "qToken"
	TradePrice          = "tradedPrice"
	MakerOrderType      = "makerOrderType"
	MakerFee            = "makerFee"
	TakerFee            = "takerFee"
)

type Trade struct {
	Taker          common.Address `json:"taker" bson:"taker"`
	Maker          common.Address `json:"maker" bson:"maker"`
	BaseToken      common.Address `json:"baseToken" bson:"baseToken"`
	QuoteToken     common.Address `json:"quoteToken" bson:"quoteToken"`
	MakerOrderHash common.Hash    `json:"makerOrderHash" bson:"makerOrderHash"`
	TakerOrderHash common.Hash    `json:"takerOrderHash" bson:"takerOrderHash"`
	MakerExchange  common.Address `json:"makerExchange" bson:"makerExchange"`
	TakerExchange  common.Address `json:"takerExchange" bson:"takerExchange"`
	Hash           common.Hash    `json:"hash" bson:"hash"`
	TxHash         common.Hash    `json:"txHash" bson:"txHash"`
	PricePoint     *big.Int       `json:"pricepoint" bson:"pricepoint"`
	Amount         *big.Int       `json:"amount" bson:"amount"`
	MakeFee        *big.Int       `json:"makeFee" bson:"makeFee"`
	TakeFee        *big.Int       `json:"takeFee" bson:"takeFee"`
	Status         string         `json:"status" bson:"status"`
	CreatedAt      time.Time      `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt" bson:"updatedAt"`
	TakerOrderSide string         `json:"takerOrderSide" bson:"takerOrderSide"`
	TakerOrderType string         `json:"takerOrderType" bson:"takerOrderType"`
	MakerOrderType string         `json:"makerOrderType" bson:"makerOrderType"`
}

type TradeBSON struct {
	Taker          string    `json:"taker" bson:"taker"`
	Maker          string    `json:"maker" bson:"maker"`
	BaseToken      string    `json:"baseToken" bson:"baseToken"`
	QuoteToken     string    `json:"quoteToken" bson:"quoteToken"`
	MakerOrderHash string    `json:"makerOrderHash" bson:"makerOrderHash"`
	TakerOrderHash string    `json:"takerOrderHash" bson:"takerOrderHash"`
	MakerExchange  string    `json:"makerExchange" bson:"makerExchange"`
	TakerExchange  string    `json:"takerExchange" bson:"takerExchange"`
	Hash           string    `json:"hash" bson:"hash"`
	TxHash         string    `json:"txHash" bson:"txHash"`
	Amount         string    `json:"amount" bson:"amount"`
	MakeFee        string    `json:"makeFee" bson:"makeFee"`
	TakeFee        string    `json:"takeFee" bson:"takeFee"`
	PricePoint     string    `json:"pricepoint" bson:"pricepoint"`
	Status         string    `json:"status" bson:"status"`
	CreatedAt      time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt" bson:"updatedAt"`
	TakerOrderSide string    `json:"takerOrderSide" bson:"takerOrderSide"`
	TakerOrderType string    `json:"takerOrderType" bson:"takerOrderType"`
	MakerOrderType string    `json:"makerOrderType" bson:"makerOrderType"`
}

func (t *Trade) GetBSON() (interface{}, error) {
	tr := TradeBSON{
		Maker:          t.Maker.Hex(),
		Taker:          t.Taker.Hex(),
		BaseToken:      t.BaseToken.Hex(),
		QuoteToken:     t.QuoteToken.Hex(),
		MakerOrderHash: t.MakerOrderHash.Hex(),
		TakerOrderHash: t.TakerOrderHash.Hex(),
		MakerExchange:  t.MakerExchange.Hex(),
		TakerExchange:  t.TakerExchange.Hex(),
		Hash:           t.Hash.Hex(),
		TxHash:         t.TxHash.Hex(),
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
		PricePoint:     t.PricePoint.String(),
		Status:         t.Status,
		Amount:         t.Amount.String(),
		MakeFee:        t.MakeFee.String(),
		TakeFee:        t.TakeFee.String(),
		TakerOrderSide: t.TakerOrderSide,
		TakerOrderType: t.TakerOrderType,
		MakerOrderType: t.MakerOrderType,
	}

	return tr, nil
}

func (t *Trade) SetBSON(raw bson.Raw) error {
	decoded := &TradeBSON{}

	err := raw.Unmarshal(decoded)
	if err != nil {
		return err
	}

	t.Taker = common.HexToAddress(decoded.Taker)
	t.Maker = common.HexToAddress(decoded.Maker)
	t.BaseToken = common.HexToAddress(decoded.BaseToken)
	t.QuoteToken = common.HexToAddress(decoded.QuoteToken)
	t.MakerOrderHash = common.HexToHash(decoded.MakerOrderHash)
	t.TakerOrderHash = common.HexToHash(decoded.TakerOrderHash)
	t.MakerExchange = common.HexToAddress(decoded.MakerExchange)
	t.TakerExchange = common.HexToAddress(decoded.TakerExchange)
	t.Hash = common.HexToHash(decoded.Hash)
	t.TxHash = common.HexToHash(decoded.TxHash)
	t.Status = decoded.Status
	t.Amount = ToBigInt(decoded.Amount)
	t.PricePoint = ToBigInt(decoded.PricePoint)

	t.MakeFee = ToBigInt(decoded.MakeFee)
	t.TakeFee = ToBigInt(decoded.TakeFee)

	t.CreatedAt = decoded.CreatedAt
	t.UpdatedAt = decoded.UpdatedAt
	t.TakerOrderSide = decoded.TakerOrderSide
	t.TakerOrderType = decoded.TakerOrderType
	t.MakerOrderType = decoded.MakerOrderType
	return nil
}

// ComputeHash returns hashes the trade
// The OrderHash, Amount, Taker and TradeNonce attributes must be
// set before attempting to compute the trade orderBookHash
func (t *Trade) ComputeHash() common.Hash {
	sha := sha3.NewKeccak256()
	sha.Write(t.MakerOrderHash.Bytes())
	sha.Write(t.TakerOrderHash.Bytes())
	return common.BytesToHash(sha.Sum(nil))
}

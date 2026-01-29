// Copyright 2019 XDC Network
// This file is part of the XDC library.

package XDCx

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/XDCx/tradingstate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// PublicXDCxAPI provides public XDCx APIs
type PublicXDCxAPI struct {
	xdcx *XDCx
}

// NewPublicXDCxAPI creates a new public XDCx API
func NewPublicXDCxAPI(xdcx *XDCx) *PublicXDCxAPI {
	return &PublicXDCxAPI{xdcx: xdcx}
}

// Version returns the XDCx version
func (api *PublicXDCxAPI) Version() string {
	return "1.0"
}

// OrderBookResult represents an order book API result
type OrderBookResult struct {
	BaseToken  common.Address `json:"baseToken"`
	QuoteToken common.Address `json:"quoteToken"`
	Bids       []OrderResult  `json:"bids"`
	Asks       []OrderResult  `json:"asks"`
}

// OrderResult represents an order API result
type OrderResult struct {
	ID              common.Hash    `json:"id"`
	UserAddress     common.Address `json:"userAddress"`
	ExchangeAddress common.Address `json:"exchangeAddress"`
	Price           *hexutil.Big   `json:"price"`
	Quantity        *hexutil.Big   `json:"quantity"`
	FilledQuantity  *hexutil.Big   `json:"filledQuantity"`
	Status          string         `json:"status"`
	Side            string         `json:"side"`
}

// TradeResult represents a trade API result
type TradeResult struct {
	Hash       common.Hash    `json:"hash"`
	Maker      common.Address `json:"maker"`
	Taker      common.Address `json:"taker"`
	BaseToken  common.Address `json:"baseToken"`
	QuoteToken common.Address `json:"quoteToken"`
	Price      *hexutil.Big   `json:"price"`
	Quantity   *hexutil.Big   `json:"quantity"`
	Amount     *hexutil.Big   `json:"amount"`
	MakerFee   *hexutil.Big   `json:"makerFee"`
	TakerFee   *hexutil.Big   `json:"takerFee"`
}

// GetOrderBook returns the order book for a trading pair
func (api *PublicXDCxAPI) GetOrderBook(ctx context.Context, baseToken, quoteToken common.Address) (*OrderBookResult, error) {
	if !api.xdcx.IsRunning() {
		return nil, ErrXDCxServiceNotRunning
	}

	// Get current trading state
	tradingState, err := tradingstate.New(common.Hash{}, api.xdcx.StateCache())
	if err != nil {
		return nil, err
	}

	ob, err := api.xdcx.GetOrderBook(baseToken, quoteToken, tradingState)
	if err != nil {
		return nil, err
	}

	result := &OrderBookResult{
		BaseToken:  baseToken,
		QuoteToken: quoteToken,
		Bids:       make([]OrderResult, 0),
		Asks:       make([]OrderResult, 0),
	}

	for _, bid := range ob.Bids {
		result.Bids = append(result.Bids, orderToResult(bid))
	}

	for _, ask := range ob.Asks {
		result.Asks = append(result.Asks, orderToResult(ask))
	}

	return result, nil
}

// GetBestBid returns the best bid price
func (api *PublicXDCxAPI) GetBestBid(ctx context.Context, baseToken, quoteToken common.Address) (*hexutil.Big, error) {
	if !api.xdcx.IsRunning() {
		return nil, ErrXDCxServiceNotRunning
	}

	tradingState, err := tradingstate.New(common.Hash{}, api.xdcx.StateCache())
	if err != nil {
		return nil, err
	}

	price, err := api.xdcx.GetBestBid(baseToken, quoteToken, tradingState)
	if err != nil {
		return nil, err
	}

	return (*hexutil.Big)(price), nil
}

// GetBestAsk returns the best ask price
func (api *PublicXDCxAPI) GetBestAsk(ctx context.Context, baseToken, quoteToken common.Address) (*hexutil.Big, error) {
	if !api.xdcx.IsRunning() {
		return nil, ErrXDCxServiceNotRunning
	}

	tradingState, err := tradingstate.New(common.Hash{}, api.xdcx.StateCache())
	if err != nil {
		return nil, err
	}

	price, err := api.xdcx.GetBestAsk(baseToken, quoteToken, tradingState)
	if err != nil {
		return nil, err
	}

	return (*hexutil.Big)(price), nil
}

// GetOrder returns an order by ID
func (api *PublicXDCxAPI) GetOrder(ctx context.Context, orderID common.Hash) (*OrderResult, error) {
	if !api.xdcx.IsRunning() {
		return nil, ErrXDCxServiceNotRunning
	}

	tradingState, err := tradingstate.New(common.Hash{}, api.xdcx.StateCache())
	if err != nil {
		return nil, err
	}

	order := tradingState.GetOrder(orderID)
	if order == nil {
		return nil, ErrOrderNotFound
	}

	result := orderStateToResult(order)
	return &result, nil
}

// orderToResult converts an order to API result
func orderToResult(order *Order) OrderResult {
	sideStr := "buy"
	if order.Side == Sell {
		sideStr = "sell"
	}

	statusStr := "new"
	switch order.Status {
	case OrderStatusPartialFilled:
		statusStr = "partial"
	case OrderStatusFilled:
		statusStr = "filled"
	case OrderStatusCancelled:
		statusStr = "cancelled"
	case OrderStatusRejected:
		statusStr = "rejected"
	}

	return OrderResult{
		ID:              order.ID,
		UserAddress:     order.UserAddress,
		ExchangeAddress: order.ExchangeAddress,
		Price:           (*hexutil.Big)(order.Price),
		Quantity:        (*hexutil.Big)(order.Quantity),
		FilledQuantity:  (*hexutil.Big)(order.FilledQuantity),
		Status:          statusStr,
		Side:            sideStr,
	}
}

// orderStateToResult converts an order state to API result
func orderStateToResult(order *tradingstate.OrderState) OrderResult {
	sideStr := "buy"
	if order.Side == 1 {
		sideStr = "sell"
	}

	statusStr := "new"
	switch order.Status {
	case 1:
		statusStr = "partial"
	case 2:
		statusStr = "filled"
	case 3:
		statusStr = "cancelled"
	case 4:
		statusStr = "rejected"
	}

	return OrderResult{
		ID:              order.ID,
		UserAddress:     order.UserAddress,
		ExchangeAddress: order.ExchangeAddress,
		Price:           (*hexutil.Big)(order.Price),
		Quantity:        (*hexutil.Big)(order.Quantity),
		FilledQuantity:  (*hexutil.Big)(order.FilledQuantity),
		Status:          statusStr,
		Side:            sideStr,
	}
}

// PrivateXDCxAPI provides private XDCx APIs
type PrivateXDCxAPI struct {
	xdcx *XDCx
}

// NewPrivateXDCxAPI creates a new private XDCx API
func NewPrivateXDCxAPI(xdcx *XDCx) *PrivateXDCxAPI {
	return &PrivateXDCxAPI{xdcx: xdcx}
}

// SendOrder sends a new order
func (api *PrivateXDCxAPI) SendOrder(ctx context.Context, args SendOrderArgs) (common.Hash, error) {
	if !api.xdcx.IsRunning() {
		return common.Hash{}, ErrXDCxServiceNotRunning
	}

	// Validate and convert args to order
	order, err := args.ToOrder()
	if err != nil {
		return common.Hash{}, err
	}

	// Process order (stub - needs full implementation)
	return order.ID, nil
}

// CancelOrder cancels an order
func (api *PrivateXDCxAPI) CancelOrder(ctx context.Context, orderID common.Hash) error {
	if !api.xdcx.IsRunning() {
		return ErrXDCxServiceNotRunning
	}

	// Cancel order (stub - needs full implementation)
	return nil
}

// SendOrderArgs represents the arguments for sending an order
type SendOrderArgs struct {
	BaseToken       common.Address `json:"baseToken"`
	QuoteToken      common.Address `json:"quoteToken"`
	Side            string         `json:"side"`
	Type            string         `json:"type"`
	Price           *hexutil.Big   `json:"price"`
	Quantity        *hexutil.Big   `json:"quantity"`
	ExchangeAddress common.Address `json:"exchangeAddress"`
	Nonce           hexutil.Uint64 `json:"nonce"`
	Signature       hexutil.Bytes  `json:"signature"`
}

// ToOrder converts args to an order
func (args *SendOrderArgs) ToOrder() (*Order, error) {
	var side OrderSide
	switch args.Side {
	case "buy":
		side = Buy
	case "sell":
		side = Sell
	default:
		return nil, errors.New("invalid order side")
	}

	var orderType OrderType
	switch args.Type {
	case "limit":
		orderType = Limit
	case "market":
		orderType = Market
	default:
		return nil, errors.New("invalid order type")
	}

	order := NewOrder(
		common.Address{}, // User address from signature recovery
		args.BaseToken,
		args.QuoteToken,
		side,
		orderType,
		(*big.Int)(args.Price),
		(*big.Int)(args.Quantity),
		uint64(args.Nonce),
		args.ExchangeAddress,
	)
	order.Signature = args.Signature

	return order, nil
}

// APIs returns the collection of XDCx APIs
func (x *XDCx) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "xdcx",
			Service:   NewPublicXDCxAPI(x),
		},
		{
			Namespace: "xdcx",
			Service:   NewPrivateXDCxAPI(x),
		},
	}
}

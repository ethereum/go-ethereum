// Copyright 2019 XDC Network
// This file is part of the XDC library.

package XDCxlending

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/XDCxlending/lendingstate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// PublicXDCxLendingAPI provides public XDCxLending APIs
type PublicXDCxLendingAPI struct {
	lending *XDCxLending
}

// NewPublicXDCxLendingAPI creates a new public XDCxLending API
func NewPublicXDCxLendingAPI(lending *XDCxLending) *PublicXDCxLendingAPI {
	return &PublicXDCxLendingAPI{lending: lending}
}

// Version returns the XDCxLending version
func (api *PublicXDCxLendingAPI) Version() string {
	return "1.0"
}

// LendingOrderBookResult represents a lending order book API result
type LendingOrderBookResult struct {
	LendingToken    common.Address       `json:"lendingToken"`
	CollateralToken common.Address       `json:"collateralToken"`
	Term            uint64               `json:"term"`
	Borrows         []LendingOrderResult `json:"borrows"`
	Lends           []LendingOrderResult `json:"lends"`
}

// LendingOrderResult represents a lending order API result
type LendingOrderResult struct {
	ID             common.Hash    `json:"id"`
	UserAddress    common.Address `json:"userAddress"`
	RelayerAddress common.Address `json:"relayerAddress"`
	InterestRate   *hexutil.Big   `json:"interestRate"`
	Quantity       *hexutil.Big   `json:"quantity"`
	FilledQuantity *hexutil.Big   `json:"filledQuantity"`
	Status         string         `json:"status"`
	Side           string         `json:"side"`
}

// LoanResult represents a loan API result
type LoanResult struct {
	ID               common.Hash    `json:"id"`
	Borrower         common.Address `json:"borrower"`
	Lender           common.Address `json:"lender"`
	LendingToken     common.Address `json:"lendingToken"`
	CollateralToken  common.Address `json:"collateralToken"`
	Principal        *hexutil.Big   `json:"principal"`
	CollateralAmount *hexutil.Big   `json:"collateralAmount"`
	InterestRate     *hexutil.Big   `json:"interestRate"`
	Term             uint64         `json:"term"`
	StartTime        uint64         `json:"startTime"`
	ExpiryTime       uint64         `json:"expiryTime"`
	Status           string         `json:"status"`
}

// GetLendingOrderBook returns the lending order book
func (api *PublicXDCxLendingAPI) GetLendingOrderBook(ctx context.Context, lendingToken, collateralToken common.Address, term uint64) (*LendingOrderBookResult, error) {
	if !api.lending.IsRunning() {
		return nil, ErrLendingServiceNotRunning
	}

	// Get current lending state
	lendingState, err := lendingstate.New(common.Hash{}, api.lending.StateCache())
	if err != nil {
		return nil, err
	}

	pairKey := GetLendingPairKey(lendingToken, collateralToken, term)
	ob := lendingState.GetLendingOrderBook(pairKey)
	if ob == nil {
		return nil, errors.New("order book not found")
	}

	result := &LendingOrderBookResult{
		LendingToken:    lendingToken,
		CollateralToken: collateralToken,
		Term:            term,
		Borrows:         make([]LendingOrderResult, 0),
		Lends:           make([]LendingOrderResult, 0),
	}

	for _, borrow := range ob.Borrows {
		result.Borrows = append(result.Borrows, orderStateToResult(borrow))
	}

	for _, lend := range ob.Lends {
		result.Lends = append(result.Lends, orderStateToResult(lend))
	}

	return result, nil
}

// GetLoan returns a loan by ID
func (api *PublicXDCxLendingAPI) GetLoan(ctx context.Context, loanID common.Hash) (*LoanResult, error) {
	if !api.lending.IsRunning() {
		return nil, ErrLendingServiceNotRunning
	}

	lendingState, err := lendingstate.New(common.Hash{}, api.lending.StateCache())
	if err != nil {
		return nil, err
	}

	loan := lendingState.GetLoan(loanID)
	if loan == nil {
		return nil, ErrLoanNotFound
	}

	return loanStateToResult(loan), nil
}

// GetLoansByBorrower returns all loans for a borrower
func (api *PublicXDCxLendingAPI) GetLoansByBorrower(ctx context.Context, borrower common.Address) ([]*LoanResult, error) {
	if !api.lending.IsRunning() {
		return nil, ErrLendingServiceNotRunning
	}

	lendingState, err := lendingstate.New(common.Hash{}, api.lending.StateCache())
	if err != nil {
		return nil, err
	}

	allLoans := lendingState.GetAllLoans()
	results := make([]*LoanResult, 0)

	for _, loan := range allLoans {
		if loan.BorrowerAddress == borrower {
			results = append(results, loanStateToResult(loan))
		}
	}

	return results, nil
}

// orderStateToResult converts a lending order state to API result
func orderStateToResult(order *lendingstate.LendingOrderState) LendingOrderResult {
	sideStr := "borrow"
	if order.Side == 1 {
		sideStr = "lend"
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

	return LendingOrderResult{
		ID:             order.ID,
		UserAddress:    order.UserAddress,
		RelayerAddress: order.RelayerAddress,
		InterestRate:   (*hexutil.Big)(order.InterestRate),
		Quantity:       (*hexutil.Big)(order.Quantity),
		FilledQuantity: (*hexutil.Big)(order.FilledQuantity),
		Status:         statusStr,
		Side:           sideStr,
	}
}

// loanStateToResult converts a loan state to API result
func loanStateToResult(loan *lendingstate.LoanState) *LoanResult {
	statusStr := "active"
	switch loan.Status {
	case 1:
		statusStr = "repaid"
	case 2:
		statusStr = "liquidated"
	case 3:
		statusStr = "defaulted"
	}

	return &LoanResult{
		ID:               loan.ID,
		Borrower:         loan.BorrowerAddress,
		Lender:           loan.LenderAddress,
		LendingToken:     loan.LendingToken,
		CollateralToken:  loan.CollateralToken,
		Principal:        (*hexutil.Big)(loan.Principal),
		CollateralAmount: (*hexutil.Big)(loan.CollateralAmount),
		InterestRate:     (*hexutil.Big)(loan.InterestRate),
		Term:             loan.Term,
		StartTime:        loan.StartTime,
		ExpiryTime:       loan.ExpiryTime,
		Status:           statusStr,
	}
}

// PrivateXDCxLendingAPI provides private XDCxLending APIs
type PrivateXDCxLendingAPI struct {
	lending *XDCxLending
}

// NewPrivateXDCxLendingAPI creates a new private XDCxLending API
func NewPrivateXDCxLendingAPI(lending *XDCxLending) *PrivateXDCxLendingAPI {
	return &PrivateXDCxLendingAPI{lending: lending}
}

// SendLendingOrder sends a new lending order
func (api *PrivateXDCxLendingAPI) SendLendingOrder(ctx context.Context, args SendLendingOrderArgs) (common.Hash, error) {
	if !api.lending.IsRunning() {
		return common.Hash{}, ErrLendingServiceNotRunning
	}

	// Convert args to order
	order, err := args.ToOrder()
	if err != nil {
		return common.Hash{}, err
	}

	return order.ID, nil
}

// SendLendingOrderArgs represents the arguments for sending a lending order
type SendLendingOrderArgs struct {
	LendingToken    common.Address `json:"lendingToken"`
	CollateralToken common.Address `json:"collateralToken"`
	Side            string         `json:"side"`
	Type            string         `json:"type"`
	InterestRate    *hexutil.Big   `json:"interestRate"`
	Term            hexutil.Uint64 `json:"term"`
	Quantity        *hexutil.Big   `json:"quantity"`
	RelayerAddress  common.Address `json:"relayerAddress"`
	Nonce           hexutil.Uint64 `json:"nonce"`
	Signature       hexutil.Bytes  `json:"signature"`
}

// ToOrder converts args to a lending order
func (args *SendLendingOrderArgs) ToOrder() (*LendingOrder, error) {
	var side OrderSide
	switch args.Side {
	case "borrow":
		side = Borrow
	case "lend":
		side = Lend
	default:
		return nil, errors.New("invalid order side")
	}

	var orderType OrderType
	switch args.Type {
	case "limit":
		orderType = LimitOrder
	case "market":
		orderType = MarketOrder
	default:
		return nil, errors.New("invalid order type")
	}

	order := NewLendingOrder(
		common.Address{},
		args.LendingToken,
		args.CollateralToken,
		side,
		orderType,
		(*big.Int)(args.InterestRate),
		uint64(args.Term),
		(*big.Int)(args.Quantity),
		uint64(args.Nonce),
		args.RelayerAddress,
	)
	order.Signature = args.Signature

	return order, nil
}

// APIs returns the collection of XDCxLending APIs
func (l *XDCxLending) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "xdcxlending",
			Service:   NewPublicXDCxLendingAPI(l),
		},
		{
			Namespace: "xdcxlending",
			Service:   NewPrivateXDCxLendingAPI(l),
		},
	}
}
